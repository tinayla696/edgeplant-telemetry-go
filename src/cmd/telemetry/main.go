package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/canhandler"
	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/cfg"
	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/dispatcher"
	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/gps"
	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/logger"
	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/mq"
	"go.uber.org/zap"
)

const (
	logFileCount int = 5
)

var (
	def_DeviceID string = "telemetry_device"
	def_LogDir   string = "logs"
	def_CfgPath  string = "config/config.yaml"

	deviceIDStr = flag.String("device", def_DeviceID, "Device ID")               // device ID flag
	confPathStr = flag.String("conf", def_CfgPath, "Path to configuration file") // configuration file path flag
	logPathStr  = flag.String("logPath", def_LogDir, "Path to log directory")    // log path flag
	isDevFlag   = flag.Bool("dev", false, "Development mode")                    // development mode flag

	logLevel string = "info"
)

func main() {
	// Graceful Shutdown
	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()
	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt, syscall.SIGTERM)

	// Start Program
	flag.Parse()

	// load configuration
	cfg, err := cfg.LoadConfig(*confPathStr)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// development mode
	if *isDevFlag {
		log.Printf("Development mode enabled.")
		logLevel = "debug"
	}
	appLogger, err := logger.New(*logPathStr, logFileCount, logLevel, *isDevFlag)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	zap.ReplaceGlobals(appLogger)

	zap.S().Info("Telemetry program started.")

	zap.S().Debug("Starting telemetry program...")
	zap.S().Debugf("Device ID: %s", *deviceIDStr)
	zap.S().Debugf("Configuration Path: %s", *confPathStr)
	zap.S().Debugf("Log Path: %s", *logPathStr)
	zap.S().Debugf("Development Mode: %v", *isDevFlag)

	// Working Grp
	var wg sync.WaitGroup
	gpsRxCh := make(chan gps.Data, 100)
	canRxCh := make(chan canhandler.DecodeMsg, 200)
	canTxCh := make(chan canhandler.DecodeMsg, 100)

	// Message Broker Handler
	broker, err := mq.NewBroker(*deviceIDStr, cfg.Broker.Type, cfg.Mosquitto, cfg.RabbitMQ, zap.S())
	if err != nil {
		zap.S().Fatalf("Failed to initialize broker handler: %v", err)
	}
	defer broker.Close()

	// SocketCAN Handler
	canHandlers := make(map[string]*canhandler.Handler, len(cfg.SocketCAN))
	latchFrameIDSet := make(map[string]map[uint32]bool, len(cfg.SocketCAN))
	for ifName, canCfg := range cfg.SocketCAN {
		zap.S().Infof("Starting SocketCAN handler for interface: %s", ifName)
		idSet := make(map[uint32]bool, len(canCfg.LatchFrameIDs))
		for _, frameID := range canCfg.LatchFrameIDs {
			idSet[frameID] = true
		}
		latchFrameIDSet[ifName] = idSet
		zap.S().Debugf("Latch whitelist for %s initialized with %d frame IDs", ifName, len(idSet))

		h, err := canhandler.New(ctx, ifName, canCfg, zap.S())
		if err != nil {
			zap.S().Fatalf("Failed to initialize SocketCAN handler for interface %s: %v", ifName, err)
		}
		canHandlers[ifName] = h

		// Start Listening to CAN messages
		wg.Add(1)
		go func(h *canhandler.Handler) {
			defer wg.Done()
			h.StartListening(ctx, canRxCh)
		}(h)
	}

	// GPSD Handler
	wg.Add(1)
	gpsH := gps.New(cfg.Gpsd.Endpoint, zap.S())
	go func() {
		defer wg.Done()
		gpsH.Start(ctx, gpsRxCh)
	}()

	// MQTT subscriber for control commands.
	rxTopics := make(map[string]byte)
	for _, prefix := range cfg.Telemetry.SubscribeMsgs.TopicPrefixes {
		topic := mq.BuildTopic("rx", *deviceIDStr, prefix)
		rxTopics[topic] = 0
	}
	rxTopicList := make([]string, 0, len(rxTopics))
	for topic := range rxTopics {
		rxTopicList = append(rxTopicList, topic)
	}
	if err := broker.Subscribe(rxTopicList, func(topic string, payload []byte) {
		parsed, err := mq.ParseControlCommand(payload)
		if err != nil {
			zap.S().Warnf("Invalid command payload on topic %s: %v", topic, err)
			return
		}
		select {
		case canTxCh <- parsed:
		case <-ctx.Done():
		}
	}); err != nil {
		zap.S().Fatalf("Failed to subscribe control topics: %v", err)
	}

	// TX worker.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-canTxCh:
				h, ok := canHandlers[msg.BusID]
				if !ok {
					zap.S().Warnf("No CAN handler for bus_id=%s", msg.BusID)
					continue
				}
				if err := h.SendFrame(ctx, msg); err != nil {
					zap.S().Warnf("Failed to send frame to %s: %v", msg.BusID, err)
				}
			}
		}
	}()

	// Publisher worker.
	txTopic := mq.BuildTopic("tx", *deviceIDStr, cfg.Telemetry.PublishMsgs.TopicPrefix)
	agg := dispatcher.NewAggregator()
	pubInterval := time.Duration(cfg.Telemetry.PublishMsgs.IntervalMs) * time.Millisecond
	ticker := time.NewTicker(pubInterval)
	defer ticker.Stop()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case gpsMsg := <-gpsRxCh:
				agg.UpdateGPS(gpsMsg)
			case canMsg := <-canRxCh:
				ifIDs, ok := latchFrameIDSet[canMsg.BusID]
				if !ok {
					zap.S().Debugf("Skip CAN latch: unknown bus_id=%s frame_id=%d", canMsg.BusID, canMsg.FrameID)
					continue
				}
				if !ifIDs[canMsg.FrameID] {
					zap.S().Debugf("Skip CAN latch: bus_id=%s frame_id=%d not in whitelist", canMsg.BusID, canMsg.FrameID)
					continue
				}
				agg.UpdateCAN(canMsg)
			case <-ticker.C:
				payload, err := agg.MarshalSnapshot(time.Now())
				if err != nil {
					zap.S().Warnf("Failed to build snapshot payload: %v", err)
					continue
				}
				if err := broker.Publish(txTopic, payload); err != nil {
					zap.S().Warnf("Failed to publish snapshot: %v", err)
				}
			}
		}
	}()

	// End Program on Interrupt
	<-interruptCh
	cancelFn()
	for ifName, h := range canHandlers {
		zap.S().Debugf("Closing SocketCAN handler: %s", ifName)
		h.Close()
	}
	wg.Wait()
	zap.S().Info("Telemetry program terminated gracefully.")
}
