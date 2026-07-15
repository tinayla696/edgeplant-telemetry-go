# 制御オーケストレーション

## チャネル構成

- `canRxCh`: CAN受信
- `gpsRxCh`: GPS受信
- `canTxCh`: 下り制御コマンド

## 主要ロジック

1. CAN/GPS受信 goroutine を起動
2. broker subscribe で制御コマンド受信（接続瞬断時は再試行）
3. JSONを内部構造体へ変換して `canTxCh` に投入
4. Publisher が定周期でスナップショットを送信

## 実コード同期（snippets）

```go
--8<-- "src/cmd/telemetry/main.go"
```
