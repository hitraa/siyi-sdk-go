package models

// ProductID identifies the SIYI gimbal camera model.
type ProductID uint8

const (
	ProductZR10Alternate ProductID = 0x63
	ProductZR10          ProductID = 0x6B
	ProductA8Mini        ProductID = 0x73
	ProductA2Mini        ProductID = 0x75
	ProductZR30          ProductID = 0x78
	ProductQuadSpectrum  ProductID = 0x7A
	ProductZT30          ProductID = 0x80
	ProductZT6Alternate  ProductID = 0x82
	ProductZT6           ProductID = 0x83
	ProductZT30Alternate ProductID = 0x85
)

func (p ProductID) String() string {
	switch p {
	case ProductZR10, ProductZR10Alternate:
		return "ZR10"
	case ProductA8Mini:
		return "A8 Mini"
	case ProductA2Mini:
		return "A2 Mini"
	case ProductZR30:
		return "ZR30"
	case ProductQuadSpectrum:
		return "Quad Spectrum"
	case ProductZT30, ProductZT30Alternate:
		return "ZT30"
	case ProductZT6, ProductZT6Alternate:
		return "ZT6"
	default:
		return "Unknown"
	}
}

// Generation returns the camera streaming generation (Old vs New) for this product ID.
func (p ProductID) Generation() CameraGeneration {
	switch p {
	case ProductZT30, ProductZT30Alternate, ProductZT6, ProductZT6Alternate:
		return CameraGenerationNew
	default:
		return CameraGenerationOld
	}
}

// CameraGeneration determines the RTSP streaming scheme.
type CameraGeneration int

const (
	CameraGenerationOld CameraGeneration = iota // ZR30/ZR10/A8Mini/A2Mini/R1M (/main.264)
	CameraGenerationNew                         // ZT30/ZT6 (/video1, /video2)
)

// GimbalMotionMode represents the stabilization mode of the gimbal.
type GimbalMotionMode uint8

const (
	GimbalMotionModeLock   GimbalMotionMode = 0
	GimbalMotionModeFollow GimbalMotionMode = 1
	GimbalMotionModeFPV    GimbalMotionMode = 2
)

func (m GimbalMotionMode) String() string {
	switch m {
	case GimbalMotionModeLock:
		return "Lock"
	case GimbalMotionModeFollow:
		return "Follow"
	case GimbalMotionModeFPV:
		return "FPV"
	default:
		return "Unknown"
	}
}

// MountingDirection indicates whether the gimbal is upright or inverted.
type MountingDirection uint8

const (
	MountingDirectionReserved MountingDirection = 0
	MountingDirectionNormal   MountingDirection = 1
	MountingDirectionInverted MountingDirection = 2
)

func (m MountingDirection) String() string {
	switch m {
	case MountingDirectionNormal:
		return "Normal"
	case MountingDirectionInverted:
		return "Inverted"
	default:
		return "Reserved"
	}
}

// CenteringAction represents one-key centering actions.
type CenteringAction uint8

const (
	CenteringActionOneKeyCenter   CenteringAction = 1
	CenteringActionCenterDownward CenteringAction = 2
	CenteringActionCenter         CenteringAction = 3
	CenteringActionDownward       CenteringAction = 4
)

func (c CenteringAction) String() string {
	switch c {
	case CenteringActionOneKeyCenter:
		return "OneKeyCenter"
	case CenteringActionCenterDownward:
		return "CenterDownward"
	case CenteringActionCenter:
		return "Center"
	case CenteringActionDownward:
		return "Downward"
	default:
		return "Unknown"
	}
}

// ControlMode represents gimbal control mode (used in ArduPilot integration).
type ControlMode uint8

const (
	ControlModeAttitude   ControlMode = 0
	ControlModeWeak       ControlMode = 1
	ControlModeMiddle     ControlMode = 2
	ControlModeFPV        ControlMode = 3
	ControlModeMotorClose ControlMode = 4
)

// HDMICVBSOutput indicates video port output state.
type HDMICVBSOutput uint8

const (
	HDMICVBSOutputHDMIOnCVBSOff HDMICVBSOutput = 0
	HDMICVBSOutputHDMIOffCVBSOn HDMICVBSOutput = 1
)

// RecordingState indicates the camera TF card recording state.
type RecordingState uint8

