# システム要件

## ハードウェア

- EDGEPLANT T1 (ET1-128NJA)
- CAN-USB Interface (EP1-CH02A)
- GNSS: u-blox NEO-M8U

## ソフトウェア

- Linux for Tegra (L4T)
- gpsd (`127.0.0.1:2947`)
- SocketCAN (`can0`, `can1`)
- Go 1.25+

## 通信要件

- MQTT または RabbitMQ
- トピック命名規則: `/${direction}/${DEVICE_ID}/${prefix}`
