package decoder

import (
	"fmt"
	"math"
	"os"
	"time"
)

func getADSBType(b byte) (int, adsbMessageType) {
	tc := int((b & 0xF8) >> 3)
	typeStr := msgUnknown
	if int(tc) < len(adsbTypes) {
		typeStr = adsbTypes[tc]
	}
	return int(tc), typeStr
}

func getAdsbIdentification(data []byte) AdsbIdentification {
	result := AdsbIdentification{}
	typecode := int(data[0] & 0xF8 >> 3)
	emittercat := int(data[0] & 0x07)

	// Format type and aircraft category from ICAO Doc 9871 Table A-2-8
	switch typecode {
	case 1: // category set D
		result.Type = ACTypeUnknown
	case 2: // category set C
		switch emittercat {
		case 0:
			result.Type = ACTypeNoInfo
		case 1:
			result.Type = ACTypeSurfaceEmergency
		case 2:
			result.Type = ACTypeSurfaceService
		case 3:
			result.Type = ACTypeObstruction
		case 4:
			result.Type = ACTypeClusterObstacle
		case 5:
			result.Type = ACTypeLineObstacle
		default:
			result.Type = ACTypeUnknown
		}
	case 3: // category set B
		switch emittercat {
		case 0:
			result.Type = ACTypeNoInfo
		case 1:
			result.Type = ACTypeGlider
		case 2:
			result.Type = ACTypeLighterThanAir
		case 3:
			result.Type = ACTypeParachutist
		case 4:
			result.Type = ACTypeUltralight
		case 6:
			result.Type = ACTypeUAV
		case 7:
			result.Type = ACTypeSpaceVehicle
		default:
			result.Type = ACTypeUnknown
		}
	case 4: // category set A
		switch emittercat {
		case 0:
			result.Type = ACTypeNoInfo
		case 1:
			result.Type = ACTypeLight
		case 2:
			result.Type = ACTypeSmall
		case 3:
			result.Type = ACTypeLarge
		case 4:
			result.Type = ACTypeHighVortexLarge
		case 5:
			result.Type = ACTypeHeavy
		case 6:
			result.Type = ACTypeHighPerformance
		case 7:
			result.Type = ACTypeRotocraft
		default:
			result.Type = ACTypeUnknown
		}
	default:
		result.Type = ACTypeUnknown
	}

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
		fmt.Fprintf(os.Stderr, "Unexpected velocity subtype: %d\n", result.ST)
	}

	sVR := data[4] & 0x08 >> 3
	vr := (int(data[4]) & 0x07 << 6) | (int(data[5]) & 0xfc >> 2)
	if vr != 0 {
		result.VerticalRateAvailable = true
		vr = (vr - 1) * 64
		if sVR == 1 {
			vr = -1 * vr
		}
		result.VerticalRate = vr
	}

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

func getAdsbPosition(msg []byte, tm time.Time) AdsbPosition {
	result := AdsbPosition{Timestamp: tm}
	result.TC = int(msg[0] & 0xF8 >> 3)
	result.SS = int(msg[0] & 0x06 >> 1)

	var alt int
	q := msg[1] & 0x01
	if q == 1 {
		alt = (int(msg[1]) & 0xfe << 3) | (int(msg[2]) & 0xf0 >> 4)
		result.Altitude = alt*25 - 1000
	} else {
		// The altitude bits are encoded in the following order:
		// C1 A1 C2 A2 C4 A4 B1 [Q] B2 D2 B4 D4
		// And need to be rearranged to:
		// D2 D4 A1 A2 A4 B1 B2 B4 C1 C2 C4
		alt = int(msg[2]&0x40)<<4 | // D2
			int(msg[2]&0x10)<<5 | // D4
			int(msg[1]&0x40)<<2 | // A1
			int(msg[1]&0x10)<<3 | // A2
			int(msg[1]&0x04)<<4 | // A4
			int(msg[1]&0x02)<<5 | // B1
			int(msg[2]&0x02)>>3 | // B2
			int(msg[2]&0x20)>>2 | // B4
			int(msg[1]&0x80)>>5 | // C1
			int(msg[1]&0x20)>>4 | // C2
			int(msg[1]&0x08)>>3 // C4
		result.Altitude = gillhamToAltitude(alt)
	}

	result.Frame = int(msg[2]) & 0x04 >> 2
	result.LatCPR = (int(msg[2]) & 0x03 << 15) | (int(msg[3]) << 7) | (int(msg[4]) & 0xfe >> 1)
	result.LonCPR = (int(msg[4]) & 0x01 << 16) | (int(msg[5]) << 8) | (int(msg[6]))

	return result
}

const dLatEven float64 = 360.0 / 60.0
const dLatOdd float64 = 360.0 / 59.0

func CalcPosition(oddFrame, evenFrame AdsbPosition) (float64, float64, bool) {
	//fmt.Printf("ELat(%d) OLat(%d) ELon(%d) OLon(%d)\n", evenFrame.LatCPR, oddFrame.LatCPR, evenFrame.LonCPR, oddFrame.LonCPR)

	cprLatEven := float64(evenFrame.LatCPR) / 131072
	cprLonEven := float64(evenFrame.LonCPR) / 131072
	cprLatOdd := float64(oddFrame.LatCPR) / 131072
	cprLonOdd := float64(oddFrame.LonCPR) / 131072

	j := math.Floor(59*float64(cprLatEven) - 60*float64(cprLatOdd) + 0.5)
	latEven := dLatEven * (mod(j, 60) + cprLatEven)
	latOdd := dLatOdd * (mod(j, 59) + cprLatOdd)
	if latEven >= 270 {
		latEven = latEven - 360
	}
	if latOdd >= 270 {
		latOdd = latOdd - 360
	}

	var lat float64
	if oddFrame.Timestamp.Before(evenFrame.Timestamp) {
		lat = latEven
	} else {
		lat = latOdd
	}

	if nl(latEven) != nl(latOdd) {
		return 0, 0, false
	}

	// Longitude
	var lon float64
	if oddFrame.Timestamp.Before(evenFrame.Timestamp) {
		ni := math.Max(float64(nl(latEven)), 1)
		dLon := 360.0 / ni
		m := math.Floor(cprLonEven*(float64(nl(latEven))-1) - cprLonOdd*float64(nl(latEven)) + 0.5)
		lon = dLon*mod(m, ni) + cprLonEven
	} else {
		ni := math.Max(float64(nl(latOdd))-1, 1)
		dLon := 360.0 / ni
		m := math.Floor(cprLonEven*(float64(nl(latOdd))-1) - cprLonOdd*float64(nl(latOdd)) + 0.5)
		lon = dLon * (mod(m, ni) + cprLonOdd)
	}
	if lon >= 180 {
		lon = lon - 360
	}

	return lat, lon, true
}

func nl(lat float64) int {
	result := (2 * math.Pi) /
		math.Acos(1-((1-math.Cos(math.Pi/30))/(math.Pow(math.Cos(math.Pi/180*lat), 2))))
	return int(math.Floor(result))
}

func mod(x, y float64) float64 {
	return x - y*math.Floor(x/y)
}
