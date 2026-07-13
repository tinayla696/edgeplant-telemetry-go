package canhandler

import (
	"path/filepath"
	"testing"
)

func TestParseDbcFile(t *testing.T) {
	can0Path := filepath.Join("..", "..", "..", "config", "can0.dbc")
	store0, err := ParseDbcFile(can0Path)
	if err != nil {
		t.Fatalf("ParseDbcFile(can0) failed: %v", err)
	}
	if _, ok := store0.Msgs[1]; !ok {
		t.Fatalf("expected frame 1 in can0 DBC")
	}
	if _, ok := store0.Msgs[2]; !ok {
		t.Fatalf("expected frame 2 in can0 DBC")
	}

	can1Path := filepath.Join("..", "..", "..", "config", "can1.dbc")
	store1, err := ParseDbcFile(can1Path)
	if err != nil {
		t.Fatalf("ParseDbcFile(can1) failed: %v", err)
	}
	if _, ok := store1.Msgs[257]; !ok {
		t.Fatalf("expected frame 257 in can1 DBC")
	}
	if _, ok := store1.Msgs[258]; !ok {
		t.Fatalf("expected frame 258 in can1 DBC")
	}
}
