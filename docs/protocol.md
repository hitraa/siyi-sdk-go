# SIYI Gimbal Camera External SDK Protocol v0.1.1 (Go Implementation)

This document describes the SIYI Gimbal Camera External SDK Protocol v0.1.1 and how it is implemented in `siyi-sdk-go`. The protocol governs communication with SIYI optical and thermal gimbal cameras including **ZT30**, **ZT6**, **ZR10**, **ZR30**, **A8 mini**, **A2 mini**, and **Quad-Spectrum** payloads.

---

## 1. Frame Structure

All communication uses a deterministic binary frame structure with byte-level serialization:

```
┌───────────┬───────────┬─────────────┬───────────┬───────────┬───────────┬───────────┐
│ STX (2B)  │ CTRL (1B) │ DataLen(2B) │  SEQ (2B) │CMD_ID(1B) │PAYLOAD(nB)│  CRC (2B) │
├───────────┼───────────┼─────────────┼───────────┼───────────┼───────────┼───────────┤
│ 0x55 0x66 │ Bitfield  │ Len (LE)    │  Seq (LE) │ 0x00-0x82 │ 0..4096 B │CRC16(XMOD)│
└───────────┴───────────┴─────────────┴───────────┴───────────┴───────────┴───────────┘
```

### Field Descriptions

| Field       | Size      | Description                                                                                                                                              |
| ----------- | --------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **STX**     | 2 Bytes   | Fixed start marker `0x6655` (`0x55 0x66` little-endian on the wire).                                                                                     |
| **CTRL**    | 1 Byte    | Control byte flags. `bit[0] = 1`: sender requests an ACK response (`CTRLNeedACK = 0x01`). `bit[1] = 1`: frame is an ACK response (`CTRLACKPack = 0x02`). |
| **DataLen** | 2 Bytes   | Payload byte length in uint16 little-endian format.                                                                                                      |
| **SEQ**     | 2 Bytes   | Sequence number in uint16 little-endian format (0x0000–0xFFFF, free-running counter). Echoed in device responses.                                        |
| **CMD_ID**  | 1 Byte    | Command identifier byte (0x00–0x82).                                                                                                                     |
| **PAYLOAD** | _n_ Bytes | Command-specific serialized parameters (0 bytes for query commands).                                                                                     |
| **CRC**     | 2 Bytes   | CRC-16/XMODEM checksum calculated over all preceding bytes (`STX` through end of `PAYLOAD`). Little-endian on the wire.                                  |

Minimum frame size is **10 bytes** (8-byte header + 2-byte CRC, with 0 bytes payload).

---

## 2. CRC-16/XMODEM Algorithm

The protocol uses the **CRC-16/XMODEM** checksum:

- **Polynomial**: `0x1021` ($X^{16} + X^{12} + X^5 + 1$)
- **Initial Seed**: `0x0000`
- **Final XOR**: None (`0x0000`)
- **Endianness**: Appended low byte first, high byte second.

### Go Implementation

In `pkg/protocol/crc.go`, `siyi-sdk-go` implements zero-allocation, pre-computed table lookup:

```go
func CalculateCRC(data []byte) uint16 {
    var crc uint16 = CRC16Init
    for _, b := range data {
        tblIdx := byte((crc >> 8) ^ uint16(b))
        crc = (crc << 8) ^ CRC16Table[tblIdx]
    }
    return crc
}
```

---

## 3. Command Catalogue & ID Reference Table

The SDK provides encoders, decoders, and typed client methods for all protocol commands:

