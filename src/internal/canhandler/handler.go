package canhandler

import (
	"context"
	"fmt"
	"net"
	"time"

	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/cfg"
	"go.einride.tech/can"
	"go.einride.tech/can/pkg/candevice"
	"go.einride.tech/can/pkg/socketcan"
	"go.uber.org/zap"
)

// CAN-Signal struct
type Signal struct {
	Name  string
	Value float64
	Unit  string
}

// Decode Message struct
type DecodeMsg struct {
	TimeStamp time.Time
	BusID     string
	FrameID   uint32
	Signals   []Signal
}

// SocketCAN Handler struct
type Handler struct {
	ifName string
	sigDB  *DbcStore
	logger *zap.SugaredLogger

	conn net.Conn
	rx   *socketcan.Receiver
	tx   *socketcan.Transmitter
}

// Create a new SocketCAN handler
func New(ctx context.Context, ifName string, cfg cfg.SocketCanCfg, logger *zap.SugaredLogger) (*Handler, error) {
	// Setup SocketCAN interface
	device, err := candevice.New(ifName)
	if err != nil {
		return nil, err
	}

	// Check interface up
	isUp, err := device.IsUp()
	if err != nil {
		return nil, err
	}
	if !isUp {
		logger.Warnf("SocketCAN interface %s is down. Attempting to bring it up...", ifName)
		if cfg.Bitrate > 0 {
			if err := device.SetBitrate(cfg.Bitrate); err != nil {
				return nil, err
			}
		}
		if err := device.SetUp(); err != nil {
			return nil, err
		}
		logger.Infof("SocketCAN interface %s is now up.", ifName)
	} else {
		logger.Warnf("SocketCAN interface %s is already up.", ifName)
	}

	// Parse DBC file
	var sigDB *DbcStore
	if cfg.DbcPath != "" {
		var err error
		sigDB, err = ParseDbcFile(cfg.DbcPath)
		if err != nil {
			return nil, err
		}
		logger.Infof("Parsed DBC file: %s", cfg.DbcPath)
	}

	// Connection to SocketCAN interface
	conn, err := socketcan.DialContext(ctx, "can", ifName)
	if err != nil {
		logger.Errorf("Failed to connect to SocketCAN interface %s: %v", ifName, err)
		return nil, err
	}
	rx := socketcan.NewReceiver(conn)
	tx := socketcan.NewTransmitter(conn)

	return &Handler{
		ifName: ifName,
		sigDB:  sigDB,
		logger: logger,
		conn:   conn,
		rx:     rx,
		tx:     tx,
	}, nil

}

// Connection Close function
func (h *Handler) Close() {
	if h.rx != nil {
		h.rx.Close()
	}
	if h.tx != nil {
		h.tx.Close()
	}
	if h.conn != nil {
		h.conn.Close()
	}
}

// Start Listening to CAN messages
func (h *Handler) StartListening(ctx context.Context, rxCh chan<- DecodeMsg) {
	h.logger.Infof("Starting to listen on SocketCAN interface: %s", h.ifName)

	// Listening loop
	for h.rx.Receive() {
		frame := h.rx.Frame()
		sigs := h.ParseFrame(frame)
		if len(sigs) > 0 {
			decodeMsg := DecodeMsg{
				TimeStamp: time.Now(),
				BusID:     h.ifName,
				FrameID:   frame.ID,
				Signals:   sigs,
			}
			h.logger.Debugf("Received frame: ID=%d, Signals=%v", frame.ID, sigs)
			select {
			case rxCh <- decodeMsg:
			case <-ctx.Done():
				h.logger.Infof("Stopping listening on SocketCAN interface: %s", h.ifName)
				return
			}
		}
	}

	// Check for errors in the receiver
	if err := h.rx.Err(); err != nil {
		h.logger.Errorf("Error while receiving from SocketCAN interface %s: %v", h.ifName, err)
	}
}

