package dispatcher

import (
	"encoding/json"
	"time"

	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/canhandler"
	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/gps"
)

type Snapshot struct {
	Timestamp string                        `json:"Timestamp"`
	Vehicle   map[string]map[string]float64 `json:"vehicle"`
	Location  *LocationSnapshot             `json:"location,omitempty"`
}

type LocationSnapshot struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
	Speed     float64 `json:"speed"`
	Timestamp string  `json:"timestamp"`
}

type Aggregator struct {
	vehicle map[string]map[string]float64
	gpsData *gps.Data
}

func NewAggregator() *Aggregator {
	return &Aggregator{vehicle: make(map[string]map[string]float64)}
}

func (a *Aggregator) UpdateCAN(msg canhandler.DecodeMsg) {
	if _, ok := a.vehicle[msg.BusID]; !ok {
		a.vehicle[msg.BusID] = make(map[string]float64)
	}
	for _, sig := range msg.Signals {
		a.vehicle[msg.BusID][sig.Name] = sig.Value
	}
}

func (a *Aggregator) UpdateGPS(data gps.Data) {
	clone := data
	a.gpsData = &clone
}

func (a *Aggregator) MarshalSnapshot(now time.Time) ([]byte, error) {
	s := Snapshot{
		Timestamp: now.Format(time.RFC3339),
		Vehicle:   a.vehicle,
	}
	if a.gpsData != nil {
		s.Location = &LocationSnapshot{
			Latitude:  a.gpsData.Latitude,
			Longitude: a.gpsData.Longitude,
			Altitude:  a.gpsData.Altitude,
			Speed:     a.gpsData.Speed,
			Timestamp: a.gpsData.Timestamp.Format(time.RFC3339),
		}
	}
	return json.Marshal(s)
}
