package main

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/racingmars/flighttrack/beast"
	"github.com/racingmars/flighttrack/consolehandler"
	"github.com/racingmars/flighttrack/decoder"
	"github.com/racingmars/flighttrack/tracker"
)

func main() {
	rdr := beast.New(os.Stdin)
	tracker := tracker.New(new(consolehandler.ConsoleHandler), false)
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
		icao, decoded := decoder.DecodeMessage(msg.Message, time.Now().UTC())
		if icao != "" && icao != "000000" {
			tracker.Message(icao, time.Now(), decoded)
		}
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
