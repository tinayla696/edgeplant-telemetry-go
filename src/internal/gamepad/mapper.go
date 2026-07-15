package gamepad

import (
	"encoding/json"
	"fmt"
	"math"
)

// State is a normalized gamepad input snapshot.
type State struct {
	Axes    []float64     `json:"axes"`
	Buttons []ButtonState `json:"buttons"`
}

// ButtonState contains both digital and analog states for a button/trigger.
type ButtonState struct {
	Pressed bool    `json:"pressed"`
	Value   float64 `json:"value"`
}

// UnmarshalJSON supports bool, number, and {pressed,value} button formats.
func (b *ButtonState) UnmarshalJSON(data []byte) error {
	var asBool bool
	if err := json.Unmarshal(data, &asBool); err == nil {
		b.Pressed = asBool
		if asBool {
			b.Value = 1.0
		} else {
			b.Value = 0
		}
		return nil
	}

	var asNumber float64
	if err := json.Unmarshal(data, &asNumber); err == nil {
		b.Value = clamp(asNumber, 0, 1)
		b.Pressed = b.Value > 0.5
		return nil
	}

	type buttonAlias struct {
		Pressed bool    `json:"pressed"`
		Value   float64 `json:"value"`
	}
	var asObj buttonAlias
	if err := json.Unmarshal(data, &asObj); err != nil {
		return fmt.Errorf("invalid button format: %w", err)
	}
	b.Pressed = asObj.Pressed
	b.Value = clamp(asObj.Value, 0, 1)
	if b.Value == 0 && b.Pressed {
		b.Value = 1.0
	}
	if b.Value > 0.5 {
		b.Pressed = true
	}
	return nil
}

// CommandSignals is the signal payload to map into CAN command JSON.
type CommandSignals struct {
	ControlCommand bool
	ControlValue   float64
}

// CommandMap binds one mapped signal set to a target frame ID.
type CommandMap struct {
	FrameID string
	Signals map[string]interface{}
}

// MultiMapOptions controls optional multi-frame mapping outputs.
type MultiMapOptions struct {
	BaseFrameID              string
	SteerFrameID             string
	TriggerFrameID           string
	ButtonsFrameID           string
	GamepadMsgFrameID        string
	AllAxes0FrameID          string
	AllAxes1FrameID          string
	AllButtonsPressedFrameID string
	AllButtonsValue0FrameID  string
	AllButtonsValue1FrameID  string

	BaseControlCommandSignal string
	BaseControlValueSignal   string
	SteerCommandSignal       string
	SteerEnableSignal        string
	TriggerBrakeSignal       string
	TriggerThrottleSignal    string
	ButtonsHornSignal        string
	ButtonsLightSignal       string
	ButtonsModeNextSignal    string
	ButtonsModePrevSignal    string
}

// Mapper converts gamepad state to command signals.
type Mapper struct {
	DeadZone float64
	MaxValue float64
}

// NewMapper creates a mapper with conservative defaults for control tests.
func NewMapper() Mapper {
	return Mapper{
		DeadZone: 0.12,
		MaxValue: 100.0,
	}
}

// ToSignals maps left-stick Y axis and A button into control signals.
// Axes[1] is commonly left-stick Y in browser Gamepad API.
func (m Mapper) ToSignals(s State) CommandSignals {
	throttle := 0.0
	if len(s.Axes) > 1 {
		v := clamp(s.Axes[1], -1.0, 1.0)
		if math.Abs(v) >= m.DeadZone {
			throttle = -v * m.MaxValue
		}
	}

	active := false
	if len(s.Buttons) > 0 {
		active = s.Buttons[0].Pressed
	}

	return CommandSignals{
		ControlCommand: active,
		ControlValue:   roundTo(throttle, 1),
	}
}

