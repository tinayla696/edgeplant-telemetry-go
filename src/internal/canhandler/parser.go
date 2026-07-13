package canhandler

import (
	"os"

	"go.einride.tech/can/pkg/dbc"
	"go.einride.tech/can/pkg/descriptor"
)

// DBC File Parser and Store
type DbcStore struct {
	Version string
	Msgs    map[uint32]*descriptor.Message
}

// Parse DBC file and return a DbcStore
func ParseDbcFile(path string) (*DbcStore, error) {
	// read the DBC file
	datas, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// parse the DBC file
	parser := dbc.NewParser("", datas)
	if err := parser.Parse(); err != nil {
		return nil, err
	}

	// make store
	store := &DbcStore{
		Msgs: make(map[uint32]*descriptor.Message),
	}

	// populate store with messages
	for _, def := range parser.Defs() {
		switch def := def.(type) {
		case *dbc.VersionDef:
			store.Version = def.Version
		case *dbc.MessageDef:
			msg := &descriptor.Message{
				Name:       string(def.Name),
				ID:         def.MessageID.ToCAN(),
				IsExtended: def.MessageID.IsExtended(),
				Length:     uint8(def.Size),
				SenderNode: string(def.Transmitter),
				Signals:    make([]*descriptor.Signal, 0, len(def.Signals)),
			}
			// (signalのパース処理は変更なし)
			for _, sigDef := range def.Signals {
				signal := &descriptor.Signal{
					Name:             string(sigDef.Name),
					IsBigEndian:      sigDef.IsBigEndian,
					IsSigned:         sigDef.IsSigned,
					IsMultiplexer:    sigDef.IsMultiplexerSwitch,
					IsMultiplexed:    sigDef.IsMultiplexed,
					MultiplexerValue: uint(sigDef.MultiplexerSwitch),
					Start:            uint8(sigDef.StartBit),
					Length:           uint8(sigDef.Size),
					Scale:            sigDef.Factor,
					Offset:           sigDef.Offset,
					Min:              sigDef.Minimum,
					Max:              sigDef.Maximum,
					Unit:             string(sigDef.Unit),
					ReceiverNodes:    make([]string, len(sigDef.Receivers)),
				}
				for i, receiver := range sigDef.Receivers {
					signal.ReceiverNodes[i] = string(receiver)
				}
				msg.Signals = append(msg.Signals, signal)
			}
			// msg.ID は uint32 なので、正しくマップに格納できる
			store.Msgs[msg.ID] = msg
		}
	}
	return store, nil
}
