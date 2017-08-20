package main

import (
	"github.com/tjclement/pixelflood_client"
	"image"
	_ "image/jpeg"
	_ "image/png"
	_ "image/gif"
	"os"
	"fmt"
	"github.com/nfnt/resize"
	"flag"
	"time"
	"os/signal"
	"io"
	"strings"
	"image/gif"
)

func main() {
	image_file := flag.String("image", "./image.jpg", "X value of the screen to begin drawing on")
	x_start := flag.Int("x_start", 0, "X value of the screen to begin drawing on")
	y_start := flag.Int("y_start", 0, "y value of the screen to begin drawing on")
	x_size := flag.Int("x_size", 320, "X size of the image we want to draw on the screen")
	y_size := flag.Int("y_size", 320, "Y size of the image we want to draw on the screen")
	worker_amount := flag.Int("worker_amount", 16, "The amount of workers that concurrently sends pixel values to the server")
	worker_type := flag.String("worker_type", "squares", "The type of pixel replacement strategy ['random', 'squares']")
	screen := flag.String("screen", "", "The screen to target ['topleft', 'center', 'topright', 'externalright']. Replaces manual x,y start and size.")
	ignore_black := flag.Bool("ignore_black", false, "Set to true to skip sending RGB(0, 0, 0) pixels. This allows for partially transparent images on the screen.")
	flag.Parse()

	if len(*screen) > 0 {
		switch *screen {
		case "center":
			*x_start, *y_start, *x_size, *y_size = 0, 0, 320, 320
		case "bottomleft":
			*x_start, *y_start, *x_size, *y_size = 0, 320, 80, 80
		case "bottomright":
			*x_start, *y_start, *x_size, *y_size = 240, 320, 80, 80
		}
	}

	reader, err := os.Open(*image_file)

	if err != nil {
		fmt.Printf("Error reading image file: %s\r\n", err.Error())
		return
	}

	imageFrames, delays, err := getImageFrames(*image_file, reader, *x_size, *y_size)

	if err != nil {
		fmt.Printf("Error reading image frame(s): %s\r\n", err.Error())
		return
	}

	sender := pixelflood_client.CreateSender(*x_start, *y_start, *worker_amount, *ignore_black)
	sender.SetImage(imageFrames[0])
	sender.Start(*worker_type)

	go loopThroughFrames(imageFrames, delays, sender)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<- c

	fmt.Println("Exiting")
	sender.Stop()
	time.Sleep(2 * time.Second)
}

func getImageFrames(fileName string, reader io.Reader, x_size int, y_size int) ([]image.Image, []int, error) {
	if strings.HasSuffix(fileName, ".gif"){
		gifData, err := gif.DecodeAll(reader)

		if err != nil {
			fmt.Printf("Error parsing image file: %s\r\n", err.Error())
			return nil, nil, err
		}

		frames := make([]image.Image, len(gifData.Image))
		for index, frame := range gifData.Image {
			frames[index] = resize.Resize(uint(x_size), uint(y_size), frame, resize.NearestNeighbor)
		}

		return frames, gifData.Delay, nil
	} else {
		imageData, _, err := image.Decode(reader)

		if err != nil {
			fmt.Printf("Error parsing image file: %s\r\n", err.Error())
			return nil, nil, err
		}

		return []image.Image{resize.Resize(uint(x_size), uint(y_size), imageData, resize.NearestNeighbor)}, []int{10000}, nil
	}
}

func loopThroughFrames(frames []image.Image, delays []int, sender *pixelflood_client.Sender) {
	for true {
		for index, frame := range frames {
			delay := delays[index]
			sender.SetImage(frame)
			time.Sleep(time.Duration(delay) * 10 * time.Millisecond)
		}
	}
}