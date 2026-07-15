# Gamepad MQTT->CAN サンプルフロー

## 目的

USB接続した Gamepad の入力を Web 経由で受け取り、MQTT 下りコマンドへ変換してテレメトリアプリに渡し、最終的に CAN フレーム送信まで確認するためのサンプルです。

## 全体フロー

```mermaid
graph LR
  GP[USB Gamepad] --> BR[Browser Gamepad API]
  BR -->|WebSocket /ws| GW[gamepadweb]
  GW -->|MQTT Publish /rx/{device}/ctrl| MB[MQTT Broker]
  MB -->|Subscribe| TM[telemetry]
  TM -->|ParseControlCommand + DBC Encode| CAN[SocketCAN can0/can1]
```

`gamepadweb` は入力値を `GamepadMsg` (`0x210`) に集約して publish します。

すべてのボタン・スティック入力を CAN 化する専用定義として、以下を追加しています。

- `config/gamepad_all_inputs.dbc`
- `config/gamepad_mapping_all_inputs.yaml`
- `config/config.vcan.gamepad_all.yaml`

## 追加した実装

- `src/cmd/gamepadweb/main.go`
- `src/internal/gamepad/mapper.go`
- `src/internal/gamepad/mapper_test.go`

## 起動手順

1. MQTT ブローカーと telemetry を起動

```bash
docker compose up -d mqtt-broker gpsd-mock

# telemetry を full-input DBC で起動する場合
cd src
go run ./cmd/telemetry -device vcan-e2e -conf ../config/config.vcan.gamepad_all.yaml -logPath ../logs
```

2. Gamepad Web サーバーを起動

```bash
cd src
go run ./cmd/gamepadweb \
  -listen :8088 \
  -mqtt 127.0.0.1:1883 \
  -device vcan-e2e \
  -topic-prefix ctrl \
  -bus can0 \
  -map-config ../config/gamepad_mapping_all_inputs.yaml
```

3. ブラウザでアクセス

- `http://127.0.0.1:8088`
- USB接続した Gamepad を認識させ、`送信開始` を押下

### Full Input 用 YAML設定例

`config/gamepad_mapping_all_inputs.yaml`

```yaml
all_inputs:
  gamepad_msg_frame_id: "0x210"
```

`gamepad_all_inputs.dbc` 側では下記 signal 名を受け取ります。

- GamepadMsg (`0x210`)
- Stick: `StickLX`,`StickLY`,`StickRX`,`StickRY` (`-1.0..1.0` を `-128..127` に変換)
- Button Pressed: `Btn00_P` ... `Btn15_P`

`gamepadweb` の送信周期は 100ms（10Hz）です。

## テストフロー

### フローA: ブラウザ実機テスト

1. `candump` で CAN 受信を監視

  `candump` は telemetry と同じ実行空間で実行してください。

  - telemetry がホスト実行: ホストで `candump can0`
  - telemetry がコンテナ実行: コンテナ内で `candump can0`

```bash
candump can0
```

2. ブラウザで A ボタンと左スティックを操作
3. `candump` に `210` フレームが継続出力されることを確認

### フローA-2: MQTT `#` SubscribeでPublish確認

`candump` に何も出ない場合は、まず MQTT へ publish されているかを確認します。

1. 別ターミナルで wildcard subscribe

```bash
mosquitto_sub -h 127.0.0.1 -p 1883 -t '#'
```

2. ブラウザでコントローラーを操作して `送信開始`
3. `/rx/vcan-e2e/ctrl` に JSON が流れることを確認

確認例:

```json
{"Timestamp":"...","bus_id":"can0","frame_id":"0x210","signals":{"StickLX":32,"StickLY":-128,"StickRX":64,"StickRY":127,"Btn00_P":true}}
```

4. 出ているのに `candump` が無出力なら、次を確認

- `frame_id` が DBC に存在するか
- YAML の signal 名が DBC の signal 名と一致しているか
- `bus_id` が対象インターフェース (`can0` / `can1`) と一致しているか

### フローB: API 擬似入力テスト

`gamepadweb` は `POST /api/gamepad` でも入力を受け取れるため、Gamepad実機なしで同じ変換を検証できます。

```bash
curl -sS -X POST http://127.0.0.1:8088/api/gamepad \
  -H 'Content-Type: application/json' \
  -d '{"axes":[0,-0.75],"buttons":[true]}'
```

期待値:

- MQTT payload の `frame_id = 0x210`
- telemetry が `frame_id=0x210` を CAN 送信

## ワンショット検証スクリプト

ブラウザ無しで疎通確認する場合は以下を利用します。

```bash
./scripts/gamepad_mqtt_can_test.sh
```

このスクリプトは次を自動で実行します。

1. 必要サービス起動 (`mqtt-broker`, `gpsd-mock`, `telemetry`)
2. `gamepadweb` 起動
3. `POST /api/gamepad` で擬似入力送信
4. `candump` で `0x210` フレーム受信確認

## 注意事項

- 実機テストではブラウザの Gamepad API が有効な環境（通常は HTTPS または localhost）を使用してください。
- `frame_id` と `signals` 名は、対象 DBC の定義に合わせて調整してください。
- `buttons` は `bool` / `number` / `{pressed,value}` のいずれでも受け付けます（互換維持）。
