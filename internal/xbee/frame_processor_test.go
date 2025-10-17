package xbee

import (
	"encoding/hex"
	"strings"
	"testing"
)

func decodeHexString(t *testing.T, s string) []byte {
	t.Helper()
	cleaned := strings.ReplaceAll(s, " ", "")
	data, err := hex.DecodeString(cleaned)
	if err != nil {
		t.Fatalf("failed to decode hex string: %v", err)
	}
	return data
}

func TestProcessByteExplicitRXEscaped(t *testing.T) {
	sample := "7E 00 44 91 00 7D 33 A2 00 42 36 7D 5E BB D5 93 E8 E8 00 03 C1 05 01 43 4D 44 5F 45 43 48 4F 3A 4D 43 55 3A 52 41 4D 5F 46 52 45 45 3A 36 34 2E 30 20 4B 42 3A 55 50 54 49 4D 45 3A 37 30 30 3A 54 45 4D 50 5F 43 3A 35 34 B7"

	fp := NewFrameProcessor()
	var received *XBeeFrameData
	fp.SetFrameHandler(func(frame XBeeFrameData) error {
		frameCopy := frame
		received = &frameCopy
		return nil
	})

	data := decodeHexString(t, sample)

	if err := fp.ProcessByte(data); err != nil {
		t.Fatalf("ProcessByte returned error: %v", err)
	}

	if received == nil {
		t.Fatal("expected frame to be received")
	}

	if received.FrameType != 0x91 {
		t.Fatalf("unexpected frame type: got %X", received.FrameType)
	}

	if received.ExplicitMetadata == nil {
		t.Fatal("expected explicit metadata to be populated")
	}

	if received.ExplicitMetadata.ClusterId != packetTypeCmdResponse {
		t.Fatalf("unexpected cluster ID: got 0x%04X", received.ExplicitMetadata.ClusterId)
	}

	if received.PacketType != "CMD_RESPONSE" {
		t.Fatalf("unexpected packet type: got %s", received.PacketType)
	}

	expectedPrefix := "CMD_ECHO:MCU:RAM_FREE"
	if !strings.HasPrefix(received.Data, expectedPrefix) {
		t.Fatalf("unexpected data prefix: got %q", received.Data)
	}
}
