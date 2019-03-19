package decoder

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
	result.TC = int((data[0] & 0xF8) >> 3)
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
