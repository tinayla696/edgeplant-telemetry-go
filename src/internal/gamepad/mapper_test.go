package gamepad

import (
	"encoding/json"
	"testing"
)

func TestToSignals_ButtonAndAxis(t *testing.T) {
	mapper := NewMapper()

	got := mapper.ToSignals(State{
		Axes:    []float64{0, -0.70},
		Buttons: []ButtonState{{Pressed: true, Value: 1}},
	})

	if !got.ControlCommand {
		t.Fatalf("expected ControlCommand=true")
	}
	if got.ControlValue != 70.0 {
		t.Fatalf("expected ControlValue=70.0, got=%v", got.ControlValue)
	}
}

func TestToSignals_DeadZone(t *testing.T) {
	mapper := NewMapper()

	got := mapper.ToSignals(State{
		Axes:    []float64{0, 0.05},
		Buttons: []ButtonState{{Pressed: false, Value: 0}},
	})

	if got.ControlCommand {
		t.Fatalf("expected ControlCommand=false")
	}
	if got.ControlValue != 0 {
		t.Fatalf("expected ControlValue=0, got=%v", got.ControlValue)
	}
}

func TestToSignals_Clamp(t *testing.T) {
	mapper := NewMapper()

	got := mapper.ToSignals(State{
		Axes:    []float64{0, -2.0},
		Buttons: []ButtonState{{Pressed: true, Value: 1}},
	})

	if got.ControlValue != 100.0 {
		t.Fatalf("expected ControlValue=100.0, got=%v", got.ControlValue)
	}
}

func TestToCommandMaps_MultiFrame(t *testing.T) {
	mapper := NewMapper()
	state := State{
		Axes: []float64{0.5, -0.4},
		Buttons: []ButtonState{
			{Pressed: true, Value: 1},  // A
			{Pressed: true, Value: 1},  // B
			{Pressed: false, Value: 0}, // Xbox X
			{Pressed: true, Value: 1},  // Y
			{Pressed: true, Value: 1},  // LB
			{Pressed: false, Value: 0}, // RB
			{Pressed: true, Value: 0.8},
			{Pressed: true, Value: 0.2},
		},
	}

	maps := mapper.ToCommandMaps(state, MultiMapOptions{
		BaseFrameID:    "0x2",
		SteerFrameID:   "0x3",
		TriggerFrameID: "0x4",
		ButtonsFrameID: "0x5",
	})

	if len(maps) != 4 {
		t.Fatalf("expected 4 mapped commands, got=%d", len(maps))
	}
	if maps[0].FrameID != "0x2" || maps[0].Signals["ControlValue"] != 40.0 {
		t.Fatalf("unexpected base mapping: %#v", maps[0])
	}
	if maps[1].FrameID != "0x3" || maps[1].Signals["SteeringCommand"] != 50.0 {
		t.Fatalf("unexpected steer mapping: %#v", maps[1])
	}
	if maps[2].FrameID != "0x4" || maps[2].Signals["BrakeCommand"] != 80.0 || maps[2].Signals["ThrottleCommand"] != 20.0 {
		t.Fatalf("unexpected trigger mapping: %#v", maps[2])
	}
	if maps[3].FrameID != "0x5" || maps[3].Signals["HornCommand"] != true || maps[3].Signals["ModePrevious"] != false {
		t.Fatalf("unexpected button mapping: %#v", maps[3])
	}
}

func TestStateUnmarshal_ButtonFormats(t *testing.T) {
	var st State
	payload := []byte(`{"axes":[0,-0.2],"buttons":[true,0.7,{"pressed":false,"value":0.9}]}`)
	if err := json.Unmarshal(payload, &st); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if len(st.Buttons) != 3 {
		t.Fatalf("unexpected button count: %d", len(st.Buttons))
	}
	if !st.Buttons[0].Pressed || st.Buttons[0].Value != 1.0 {
		t.Fatalf("unexpected bool-button mapping: %#v", st.Buttons[0])
	}
	if !st.Buttons[1].Pressed || st.Buttons[1].Value != 0.7 {
		t.Fatalf("unexpected numeric-button mapping: %#v", st.Buttons[1])
	}
	if !st.Buttons[2].Pressed || st.Buttons[2].Value != 0.9 {
		t.Fatalf("unexpected object-button mapping: %#v", st.Buttons[2])
	}
}

func TestToCommandMaps_AllInputs(t *testing.T) {
	mapper := NewMapper()
	state := State{
		Axes: []float64{-1.0, -0.5, 0.25, 1.0, -0.75},
		Buttons: []ButtonState{
			{Pressed: true, Value: 1.0},
			{Pressed: false, Value: 0.0},
			{Pressed: true, Value: 0.6},
		},
	}

	maps := mapper.ToCommandMaps(state, MultiMapOptions{
		GamepadMsgFrameID: "0x210",
	})

	if len(maps) != 1 {
		t.Fatalf("expected 1 mapped frame, got=%d", len(maps))
	}
	if maps[0].Signals["StickLX"] != -128.0 || maps[0].Signals["StickLY"] != -64.0 || maps[0].Signals["StickRX"] != 32.0 || maps[0].Signals["StickRY"] != 127.0 {
		t.Fatalf("unexpected gamepad msg mapping: %#v", maps[0].Signals)
	}
	if maps[0].Signals["Btn00_P"] != true || maps[0].Signals["Btn15_P"] != false {
		t.Fatalf("unexpected gamepad button bits: %#v", maps[0].Signals)
	}
}

func TestAxisToInt8Physical(t *testing.T) {
	state := State{Axes: []float64{-1.0, -0.5, 0, 0.5, 1.0, 2.0, -2.0}}

	cases := []struct {
		idx int
		exp float64
	}{
		{0, -128},
		{1, -64},
		{2, 0},
		{3, 64},
		{4, 127},
		{5, 127},
		{6, -128},
		{100, 0},
	}

	for _, tc := range cases {
		got := axisToInt8Physical(state, tc.idx)
		if got != tc.exp {
			t.Fatalf("idx=%d expected=%v got=%v", tc.idx, tc.exp, got)
		}
	}
}
