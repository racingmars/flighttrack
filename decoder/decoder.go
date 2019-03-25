package decoder

import (
	"encoding/hex"
	"time"

	"github.com/rs/zerolog/log"
)

func DecodeMessage(msg []byte, tm time.Time) (string, interface{}) {
	df, _ := GetDFCA(msg[0])
	//fmt.Printf("DF/CA: %d/%d", df, ca)

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
		//fmt.Printf(", ICAO ID: %s", hex.EncodeToString(icaoid))
		return hex.EncodeToString(icaoid), nil
	}

	//fmt.Printf("\n")
	return "", nil
}