const (
	RecordingStateNotRecording RecordingState = 0
	RecordingStateRecording    RecordingState = 1
	RecordingStateNoTFCard     RecordingState = 2
	RecordingStateDataLoss     RecordingState = 3
)

func (r RecordingState) String() string {
	switch r {
	case RecordingStateNotRecording:
		return "NotRecording"
	case RecordingStateRecording:
		return "Recording"
	case RecordingStateNoTFCard:
		return "NoTFCard"
	case RecordingStateDataLoss:
		return "DataLoss"
	default:
		return "Unknown"
	}
}

// FunctionFeedback response codes from camera.
type FunctionFeedback uint8

const (
	FunctionFeedbackPhotoOK          FunctionFeedback = 0
	FunctionFeedbackPhotoFailed      FunctionFeedback = 1
	FunctionFeedbackHDROn            FunctionFeedback = 2
	FunctionFeedbackHDROff           FunctionFeedback = 3
	FunctionFeedbackRecordingFailed  FunctionFeedback = 4
	FunctionFeedbackRecordingStarted FunctionFeedback = 5
	FunctionFeedbackRecordingStopped FunctionFeedback = 6
)

// CaptureFuncType selects photo capture or video recording commands.
type CaptureFuncType uint8

const (
	CaptureFuncPhoto           CaptureFuncType = 0
	CaptureFuncHDRToggle       CaptureFuncType = 1
	CaptureFuncStartRecord     CaptureFuncType = 2
	CaptureFuncLockMode        CaptureFuncType = 3
	CaptureFuncFollowMode      CaptureFuncType = 4
	CaptureFuncFPVMode         CaptureFuncType = 5
	CaptureFuncEnableHDMI      CaptureFuncType = 6
	CaptureFuncEnableCVBS      CaptureFuncType = 7
	CaptureFuncDisableHDMICVBS CaptureFuncType = 8
	CaptureFuncTiltDownward    CaptureFuncType = 9
	CaptureFuncZoomLinkage     CaptureFuncType = 10
)

// VideoEncType represents video encoding compression algorithm.
type VideoEncType uint8

const (
	VideoEncH264 VideoEncType = 1
	VideoEncH265 VideoEncType = 2
)

// StreamType selects video stream slot.
type StreamType uint8

const (
	StreamTypeRecording StreamType = 0
	StreamTypeMain      StreamType = 1
	StreamTypeSub       StreamType = 2
)

func (s StreamType) String() string {
	switch s {
	case StreamTypeRecording:
		return "RECORDING"
	case StreamTypeMain:
		return "MAIN"
	case StreamTypeSub:
		return "SUB"
	default:
		return "UNKNOWN"
	}
}

// VideoStitchingMode specifies stitching combinations on multi-sensor payloads.
type VideoStitchingMode uint8

const (
	VideoStitchingMode0 VideoStitchingMode = 0 // Stitching: Main=Zoom&Thermal, Sub=Wide
	VideoStitchingMode1 VideoStitchingMode = 1 // Stitching: Main=Wide&Thermal, Sub=Zoom
	VideoStitchingMode2 VideoStitchingMode = 2 // Stitching: Main=Zoom&Wide, Sub=Thermal
	VideoStitchingMode3 VideoStitchingMode = 3 // Non-stitching: Main=Zoom, Sub=Thermal
	VideoStitchingMode4 VideoStitchingMode = 4 // Non-stitching: Main=Zoom, Sub=Wide
	VideoStitchingMode5 VideoStitchingMode = 5 // Non-stitching: Main=Wide, Sub=Thermal
	VideoStitchingMode6 VideoStitchingMode = 6 // Non-stitching: Main=Wide, Sub=Zoom
	VideoStitchingMode7 VideoStitchingMode = 7 // Non-stitching: Main=Thermal, Sub=Zoom
	VideoStitchingMode8 VideoStitchingMode = 8 // Non-stitching: Main=Thermal, Sub=Wide
)

// PseudoColor selects thermal palette representation.
type PseudoColor uint8

