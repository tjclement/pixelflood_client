package pixelflood_client

import (
	image2 "image"
	"net"
	"fmt"
	"math/rand"
	"math"
	"time"
	"sync"
)

type Pixel struct {
	R uint8
	G uint8
	B uint8
	Ignore bool
}

type Sender struct {
	conn         []net.Conn
	currentImage image2.Image
	pixels       [][]Pixel
	image_x      int
	image_y      int
	x_start      int
	y_start      int
	concurrency  int
	ignore_black bool
	started      bool
	lock         *sync.Mutex
}

func CreateSender(x_start int, y_start int, concurrency int, ignore_black bool) (*Sender) {
	return &Sender{[]net.Conn{}, nil, nil, 0, 0, x_start, y_start, concurrency, ignore_black, false, &sync.Mutex{}}
}

func (sender *Sender) SetImage(image image2.Image) {
	size := image.Bounds().Size()
	sender.currentImage = image
	sender.image_x = size.X
	sender.image_y = size.Y

	pixels := make([][]Pixel, size.X)
	for column := 0; column < size.Y; column++ {
		pixels[column] = make([]Pixel, size.Y)
	}

	for x := 0; x < size.X; x++ {
		for y := 0; y < size.Y; y++ {
			R, G, B := sender.getNormalisedRgbaAt(x, y)

			pixel := &pixels[x][y]
			pixel.R = R
			pixel.G = G
			pixel.B = B

			if R == 0 && G == 0 && B == 0 {
				pixel.Ignore = true
			}
		}
	}

	sender.pixels = pixels
}

func (sender *Sender) Start(worker_type string, address string) {
	if sender.started {
		return
	}

	sender.started = true

	for i := 0; i < sender.concurrency; i ++ {
		go func() {
			conn, err := net.DialTimeout("tcp", address, 1*time.Second)

			if err != nil {
				fmt.Printf("Error setting up TCP connection: %s\r\n", err.Error())
				return
			}

			sender.lock.Lock()
			sender.conn = append(sender.conn, conn)
			index := len(sender.conn) - 1
			sender.lock.Unlock()

			switch worker_type {
			case "random":
				go sender.launchRandomWorker(index)
			case "squares":
				go sender.launchSquaresWorker(index)
			}
		}()
	}
}

func (sender *Sender) launchRandomWorker(index int) {
	for sender.started {
		random_x, random_y := rand.Intn(sender.image_x), rand.Intn(sender.image_y)
		pixel := sender.pixels[random_x][random_y]
		R, G, B := pixel.R, pixel.G, pixel.B
		screen_x, screen_y := sender.x_start+random_x, sender.y_start+random_y

		sender.conn[index].Write([]byte(fmt.Sprintf("PX %d %d %02x%02x%02x\n", screen_x, screen_y, R, G, B)))
	}
}

// Divides the screen into multiple equally sized squares (2x2, 3x3, 4x4, etc), and renders the square
// of a given index as a separate worker. Sender.concurrency must be a number that has an integer square root.
func (sender *Sender) launchSquaresWorker(index int) {
	for sender.started {
		// The amount of cells in a single row or column
		screen_dim_cells := int(math.Sqrt(float64(sender.concurrency)))

		worker_column := index % (screen_dim_cells)
		worker_row := int(math.Floor(float64(index) / float64(screen_dim_cells)))

		square_x_size := int(math.Ceil(float64(sender.image_x / screen_dim_cells)))
		square_y_size := int(math.Ceil(float64(sender.image_y / screen_dim_cells)))

		for y := 0; y < square_y_size; y++ {
			for x := 0; x < square_x_size; x++ {
				if !sender.started {
					break
				}

				x_position, y_position := (x + (worker_column * square_x_size)), (y + (worker_row * square_y_size))
				pixel := sender.pixels[x_position][y_position]

				if pixel.Ignore {
					continue
				}

				screen_x, screen_y := sender.x_start+x_position, sender.y_start+y_position
				sender.conn[index].Write([]byte(fmt.Sprintf("PX %d %d %02x%02x%02x\n", screen_x, screen_y, pixel.R, pixel.G, pixel.B)))
			}
		}
	}
}

func (sender *Sender) Stop() {
	sender.started = false
}

func (sender *Sender) getNormalisedRgbaAt(x int, y int) (uint8, uint8, uint8) {
	color := sender.currentImage.At(x, y)
	R, G, B, _ := color.RGBA()
	return uint8(R / 256), uint8(G / 256), uint8(B / 256)
}
