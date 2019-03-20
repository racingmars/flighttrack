package main

import (
	"encoding/hex"
	"io"
	"log"
	"os"

	"github.com/racingmars/flighttrack/beast"
	"github.com/racingmars/flighttrack/decoder"
)

func main() {
	rdr := beast.New(os.Stdin)
	for {
		msg, startoffset, err := rdr.Read()
		if err == io.EOF {
			break
		}
		if _, ok := err.(beast.UnknownFormatError); ok {
			log.Print(startoffset, err)
			continue
		}
		if err != nil {
			log.Fatal(startoffset, err)
		}
		//fmt.Println(hex.EncodeToString(msg.Message))
		decoder.DecodeMessage(hex.EncodeToString(msg.Message))
	}
}

/*
// This alternate main function reads "AVR" format messages instead of beast
func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		msg := scanner.Text()
		msg = msg[1 : len(msg)-1]
		decoder.DecodeMessage(msg)
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}
*/
