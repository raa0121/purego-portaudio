// Command sine plays a sine wave on the default output device using a stream
// callback.
package main

import (
	"flag"
	"log"
	"math"
	"time"

	"github.com/raa0121/purego-portaudio"
)

const sampleRate = 48000

func main() {
	lib := flag.String("lib", "", "path to the PortAudio shared library")
	freq := flag.Float64("freq", 440, "frequency in Hz")
	duration := flag.Duration("duration", 2*time.Second, "how long to play")
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

	var phase float64
	step := 2 * math.Pi * *freq / sampleRate
	stream, err := portaudio.OpenDefaultStream(0, 2, portaudio.Float32, sampleRate, portaudio.FramesPerBufferUnspecified,
		func(_, out portaudio.Buffer, _ portaudio.StreamCallbackTimeInfo, _ portaudio.StreamCallbackFlags) portaudio.StreamCallbackResult {
			samples := portaudio.Samples[float32](out)
			for i := 0; i < len(samples); i += 2 {
				v := float32(0.2 * math.Sin(phase))
				samples[i], samples[i+1] = v, v
				phase = math.Mod(phase+step, 2*math.Pi)
			}
			return portaudio.Continue
		})
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	if err := stream.Start(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(*duration)
	if err := stream.Stop(); err != nil {
		log.Fatal(err)
	}
}
