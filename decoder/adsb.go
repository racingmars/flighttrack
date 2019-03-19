package decoder

import (
	"fmt"
	"math"
)

func getADSBType(b byte) (int, adsbMessageType) {
	tc := int((b & 0xF8) >> 3)
	typeStr := msgUnknown
	if int(tc-1) < len(adsbTypes) {
		typeStr = adsbTypes[tc-1]
	}
	return int(tc), typeStr
}

func getAdsbIdentification(data []byte) AdsbIdentification {
	result := AdsbIdentification{}
	result.TC = int(data[0] & 0xF8 >> 3)
	result.EC = int(data[0] & 0x07)

	cs1raw := data[1] & 0xfc >> 2
	cs2raw := data[1]&0x03<<4 | data[2]&0xf0>>4
	cs3raw := data[2]&0xf<<2 | data[3]&0xf0>>6
	cs4raw := data[3] & 0x3f
	cs5raw := data[4] & 0xfc >> 2
	cs6raw := data[4]&0x03<<4 | data[5]&0xf0>>4
	cs7raw := data[5]&0xf<<2 | data[6]&0xf0>>6
	cs8raw := data[6] & 0x3f

	result.Callsign += string([]rune{charmap[cs1raw], charmap[cs2raw], charmap[cs3raw],
		charmap[cs4raw], charmap[cs5raw], charmap[cs6raw], charmap[cs7raw], charmap[cs8raw]})

	return result
}

func getAdsbVelocity(data []byte) AdsbVelocity {
	result := AdsbVelocity{}
	result.TC = int(data[0] & 0xF8 >> 3)
	result.ST = int(data[0] & 0x07)

	if data[1]&0x80 > 0 {
		result.IntentChange = true
	}

	switch result.ST {
	case 1:
		fillAdsbVelocityGroundspeed(data, &result)
	case 3:
		fillAdsbVelocityAirspeed(data, &result)
	default:
		panic(fmt.Errorf("Bad velocity subtype: %d", result.ST))
	}

	sVR := data[4] & 0x08 >> 3
	vr := (int(data[4]) & 0x07 << 6) | (int(data[5]) & 0xfc >> 2)
	vr = (vr - 1) * 64
	if sVR == 1 {
		vr = -1 * vr
	}

	result.VerticalRate = vr

	return result
}

func fillAdsbVelocityGroundspeed(data []byte, result *AdsbVelocity) {
	sEW := data[1] & 0x04 >> 2
	vEW := (int(data[1]) & 0x03 << 8) | int(data[2])
	sNS := data[3] & 0x80 >> 7
	vNS := (int(data[3]) & 0x7f << 3) | (int(data[4]) & 0xe0 >> 5)

	vWE := float64(vEW - 1)
	if sEW == 1 {
		vWE = -1 * vWE
	}

	vSN := float64(vNS - 1)
	if sNS == 1 {
		vSN = -1 * vSN
	}

	v := math.Sqrt(vWE*vWE + vSN*vSN)
	h := math.Atan2(vWE, vSN) * (360 / (2 * math.Pi))
	if h < 0 {
		h = h + 360
	}

	result.SpeedType = SpeedGS
	result.Speed = int(math.Round(v))
	result.HeadingAvailable = true
	result.Heading = int(math.Round(h))
}

func fillAdsbVelocityAirspeed(data []byte, result *AdsbVelocity) {
	sHdg := data[1] & 0x04 >> 2
	hdg := (int(data[1]) & 0x03 << 8) | int(data[2])
	asT := data[3] & 0x80 >> 7
	as := (int(data[3]) & 0x7f << 3) | (int(data[4]) & 0xe0 >> 5)

	if asT == 0 {
		result.SpeedType = SpeedIAS
	} else {
		result.SpeedType = SpeedTAS
	}
	result.Speed = as
	if sHdg == 1 {
		result.HeadingAvailable = true
		result.Heading = int(math.Round(float64(hdg) / 1024.0 * 360.0))
	}
}
