package gamepad

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMultiMapOptionsFromYAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "map.yaml")
	content := strings.Join([]string{
		"base:",
		"  frame_id: \"0x22\"",
		"  signals:",
		"    control_command: \"DrvEnable\"",
		"    control_value: \"DrvTorque\"",
		"steer:",
		"  frame_id: \"0x23\"",
		"  signals:",
		"    steering_command: \"StrCmd\"",
		"    steering_enable: \"StrEn\"",
		"trigger:",
		"  frame_id: \"0x24\"",
		"  signals:",
		"    brake_command: \"BrkCmd\"",
		"    throttle_command: \"ThrtlCmd\"",
		"buttons:",
		"  frame_id: \"0x25\"",
		"  signals:",
		"    horn_command: \"Horn\"",
		"    light_toggle: \"LampToggle\"",
		"    mode_next: \"ModeUp\"",
		"    mode_previous: \"ModeDown\"",
		"all_inputs:",
		"  gamepad_msg_frame_id: \"0x210\"",
		"  axes0_frame_id: \"0x200\"",
		"  axes1_frame_id: \"0x201\"",
		"  buttons_pressed_frame_id: \"0x202\"",
		"  buttons_value0_frame_id: \"0x203\"",
		"  buttons_value1_frame_id: \"0x204\"",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp yaml failed: %v", err)
	}

	got, err := LoadMultiMapOptionsFromYAML(path)
	if err != nil {
		t.Fatalf("load yaml failed: %v", err)
	}

	if got.BaseFrameID != "0x22" || got.BaseControlCommandSignal != "DrvEnable" || got.BaseControlValueSignal != "DrvTorque" {
		t.Fatalf("unexpected base config: %#v", got)
	}
	if got.SteerFrameID != "0x23" || got.SteerCommandSignal != "StrCmd" || got.SteerEnableSignal != "StrEn" {
		t.Fatalf("unexpected steer config: %#v", got)
	}
	if got.TriggerFrameID != "0x24" || got.TriggerBrakeSignal != "BrkCmd" || got.TriggerThrottleSignal != "ThrtlCmd" {
		t.Fatalf("unexpected trigger config: %#v", got)
	}
	if got.ButtonsFrameID != "0x25" || got.ButtonsHornSignal != "Horn" || got.ButtonsModePrevSignal != "ModeDown" {
		t.Fatalf("unexpected buttons config: %#v", got)
	}
	if got.GamepadMsgFrameID != "0x210" || got.AllAxes0FrameID != "0x200" || got.AllButtonsPressedFrameID != "0x202" || got.AllButtonsValue1FrameID != "0x204" {
		t.Fatalf("unexpected all-input config: %#v", got)
	}
}