| CMD_ID | Command Name                    | Go Package Function / Client Method                                                | Description                                  |
| ------ | ------------------------------- | ---------------------------------------------------------------------------------- | -------------------------------------------- |
| `0x00` | `CmdTCPHeartbeat`               | `client.Heartbeat(ctx)`                                                            | 1Hz TCP keepalive frame                      |
| `0x01` | `CmdRequestFirmwareVersion`     | `client.GetFirmwareVersion(ctx)`                                                   | Queries camera, gimbal, zoom firmware        |
| `0x02` | `CmdRequestHardwareID`          | `client.GetHardwareID(ctx)`                                                        | Queries hardware ID string (12 bytes)        |
| `0x04` | `CmdAutoFocus`                  | `client.AutoFocus(ctx, flag, x, y)`                                                | Touch-to-focus at screen coordinates         |
| `0x05` | `CmdManualZoomAutoFocus`        | `client.ManualZoom(ctx, dir)`                                                      | Zoom in (`+1`), stop (`0`), or out (`-1`)    |
| `0x06` | `CmdManualFocus`                | `client.ManualFocus(ctx, dir)`                                                     | Focus far (`+1`), stop (`0`), or near (`-1`) |
| `0x07` | `CmdGimbalRotation`             | `client.Rotate(ctx, yaw, pitch)` / `client.RotateNoWait`                           | Velocity rotation rate [-100..100]           |
| `0x08` | `CmdOneKeyCentering`            | `client.OneKeyCentering(ctx, action)`                                              | Center, center downward, or forward          |
| `0x0A` | `CmdRequestCameraSystemInfo`    | `client.GetCameraSystemInfo(ctx)`                                                  | Recording status, gimbal mount, HDMI/CVBS    |
| `0x0B` | `CmdFunctionFeedback`           | `client.OnFunctionFeedback(cb)`                                                    | Asynchronous execution feedback push         |
| `0x0C` | `CmdCapturePhotoRecordVideo`    | `client.TakePhoto(ctx)`, `client.StartRecording(ctx)`, `client.StopRecording(ctx)` | Take photo or toggle video recording         |
| `0x0D` | `CmdRequestGimbalAttitude`      | `client.GetGimbalAttitude(ctx)` / `client.OnAttitude(cb)`                          | Query or push stream Euler angles & rates    |
| `0x0E` | `CmdSetGimbalAttitude`          | `client.SetGimbalAttitude(ctx, y, p)` / `client.SetGimbalAttitudeNoWait`           | Move gimbal to target yaw and pitch angles   |
| `0x0F` | `CmdAbsoluteZoomAutoFocus`      | `client.AbsoluteZoom(ctx, zoom)`                                                   | Set exact optical zoom level (e.g. 2.5x)     |
| `0x10` | `CmdRequestVideoStitchingMode`  | `client.GetVideoStitchingMode(ctx)`                                                | Query multi-sensor display stitching mode    |
| `0x11` | `CmdSetVideoStitchingMode`      | `client.SetVideoStitchingMode(ctx, mode)`                                          | Configure video stitching layout             |
| `0x12` | `CmdGetTempAtPoint`             | `client.GetTempAtPoint(ctx, x, y, flag)`                                           | Query spot temperature at pixel (x, y)       |
| `0x13` | `CmdLocalTempMeasurement`       | `client.GetLocalTemp(ctx, x1, y1, x2, y2, flag)`                                   | Query regional temperature box (min/max)     |
| `0x14` | `CmdGlobalTempMeasurement`      | `client.GetGlobalTemp(ctx, flag)`                                                  | Query full-frame temperature extrema         |
| `0x15` | `CmdRequestLaserDistance`       | `client.GetLaserDistance(ctx)` / `client.OnLaser(cb)`                              | Query or push target distance in meters      |
| `0x16` | `CmdRequestZoomRange`           | `client.GetZoomRange(ctx)`                                                         | Query maximum optical zoom magnification     |
| `0x17` | `CmdRequestLaserLatLon`         | `client.GetLaserTargetLatLon(ctx)`                                                 | Target WGS-84 geographic coordinates         |
| `0x18` | `CmdRequestZoomMagnification`   | `client.GetCurrentZoom(ctx)`                                                       | Current zoom magnification level             |
| `0x19` | `CmdRequestGimbalMode`          | `client.GetGimbalMode(ctx)`                                                        | Lock, Follow, or FPV mode                    |
| `0x1A` | `CmdRequestPseudoColor`         | `client.GetPseudoColor(ctx)`                                                       | Query thermal pseudo-color palette           |
| `0x1B` | `CmdSetPseudoColor`             | `client.SetPseudoColor(ctx, color)`                                                | Select pseudo-color palette                  |
| `0x20` | `CmdRequestEncodingParams`      | `client.GetEncodingParams(ctx, stream)`                                            | Query video encoding bitrate and fps         |
| `0x21` | `CmdSetEncodingParams`          | `client.SetEncodingParams(ctx, params)`                                            | Configure video encoding parameters          |
| `0x22` | `CmdSendAircraftAttitude`       | `client.SendAircraftAttitude(ctx, att)`                                            | Transmit drone vehicle orientation to gimbal |
| `0x23` | `CmdSendRCChannels`             | `client.SendRCChannels(ctx, ch)`                                                   | Transmit 18-channel RC joystick values       |
| `0x24` | `CmdRequestFCDataStream`        | `client.RequestFCStream(ctx, type, freq)`                                          | Subscribe flight controller telemetry stream |
| `0x25` | `CmdRequestGimbalDataStream`    | `client.RequestGimbalStream(ctx, type, freq)`                                      | Subscribe gimbal attitude telemetry stream   |
| `0x26` | `CmdRequestMagneticEncoder`     | `client.GetMagneticEncoder(ctx)`                                                   | Raw magnetic encoder angles in degrees       |
| `0x27` | `CmdRequestControlMode`         | `client.GetControlMode(ctx)`                                                       | ArduPilot gimbal control mode                |
| `0x28` | `CmdRequestWeakThreshold`       | `client.GetWeakThreshold(ctx)`                                                     | ArduPilot weak control threshold limits      |
| `0x29` | `CmdSetWeakThreshold`           | `client.SetWeakThreshold(ctx, t)`                                                  | Set weak control threshold limits            |
| `0x2A` | `CmdRequestMotorVoltage`        | `client.GetMotorVoltage(ctx)`                                                      | Motor drive electrical voltages              |
| `0x30` | `CmdSetUTCTime`                 | `client.SetUTCTime(ctx, usec)`                                                     | Synchronize RTC clock with Unix epoch        |
| `0x31` | `CmdRequestGimbalSystemInfo`    | `client.GetGimbalSystemInfo(ctx)`                                                  | Pan limits and hardware status               |
| `0x32` | `CmdSetLaserRangingState`       | `client.SetLaserRangingState(ctx, on)`                                             | Turn laser rangefinder beam on/off           |
| `0x33` | `CmdRequestThermalOutputMode`   | `client.GetThermalOutputMode(ctx)`                                                 | Thermal framerate and sensor output mode     |
| `0x34` | `CmdSetThermalOutputMode`       | `client.SetThermalOutputMode(ctx, mode)`                                           | Set thermal sensor output mode               |
| `0x35` | `CmdGetSingleTempFrame`         | `client.GetSingleTempFrame(ctx)`                                                   | Request single raw temperature frame         |
| `0x37` | `CmdRequestThermalGain`         | `client.GetThermalGain(ctx)`                                                       | Thermal gain mode (low/high)                 |
| `0x38` | `CmdSetThermalGain`             | `client.SetThermalGain(ctx, gain)`                                                 | Configure thermal gain sensitivity           |
| `0x39` | `CmdRequestEnvCorrectionParams` | `client.GetEnvCorrectionParams(ctx)`                                               | Atmospheric and environmental parameters     |
| `0x3A` | `CmdSetEnvCorrectionParams`     | `client.SetEnvCorrectionParams(ctx, p)`                                            | Set environmental thermal compensation       |
| `0x3B` | `CmdRequestEnvCorrectionSwitch` | `client.GetEnvCorrectionSwitch(ctx)`                                               | Environmental compensation state             |
| `0x3C` | `CmdSetEnvCorrectionSwitch`     | `client.SetEnvCorrectionSwitch(ctx, on)`                                           | Enable/disable environmental compensation    |
| `0x3E` | `CmdSendRawGPS`                 | `client.SendRawGPS(ctx, gps)`                                                      | Transmit GPS lat/lon/alt to camera           |
| `0x40` | `CmdRequestSystemTime`          | `client.GetSystemTime(ctx)`                                                        | Query camera system clock (usec)             |
| `0x41` | `CmdSingleAxisAttitude`         | `client.SetSingleAxis(ctx, angle, axis)` / `NoWait`                                | Move single gimbal axis (yaw or pitch)       |
| `0x42` | `CmdGetIRThreshMapSta`          | `client.GetIRThreshMapState(ctx)`                                                  | IR isotherm threshold map state              |
| `0x43` | `CmdSetIRThreshMapSta`          | `client.SetIRThreshMapState(ctx, on)`                                              | Enable/disable IR isotherm mapping           |
| `0x44` | `CmdGetIRThreshParam`           | `client.GetIRThreshParams(ctx)`                                                    | IR isotherm temperature threshold bands      |
| `0x45` | `CmdSetIRThreshParam`           | `client.SetIRThreshParams(ctx, p)`                                                 | Configure IR isotherm temperature bands      |
| `0x46` | `CmdGetIRThreshPrecision`       | `client.GetIRThreshPrecision(ctx)`                                                 | IR threshold precision level                 |
| `0x47` | `CmdSetIRThreshPrecision`       | `client.SetIRThreshPrecision(ctx, p)`                                              | Set IR threshold precision level             |
| `0x48` | `CmdSDFormat`                   | `client.FormatSDCard(ctx)`                                                         | Format camera TF storage card                |
| `0x49` | `CmdGetPicNameType`             | `client.GetPicNameType(ctx, ft)`                                                   | Query file naming convention                 |
| `0x4A` | `CmdSetPicNameType`             | `client.SetPicNameType(ctx, ft, nt)`                                               | Set file naming convention                   |
| `0x4B` | `CmdGetMavlinkOSDFlag`          | `client.GetMavlinkOSDFlag(ctx)`                                                    | Query HDMI MAVLink OSD overlay state         |
| `0x4C` | `CmdSetMavlinkOSDFlag`          | `client.SetMavlinkOSDFlag(ctx, on)`                                                | Enable/disable HDMI MAVLink OSD overlay      |
| `0x4D` | `CmdGetAIModeSta`               | `client.GetAIMode(ctx)`                                                            | Query AI recognition & tracking state        |
| `0x4E` | `CmdGetAITrackStreamSta`        | `client.GetAIStreamStatus(ctx)`                                                    | Query AI tracking bounding box output        |
| `0x4F` | `CmdManualThermalShutter`       | `client.ManualThermalShutter(ctx)`                                                 | Trigger manual thermal shutter calibration   |
| `0x50` | `CmdAITrackStream`              | `client.OnAITracking(cb)`                                                          | AI bounding-box tracking push stream         |
| `0x51` | `CmdSetAITrackStreamSta`        | `client.SetAIStreamOutput(ctx, on)`                                                | Enable continuous AI coordinate output       |
| `0x70` | `CmdRequestWeakControlMode`     | `client.GetWeakControlMode(ctx)`                                                   | Weak control mode activation status          |
| `0x71` | `CmdSetWeakControlMode`         | `client.SetWeakControlMode(ctx, on)`                                               | Enable/disable weak control mode             |
| `0x80` | `CmdSoftReboot`                 | `client.SoftReboot(ctx, camera, gimbal)`                                           | Reboot camera module and/or gimbal core      |
| `0x81` | `CmdGetIP`                      | `client.GetIP(ctx)`                                                                | Read camera IP, netmask, gateway             |
| `0x82` | `CmdSetIP`                      | `client.SetIP(ctx, cfg)`                                                           | Configure camera IP, netmask, gateway        |

---

## 4. Concurrency, Retries & Timing

1. **Per-Command Concurrency Locking**:
   - Each `CMD_ID` has an isolated mutex (`sync.Mutex`).
   - Distinct commands (e.g., `SetGimbalAttitude` and `Rotate`) execute concurrently without blocking each other.
   - Duplicate concurrent requests for the same `CMD_ID` are cleanly serialized.

2. **Idempotent Retry Policy**:
   - Query/Read commands (such as `GetFirmwareVersion`, `GetGimbalAttitude`, `GetHardwareID`) automatically retry up to `MaxRetries` with exponential backoff on timeout.
   - Non-idempotent write commands (such as `TakePhoto`, `FormatSDCard`, `SoftReboot`) fail immediately without retry to avoid unintended side effects.

3. **High-Rate NoWait Variants**:
   - Fast closed-loop control algorithms (50–100Hz visual servoing) should use:
     - `client.RotateNoWait(ctx, yawRate, pitchRate)`
     - `client.SetGimbalAttitudeNoWait(ctx, yawDeg, pitchDeg)`
     - `client.SetSingleAxisNoWait(ctx, angleDeg, axis)`
   - These bypass ACK waiting and send directly, avoiding round-trip serialization stalls.
