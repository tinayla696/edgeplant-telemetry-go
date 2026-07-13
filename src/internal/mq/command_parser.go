package mq

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/canhandler"
)

type controlCommandPayload struct {
	Timestamp string                 `json:"Timestamp"`
	BusID     string                 `json:"bus_id"`
	FrameID   string                 `json:"frame_id"`
	Signals   map[string]interface{} `json:"signals"`
}

// ParseControlCommand converts control JSON payload into internal CAN command format.
func ParseControlCommand(payload []byte) (canhandler.DecodeMsg, error) {
	var in controlCommandPayload
	if err := json.Unmarshal(payload, &in); err != nil {
		return canhandler.DecodeMsg{}, err
	}

	if strings.TrimSpace(in.BusID) == "" {
		return canhandler.DecodeMsg{}, fmt.Errorf("bus_id is required")
	}
	if strings.TrimSpace(in.FrameID) == "" {
		return canhandler.DecodeMsg{}, fmt.Errorf("frame_id is required")
	}

	frameID, err := parseFrameID(in.FrameID)
	if err != nil {
		return canhandler.DecodeMsg{}, err
	}

	ts := time.Now()
	if in.Timestamp != "" {
		if parsed, err := time.Parse(time.RFC3339, in.Timestamp); err == nil {
			ts = parsed
		}
	}

	signals := make([]canhandler.Signal, 0, len(in.Signals))
	for key, val := range in.Signals {
		converted, ok := signalToFloat64(val)
		if !ok {
			return canhandler.DecodeMsg{}, fmt.Errorf("unsupported signal type for %s", key)
		}
		signals = append(signals, canhandler.Signal{Name: key, Value: converted})
	}

	return canhandler.DecodeMsg{
		TimeStamp: ts,
		BusID:     in.BusID,
		FrameID:   frameID,
		Signals:   signals,
	}, nil
}

func parseFrameID(raw string) (uint32, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("frame_id is empty")
	}
	base := 10
	if strings.HasPrefix(strings.ToLower(s), "0x") {
		base = 16
		s = s[2:]
	}
	n, err := strconv.ParseUint(s, base, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid frame_id: %w", err)
	}
	return uint32(n), nil
}

func signalToFloat64(v interface{}) (float64, bool) {
	switch vv := v.(type) {
	case bool:
		if vv {
			return 1.0, true
		}
		return 0.0, true
	case float64:
		return vv, true
	case float32:
		return float64(vv), true
	case int:
		return float64(vv), true
	case int8:
		return float64(vv), true
	case int16:
		return float64(vv), true
	case int32:
		return float64(vv), true
	case int64:
		return float64(vv), true
	case uint:
		return float64(vv), true
	case uint8:
		return float64(vv), true
	case uint16:
		return float64(vv), true
	case uint32:
		return float64(vv), true
	case uint64:
		return float64(vv), true
	default:
		return 0, false
	}
}
