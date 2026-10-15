package mavlink_test

import (
	"math"
	"testing"

	"github.com/hitraa/siyi-sdk-go/pkg/mavlink"
)

func TestMAVLink_FrameEncodeDecode(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03, 0x04}
	frame := &mavlink.Frame{
		Sequence:    42,
		SystemID:    255,
		ComponentID: 190,
		MessageID:   mavlink.MsgHeartbeat,
		Payload:     payload,
	}

	encoded := frame.Encode()
	if len(encoded) != 10+len(payload)+2 {
		t.Fatalf("unexpected encoded length: %d", len(encoded))
	}

	decoded, err := mavlink.DecodeFrame(encoded)
	if err != nil {
		t.Fatalf("DecodeFrame failed: %v", err)
	}

	if decoded.Sequence != 42 || decoded.SystemID != 255 || decoded.ComponentID != 190 {
		t.Errorf("header mismatch: seq=%d, sys=%d, comp=%d", decoded.Sequence, decoded.SystemID, decoded.ComponentID)
	}

	if decoded.MessageID != mavlink.MsgHeartbeat {
		t.Errorf("message ID mismatch: %d", decoded.MessageID)
	}

	if len(decoded.Payload) != len(payload) {
		t.Errorf("payload length mismatch: %d", len(decoded.Payload))
	}
}

func TestMAVLink_GimbalAttitudeStatus(t *testing.T) {
	// 40 bytes payload for GIMBAL_DEVICE_ATTITUDE_STATUS
	payload := make([]byte, 40)

	// Quaternion for identity (roll=0, pitch=0, yaw=0): w=1, x=0, y=0, z=0
	q := mavlink.EulerToQuaternion(0, 0, 0)
	if math.Abs(float64(q[0])-1.0) > 0.01 {
		t.Errorf("expected w=1 for identity, got %.2f", q[0])
	}

	// Quaternion for yaw=90 degrees: w=0.7071, z=0.7071
	qYaw90 := mavlink.EulerToQuaternion(0, 0, 90.0)
	if math.Abs(float64(qYaw90[0])-0.7071) > 0.01 || math.Abs(float64(qYaw90[3])-0.7071) > 0.01 {
		t.Errorf("unexpected quaternion for yaw=90: %+v", qYaw90)
	}

	setAttPayload := mavlink.EncodeGimbalDeviceSetAttitude(1, 154, 0, qYaw90, 0, 0, 0)
	if len(setAttPayload) != 32 {
		t.Errorf("expected 32 bytes for SetAttitude payload, got %d", len(setAttPayload))
	}

	_ = payload
}
