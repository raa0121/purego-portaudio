// Command devices lists the host APIs and devices PortAudio can see.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/raa0121/purego-portaudio"
)

func main() {
	lib := flag.String("lib", "", "path to the PortAudio shared library")
	flag.Parse()

	if *lib != "" {
		if err := portaudio.Load(*lib); err != nil {
			log.Fatal(err)
		}
	}
	if err := portaudio.Initialize(); err != nil {
		log.Fatal(err)
	}
	defer portaudio.Terminate()

	v, err := portaudio.Version()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(v.Text)

	apis, err := portaudio.HostAPIs()
	if err != nil {
		log.Fatal(err)
	}
	devices, err := portaudio.Devices()
	if err != nil {
		log.Fatal(err)
	}

	for _, api := range apis {
		fmt.Printf("\n%s (%v)\n", api.Name, api.Type)
		for _, d := range devices {
			if d.HostAPI != api.Index || (d.MaxInputChannels == 0 && d.MaxOutputChannels == 0) {
				continue
			}
			mark := " "
			if d.Index == api.DefaultInputDevice || d.Index == api.DefaultOutputDevice {
				mark = "*"
			}
			fmt.Printf("%s %3d  %-50s in:%-2d out:%-2d %6.0f Hz\n",
				mark, d.Index, d.Name, d.MaxInputChannels, d.MaxOutputChannels, d.DefaultSampleRate)
		}
	}
}
