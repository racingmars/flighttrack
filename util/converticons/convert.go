package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Marker struct {
	Svg  string `json:"svg"`
	Size []int  `json:"size"`
}

const color = "rgb(0,0,200)"

func main() {
	f, _ := os.Open("markers.json")
	defer f.Close()

	markers := make(map[string]Marker)

	d := json.NewDecoder(f)
	err := d.Decode(&markers)
	if err != nil {
		panic(err)
	}

	for name := range markers {
		f, err := os.Create(fmt.Sprintf("../../web/static/icons/%s.svg", name))
		if err != nil {
			panic(err)
		}
		defer f.Close()
		svg := strings.ReplaceAll(markers[name].Svg, "add_stroke_selected", "")
		svg = strings.ReplaceAll(svg, "aircraft_color_fill", color)
		svg = strings.ReplaceAll(svg, "aircraft_color_stroke", color)
		_, err = f.WriteString(svg)
		if err != nil {
			panic(err)
		}
	}

}
