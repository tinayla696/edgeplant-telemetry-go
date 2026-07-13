package gps

import (
	"context"
	"time"

	"github.com/stratoberry/go-gpsd"
	"go.uber.org/zap"
)

// GPS Data struct
type Data struct {
	Timestamp time.Time
	Latitude  float64
	Longitude float64
	Altitude  float64
	Speed     float64
	Climb     float64
	Track     float64
}

// GPS Handler struct
type Handler struct {
	host   string
	logger *zap.SugaredLogger // Add any necessary fields for the GPS handler
}

func New(host string, logger *zap.SugaredLogger) *Handler {
	if host == "" {
		host = "localhost:2947"
	}
	return &Handler{host: host, logger: logger}
}

// Start はgpsdへの接続とデータ受信を開始します。
func (h *Handler) Start(ctx context.Context, dataChan chan<- Data) {
	h.logger.Infof("Starting GPSD handler for host %s", h.host)

	// 無限ループで再接続を試みる
	for {
		// まず、アプリケーションの終了が要求されていないかチェック
		select {
		case <-ctx.Done():
			h.logger.Info("Stopping GPSD handler due to context cancellation.")
			return
		default:
			// 継続
		}

		h.logger.Info("Attempting to connect to GPSD...")
		session, err := gpsd.Dial(h.host)
		if err != nil {
			h.logger.Warnf("Failed to connect to GPSD at %s: %v. Retrying in 5 seconds...", h.host, err)
			time.Sleep(5 * time.Second)
			continue // ループの先頭に戻ってリトライ
		}

		// 接続成功
		h.logger.Info("Successfully connected to GPSD.")

		// AddFilterとWatchのロジック (ここは変更なし)
		session.AddFilter("TPV", func(report interface{}) {
			if tpvReport, ok := report.(*gpsd.TPVReport); ok {
				if tpvReport.Mode >= 2 {
					dataChan <- Data{
						Timestamp: tpvReport.Time,
						Latitude:  tpvReport.Lat,
						Longitude: tpvReport.Lon,
						Altitude:  tpvReport.Alt,
						Speed:     tpvReport.Speed,
						Climb:     tpvReport.Climb,
						Track:     tpvReport.Track,
					}
				}
			}
		})

		done := session.Watch()

		// 接続が維持されている間、ここでブロックされる
		// DEV: コンテキストがキャンセルされるか、接続が切れるとブロックが解除される
		select {
		case <-ctx.Done():
			h.logger.Info("Stopping GPSD handler due to context cancellation.")
			session.Close()
			return
		case <-done:
			// 接続が切れた場合
			h.logger.Warn("GPSD watch has finished, connection may be lost. Attempting to reconnect...")
			session.Close()
			// DEV: この後、forループの次のイテレーションで再接続が試みられる
		}
	}
}