// ToCommandMaps maps one gamepad state to one or multiple CAN command frames.
func (m Mapper) ToCommandMaps(s State, opts MultiMapOptions) []CommandMap {
	results := make([]CommandMap, 0, 4)
	baseCmdName := nameOrDefault(opts.BaseControlCommandSignal, "ControlCommand")
	baseValName := nameOrDefault(opts.BaseControlValueSignal, "ControlValue")
	steerCmdName := nameOrDefault(opts.SteerCommandSignal, "SteeringCommand")
	steerEnableName := nameOrDefault(opts.SteerEnableSignal, "SteeringEnable")
	trigBrakeName := nameOrDefault(opts.TriggerBrakeSignal, "BrakeCommand")
	trigThrottleName := nameOrDefault(opts.TriggerThrottleSignal, "ThrottleCommand")
	btnHornName := nameOrDefault(opts.ButtonsHornSignal, "HornCommand")
	btnLightName := nameOrDefault(opts.ButtonsLightSignal, "LightToggle")
	btnModeNextName := nameOrDefault(opts.ButtonsModeNextSignal, "ModeNext")
	btnModePrevName := nameOrDefault(opts.ButtonsModePrevSignal, "ModePrevious")

	if opts.BaseFrameID != "" {
		base := m.ToSignals(s)
		results = append(results, CommandMap{
			FrameID: opts.BaseFrameID,
			Signals: map[string]interface{}{
				baseCmdName: base.ControlCommand,
				baseValName: base.ControlValue,
			},
		})
	}

	if opts.SteerFrameID != "" {
		steer := 0.0
		if len(s.Axes) > 0 {
			v := clamp(s.Axes[0], -1.0, 1.0)
			if math.Abs(v) >= m.DeadZone {
				steer = v * m.MaxValue
			}
		}
		results = append(results, CommandMap{
			FrameID: opts.SteerFrameID,
			Signals: map[string]interface{}{
				steerCmdName:    roundTo(steer, 1),
				steerEnableName: buttonPressed(s, 2),
			},
		})
	}

	if opts.TriggerFrameID != "" {
		leftTrigger := buttonValue(s, 6) * m.MaxValue
		rightTrigger := buttonValue(s, 7) * m.MaxValue
		results = append(results, CommandMap{
			FrameID: opts.TriggerFrameID,
			Signals: map[string]interface{}{
				trigBrakeName:    roundTo(leftTrigger, 1),
				trigThrottleName: roundTo(rightTrigger, 1),
			},
		})
	}

	if opts.ButtonsFrameID != "" {
		results = append(results, CommandMap{
			FrameID: opts.ButtonsFrameID,
			Signals: map[string]interface{}{
				btnHornName:     buttonPressed(s, 1),
				btnLightName:    buttonPressed(s, 3),
				btnModeNextName: buttonPressed(s, 4),
				btnModePrevName: buttonPressed(s, 5),
			},
		})
	}

	if opts.GamepadMsgFrameID != "" {
		results = append(results, CommandMap{
			FrameID: opts.GamepadMsgFrameID,
			Signals: gamepadMsgSignalMap(s),
		})
	}

	if opts.AllAxes0FrameID != "" {
		results = append(results, CommandMap{
			FrameID: opts.AllAxes0FrameID,
			Signals: axisSignalMap(s, 0, 4),
		})
	}

	if opts.AllAxes1FrameID != "" {
		results = append(results, CommandMap{
			FrameID: opts.AllAxes1FrameID,
			Signals: axisSignalMap(s, 4, 4),
		})
	}

	if opts.AllButtonsPressedFrameID != "" {
		results = append(results, CommandMap{
			FrameID: opts.AllButtonsPressedFrameID,
			Signals: buttonPressedSignalMap(s, 0, 16),
		})
	}

	if opts.AllButtonsValue0FrameID != "" {
		results = append(results, CommandMap{
			FrameID: opts.AllButtonsValue0FrameID,
			Signals: buttonValueSignalMap(s, 0, 8),
		})
	}

	if opts.AllButtonsValue1FrameID != "" {
		results = append(results, CommandMap{
			FrameID: opts.AllButtonsValue1FrameID,
			Signals: buttonValueSignalMap(s, 8, 8),
		})
	}

	return results
}

func axisSignalMap(s State, start, count int) map[string]interface{} {
	res := make(map[string]interface{}, count)
	for i := 0; i < count; i++ {
		idx := start + i
		name := fmt.Sprintf("Axis%02d", idx)
		value := 0.0
		if idx >= 0 && idx < len(s.Axes) {
			value = clamp(s.Axes[idx], -1.0, 1.0)
		}
		res[name] = roundTo(value, 3)
	}
	return res
}

func buttonPressedSignalMap(s State, start, count int) map[string]interface{} {
	res := make(map[string]interface{}, count)
	for i := 0; i < count; i++ {
		idx := start + i
		name := fmt.Sprintf("Btn%02d_P", idx)
		res[name] = buttonPressed(s, idx)
	}
	return res
}

func buttonValueSignalMap(s State, start, count int) map[string]interface{} {
	res := make(map[string]interface{}, count)
	for i := 0; i < count; i++ {
		idx := start + i
		name := fmt.Sprintf("Btn%02d_V", idx)
		res[name] = roundTo(buttonValue(s, idx), 3)
	}
	return res
}

func gamepadMsgSignalMap(s State) map[string]interface{} {
	res := map[string]interface{}{
		"StickLX": axisToInt8Physical(s, 0),
		"StickLY": axisToInt8Physical(s, 1),
		"StickRX": axisToInt8Physical(s, 2),
		"StickRY": axisToInt8Physical(s, 3),
	}
	for i := 0; i < 16; i++ {
		name := fmt.Sprintf("Btn%02d_P", i)
		res[name] = buttonPressed(s, i)
	}
	return res
}

// axisToInt8Physical converts normalized stick input [-1, 1] to signed byte range [-128, 127].
func axisToInt8Physical(s State, idx int) float64 {
	v := 0.0
	if idx >= 0 && idx < len(s.Axes) {
		v = clamp(s.Axes[idx], -1.0, 1.0)
	}

	if v >= 0 {
		iv := int(math.Round(v * 127.0))
		if iv > 127 {
			iv = 127
		}
		return float64(iv)
	}

	iv := int(math.Round(v * 128.0))
	if iv < -128 {
		iv = -128
	}
	return float64(iv)
}

func buttonPressed(s State, idx int) bool {
	if idx < 0 || idx >= len(s.Buttons) {
		return false
	}
	return s.Buttons[idx].Pressed
}

func buttonValue(s State, idx int) float64 {
	if idx < 0 || idx >= len(s.Buttons) {
		return 0
	}
	return clamp(s.Buttons[idx].Value, 0, 1)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func roundTo(v float64, digits int) float64 {
	if digits <= 0 {
		return math.Round(v)
	}
	pow := math.Pow(10, float64(digits))
	return math.Round(v*pow) / pow
}

func nameOrDefault(name, fallback string) string {
	if name == "" {
		return fallback
	}
	return name
}
