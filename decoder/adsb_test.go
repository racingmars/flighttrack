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

func TestGroundspeed(t *testing.T) {
	msg, _ := hex.DecodeString("8D485020994409940838175B284F")
	result := getAdsbVelocity(msg[4:])
	if result.Heading != 183 {
		t.Errorf("Bad heading: %d should be 183", result.Heading)
	}
	if result.Speed != 159 {
		t.Errorf("Bad speed: %d should be 159", result.Speed)
	}
}

func TestAirspeed(t *testing.T) {
	msg, _ := hex.DecodeString("8DA05F219B06B6AF189400CBC33F")
	result := getAdsbVelocity(msg[4:])
	if result.Heading != 244 {
		t.Errorf("Bad heading: %d should be 244", result.Heading)
	}
	if result.Speed != 376 {
		t.Errorf("Bad speed: %d should be 376", result.Speed)
	}
}