const (
	PseudoColorWhiteHot  PseudoColor = 0
	PseudoColorReserved  PseudoColor = 1
	PseudoColorSepia     PseudoColor = 2
	PseudoColorIronbow   PseudoColor = 3
	PseudoColorRainbow   PseudoColor = 4
	PseudoColorNight     PseudoColor = 5
	PseudoColorAurora    PseudoColor = 6
	PseudoColorRedHot    PseudoColor = 7
	PseudoColorJungle    PseudoColor = 8
	PseudoColorMedical   PseudoColor = 9
	PseudoColorBlackHot  PseudoColor = 10
	PseudoColorGloryHot  PseudoColor = 11
)

// TempMeasureFlag represents temperature measurement mode.
type TempMeasureFlag uint8

const (
	TempMeasureDisable       TempMeasureFlag = 0
	TempMeasureOnce          TempMeasureFlag = 1
	TempMeasureContinuous5Hz TempMeasureFlag = 2
)

// ThermalOutputMode represents thermal camera output mode.
type ThermalOutputMode uint8

const (
	ThermalOutputFPS30        ThermalOutputMode = 0
	ThermalOutputFPS25PlusTemp ThermalOutputMode = 1
)

// ThermalGain represents thermal gain sensitivity.
type ThermalGain uint8

const (
	ThermalGainLow  ThermalGain = 0
	ThermalGainHigh ThermalGain = 1
)

// IRThreshPrecision represents IR threshold precision level.
type IRThreshPrecision uint8

const (
	IRThreshPrecisionMax IRThreshPrecision = 1
	IRThreshPrecisionMid IRThreshPrecision = 2
	IRThreshPrecisionMin IRThreshPrecision = 3
)

// FCDataType selects Flight Controller data stream type.
type FCDataType uint8

const (
	FCDataAttitude   FCDataType = 1
	FCDataRCChannels FCDataType = 2
)

// GimbalDataType selects Gimbal data stream type.
type GimbalDataType uint8

const (
	GimbalDataAttitude        GimbalDataType = 1
	GimbalDataLaserRange      GimbalDataType = 2
	GimbalDataMagneticEncoder GimbalDataType = 3
	GimbalDataMotorVoltage    GimbalDataType = 4
)

// DataStreamFreq selects data stream output rate.
type DataStreamFreq uint8

const (
	DataStreamFreqOff   DataStreamFreq = 0
	DataStreamFreq2Hz   DataStreamFreq = 1
	DataStreamFreq4Hz   DataStreamFreq = 2
	DataStreamFreq5Hz   DataStreamFreq = 3
	DataStreamFreq10Hz  DataStreamFreq = 4
	DataStreamFreq20Hz  DataStreamFreq = 5
	DataStreamFreq50Hz  DataStreamFreq = 6
	DataStreamFreq100Hz DataStreamFreq = 7
)

// AITargetID identifies the recognized AI target object.
type AITargetID uint8

const (
	AITargetHuman AITargetID = 0
	AITargetCar   AITargetID = 1
	AITargetBus   AITargetID = 2
	AITargetTruck AITargetID = 3
	AITargetAny   AITargetID = 255
)

// AITrackStatus indicates AI tracking lock status.
type AITrackStatus uint8

const (
	AITrackNormalAI        AITrackStatus = 0
	AITrackIntermittentLoss AITrackStatus = 1
	AITrackLost            AITrackStatus = 2
	AITrackUserCanceled    AITrackStatus = 3
	AITrackNormalAny       AITrackStatus = 4
)

// AIStreamStatus indicates status of the AI coordinate output stream.
type AIStreamStatus uint8

const (
	AIStreamDisabled           AIStreamStatus = 0
	AIStreamStreaming          AIStreamStatus = 1
	AIStreamAINotEnabled       AIStreamStatus = 2
	AIStreamTrackingNotEnabled AIStreamStatus = 3
)

// FileType identifies file storage types.
type FileType uint8

const (
	FileTypePicture     FileType = 0
	FileTypeTempRaw     FileType = 1
	FileTypeRecordVideo FileType = 2
)

// FileNameType represents file naming schema on SD card.
type FileNameType uint8

const (
	FileNameTypeReserve   FileNameType = 0
	FileNameTypeIndex     FileNameType = 1
	FileNameTypeTimestamp FileNameType = 2
)

// MediaType selects media category for camera web server API.
type MediaType int

const (
	MediaTypeImages MediaType = 0
	MediaTypeVideos MediaType = 1
)
