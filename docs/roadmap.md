# ロードマップ戦略

## 目標

- Phase 01: EDGEPLANT T1 (ARM64) で安定運用
- 最終: Linuxベースでアーキテクチャ非依存（x86_64 / ARM64）

## ポータビリティ方針

1. デバイスパス外部設定化
- `/dev/ttyTHS1` などをコード固定しない
- `config/config.yaml` または環境変数で注入

2. ブローカー抽象化
- `broker.type` で `mqtt` / `rabbitmq` を切替
- 送受信インターフェースは `internal/mq` で統一

3. マルチアーキテクチャビルド
- CIで `linux/amd64` と `linux/arm64` をクロスビルド
- 実機差分は設定ファイル側で吸収
