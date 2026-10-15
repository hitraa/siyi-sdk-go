package thermal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/hitraa/siyi-sdk-go/pkg/models"
)

var (
	// ErrInvalidDataSize is returned when raw radiometric frame length does not match expected dimensions.
	ErrInvalidDataSize = errors.New("invalid radiometric data size")

	// ErrOutOfBounds is returned when pixel coordinates are outside frame bounds.
	ErrOutOfBounds = errors.New("coordinates out of frame bounds")
)

// RadiometricFrame represents an uncompressed 2D radiometric thermal temperature matrix.
// Standard SIYI thermal cores (ZT30, ZT6, Quad-Spectrum) output 640x512 resolution.
type RadiometricFrame struct {
	Width    int       `json:"width"`
	Height   int       `json:"height"`
	Matrix   []float64 `json:"-"` // Temperature in Celsius for each pixel: row-major [y * Width + x]
	MinTempC float64   `json:"min_temp_c"`
	MaxTempC float64   `json:"max_temp_c"`
	AvgTempC float64   `json:"avg_temp_c"`
	MinX     int       `json:"min_x"`
	MinY     int       `json:"min_y"`
	MaxX     int       `json:"max_x"`
	MaxY     int       `json:"max_y"`
}

// DecodeY16 decodes raw 16-bit little-endian radiometric frame bytes into a RadiometricFrame.
// If width and height are <= 0, standard SIYI thermal resolution 640x512 is assumed.
// scale and offset convert raw 16-bit values to degrees Celsius: TempC = raw * scale + offset.
// For standard centikelvin sensors (raw = cK): scale = 0.01, offset = -273.15.
// If scale is 0, default scale=0.01 and offset=-273.15 are used.
func DecodeY16(data []byte, width, height int, scale, offset float64) (*RadiometricFrame, error) {
	if width <= 0 {
		width = 640
	}
	if height <= 0 {
		height = 512
	}
	expectedBytes := width * height * 2
	if len(data) < expectedBytes {
		return nil, fmt.Errorf("%w: got %d bytes, expected %d for %dx%d 16-bit",
			ErrInvalidDataSize, len(data), expectedBytes, width, height)
	}

	if scale == 0 {
		scale = 0.01
		offset = -273.15
	}

	totalPixels := width * height
	matrix := make([]float64, totalPixels)

	minTemp := math.MaxFloat64
	maxTemp := -math.MaxFloat64
	var sumTemp float64
	minX, minY := 0, 0
	maxX, maxY := 0, 0

	for i := 0; i < totalPixels; i++ {
		rawVal := binary.LittleEndian.Uint16(data[i*2 : i*2+2])
		tempC := float64(rawVal)*scale + offset
		matrix[i] = tempC
		sumTemp += tempC

		x := i % width
		y := i / width

		if tempC < minTemp {
			minTemp = tempC
			minX, minY = x, y
		}
		if tempC > maxTemp {
			maxTemp = tempC
			maxX, maxY = x, y
		}
	}

	avgTemp := sumTemp / float64(totalPixels)

	return &RadiometricFrame{
		Width:    width,
		Height:   height,
		Matrix:   matrix,
		MinTempC: minTemp,
		MaxTempC: maxTemp,
		AvgTempC: avgTemp,
		MinX:     minX,
		MinY:     minY,
		MaxX:     maxX,
		MaxY:     maxY,
	}, nil
}

// TempAt returns the measured temperature in degrees Celsius at pixel coordinate (x, y).
func (f *RadiometricFrame) TempAt(x, y int) (float64, error) {
	if x < 0 || x >= f.Width || y < 0 || y >= f.Height {
		return 0, fmt.Errorf("%w: (%d, %d) for size %dx%d", ErrOutOfBounds, x, y, f.Width, f.Height)
	}
	return f.Matrix[y*f.Width+x], nil
}

// RegionStats computes the minimum, maximum, and average temperature in a rectangular ROI.
func (f *RadiometricFrame) RegionStats(x1, y1, x2, y2 int) (minC, maxC, avgC float64, err error) {
	if x1 < 0 || x1 >= f.Width || x2 < 0 || x2 >= f.Width ||
		y1 < 0 || y1 >= f.Height || y2 < 0 || y2 >= f.Height {
		return 0, 0, 0, ErrOutOfBounds
	}

	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}

	minC = math.MaxFloat64
	maxC = -math.MaxFloat64
	var sum float64
	count := 0

	for y := y1; y <= y2; y++ {
		rowOffset := y * f.Width
		for x := x1; x <= x2; x++ {
			t := f.Matrix[rowOffset+x]
			if t < minC {
				minC = t
			}
			if t > maxC {
				maxC = t
			}
			sum += t
			count++
		}
	}

	if count > 0 {
		avgC = sum / float64(count)
	}
	return minC, maxC, avgC, nil
}

// ToGrayscale generates an 8-bit normalized grayscale image buffer (1 byte per pixel, [0..255]).
func (f *RadiometricFrame) ToGrayscale() []byte {
	out := make([]byte, f.Width*f.Height)
	span := f.MaxTempC - f.MinTempC
	if span <= 0.001 {
		span = 1.0
	}

	for i, t := range f.Matrix {
		normalized := (t - f.MinTempC) / span
		if normalized < 0 {
			normalized = 0
		}
		if normalized > 1 {
			normalized = 1
		}
		out[i] = byte(normalized * 255.0)
	}
	return out
}

// ToRGB converts the radiometric frame to a 24-bit RGB byte buffer (3 bytes per pixel: R, G, B)
// applying the selected pseudo-color palette.
func (f *RadiometricFrame) ToRGB(palette models.PseudoColor) []byte {
	gray := f.ToGrayscale()
	rgb := make([]byte, len(gray)*3)

	for i, v := range gray {
		r, g, b := applyColormap(v, palette)
		rgb[i*3] = r
		rgb[i*3+1] = g
		rgb[i*3+2] = b
	}
	return rgb
}

func applyColormap(v byte, palette models.PseudoColor) (r, g, b byte) {
	val := float64(v) / 255.0

	switch palette {
	case models.PseudoColorBlackHot:
		inv := 255 - v
		return inv, inv, inv

	case models.PseudoColorIronbow:
		// Classic Ironbow: purple/blue -> red -> orange -> yellow -> white
		if val < 0.25 {
			r = byte(val * 4.0 * 128.0)
			g = 0
			b = byte(val * 4.0 * 255.0)
		} else if val < 0.5 {
			f := (val - 0.25) * 4.0
			r = byte(128.0 + f*127.0)
			g = byte(f * 100.0)
			b = byte(255.0 * (1.0 - f))
		} else if val < 0.75 {
			f := (val - 0.5) * 4.0
			r = 255
			g = byte(100.0 + f*155.0)
			b = 0
		} else {
			f := (val - 0.75) * 4.0
			r = 255
			g = 255
			b = byte(f * 255.0)
		}
		return r, g, b

	case models.PseudoColorRainbow:
		// Jet / Rainbow: blue -> cyan -> green -> yellow -> red
		if val < 0.25 {
			r = 0
			g = byte(val * 4.0 * 255.0)
			b = 255
		} else if val < 0.5 {
			r = 0
			g = 255
			b = byte((0.5 - val) * 4.0 * 255.0)
		} else if val < 0.75 {
			r = byte((val - 0.5) * 4.0 * 255.0)
			g = 255
			b = 0
		} else {
			r = 255
			g = byte((1.0 - val) * 4.0 * 255.0)
			b = 0
		}
		return r, g, b

	default: // White Hot
		return v, v, v
	}
}
