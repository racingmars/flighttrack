package decoder

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

func DecodeMessage(msg []byte, tm time.Time) (string, interface{}) {
	df := msg[0] & 0xF8 >> 3
	//fmt.Printf("DF: %d", df)

	if df == 17 || df == 18 {
		icaoid := msg[1:4]
		//fmt.Printf(", ICAO ID: %s", hex.EncodeToString(icaoid))

		if !CheckCRC(msg) {
			log.Warn().Msgf("parity failed for message from %s", hex.EncodeToString(icaoid))
			return hex.EncodeToString(icaoid), nil
		}
		// if CheckCRC(msg) {
		// 	fmt.Printf(", parity passed")
		// } else {
		// 	fmt.Printf(", parity FAILED")
		// }

		_, typeStr := getADSBType(msg[4])
		//fmt.Printf(". ADS-B: %d - %s", typeCode, typeStr)

		if typeStr == msgAircraftID {
			id := getAdsbIdentification(msg[4:])
			//fmt.Printf(" | IDENT: %s (%d)", id.Callsign, id.EC)
			return hex.EncodeToString(icaoid), &id
		}

		if typeStr == msgAirbornVelocities {
			vel := getAdsbVelocity(msg[4:])
			// var speedtype string
			// switch vel.SpeedType {
			// case SpeedGS:
			// 	speedtype = "groundspeed"
			// case SpeedIAS:
			// 	speedtype = "indicated airspeed"
			// case SpeedTAS:
			// 	speedtype = "true airspeed"
			// }
			//fmt.Printf(" | HDG: %03d ; VS: %5dfpm ; SPEED: %3dkt (%s)", vel.Heading, vel.VerticalRate, vel.Speed, speedtype)
			return hex.EncodeToString(icaoid), &vel
		}

		if typeStr == msgAirbornPosWithBaroAlt {
			pos := getAdsbPosition(msg[4:], tm)
			//fmt.Printf(" | ALT: %dft", pos.Altitude)
			//fmt.Printf(" | LatCPR: %6d | LonCPR: %6d | Frame: %d", pos.LatCPR, pos.LonCPR, pos.Frame)
			return hex.EncodeToString(icaoid), &pos
		}

	} else if df == 20 || df == 21 {
		crc := CalcCRC(msg)
		origcrc := msg[len(msg)-3:]
		icaoid := []byte{crc[0] ^ origcrc[0], crc[1] ^ origcrc[1], crc[2] ^ origcrc[2]}
		if msg[4] == 0x20 {
			fmt.Printf("Potential aircraft identification: ")
			ident := getAdsbIdentification(msg[4:])
			fmt.Printf("%s\n", ident.Callsign)
		} else {
			tryTypeDecode(icaoid, msg[4:])
		}

		//tryTypeDecode(icaoid, msg[4:])
		//fmt.Printf(", ICAO ID: %s", hex.EncodeToString(icaoid))
		return hex.EncodeToString(icaoid), nil
	}

	//fmt.Printf("\n")
	return "", nil
}

func tryTypeDecode(icaoid, msg []byte) {
	if msg[6]&0x1F != 0 {
		// data in reserved bits; ignore
		return
	}

	chars := make([]byte, 5)
	chars[0] = (msg[1] & 0x01 << 5) | (msg[2] & 0xf8 >> 3)
	chars[1] = (msg[2] & 0x07 << 3) | (msg[3] & 0xe0 >> 5)
	chars[2] = (msg[3] & 0x1f << 1) | (msg[4] & 0x80 >> 7)
	chars[3] = (msg[4] & 0x7e >> 1)
	chars[4] = (msg[4] & 0x01 << 5) | (msg[5] & 0xf8 >> 3)

	candidateType := string([]rune{charmap[chars[0]], charmap[chars[1]], charmap[chars[2]],
		charmap[chars[3]], charmap[chars[4]]})
	for _, c := range chars {
		if !((c >= 1 && c <= 26) || (c >= 48 && c <= 57) || c == 32) {
			// not a valid character
			fmt.Printf("Rejected (bad chars) %s\n", candidateType)
			return
		}
	}

	fmt.Printf("Candidate type for %s: %s\n", hex.EncodeToString(icaoid), candidateType)
}
