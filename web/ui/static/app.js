class SIYIApp {
    constructor() {
        this.ws = null;
        this.joystickActive = false;
        this.currentMediaMode = 0; // 0: images, 1: videos
        this.currentPath = '';
        this.theme = localStorage.getItem('theme') || 'dark';
        this.liveViewEnabled = localStorage.getItem('liveViewEnabled') !== 'false';
        this.init();
    }

    init() {
        this.applyTheme();
        this.bindEvents();
        this.setupJoystick();
        this.connectWebSocket();
        this.loadInitialData();
    }

    toast(msg, type = 'success') {
        const container = document.getElementById('toast-container');
        if (!container) return;
        const el = document.createElement('div');
        el.className = `toast ${type}`;
        el.innerText = msg;
        container.appendChild(el);
        setTimeout(() => el.remove(), 3200);
    }

    applyTheme() {
        const icon = document.querySelector('#theme-toggle-btn i');
        if (this.theme === 'light') {
            document.body.classList.add('light-mode');
            if (icon) icon.className = 'fas fa-sun';
        } else {
            document.body.classList.remove('light-mode');
            if (icon) icon.className = 'fas fa-moon';
        }
    }

    toggleTheme() {
        this.theme = this.theme === 'dark' ? 'light' : 'dark';
        localStorage.setItem('theme', this.theme);
        this.applyTheme();
    }

    switchView(view) {
        const dash = document.getElementById('dashboard-view');
        const settings = document.getElementById('settings-view');
        const dashBtn = document.getElementById('nav-dashboard-btn');
        const settingsBtn = document.getElementById('nav-settings-btn');
        if (view === 'settings') {
            dash.style.display = 'none';
            settings.style.display = 'block';
            dashBtn.classList.remove('active');
            settingsBtn.classList.add('active');
            this.loadAllSettings();
        } else {
            settings.style.display = 'none';
            dash.style.display = 'grid';
            settingsBtn.classList.remove('active');
            dashBtn.classList.add('active');
            if (this.liveViewEnabled) {
                const stream = document.getElementById('video-stream');
                if (stream) stream.src = `/api/stream/video?t=${Date.now()}`;
            }
        }
    }

    async loadInitialData() {
        await Promise.allSettled([this.loadAllSettings(), this.loadMedia()]);
        this.updateLiveView();
    }

    updateLiveView() {
        const stream = document.getElementById('video-stream');
        const toggle = document.getElementById('config-live-view-toggle');
        if (toggle) toggle.checked = this.liveViewEnabled;

        if (this.liveViewEnabled) {
            stream.src = `/api/stream/video?t=${Date.now()}`;
            this.post('/api/stream/toggle', null, { enabled: true });
        } else {
            stream.src = '';
            this.post('/api/stream/toggle', null, { enabled: false });
        }
        localStorage.setItem('liveViewEnabled', this.liveViewEnabled);
    }

    bindEvents() {
        // Navigation
        document.getElementById('nav-dashboard-btn').onclick = () => this.switchView('dashboard');
        document.getElementById('nav-settings-btn').onclick = () => this.switchView('settings');
        document.getElementById('theme-toggle-btn').onclick = () => this.toggleTheme();
        document.getElementById('refresh-settings-btn').onclick = () => this.loadAllSettings(true);

        document.addEventListener('visibilitychange', () => {
            if (!document.hidden && this.liveViewEnabled) {
                const stream = document.getElementById('video-stream');
                if (stream) stream.src = `/api/stream/video?t=${Date.now()}`;
            }
        });

        const videoEl = document.getElementById('video-stream');
        if (videoEl) {
            videoEl.onerror = () => {
                if (this.liveViewEnabled) {
                    setTimeout(() => {
                        if (this.liveViewEnabled) {
                            videoEl.src = `/api/stream/video?t=${Date.now()}`;
                        }
                    }, 1500);
                }
            };
        }

        document.getElementById('config-live-view-toggle').onchange = (e) => {
            this.liveViewEnabled = e.target.checked;
            this.updateLiveView();
        };

        // Gimbal controls
        document.getElementById('center-btn').onclick = () => this.post('/api/gimbal/center');
        document.getElementById('downward-btn').onclick = () =>
            this.post('/api/gimbal/center', null, { action: 4 });
        document.getElementById('lock-btn').onclick = () =>
            this.post('/api/gimbal/mode', { mode: 'lock' });
        document.getElementById('follow-btn').onclick = () =>
            this.post('/api/gimbal/mode', { mode: 'follow' });
        document.getElementById('fpv-btn').onclick = () =>
            this.post('/api/gimbal/mode', { mode: 'fpv' });
        document.getElementById('set-att-btn').onclick = () => {
            const yaw = parseFloat(document.getElementById('att-yaw-input').value) || 0;
            const pitch = parseFloat(document.getElementById('att-pitch-input').value) || 0;
            this.post('/api/gimbal/attitude', { yaw, pitch });
        };

        // Camera actions
        document.getElementById('photo-btn').onclick = () => this.post('/api/camera/photo');
        document.getElementById('record-btn').onclick = () => this.toggleRecord();
        document.getElementById('zoom-in-btn').onmousedown = () =>
            this.post('/api/camera/zoom', null, { direction: 1 });
        document.getElementById('zoom-in-btn').onmouseup = () =>
            this.post('/api/camera/zoom', null, { direction: 0 });
        document.getElementById('zoom-out-btn').onmousedown = () =>
            this.post('/api/camera/zoom', null, { direction: -1 });
        document.getElementById('zoom-out-btn').onmouseup = () =>
            this.post('/api/camera/zoom', null, { direction: 0 });
        document.getElementById('abs-zoom-btn').onclick = () => {
            const val = parseFloat(document.getElementById('abs-zoom-input').value);
            if (!isNaN(val) && val >= 1) {
                this.post('/api/camera/zoom', null, { absolute: val });
            }
        };
        document.getElementById('focus-near-btn').onmousedown = () =>
            this.post('/api/camera/focus', null, { direction: 1 });
        document.getElementById('focus-near-btn').onmouseup = () =>
            this.post('/api/camera/focus', null, { direction: 0 });
        document.getElementById('focus-far-btn').onmousedown = () =>
            this.post('/api/camera/focus', null, { direction: -1 });
        document.getElementById('focus-far-btn').onmouseup = () =>
            this.post('/api/camera/focus', null, { direction: 0 });

        // Media Browser
        document.getElementById('refresh-media-btn').onclick = () => this.loadMedia();
        document.getElementById('media-tab-img').onclick = () => this.setMediaMode(0);
        document.getElementById('media-tab-vid').onclick = () => this.setMediaMode(1);

        // Settings Page Actions
        document.getElementById('btn-connect-ip').onclick = async () => {
            const ip = document.getElementById('set-target-ip').value.trim();
            if (!ip) return;
            const res = await this.post('/api/config/ip', { ip });
            if (res?.status === 'ok') {
                document.getElementById('camera-ip-display').innerText = ip;
                this.toast(`Connecting to ${ip}...`);
                setTimeout(() => this.loadInitialData(), 1500);
            }
        };
        document.getElementById('btn-save-static-ip').onclick = async () => {
            const ip = document.getElementById('set-static-ip').value.trim();
            const mask = document.getElementById('set-static-mask').value.trim();
            const gateway = document.getElementById('set-static-gw').value.trim();
            await this.post('/api/system/ip', { ip, mask, gateway });
            this.toast('Static IP updated');
        };
        document.getElementById('btn-sync-utc').onclick = async () => {
            await this.post('/api/system/time', {});
            this.toast('UTC time synchronized');
            this.loadAllSettings();
        };

        document.getElementById('btn-save-rec').onclick = () =>
            this.saveStream(0, 'rec-enc-type', 'rec-res', 'rec-bitrate', 'Recording');
        document.getElementById('btn-save-main').onclick = () =>
            this.saveStream(1, 'main-enc-type', 'main-res', 'main-bitrate', 'Main');
        document.getElementById('btn-save-sub').onclick = () =>
            this.saveStream(2, 'sub-enc-type', 'sub-res', 'sub-bitrate', 'Sub');

        document.getElementById('btn-save-cam-prefs').onclick = async () => {
            await this.post('/api/camera/picname', {
                file_type: 0,
                name_type: parseInt(document.getElementById('set-picname-photo').value, 10),
            });
            await this.post('/api/camera/picname', {
                file_type: 2,
                name_type: parseInt(document.getElementById('set-picname-video').value, 10),
            });
            await this.post('/api/camera/osd', {
                enabled: document.getElementById('set-osd').value === 'true',
            });
            await this.post('/api/camera/stitching', {
                mode: parseInt(document.getElementById('set-stitching').value, 10),
            });
            this.toast('Camera preferences saved');
        };

        document.getElementById('btn-hdr-toggle').onclick = () =>
            this.post('/api/camera/capture', { func: 1 });
        document.getElementById('btn-hdmi-on').onclick = () =>
            this.post('/api/camera/capture', { func: 6 });
        document.getElementById('btn-cvbs-on').onclick = () =>
            this.post('/api/camera/capture', { func: 7 });
        document.getElementById('btn-zoom-linkage').onclick = () =>
            this.post('/api/camera/capture', { func: 10 });

        document.getElementById('btn-save-gimbal').onclick = async () => {
            await this.post('/api/gimbal/mode', {
                mode: document.getElementById('set-gimbal-mode').value,
            });
            await this.post('/api/gimbal/weak', {
                enabled: document.getElementById('set-weak-mode').value === 'true',
            });
            this.toast('Gimbal settings applied');
        };

        document.getElementById('btn-save-payload').onclick = async () => {
            await this.post('/api/thermal', {
                pseudo_color: parseInt(document.getElementById('set-thermal-palette').value, 10),
                gain: parseInt(document.getElementById('set-thermal-gain').value, 10),
            });
            await this.post('/api/ai', {
                enabled: document.getElementById('set-ai-stream').value === 'true',
            });
            this.toast('Payload settings applied');
        };
        document.getElementById('btn-ir-shutter').onclick = () =>
            this.post('/api/thermal', { shutter: true });
        document.getElementById('btn-read-laser').onclick = async () => {
            const res = await this.get('/api/laser');
            if (res?.distance_m != null) {
                this.toast(`Laser Distance: ${res.distance_m} m`);
            } else {
                this.toast('Laser unavailable or out of range', 'error');
            }
        };

        document.getElementById('format-sd-btn').onclick = () => this.formatSD();
        document.getElementById('reboot-camera-btn').onclick = () => this.reboot(true, false);
        document.getElementById('reboot-gimbal-btn').onclick = () => this.reboot(false, true);
    }

    async saveStream(streamType, encId, resId, bitrateId, label) {
        const enc_type = parseInt(document.getElementById(encId).value, 10);
        const resolution = document.getElementById(resId).value;
        const bitrate_kbps = parseInt(document.getElementById(bitrateId).value, 10) || 4000;
        const res = await this.post('/api/camera/encoding', {
            stream_type: streamType,
            enc_type,
            resolution,
            bitrate_kbps,
        });
        if (res?.status === 'ok') {
            this.toast(`${label} stream saved`);
            this.loadAllSettings();
        } else {
            this.toast(`Failed to save ${label} stream`, 'error');
        }
    }

    async loadAllSettings(showToast = false) {
        const s = await this.get('/api/settings');
        if (!s) return;

        if (s.target_ip) {
            document.getElementById('camera-ip-display').innerText = s.target_ip;
            document.getElementById('set-target-ip').value = s.target_ip;
        }
        if (s.hardware) {
            document.getElementById('info-camera-type').innerText = s.hardware.camera_type || '---';
        }
        if (s.firmware) {
            document.getElementById('info-camera-fw').innerText = s.firmware.camera || '---';
            document.getElementById('info-gimbal-fw').innerText = s.firmware.gimbal || '---';
            document.getElementById('info-zoom-fw').innerText = s.firmware.zoom || 'n/a';
        }
        if (s.gimbal_mode) {
            document.getElementById('info-gimbal-mode').innerText = s.gimbal_mode.name || '---';
            const modeName = (s.gimbal_mode.name || '').toLowerCase();
            if (['follow', 'lock', 'fpv'].includes(modeName)) {
                document.getElementById('set-gimbal-mode').value = modeName;
            }
        }

        const populateStream = (enc, encId, resId, bitrateId, fpsId) => {
            if (!enc) return;
            document.getElementById(encId).value = String(enc.enc_type || 1);
            const resStr = `${enc.resolution_w}x${enc.resolution_h}`;
            const sel = document.getElementById(resId);
            if (![...sel.options].some((o) => o.value === resStr)) {
                sel.add(new Option(resStr, resStr));
            }
            sel.value = resStr;
            document.getElementById(bitrateId).value = enc.bitrate_kbps || '';
            document.getElementById(fpsId).value = enc.frame_rate ? `${enc.frame_rate} FPS` : '---';
        };

        populateStream(s.encoding_recording, 'rec-enc-type', 'rec-res', 'rec-bitrate', 'rec-fps');
        populateStream(s.encoding_main, 'main-enc-type', 'main-res', 'main-bitrate', 'main-fps');
        populateStream(s.encoding_sub, 'sub-enc-type', 'sub-res', 'sub-bitrate', 'sub-fps');

        if (s.encoding_main) {
            document.getElementById('info-resolution').innerText =
                `${s.encoding_main.resolution_w}x${s.encoding_main.resolution_h} @ ${s.encoding_main.frame_rate}fps`;
            document.getElementById('info-bitrate').innerText = `${s.encoding_main.bitrate_kbps} kbps`;
        }
        if (s.encoding_recording) {
            document.getElementById('info-rec-resolution').innerText =
                `${s.encoding_recording.resolution_w}x${s.encoding_recording.resolution_h} (${s.encoding_recording.bitrate_kbps} kbps)`;
        }

        if (s.ip_config) {
            document.getElementById('set-static-ip').value = s.ip_config.ip || '';
            document.getElementById('set-static-mask').value = s.ip_config.mask || '';
            document.getElementById('set-static-gw').value = s.ip_config.gateway || '';
        }
        if (s.system_time?.unix_usec) {
            const d = new Date(s.system_time.unix_usec / 1000);
            document.getElementById('set-sys-time-display').innerText = `System Time: ${d.toISOString()}`;
        }
        if (s.pic_name_photo != null) {
            document.getElementById('set-picname-photo').value = String(s.pic_name_photo);
        }
        if (s.pic_name_video != null) {
            document.getElementById('set-picname-video').value = String(s.pic_name_video);
        }
        if (s.osd_enabled != null) {
            document.getElementById('set-osd').value = String(s.osd_enabled);
        }
        if (s.stitching_mode != null) {
            document.getElementById('set-stitching').value = String(s.stitching_mode);
        }
        if (s.weak_control != null) {
            document.getElementById('set-weak-mode').value = String(s.weak_control);
        }
        if (s.camera_info) {
            document.getElementById('set-motor-info').innerText =
                `Mounting: ${s.camera_info.gimbal_mounting_dir === 2 ? 'UpsideDown' : 'Normal'} | Zoom Linkage: ${s.camera_info.zoom_linkage ? 'On' : 'Off'}`;
        }
        if (s.pseudo_color != null) {
            document.getElementById('set-thermal-palette').value = String(s.pseudo_color);
        }
        if (s.thermal_gain != null) {
            document.getElementById('set-thermal-gain').value = String(s.thermal_gain);
        }
        if (s.ai_mode != null) {
            document.getElementById('set-ai-stream').value = String(s.ai_mode);
        }
        if (showToast) {
            this.toast('Settings refreshed from gimbal');
        }
    }

    setMediaMode(mode) {
        this.currentMediaMode = mode;
        document.getElementById('media-tab-img').classList.toggle('active', mode === 0);
        document.getElementById('media-tab-vid').classList.toggle('active', mode === 1);
        this.loadMedia();
    }

    async post(url, body = {}, params = {}) {
        const queryParams = new URLSearchParams(params).toString();
        const fullUrl = queryParams ? `${url}?${queryParams}` : url;
        try {
            const resp = await fetch(fullUrl, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: body ? JSON.stringify(body) : null,
            });
            if (!resp.ok) return null;
            return await resp.json();
        } catch (e) {
            console.error('POST failed', e);
            return null;
        }
    }

    async get(url, params = {}) {
        const queryParams = new URLSearchParams(params).toString();
        const fullUrl = queryParams ? `${url}?${queryParams}` : url;
        try {
            const resp = await fetch(fullUrl);
            if (!resp.ok) return null;
            return await resp.json();
        } catch (e) {
            console.error('GET failed', e);
            return null;
        }
    }

    async toggleRecord() {
        await this.post('/api/camera/record');
        const badge = document.getElementById('recording-badge');
        badge.style.display = badge.style.display === 'none' ? 'block' : 'none';
        document.getElementById('record-btn').classList.toggle('btn-danger');
    }

    setupJoystick() {
        const zone = document.getElementById('joystick-zone');
        const handle = document.getElementById('joystick-handle');
        let moveInterval = null;
        let lastVel = { yaw: 0, pitch: 0 };
        let rotatePending = false;

        const handleMove = (e) => {
            if (!this.joystickActive) return;
            const rect = zone.getBoundingClientRect();
            const center = { x: rect.width / 2, y: rect.height / 2 };
            const clientX = e.touches ? e.touches[0].clientX : e.clientX;
            const clientY = e.touches ? e.touches[0].clientY : e.clientY;
            let dx = clientX - rect.left - center.x;
            let dy = clientY - rect.top - center.y;
            const dist = Math.min(60, Math.sqrt(dx * dx + dy * dy));
            const angle = Math.atan2(dy, dx);
            const x = Math.cos(angle) * dist;
            const y = Math.sin(angle) * dist;
            handle.style.transform = `translate(calc(-50% + ${x}px), calc(-50% + ${y}px))`;
            lastVel = { yaw: Math.round((x / 60) * 100), pitch: Math.round((-y / 60) * 100) };
        };

        const startMove = (e) => {
            this.joystickActive = true;
            handleMove(e);
            moveInterval = setInterval(async () => {
                if (rotatePending) return;
                rotatePending = true;
                try {
                    await this.post('/api/gimbal/rotate', lastVel);
                } finally {
                    rotatePending = false;
                }
            }, 150);
        };

        const endMove = () => {
            if (!this.joystickActive) return;
            this.joystickActive = false;
            clearInterval(moveInterval);
            handle.style.transform = 'translate(-50%, -50%)';
            this.post('/api/gimbal/rotate', { yaw: 0, pitch: 0 });
        };

        zone.onmousedown = startMove;
        window.addEventListener('mousemove', handleMove);
        window.addEventListener('mouseup', endMove);
        zone.ontouchstart = (e) => { e.preventDefault(); startMove(e); };
        window.addEventListener('touchmove', handleMove);
        window.addEventListener('touchend', endMove);
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        this.ws = new WebSocket(`${protocol}//${window.location.host}/ws/attitude`);
        this.ws.onopen = () => {
            document.getElementById('connection-indicator').classList.add('online');
            document.getElementById('camera-status').innerText = 'CONNECTED';
            document.getElementById('offline-overlay').classList.remove('active');
        };
        this.ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            document.getElementById('att-yaw').innerText = (data.yaw ?? 0).toFixed(1);
            document.getElementById('att-pitch').innerText = (data.pitch ?? 0).toFixed(1);
            document.getElementById('att-roll').innerText = (data.roll ?? 0).toFixed(1);
        };
        this.ws.onclose = () => {
            document.getElementById('connection-indicator').classList.remove('online');
            document.getElementById('camera-status').innerText = 'DISCONNECTED';
            document.getElementById('offline-overlay').classList.add('active');
            setTimeout(() => this.connectWebSocket(), 2000);
        };
    }

    async loadMedia() {
        const grid = document.getElementById('media-grid');
        const badge = document.getElementById('media-count-badge');
        if (badge) badge.innerText = '';
        grid.innerHTML = '<div class="media-card"><div class="media-thumb"><i class="fas fa-spinner fa-spin"></i></div></div>';
        document.getElementById('media-path-breadcrumb').innerText =
            this.currentMediaMode === 0 ? '/root/photo' : '/root/video';

        const dirs = await this.get('/api/media/directories', { type: this.currentMediaMode });
        if (!dirs || dirs.length === 0) {
            grid.innerHTML = '<div class="overlay-card" style="grid-column: 1/-1; text-align: center;">No media found</div>';
            return;
        }
        grid.innerHTML = '';
        for (const dir of dirs) {
            const el = document.createElement('div');
            el.className = 'media-card';
            el.innerHTML = `
                <div class="media-thumb"><i class="fas fa-folder fa-2x"></i></div>
                <div class="media-info"><div class="media-name">${dir.name}</div></div>
            `;
            el.onclick = () => this.openDirectory(dir.path);
            grid.appendChild(el);
        }
    }

    async openDirectory(path) {
        this.currentPath = path;
        document.getElementById('media-path-breadcrumb').innerText = path;
        const grid = document.getElementById('media-grid');
        grid.innerHTML = '<div class="media-card"><div class="media-thumb"><i class="fas fa-spinner fa-spin"></i></div></div>';

        const files = await this.get('/api/media/files', { path, type: this.currentMediaMode });
        grid.innerHTML = '';

        const back = document.createElement('div');
        back.className = 'media-card';
        back.innerHTML = `
            <div class="media-thumb"><i class="fas fa-arrow-left fa-2x"></i></div>
            <div class="media-info"><div class="media-name">Back</div></div>
        `;
        back.onclick = () => this.loadMedia();
        grid.appendChild(back);

        const badge = document.getElementById('media-count-badge');
        if (badge) badge.innerText = files ? `${files.length} items` : '0 items';
        if (!files) return;

        for (const file of files) {
            const el = document.createElement('div');
            el.className = 'media-card';
            const thumb = this.currentMediaMode === 0
                ? `<img src="/api/media/download?url=${encodeURIComponent(file.url)}" loading="lazy" alt="${file.name}" />`
                : `<i class="fas fa-file-video fa-2x"></i>`;
            el.innerHTML = `
                <div class="media-thumb">${thumb}</div>
                <div class="media-info">
                    <div class="media-name">${file.name}</div>
                    <a href="/api/media/download?url=${encodeURIComponent(file.url)}" download="${file.name}" class="btn btn-sm btn-primary" style="margin-top: 8px; width: 100%;">Download</a>
                </div>
            `;
            grid.appendChild(el);
        }
    }

    async formatSD() {
        if (!confirm('Format SD card? All photos and videos will be permanently deleted!')) return;
        const res = await this.post('/api/storage/format');
        this.toast(res?.status === 'ok' ? 'SD Card formatted' : 'Format failed', res?.status === 'ok' ? 'success' : 'error');
        this.loadMedia();
    }

    async reboot(camera, gimbal) {
        if (!confirm(`Reboot ${camera ? 'Camera' : ''}${camera && gimbal ? ' & ' : ''}${gimbal ? 'Gimbal' : ''}?`)) return;
        await this.post('/api/system/reboot', { camera, gimbal });
        this.toast('Reboot command sent');
    }
}

window.addEventListener('DOMContentLoaded', () => {
    window.siyiApp = new SIYIApp();
});
