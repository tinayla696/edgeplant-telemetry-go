package mq

import "testing"

func TestParseControlCommand_HexAndBool(t *testing.T) {
	payload := []byte(`{
		"Timestamp":"2024-06-10T15:30:30+09:00",
		"bus_id":"can0",
		"frame_id":"0x123",
		"signals":{"ControlCommand":true,"ControlValue":-10.0}
	}`)

	msg, err := ParseControlCommand(payload)
	if err != nil {
		t.Fatalf("ParseControlCommand failed: %v", err)
	}
	if msg.BusID != "can0" {
		t.Fatalf("unexpected bus id: %s", msg.BusID)
	}
	if msg.FrameID != 0x123 {
		t.Fatalf("unexpected frame id: %#x", msg.FrameID)
	}
	if len(msg.Signals) != 2 {
		t.Fatalf("unexpected signal count: %d", len(msg.Signals))
	}
}

func TestBuildTopic(t *testing.T) {
	got := BuildTopic("tx", "device-a", "/state")
	if got != "/tx/device-a/state" {
		t.Fatalf("unexpected topic: %s", got)
	}
}
