package gamepad

import (
	"fmt"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

type fileMapping struct {
	Base struct {
		FrameID string `yaml:"frame_id"`
		Signals struct {
			ControlCommand string `yaml:"control_command"`
			ControlValue   string `yaml:"control_value"`
		} `yaml:"signals"`
	} `yaml:"base"`
	Steer struct {
		FrameID string `yaml:"frame_id"`
		Signals struct {
			SteeringCommand string `yaml:"steering_command"`
			SteeringEnable  string `yaml:"steering_enable"`
		} `yaml:"signals"`
	} `yaml:"steer"`
	Trigger struct {
		FrameID string `yaml:"frame_id"`
		Signals struct {
			BrakeCommand    string `yaml:"brake_command"`
			ThrottleCommand string `yaml:"throttle_command"`
		} `yaml:"signals"`
	} `yaml:"trigger"`
	Buttons struct {
		FrameID string `yaml:"frame_id"`
		Signals struct {
			HornCommand  string `yaml:"horn_command"`
			LightToggle  string `yaml:"light_toggle"`
			ModeNext     string `yaml:"mode_next"`
			ModePrevious string `yaml:"mode_previous"`
		} `yaml:"signals"`
	} `yaml:"buttons"`
	AllInputs struct {
		GamepadMsgFrameID     string `yaml:"gamepad_msg_frame_id"`
		Axes0FrameID          string `yaml:"axes0_frame_id"`
		Axes1FrameID          string `yaml:"axes1_frame_id"`
		ButtonsPressedFrameID string `yaml:"buttons_pressed_frame_id"`
		ButtonsValue0FrameID  string `yaml:"buttons_value0_frame_id"`
		ButtonsValue1FrameID  string `yaml:"buttons_value1_frame_id"`
	} `yaml:"all_inputs"`
}

// LoadMultiMapOptionsFromYAML reads frame IDs and signal names from a YAML file.
func LoadMultiMapOptionsFromYAML(path string) (MultiMapOptions, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return MultiMapOptions{}, err
	}

	var cfg fileMapping
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return MultiMapOptions{}, fmt.Errorf("failed to parse map config: %w", err)
	}

	opts := MultiMapOptions{
		BaseFrameID:              strings.TrimSpace(cfg.Base.FrameID),
		SteerFrameID:             strings.TrimSpace(cfg.Steer.FrameID),
		TriggerFrameID:           strings.TrimSpace(cfg.Trigger.FrameID),
		ButtonsFrameID:           strings.TrimSpace(cfg.Buttons.FrameID),
		GamepadMsgFrameID:        strings.TrimSpace(cfg.AllInputs.GamepadMsgFrameID),
		AllAxes0FrameID:          strings.TrimSpace(cfg.AllInputs.Axes0FrameID),
		AllAxes1FrameID:          strings.TrimSpace(cfg.AllInputs.Axes1FrameID),
		AllButtonsPressedFrameID: strings.TrimSpace(cfg.AllInputs.ButtonsPressedFrameID),
		AllButtonsValue0FrameID:  strings.TrimSpace(cfg.AllInputs.ButtonsValue0FrameID),
		AllButtonsValue1FrameID:  strings.TrimSpace(cfg.AllInputs.ButtonsValue1FrameID),
		BaseControlCommandSignal: strings.TrimSpace(cfg.Base.Signals.ControlCommand),
		BaseControlValueSignal:   strings.TrimSpace(cfg.Base.Signals.ControlValue),
		SteerCommandSignal:       strings.TrimSpace(cfg.Steer.Signals.SteeringCommand),
		SteerEnableSignal:        strings.TrimSpace(cfg.Steer.Signals.SteeringEnable),
		TriggerBrakeSignal:       strings.TrimSpace(cfg.Trigger.Signals.BrakeCommand),
		TriggerThrottleSignal:    strings.TrimSpace(cfg.Trigger.Signals.ThrottleCommand),
		ButtonsHornSignal:        strings.TrimSpace(cfg.Buttons.Signals.HornCommand),
		ButtonsLightSignal:       strings.TrimSpace(cfg.Buttons.Signals.LightToggle),
		ButtonsModeNextSignal:    strings.TrimSpace(cfg.Buttons.Signals.ModeNext),
		ButtonsModePrevSignal:    strings.TrimSpace(cfg.Buttons.Signals.ModePrevious),
	}

	return opts, nil
}
