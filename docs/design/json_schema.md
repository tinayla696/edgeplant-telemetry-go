# JSONスキーマ仕様

## 下り（コマンド）

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

- `frame_id`: 10進/16進（`0x...`）を受理
- `signals`: `bool` と `number` を動的変換

## 上り（状態）

```json
{
  "Timestamp": "2024-06-10T15:30:30+09:00",
  "vehicle": {
    "can0": {"speed": 10.0}
  },
  "location": {
    "latitude": 35.6812,
    "longitude": 139.7671,
    "timestamp": "2024-06-10T15:30:30+09:00"
  }
}
```

- `vehicle`: バスごとに最新ラッチ値を保持
- `location`: GPSD由来の最新値
