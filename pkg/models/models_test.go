package models_test

import (
	"testing"

	"github.com/hitraa/siyi-sdk-go/pkg/models"
)

func TestFirmwareVersion_Decode(t *testing.T) {
	// Word: 0x6E030203 -> major=3, minor=2, patch=3
	major, minor, patch := models.DecodeWord(0x6E030203)
	if major != 3 || minor != 2 || patch != 3 {
		t.Errorf("DecodeWord = (%d,%d,%d), want (3,2,3)", major, minor, patch)
	}

	formatted := models.FormatWord(0x6E030203)
	if formatted != "v3.2.3" {
		t.Errorf("FormatWord = %s, want v3.2.3", formatted)
	}

	if models.FormatWord(0) != "n/a" {
		t.Errorf("FormatWord(0) = %s, want n/a", models.FormatWord(0))
	}
}

func TestHardwareID_ProductID(t *testing.T) {
	cases := []struct {
		hexStr   string
		wantPID  models.ProductID
		wantName string
		wantGen  models.CameraGeneration
	}{
		{"631234567890", models.ProductZR10Alternate, "ZR10", models.CameraGenerationOld},
		{"6B1234567890", models.ProductZR10, "ZR10", models.CameraGenerationOld},
		{"731234567890", models.ProductA8Mini, "A8 Mini", models.CameraGenerationOld},
		{"751234567890", models.ProductA2Mini, "A2 Mini", models.CameraGenerationOld},
		{"771234567890", models.ProductR1M, "R1M", models.CameraGenerationOld},
		{"781234567890", models.ProductZR30, "ZR30", models.CameraGenerationOld},
		{"7A1234567890", models.ProductQuadSpectrum, "UniPod MT11", models.CameraGenerationNew},
		{"801234567890", models.ProductZT30, "ZT30", models.CameraGenerationNew},
		{"821234567890", models.ProductZT6Alternate, "ZT6", models.CameraGenerationNew},
		{"831234567890", models.ProductZT6, "ZT6", models.CameraGenerationNew},
		{"851234567890", models.ProductZT30Alternate, "ZT30", models.CameraGenerationNew},
	}

	for _, tc := range cases {
		hw := models.HardwareID{Raw: []byte(tc.hexStr)}
		pid, err := hw.ProductID()
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", tc.hexStr, err)
		}
		if pid != tc.wantPID {
			t.Errorf("hex %s: got PID %v, want %v", tc.hexStr, pid, tc.wantPID)
		}
		if pid.String() != tc.wantName {
			t.Errorf("hex %s: got name %s, want %s", tc.hexStr, pid.String(), tc.wantName)
		}
		if pid.Generation() != tc.wantGen {
			t.Errorf("hex %s: got gen %v, want %v", tc.hexStr, pid.Generation(), tc.wantGen)
		}
	}

	short := models.HardwareID{Raw: []byte("7")}
	if _, err := short.ProductID(); err == nil {
		t.Errorf("expected error for short hardware ID")
	}

	invalid := models.HardwareID{Raw: []byte("ZZ1234567890")}
	if _, err := invalid.ProductID(); err == nil {
		t.Errorf("expected error for non-hex hardware ID")
	}
}

func TestZoomCalculations(t *testing.T) {
	zr := models.ZoomRange{MaxInt: 30, MaxFloat: 5}
	if zr.MaxZoom() != 30.5 {
		t.Errorf("MaxZoom = %f, want 30.5", zr.MaxZoom())
	}

	cz := models.CurrentZoom{Integer: 5, Decimal: 3}
	if cz.Zoom() != 5.3 {
		t.Errorf("Zoom = %f, want 5.3", cz.Zoom())
	}
}

func TestAngleLimits(t *testing.T) {
	lim, ok := models.GetAngleLimits(models.ProductA8Mini)
	if !ok {
		t.Fatalf("expected angle limits for A8 Mini")
	}
	if lim.YawMin != -135.0 || lim.YawMax != 135.0 || lim.PitchMin != -90.0 || lim.PitchMax != 25.0 {
		t.Errorf("unexpected angle limits: %+v", lim)
	}

	_, ok = models.GetAngleLimits(models.ProductID(0xFF))
	if ok {
		t.Errorf("expected false for unknown product ID")
	}
}
