# Telemetry Golang Project

ARM64 LinuxOS（NVIDIA L4T）で動作する高信頼性双方向テレメトリアプリケーションです。

![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Platform](https://img.shields.io/badge/Platform-EDGEPLANT%20T1-3B7EA1?style=for-the-badge)
![Protocol](https://img.shields.io/badge/Protocol-MQTT%20%2F%20RabbitMQ-3C525A?style=for-the-badge)
![CAN](https://img.shields.io/badge/CAN-SocketCAN-1D70B8?style=for-the-badge)

## 特徴

- 上り: SocketCAN + GPSD を統合して状態 JSON を配信
- 下り: JSON コマンドを DBC エンコードして CAN 送信
- `mqtt` / `rabbitmq` を設定で切替可能
- `vcan` を使った E2E 検証に対応

## ドキュメントの読み方

- 戦略と要件: ロードマップ、システム要件、要求仕様
- 基本設計: データフロー、JSONスキーマ、main.go 配線
- 開発運用: クイックスタート、開発規約、E2E検証