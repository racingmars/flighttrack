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
	if result.VerticalRate != -832 {
		t.Errorf("Bad vertical rate: %d should be -832", result.VerticalRate)
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

func TestAltitude(t *testing.T) {
	msg, _ := hex.DecodeString("8D40621D58C382D690C8AC2863A7")
	result := getAdsbPosition(msg[4:])
	if result.Altitude != 38000 {
		t.Errorf("Bad altitude: %d should be 38000", result.Altitude)
	}
}

func TestAltitudeQZero(t *testing.T) {
	msg, _ := hex.DecodeString("59a6a5b819fde2e7cfb1")
	result := getAdsbPosition(msg)
	if result.Altitude != 6100 {
		t.Errorf("Bad altitude: %d should be 6100", result.Altitude)
	}
}

func TestPosition(t *testing.T) {
	msg, _ := hex.DecodeString("8D40621D58C382D690C8AC2863A7")
	resultEven := getAdsbPosition(msg[4:])
	if resultEven.LatCPR != 93000 {
		t.Errorf("Bad LAT-CPR: %d should be 93000", resultEven.LatCPR)
	}
	if resultEven.LonCPR != 51372 {
		t.Errorf("Bad LON-CPR: %d should be 51372", resultEven.LonCPR)
	}

	msg, _ = hex.DecodeString("8D40621D58C386435CC412692AD6")
	resultOdd := getAdsbPosition(msg[4:])
	if resultOdd.LatCPR != 74158 {
		t.Errorf("Bad LAT-CPR: %d should be 74158", resultEven.LatCPR)
	}
	if resultOdd.LonCPR != 50194 {
		t.Errorf("Bad LON-CPR: %d should be 50194", resultEven.LonCPR)
	}

	calcPosition(resultOdd, resultEven)
}
