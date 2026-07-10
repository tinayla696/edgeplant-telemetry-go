package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/canhandler"
	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/cfg"
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
	mqttPubCh := make(chan mq.Msg)
	mqttSubCh := make(chan mq.Msg)
	gpsRxCh := make(chan gps.Data)
	canRxCh := make(chan canhandler.DecodeMsg)
	canTxCh := make(chan canhandler.DecodeMsg)

	// MQTT Handler
	wg.Add(1)
	mqttH, err := mq.New(*deviceIDStr, cfg.Mosquitto, zap.S())
	if err != nil {
		zap.S().Fatalf("Failed to initialize MQTT handler: %v", err)
	}
	defer mqttH.Close()

	// GPSD Handler
	wg.Add(1)
	gpsH := gps.New(cfg.Gpsd.Endpoint, zap.S())
	go func() {
		defer wg.Done()
		gpsH.Start(context.Background(), gpsRxCh)
	}()

	// SocketCAN Handler
	for ifName, canCfg := range cfg.SocketCAN {
		zap.S().Infof("Starting SocketCAN handler for interface: %s", ifName)
		h, err := canhandler.New(ctx, ifName, canCfg, zap.S())
		if err != nil {
			zap.S().Fatalf("Failed to initialize SocketCAN handler for interface %s: %v", ifName, err)
		}

		// Start Listening to CAN messages
		wg.Add(1)
		go func(h *canhandler.Handler) {
			defer wg.Done()
			h.StartListening(ctx, canRxCh)
		}(h)
	}

	// End Program on Interrupt
	<-interruptCh
	cancelFn()
	wg.Wait()
	zap.S().Info("Telemetry program terminated gracefully.")
}
