package decoder

import (
	"encoding/hex"
	"testing"
)

func TestIdentification(t *testing.T) {
	msg, _ := hex.DecodeString("8D4840D6202CC371C32CE0576098")
	result := getAdsbIdentification(msg[4:])
	if result.Callsign != "KLM1023 " {
		t.Errorf("Unexpected callsign: \"%s\" (should be \"KLM1023 \")", result.Callsign)
	}
}
