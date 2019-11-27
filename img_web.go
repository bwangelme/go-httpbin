package httpbin

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"math"
	"net/http"
	"path/filepath"
	"strings"
)

type circle struct {
	X, Y, R float64
}

func (c *circle) Brightness(x, y float64) uint8 {
	var dx, dy float64 = c.X - x, c.Y - y
	d := math.Sqrt(dx*dx+dy*dy) / c.R
	if d > 1 {
		return 0
	}

	return 255
}

// ImgGIFHandler returns an animated GIF image.
// Source: http://tech.nitoyon.com/en/blog/2016/01/07/go-animated-gif-gen/
func ImgGIFHandler(resp http.ResponseWriter, r *http.Request) {
	var w, h int = 240, 240
	var hw, hh float64 = float64(w / 2), float64(h / 2)
	circles := []*circle{{}, {}, {}}

	var palette = []color.Color{
		color.RGBA{0x00, 0x00, 0x00, 0xff},
		color.RGBA{0x00, 0x00, 0xff, 0xff},
		color.RGBA{0x00, 0xff, 0x00, 0xff},
		color.RGBA{0x00, 0xff, 0xff, 0xff},
		color.RGBA{0xff, 0x00, 0x00, 0xff},
		color.RGBA{0xff, 0x00, 0xff, 0xff},
		color.RGBA{0xff, 0xff, 0x00, 0xff},
		color.RGBA{0xff, 0xff, 0xff, 0xff},
	}

	var images []*image.Paletted
	var delays []int
	steps := 20

	for step := 0; step < steps; step++ {
		img := image.NewPaletted(image.Rect(0, 0, w, h), palette)
		images = append(images, img)
		delays = append(delays, 0)

		theta := 2.0 * math.Pi / float64(steps) * float64(step)
		for i, circle := range circles {
			theta0 := 2 * math.Pi / 3 * float64(i)
			circle.X = hw - 40*math.Sin(theta0) - 20*math.Sin(theta0+theta)
			circle.Y = hh - 40*math.Cos(theta0) - 20*math.Cos(theta0+theta)
			circle.R = 50
		}

		for x := 0; x < w; x++ {
			for y := 0; y < h; y++ {
				img.Set(x, y, color.RGBA{
					circles[0].Brightness(float64(x), float64(y)),
					circles[1].Brightness(float64(x), float64(y)),
					circles[2].Brightness(float64(x), float64(y)),
					255,
				})
			}
		}
	}

	gif.EncodeAll(resp, &gif.GIF{
		Image: images,
		Delay: delays,
	})
}

func ImgHandler(w http.ResponseWriter, r *http.Request) {
	acceptHeader := r.Header.Get("accept")

	if acceptHeader == "" {
		ImgGIFHandler(w, r)
		return
	}

	if strings.Contains(acceptHeader, "image/webp") {
		ImgWebpHandler(w, r)
		return
	} else if strings.Contains(acceptHeader, "image/gif") {
		ImgGIFHandler(w, r)
		return
	} else if strings.Contains(acceptHeader, "image/svg+xml") {
		ImgSVGHandler(w, r)
		return
	} else if strings.Contains(acceptHeader, "image/jpeg") {
		ImgJPEGHandler(w, r)
		return
	} else if strings.Contains(acceptHeader, "image/png") || strings.Contains(acceptHeader, "image/*") {
		ImgPngHandler(w, r)
		return
	} else {
		http.Error(w, "Invalid Accept", http.StatusNotAcceptable)
		return
	}

}

func ImgPngHandler(w http.ResponseWriter, r *http.Request) {
	data, err := Resource(filepath.Join("images", "pig_icon.png"))
	if err != nil {
		logger.InternalErrorPrint(w, err)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(data)
}

func ImgJPEGHandler(w http.ResponseWriter, r *http.Request) {
	data, err := Resource(filepath.Join("images", "jackal.jpg"))
	if err != nil {
		logger.InternalErrorPrint(w, err)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(data)
}

func ImgWebpHandler(w http.ResponseWriter, r *http.Request) {
	data, err := Resource(filepath.Join("images", "wolf_1.webp"))
	if err != nil {
		logger.InternalErrorPrint(w, err)
		return
	}

	w.Header().Set("Content-Type", "image/webp")
	w.Write(data)
}

func ImgSVGHandler(w http.ResponseWriter, r *http.Request) {
	data, err := Resource(filepath.Join("images", "svg_logo.svg"))
	if err != nil {
		logger.InternalErrorPrint(w, err)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Write(data)
}
