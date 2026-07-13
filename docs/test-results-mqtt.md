# MQTT E2E Test Result

- Executed at: 2026-07-13T05:10:43Z
- Mode: mqtt
- Device ID: vcan-e2e
- Result: Pass
- Command: `./scripts/e2e_vcan.sh mqtt`
- Runner: LAPTOP-CNOSSPAI
- Branch: `main`
- Commit: `8e4ec401e404`
- Workflow: local

## Validation Scope

1. Verify uplink JSON against CAN input and GPS mock output
2. Verify downlink JSON command against emitted CAN frame
3. Preserve app and compose logs for post-run inspection

## Observed Uplink JSON

```json
{"Timestamp":"2026-07-13T05:10:41Z","vehicle":{"can0":{"brake":2,"left_turn":0,"right_turn":1,"speed":2.56,"steering_deg":102.7}},"location":{"latitude":35.6812,"longitude":139.7671,"altitude":24.5,"speed":10,"timestamp":"2024-06-10T15:30:30Z"}}

```

## Observed Downlink CAN

```text
  can0  002   [8]  01 9C FF 00 00 00 00 00

```

## Saved Logs

- `logs/app.mqtt.log`
- `logs/server.mqtt.uplink.log`
- `logs/server.mqtt.downlink.log`
- `logs/e2e_mqtt_uplink.json`
- `logs/e2e_mqtt_downlink_can.log`

## Notes

The scenario completed successfully and the generated logs match the expected CAN/GPS round-trip flow.
