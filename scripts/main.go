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
	"bufio"
	"time"
)

func main() {
	image_file := flag.String("image", "./image.jpg", "X value of the screen to begin drawing on")
	x_start := *flag.Int("x_start", 0, "X value of the screen to begin drawing on")
	y_start := *flag.Int("y_start", 0, "y value of the screen to begin drawing on")
	x_size := *flag.Int("x_size", 320, "X size of the image we want to draw on the screen")
	y_size := *flag.Int("y_size", 320, "Y size of the image we want to draw on the screen")
	worker_amount := flag.Int("worker_amount", 16, "The amount of workers that concurrently sends pixel values to the server")
	worker_type := flag.String("worker_type", "squares", "The type of pixel replacement strategy ['random', 'squares']")
	screen := flag.String("screen", "", "The screen to target ['topleft', 'center', 'topright', 'externalright']. Replaces manual x,y start and size.")
	flag.Parse()

	if len(*screen) > 0 {
		switch *screen {
		case "topleft":
			x_start, y_start, x_size, y_size = 0, 0, 80, 80
		case "center":
			x_start, y_start, x_size, y_size = 80, 24, 240, 240
		case "topright":
			x_start, y_start, x_size, y_size = 320, 0, 80, 80
		case "externalright":
			x_start, y_start, x_size, y_size = 400, 0, 80, 80
		}
	}

	reader, err := os.Open(*image_file)

	if err != nil {
		fmt.Printf("Error reading image file: %s\r\n", err.Error())
		return
	}

	image, _, err := image.Decode(reader)

	if err != nil {
		fmt.Printf("Error parsing image file: %s\r\n", err.Error())
		return
	}

	// Resize the image to fit our screen's dimensions
	image = resize.Resize(uint(x_size), uint(y_size), image, resize.NearestNeighbor)

	sender := pixelflood_client.CreateSender(x_start, y_start, *worker_amount)
	sender.SetImage(image)
	sender.Start(*worker_type)

	bufio.NewScanner(os.Stdin).Scan()
	fmt.Println("Exiting")
	sender.Stop()
	time.Sleep(2 * time.Second)
}