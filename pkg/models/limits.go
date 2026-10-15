package models

// AngleLimits specifies physical rotation boundaries in degrees.
type AngleLimits struct {
	YawMin   float64 `json:"yaw_min"`
	YawMax   float64 `json:"yaw_max"`
	PitchMin float64 `json:"pitch_min"`
	PitchMax float64 `json:"pitch_max"`
}

// Product angle limits table
var productLimits = map[ProductID]AngleLimits{
	ProductZR10: {
		YawMin:   -135.0,
		YawMax:   135.0,
		PitchMin: -90.0,
		PitchMax: 25.0,
	},
	ProductA8Mini: {
		YawMin:   -135.0,
		YawMax:   135.0,
		PitchMin: -90.0,
		PitchMax: 25.0,
	},
	ProductZR30: {
		YawMin:   -270.0,
		YawMax:   270.0,
		PitchMin: -90.0,
		PitchMax: 25.0,
	},
	ProductA2Mini: {
		YawMin:   0.0,
		YawMax:   0.0,
		PitchMin: -90.0,
		PitchMax: 25.0,
	},
	ProductQuadSpectrum: {
		YawMin:   -360.0,
		YawMax:   360.0,
		PitchMin: -90.0,
		PitchMax: 25.0,
	},
}

// GetAngleLimits returns the mechanical angular limits for a given camera product.
func GetAngleLimits(product ProductID) (AngleLimits, bool) {
	limits, ok := productLimits[product]
	return limits, ok
}

// Gimbal rate command constraints for 0x07 rotation velocity command.
const (
	GimbalRateCmdMin = -100
	GimbalRateCmdMax = 100
)
