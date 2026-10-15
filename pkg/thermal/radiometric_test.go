package thermal_test

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/thermal"
)

func TestDecodeY16(t *testing.T) {
	width := 4
	height := 4
	data := make([]byte, width*height*2)

	// Fill with raw centikelvin values (e.g. 29315 cK = 20.0 C, 30315 cK = 30.0 C)
	for i := 0; i < width*height; i++ {
		// Linear gradient from 29315 (20 C) to 30815 (35 C)
		cK := uint16(29315 + i*100)
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], cK)
	}

	frame, err := thermal.DecodeY16(data, width, height, 0.01, -273.15)
	if err != nil {
		t.Fatalf("DecodeY16 failed: %v", err)
	}

	if frame.Width != width || frame.Height != height {
		t.Errorf("expected size %dx%d, got %dx%d", width, height, frame.Width, frame.Height)
	}

	// Check min/max
	if math.Abs(frame.MinTempC-20.0) > 0.01 {
		t.Errorf("expected MinTempC 20.0, got %.2f", frame.MinTempC)
	}
	expectedMax := 20.0 + float64(width*height-1)
	if math.Abs(frame.MaxTempC-expectedMax) > 0.01 {
		t.Errorf("expected MaxTempC %.2f, got %.2f", expectedMax, frame.MaxTempC)
	}

	// Check TempAt
	t00, err := frame.TempAt(0, 0)
	if err != nil || math.Abs(t00-20.0) > 0.01 {
		t.Errorf("TempAt(0, 0) = %.2f, err=%v", t00, err)
	}

	// Out of bounds check
	_, err = frame.TempAt(5, 5)
	if err == nil {
		t.Errorf("expected error for out-of-bounds TempAt")
	}

	// RegionStats check
	minC, maxC, avgC, err := frame.RegionStats(1, 1, 2, 2)
	if err != nil {
		t.Fatalf("RegionStats failed: %v", err)
	}
	if minC < 20.0 || maxC > expectedMax || avgC < minC || avgC > maxC {
		t.Errorf("unexpected region stats: min=%.2f, max=%.2f, avg=%.2f", minC, maxC, avgC)
	}

	// Grayscale generation check
	gray := frame.ToGrayscale()
	if len(gray) != width*height {
		t.Errorf("expected %d grayscale bytes, got %d", width*height, len(gray))
	}
	if gray[0] != 0 || gray[len(gray)-1] != 255 {
		t.Errorf("expected normalized extremes 0 and 255, got %d and %d", gray[0], gray[len(gray)-1])
	}

	// Color palette generation check
	rgbWhiteHot := frame.ToRGB(models.PseudoColorWhiteHot)
	if len(rgbWhiteHot) != width*height*3 {
		t.Errorf("expected %d RGB bytes, got %d", width*height*3, len(rgbWhiteHot))
	}

	rgbBlackHot := frame.ToRGB(models.PseudoColorBlackHot)
	if len(rgbBlackHot) != width*height*3 {
		t.Errorf("expected %d RGB bytes for BlackHot", len(rgbBlackHot))
	}

	rgbIron := frame.ToRGB(models.PseudoColorIronbow)
	if len(rgbIron) != width*height*3 {
		t.Errorf("expected %d RGB bytes for Ironbow", len(rgbIron))
	}
}
