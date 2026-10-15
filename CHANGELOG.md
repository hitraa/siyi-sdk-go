# Changelog

All notable changes to `siyi-sdk-go` are documented here.

This changelog follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and `siyi-sdk-go` adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v0.1.0] - 2026-10-15

### 🚀 Added

- **Core SIYI Protocol Engine (`pkg/protocol`)**:
  - Full binary framing layer (STX `0x6655`, LE payload length, 16-bit sequence tracking, command ID).
  - High-performance CRC-16/XMODEM table lookup generator and validator.
  - Robust state-machine streaming parser capable of recovering from arbitrary framing slips or noise.
- **Strongly Typed Domain Models (`pkg/models`)**:
  - Type-safe structs, angle validators, and enum decoders for all SIYI payloads.
  - Multi-camera Product IDs supporting **UniPod MT11**, **ZT30**, **ZT6**, **A8 mini**, **ZR30**, **ZR10**, **A2 mini**, and **R1M**.
  - Stream generation mapping (`CameraGenerationOld` for `/main.264`, `CameraGenerationNew` for `/video1` and `/video2`).
- **Complete Command Catalogue (`pkg/commands`)**:
  - Over 80 command serializers and deserializers covering system, gimbal, camera, zoom/focus, thermal, laser, OSD, encoding, and AI tracking.
- **Pluggable Transports (`pkg/transport`)**:
  - **UDP Transport**: Low-latency command and telemetry exchange on port `37260`.
  - **TCP Transport**: Persistent TCP connection with automatic 1Hz keepalive heartbeat ticker.
  - **UART Serial Transport**: Hardware serial communication with configurable baud rates (default 115200 8N1).
  - **Deterministic Mock Transport**: Reactive in-memory transport with `SetOnSend` hooks for unit testing without hardware.
- **Pure-Go RTSP Video Streaming (`pkg/stream`)**:
  - Zero CGo, zero OpenCV, zero GStreamer dependencies using `gortsplib/v4`.
  - H.264 and H.265 video stream demuxing with automatic reconnect.
  - URL builder for both old-generation (`/main.264`) and new-generation (`/video1`, `/video2`) camera architectures.
- **Embedded Web UI Dashboard (`cmd/siyi-web`, `web/`)**:
  - Standalone binary with modern embedded HTML5/CSS3/JavaScript assets via `embed.FS`.
  - Real-time 10Hz gimbal attitude visualization over WebSockets.
  - Virtual PTZ joystick, zoom/focus sliders, photo capture, and video recording controls.
  - Live RTSP-to-MJPEG proxy and TF card media gallery browser.
- **Full-Featured CLI Tool (`cmd/siyi-cli`)**:
  - Command-line interface with subcommands: `info`, `attitude`, `rotate`, `center`, `photo`, `record`, `zoom`, `focus`, `laser`, `format`, `reboot`, `rtsp-url`, and `discover`.
- **Advanced Autonomous & Thermal Modules**:
  - **Radiometric Thermal Matrix Decoding (`pkg/thermal`)**: 16-bit Y16 thermal buffer processing, Celsius temperature matrix generation, ROI statistics, and pseudo-color palettes (WhiteHot, BlackHot, Ironbow, Rainbow).
  - **MAVLink v2 Gimbal & Camera Client (`pkg/mavlink`)**: Support for MSG #284 (`GIMBAL_DEVICE_SET_ATTITUDE`) and MSG #285 (`GIMBAL_DEVICE_ATTITUDE_STATUS`) with quaternion and Euler angle conversions on UDP port 14550.
  - **Port 80 Web Administration Client (`pkg/admin`)**: Camera IP address, subnet mask, gateway, DNS query/configuration, and remote reboot.
  - **Multi-Camera & Dual-Payload Cluster (`pkg/multicam`)**: Coordinated control across multi-camera topologies (Primary Camera A, Secondary Camera B, and AI tracking modules).
  - **Network UDP Auto-Discovery (`pkg/discovery`)**: LAN broadcast discovery on port 37260 for SIYI devices.

### 🛡️ Fixed & Hardened (Upstream Python SDK Deficiencies Resolved)

- **Sequence Number Wrap-Around**: Solved sequence number overflows via native `uint16` masking (`(seq + 1) & 0xFFFF`), eliminating freezing and warnings when sequence exceeds 65535.
- **Gimbal Rotation vs. Attitude Separation**: Separated velocity rate commands (`0x07`, `RotateGimbal`) from absolute attitude target angles (`0x0E`, `SetGimbalAttitude`), preventing gimbals from slamming into mechanical limits.
- **A8 Mini Zoom Stop Timeout**: Added automatic fallback to `GetCurrentZoom()` when zoom stop commands (`0x05`, direction=0) do not receive an ACK on idle cameras.
- **4K Resolution Support**: Permitted 4K (`3840x2160`) and 2K (`2560x1440`) in `SetCameraEncodingParameters` (`0x21`), supporting high-resolution cameras like ZT30, ZR30, and UniPod MT11.
- **Hardware ID Coverage**: Included dual IDs for ZR10 (`0x6B` and `0x63`), ZT6 (`0x83` and `0x82`), ZT30 (`0x80` and `0x85`), UniPod MT11 (`0x7A`), and R1M (`0x77`).
- **Clean Channel Shutdown**: Transferred transport receive channel closure exclusively to reader goroutines (`readLoop`), eliminating runtime panics on connection shutdown.
- **Format SD Card Resilience**: Handled unacknowledged SD card format commands (`0x48`) gracefully across both Web UI and CLI.
