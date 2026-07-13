# E2E Validation Guide

## 目的

本ドキュメントは、実機CANとMQブローカーを使った疎通検証手順を定義します。

## 前提

- EDGEPLANT T1 で `can0` `can1` が有効
- `gpsd` 稼働中 (`127.0.0.1:2947`)
- `config/can0.dbc` と `config/can1.dbc` が配置済み
- テスト端末からMQへ Publish/Subscribe 可能

## Docker/vcan 検証

- `./scripts/e2e_vcan.sh` で compose ベースの E2E を実行
- `logs/app.log`: テレメトリアプリの実行ログ
- `logs/server.log`: compose サービス全体ログ
- `logs/e2e_uplink.json`: 取得した上り JSON
- `logs/e2e_downlink_can.log`: 取得した下り CAN フレーム
- `docs/test-results-mqtt.md` と `docs/test-results-rabbitmq.md`: 実行結果から自動生成される試験記録

この検証では `scripts/mock_gpsd.py` を使って GPSD を模擬します。

`scripts/e2e_vcan.sh` は終了時に `scripts/render_e2e_result_doc.sh` を呼び出し、現在のログと CI 実行メタデータから broker 別の試験結果ページを再生成します。self-hosted GitHub Actions では同じ Markdown を job summary と artifact にも保存します。

self-hosted runner の登録手順と main ブランチ E2E 再実行手順は `public-operations.md` を参照してください。

通常の CI では `mkdocs build --strict` を実行して、生成された試験結果ページを含むドキュメント全体が常に描画可能であることを検証します。GitHub Pages デプロイは `main` ブランチ push または手動実行時に別 workflow で実施し、公開物は GitHub Actions artifact から配信します。`gh-pages` ブランチを成果物置き場として運用しない方針です。

## ケース1: MQTT上りテレメトリ

1. 受信端末で `/tx/{DEVICE_ID}/state` を Subscribe
2. テレメトリアプリを MQTT モードで起動
3. 100ms 周期で JSON が受信されることを確認
4. `vehicle.can0.speed` と `location.latitude` が更新されることを確認

## ケース2: MQTT下りコマンド

1. `/rx/{DEVICE_ID}/ctrl` に以下を Publish

```json
{
  "Timestamp": "2024-06-10T15:30:30+09:00",
  "bus_id": "can0",
  "frame_id": "0x2",
  "signals": {
    "ControlCommand": true,
    "ControlValue": -10.0
  }
}
```

2. `candump can0` で frame id `0x2` が送出されることを確認
3. 不正 payload 送信時にアプリが落ちず warning ログで継続することを確認

## ケース3: RabbitMQ上りテレメトリ

1. `broker.type: rabbitmq` で起動
2. exchange を `topic` バインドし routing key `/tx/{DEVICE_ID}/state` を購読
3. 上り JSON が受信できることを確認

## ケース4: RabbitMQ下りコマンド

1. exchange に routing key `/rx/{DEVICE_ID}/ctrl` でコマンド JSON を Publish
2. `candump can0` で frame 送信を確認
3. `bus_id` を `can1` に変更して `can1` へのルーティングを確認

## ケース5: broker切替回帰

1. MQTTモードで起動して publish/subscribe 確認
2. 再起動して RabbitMQモードで同じ確認
3. どちらも SIGTERM で graceful shutdown できることを確認

## 記録項目

- 実施日
- 実施者
- `DEVICE_ID`
- 利用ブローカー
- ケースごとの Pass/Fail
- ログ添付先
