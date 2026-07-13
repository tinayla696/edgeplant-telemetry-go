# クイックスタート

## 1. 単体テスト

```bash
cd src
go test ./...
```

## 2. クロスビルド

```bash
cd src
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o telemetry-arm64 ./cmd/telemetry
```

## 3. 実機準備（SocketCAN）

```bash
sudo ip link set can0 up type can bitrate 500000
sudo ip link set can1 up type can bitrate 500000
```

## 4. vcan E2E（Docker Compose）

```bash
./scripts/e2e_vcan.sh
```
