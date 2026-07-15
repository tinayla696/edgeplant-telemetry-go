# 要求仕様書

## 機能要件

- REQ-01: SocketCAN（`can0`/`can1`）から並行受信
- REQ-02: DBCで物理値デコード
- REQ-03: GPSDから位置・時刻取得
- REQ-04: 自動再接続付きでMQへPublish
- REQ-05: MQ Subscribeで制御JSONを非同期受信（接続瞬断時は再試行）
- REQ-06: 受信JSONをDBCエンコードしCAN送信

## 非機能要件

- 長期稼働時の安定性（リーク抑制、再接続）
- SIGTERM検知による安全停止
- Docs as Codeの維持
