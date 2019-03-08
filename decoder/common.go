package decoder

import (
	"encoding/binary"
)

func GetDFCA(b byte) (int, int) {
	df, _ := binary.Uvarint([]byte{(b & 0xF8) >> 3})
	ca, _ := binary.Uvarint([]byte{b & 0x07})
	return int(df), int(ca)
}

func CheckCRC(msg []byte) bool {
	generator := []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 1}
	binmsg := bytesToBinary(msg)
	for i := 0; i < len(binmsg)-24; i++ {
		if binmsg[i] == 1 {
			for j := 0; j <= 24; j++ {
				binmsg[i+j] = binmsg[i+j] ^ generator[j]
			}
		}
	}
	for _, bit := range binmsg[len(binmsg)-24:] {
		if bit != 0 {
			return false
		}
	}
	return true
}

func CalcCRC(msg []byte) []byte {
	generator := []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 1}
	binmsg := bytesToBinary(msg)
	for i := len(binmsg) - 24; i < len(binmsg); i++ {
		binmsg[i] = 0
	}
	for i := 0; i < len(binmsg)-24; i++ {
		if binmsg[i] == 1 {
			for j := 0; j <= 24; j++ {
				binmsg[i+j] = binmsg[i+j] ^ generator[j]
			}
		}
	}
	return binaryToBytes(binmsg[len(binmsg)-24:])
}

func bytesToBinary(data []byte) []int {
	var result []int
	for _, b := range data {
		for i := 7; i >= 0; i-- {
			mask := 1 << uint(i)
			thisbit := int(b) & mask
			if thisbit > 0 {
				result = append(result, 1)
			} else {
				result = append(result, 0)
			}
		}
	}
	return result
}

func binaryToBytes(data []int) []byte {
	var result []byte
	bytes := len(data) / 8
	for i := 0; i < bytes; i++ {
		newbyte := data[i*8+0]<<7 +
			data[i*8+1]<<6 +
			data[i*8+2]<<5 +
			data[i*8+3]<<4 +
			data[i*8+4]<<3 +
			data[i*8+5]<<2 +
			data[i*8+6]<<1 +
			data[i*8+7]
		result = append(result, byte(newbyte))
	}
	return result
}