// Send CAN message frame
func (h *Handler) SendFrame(ctx context.Context, msg DecodeMsg) error {
	h.logger.Debugf("Sending frame: ID=%d, Signals=%v", msg.FrameID, msg.Signals)

	// Create CAN frame from DecodeMsg
	frame, err := h.MakeFrame(msg)
	if err != nil {
		h.logger.Errorf("Failed to create CAN frame for message with ID %d: %v", msg.FrameID, err)
		return err
	}

	// Transmit the frame
	if err := h.tx.TransmitFrame(ctx, frame); err != nil {
		h.logger.Errorf("Failed to transmit CAN frame with ID %d: %v", frame.ID, err)
		return err
	}
	h.logger.Debugf("Successfully transmitted CAN frame with ID %d", frame.ID)
	return nil
}

// Parse CAN-Message frame and decode signals
func (h *Handler) ParseFrame(f can.Frame) []Signal {
	if h.sigDB == nil {
		h.logger.Warnf("No DBC file loaded. Cannot decode frame with ID: %d", f.ID)
		return nil
	}

	msgDef, ok := h.sigDB.Msgs[f.ID]
	if !ok {
		h.logger.Warnf("No message definition found for frame ID: %d", f.ID)
		return nil
	}
	sigs := make([]Signal, 0, len(msgDef.Signals))
	for _, sigDef := range msgDef.Signals {
		if sigDef.IsMultiplexed {

			// Handle multiplexed signals if needed
			switchSignal, _ := msgDef.MultiplexerSignal()
			if switchSignal == nil {
				continue
			}

			// Check if the multiplexed signal matches the expected value
			switchValue := uint(switchSignal.UnmarshalPhysical(f.Data))
			if uint(switchValue) != sigDef.MultiplexerValue {
				continue
			}
		}

		// Unmarshal the signal value
		physVal := sigDef.UnmarshalPhysical(f.Data)
		sigs = append(sigs, Signal{
			Name:  sigDef.Name,
			Value: physVal,
			Unit:  sigDef.Unit,
		})
	}
	return sigs
}

// Make CAN-Frame from DecodeMsg
func (h *Handler) MakeFrame(msg DecodeMsg) (can.Frame, error) {
	// Check if the DBC store is loaded
	if h.sigDB == nil {
		return can.Frame{}, fmt.Errorf("no DBC file loaded. Cannot create frame for message with ID: %d", msg.FrameID)
	}

	// Create a new CAN data array
	var data can.Data

	// Check if the message definition exists in the DBC store
	msgDef, ok := h.sigDB.Msgs[msg.FrameID]
	if !ok {
		return can.Frame{}, fmt.Errorf("no message definition found for frame ID: %d", msg.FrameID)
	}

	// FromPhysical for each signal in the DecodeMsg
	for _, sigDef := range msgDef.Signals {
		for _, sig := range msg.Signals {
			if sigDef.Name == sig.Name {
				rawValueFloat := sigDef.FromPhysical(sig.Value)

				if sigDef.IsBigEndian {
					// Big-endian encoding
					if sigDef.IsSigned {
						// Signed big-endian encoding
						data.SetSignedBitsBigEndian(sigDef.Start, sigDef.Length, int64(rawValueFloat))
					} else {
						// Unsigned big-endian encoding
						data.SetUnsignedBitsBigEndian(sigDef.Start, sigDef.Length, uint64(rawValueFloat))
					}
				} else {
					// Little-endian encoding
					if sigDef.IsSigned {
						// Signed little-endian encoding
						data.SetSignedBitsLittleEndian(sigDef.Start, sigDef.Length, int64(rawValueFloat))
					} else {
						// Unsigned little-endian encoding
						data.SetUnsignedBitsLittleEndian(sigDef.Start, sigDef.Length, uint64(rawValueFloat))
					}
				}
			}
		}
	}

	// Create CAN frame
	frame := can.Frame{
		ID:         msg.FrameID,
		Length:     uint8(len(data)),
		Data:       data,
		IsExtended: msgDef.IsExtended,
		IsRemote:   false,
	}

	return frame, nil
}
