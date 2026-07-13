package dispatcher

import (
	"testing"
	"time"

	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/canhandler"
	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/gps"
)

func TestAggregatorMarshalSnapshot(t *testing.T) {
	a := NewAggregator()
	a.UpdateCAN(canhandler.DecodeMsg{
		BusID:   "can0",
		Signals: []canhandler.Signal{{Name: "speed", Value: 12.3}},
	})
	a.UpdateGPS(gps.Data{Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Latitude: 1.0, Longitude: 2.0})

	out, err := a.MarshalSnapshot(time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC))
	if err != nil {
		t.Fatalf("MarshalSnapshot failed: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("expected non-empty payload")
	}
}
