//go:build windows

// Command loopback captures what the default output device is playing through
// WASAPI loopback and prints its level.
//
// It requires a PortAudio build newer than v19.7.0.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/raa0121/purego-portaudio"
)

func main() {
	lib := flag.String("lib", "", "path to the PortAudio shared library")
	duration := flag.Duration("duration", 10*time.Second, "how long to capture")
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

	wasapi, err := portaudio.HostAPIIndexOf(portaudio.WASAPI)
	if err != nil {
		log.Fatal(err)
	}
	api, err := portaudio.HostAPI(wasapi)
	if err != nil {
		log.Fatal(err)
	}
	loopback, err := portaudio.WASAPILoopbackOf(api.DefaultOutputDevice)
	if err != nil {
		log.Fatal(err)
	}
	device, err := portaudio.Device(loopback)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("capturing %q at %.0f Hz\n", device.Name, device.DefaultSampleRate)

	channels := device.MaxInputChannels
	stream, err := portaudio.OpenStream(&portaudio.StreamParameters{
		Device:           loopback,
		ChannelCount:     channels,
		SampleFormat:     portaudio.Float32,
		SuggestedLatency: device.DefaultLowInputLatency,
	}, nil, device.DefaultSampleRate, portaudio.FramesPerBufferUnspecified, portaudio.NoFlag, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()
	if err := stream.Start(); err != nil {
		log.Fatal(err)
	}

	// When no application is rendering audio, loopback may deliver no data and
	// ReadStream blocks until something plays.
	frames := int(device.DefaultSampleRate / 30)
	buf := make([]float32, frames*channels)
	for end := time.Now().Add(*duration); time.Now().Before(end); {
		if err := portaudio.ReadStream(stream, buf); err != nil && !errors.Is(err, portaudio.ErrInputOverflowed) {
			log.Fatal(err)
		}
		var sum float64
		for _, s := range buf {
			sum += float64(s) * float64(s)
		}
		db := 10 * math.Log10(sum/float64(len(buf))+1e-12)
		bar := int(math.Max(0, (db+60)/60*50))
		fmt.Printf("\r%6.1f dBFS |%-50s|", db, strings.Repeat("#", min(bar, 50)))
	}
	fmt.Println()
}
