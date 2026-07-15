package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/gamepad"
	"gihub.com/tinayla696/edgeplant-telemetry-go/internal/mq"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/websocket"
)

type controlPayload struct {
	Timestamp string                 `json:"Timestamp"`
	BusID     string                 `json:"bus_id"`
	FrameID   string                 `json:"frame_id"`
	Signals   map[string]interface{} `json:"signals"`
}

type server struct {
	mapper  gamepad.Mapper
	client  mqtt.Client
	topic   string
	busID   string
	mapOpts gamepad.MultiMapOptions
}

//go:embed assets
var webAssets embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}

		u, err := url.Parse(origin)
		if err != nil {
			return false
		}

		originHost := u.Hostname()
		requestHost := r.Host
		if h, _, err := net.SplitHostPort(r.Host); err == nil {
			requestHost = h
		}

		if strings.EqualFold(originHost, requestHost) {
			return true
		}

		return isLoopbackHost(originHost) && isLoopbackHost(requestHost)
	},
}

func isLoopbackHost(host string) bool {
	h := strings.Trim(strings.ToLower(host), "[]")
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}

func main() {
	listenAddr := flag.String("listen", ":8088", "listen address")
	mqttEndpoint := flag.String("mqtt", "127.0.0.1:1883", "MQTT endpoint host:port")
	deviceID := flag.String("device", "vcan-e2e", "device id")
	topicPrefix := flag.String("topic-prefix", "ctrl", "rx topic prefix")
	mapConfig := flag.String("map-config", "", "optional YAML path for frame_id and signal mapping")
	busID := flag.String("bus", "can0", "target bus_id")
	frameID := flag.String("frame", "", "base frame_id")
	frameSteer := flag.String("frame-steer", "", "optional frame_id for steering signals")
	frameTrigger := flag.String("frame-trigger", "", "optional frame_id for trigger signals")
	frameButtons := flag.String("frame-buttons", "", "optional frame_id for button signals")
	flag.Parse()

	topic := mq.BuildTopic("rx", *deviceID, *topicPrefix)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(normalizeEndpoint(*mqttEndpoint))
	opts.SetClientID(fmt.Sprintf("gamepad-web-%d", time.Now().UnixNano()))
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(500 * time.Millisecond)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(fmt.Errorf("failed to connect MQTT: %w", token.Error()))
	}
	defer client.Disconnect(200)

	mapOpts := gamepad.MultiMapOptions{}
	if *mapConfig != "" {
		loaded, err := gamepad.LoadMultiMapOptionsFromYAML(*mapConfig)
		if err != nil {
			panic(fmt.Errorf("failed to load map config: %w", err))
		}
		mapOpts = loaded
	}

	if *frameID != "" {
		mapOpts.BaseFrameID = *frameID
	}
	if *frameSteer != "" {
		mapOpts.SteerFrameID = *frameSteer
	}
	if *frameTrigger != "" {
		mapOpts.TriggerFrameID = *frameTrigger
	}
	if *frameButtons != "" {
		mapOpts.ButtonsFrameID = *frameButtons
	}
	if mapOpts.BaseFrameID == "" && mapOpts.GamepadMsgFrameID == "" && mapOpts.AllAxes0FrameID == "" && mapOpts.AllAxes1FrameID == "" && mapOpts.AllButtonsPressedFrameID == "" && mapOpts.AllButtonsValue0FrameID == "" && mapOpts.AllButtonsValue1FrameID == "" {
		mapOpts.BaseFrameID = "0x2"
	}

	s := &server{
		mapper:  gamepad.NewMapper(),
		client:  client,
		topic:   topic,
		busID:   *busID,
		mapOpts: mapOpts,
	}

	assetsSub, err := fs.Sub(webAssets, "assets")
	if err != nil {
		panic(fmt.Errorf("failed to load embedded assets: %w", err))
	}
	assetsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=3600")
		}
		http.StripPrefix("/assets/", http.FileServer(http.FS(assetsSub))).ServeHTTP(w, r)
	})

	http.HandleFunc("/", s.handleIndex)
	http.Handle("/assets/", assetsHandler)
	http.HandleFunc("/ws", s.handleWS)
	http.HandleFunc("/api/gamepad", s.handleAPI)

	fmt.Printf("gamepad-web listening on %s\n", *listenAddr)
	fmt.Printf("publishing MQTT topic: %s\n", topic)
	if err := http.ListenAndServe(*listenAddr, nil); err != nil {
		panic(err)
	}
}

func normalizeEndpoint(endpoint string) string {
	e := strings.TrimSpace(endpoint)
	if strings.HasPrefix(e, "tcp://") || strings.HasPrefix(e, "ssl://") || strings.HasPrefix(e, "tls://") || strings.HasPrefix(e, "ws://") || strings.HasPrefix(e, "wss://") {
		return e
	}
	return "tcp://" + e
}

func (s *server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	indexBody, err := fs.ReadFile(webAssets, "assets/index.html")
	if err != nil {
		http.Error(w, "failed to load index", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexBody)
}

func (s *server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer conn.Close()

	for {
		var st gamepad.State
		if err := conn.ReadJSON(&st); err != nil {
			return
		}
		published, err := s.publishState(st)
		if err != nil {
			_ = conn.WriteJSON(map[string]interface{}{"ok": false, "error": err.Error()})
			continue
		}
		if len(published) == 1 {
			_ = conn.WriteJSON(published[0])
			continue
		}
		_ = conn.WriteJSON(published)
	}
}

func (s *server) handleAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var st gamepad.State
	if err := json.NewDecoder(r.Body).Decode(&st); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	published, err := s.publishState(st)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if len(published) == 1 {
		_ = json.NewEncoder(w).Encode(published[0])
		return
	}
	_ = json.NewEncoder(w).Encode(published)
}

func (s *server) publishState(st gamepad.State) ([]controlPayload, error) {
	mapped := s.mapper.ToCommandMaps(st, s.mapOpts)
	published := make([]controlPayload, 0, len(mapped))
	for _, cmd := range mapped {
		payload := controlPayload{
			Timestamp: time.Now().Format(time.RFC3339),
			BusID:     s.busID,
			FrameID:   cmd.FrameID,
			Signals:   cmd.Signals,
		}

		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		token := s.client.Publish(s.topic, 0, false, b)
		if token.Wait() && token.Error() != nil {
			return nil, token.Error()
		}
		published = append(published, payload)
	}
	return published, nil
}
