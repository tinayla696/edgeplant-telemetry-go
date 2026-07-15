# Gamepad MQTT-CAN 実行手順 (Runbook)

## 目的

- `gamepadweb` で入力を `/rx/vcan-e2e/ctrl` へ publish
- `telemetry` が受信JSONを DBC エンコードして `can0` へ送信
- `candump can0` で `0x210 (GamepadMsg)` を確認

## 重要ルール（最重要）

`candump` は **telemetry と同じ実行空間** で実行すること。

- telemetry がホストで動作中: `candump` もホストで実行
- telemetry がコンテナで動作中: `candump` も同じコンテナ内で実行

混在させると、MQTTに来ていても `candump` で見えません。

## 推奨パターンA（ホスト統一）

### 1. 事前クリーン

```bash
cd /home/azpeng/tinayla696/edgeplant-telemetry-go
pkill -f 'go run ./cmd/gamepadweb' || true
pkill -f 'go run ./cmd/telemetry' || true
docker compose stop telemetry rabbitmq gpsd-mock mqtt-broker 2>/dev/null || true
```

### 2. 必要サービス起動（broker + gpsd mock）

```bash
cd /home/azpeng/tinayla696/edgeplant-telemetry-go
docker compose up -d mqtt-broker gpsd-mock
```

### 3. telemetry 起動（ターミナルA）

```bash
cd /home/azpeng/tinayla696/edgeplant-telemetry-go/src
go run ./cmd/telemetry -device vcan-e2e -conf ../config/config.vcan.gamepad_all.yaml -logPath ../logs
```

期待ログ:

- `Parsed DBC file: ../config/gamepad_all_inputs.dbc`
- `Starting to listen on SocketCAN interface: can0`

### 4. gamepadweb 起動（ターミナルB）

```bash
cd /home/azpeng/tinayla696/edgeplant-telemetry-go/src
go run ./cmd/gamepadweb \
  -listen :8088 \
  -mqtt 127.0.0.1:1883 \
  -device vcan-e2e \
  -topic-prefix ctrl \
  -bus can0 \
  -map-config ../config/gamepad_mapping_all_inputs.yaml
```

期待ログ:

- `gamepad-web listening on :8088`
- `publishing MQTT topic: /rx/vcan-e2e/ctrl`

### 5. CAN監視（ターミナルC）

```bash
candump can0
```

### 6. テスト入力（ターミナルD）

```bash
curl -sS -X POST http://127.0.0.1:8088/api/gamepad \
  -H 'Content-Type: application/json' \
  -d '{"axes":[0.25,-1.0,0.5,1.0],"buttons":[true,false,true,false,true,false,true,false,true,false,true,false,true,false,true,false]}'
```

期待結果:

- APIレスポンスに `"frame_id":"0x210"`
- `candump can0` に `210` フレーム

例:

```text
can0  210   [8]  01 FE 00 FF 00 00 00 00
```

## 参考パターンB（コンテナ統一）

telemetry をコンテナで動かす場合のみ:

```bash
docker compose up -d telemetry mqtt-broker gpsd-mock
docker compose exec -T telemetry bash -lc 'candump can0'
```

## MQTT経路だけ確認したい場合

```bash
mosquitto_sub -h 127.0.0.1 -p 1883 -t '#'
```

期待結果:

- `/rx/vcan-e2e/ctrl` に publish JSON が流れる

## トラブルシュート

### `gamepadweb` が Exit 1

多くは `:8088` 競合。

```bash
ss -ltnp | rg ':8088' || true
pkill -f 'go run ./cmd/gamepadweb' || true
```

### `telemetry` が Exit 1

ログで `Failed to subscribe control topics` を確認。
MQTT接続断の瞬間に起きることがあるため、再起動して再確認。

### `candump can0` に何も出ない

1. telemetry が起動中か確認
2. `bus_id` が `can0` か確認
3. `frame_id` が DBC 定義済み (`0x210`) か確認
4. telemetry と candump の実行空間が一致しているか確認

## 停止

```bash
pkill -f 'go run ./cmd/gamepadweb' || true
pkill -f 'go run ./cmd/telemetry' || true
```
