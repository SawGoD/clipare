package ui

import (
	"clipare/internal/config"
	"strconv"
	"strings"
)

func (d *reactDesktop) Init() error { return nil }
func (d *reactDesktop) Close()      {}
func (d *reactDesktop) Poll() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.queue) == 0 {
		return eventNone
	}
	a := d.queue[0]
	d.queue = d.queue[1:]
	if a.setView {
		d.view = a.view
	}
	if a.mutate != nil {
		a.mutate(d)
	}
	if a.selectedPeer != "" {
		d.selected = -1
		for i, p := range d.peers {
			if p.ID == a.selectedPeer {
				d.selected = i
				break
			}
		}
	}
	if a.selectedHistory != "" {
		d.selected = -1
		for i, id := range d.historyIDs {
			if id == a.selectedHistory {
				d.selected = i
				break
			}
		}
	}
	if a.selectedDiscovery != "" {
		d.discoverySelected = -1
		for i, p := range d.discovered {
			if p.ID == a.selectedDiscovery {
				d.discoverySelected = i
				break
			}
		}
	}
	return a.event
}
func (d *reactDesktop) Read() form    { d.mu.RLock(); defer d.mu.RUnlock(); return d.form }
func (d *reactDesktop) Selected() int { d.mu.RLock(); defer d.mu.RUnlock(); return d.selected }
func (d *reactDesktop) DiscoveredSelected() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.discoverySelected
}
func (d *reactDesktop) CurrentView() ViewState { d.mu.RLock(); defer d.mu.RUnlock(); return d.view }
func (d *reactDesktop) SetPeer(config.Peer)    {}
func (d *reactDesktop) Prepare(f form, peers []config.Peer, addresses []string) {
	d.Show(f, peers, addresses)
}

func (d *reactDesktop) Show(f form, peers []config.Peer, addresses []string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.form, d.peers = f, append([]config.Peer(nil), peers...)
	d.snapshot.DeviceName = f.Values[0]
	d.snapshot.Advanced = reactAdvanced{
		DeviceID: f.Values[1], ListenAddress: f.Values[2], Addresses: append([]string{}, addresses...),
		Port: parsePort(f.Values[3]), PublicKeyFingerprint: strings.TrimPrefix(f.Values[5], "Протокол 1 · Public key SHA-256: "),
		LegacyKeyPresent: f.Values[4] != "",
	}
	d.emitLocked()
}
func parsePort(v string) int { n, _ := strconv.Atoi(v); return n }
func (d *reactDesktop) Update(message string, running bool, peers []config.Peer, online map[string]bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.peers = append([]config.Peer(nil), peers...)
	devices := make([]reactDevice, 0, len(peers))
	for _, p := range peers {
		platform := "unknown"
		if v, ok := d.platforms[p.ID]; ok {
			platform = v
		}
		devices = append(devices, reactDevice{ID: p.ID, Name: peerLabel(p), Platform: platform, Online: online[p.ID], Address: p.Address})
	}
	d.snapshot.Devices = devices
	d.snapshot.DeviceName = d.form.Values[0]
	d.snapshot.Advanced.ListenAddress = d.form.Values[2]
	d.snapshot.Advanced.Port = parsePort(d.form.Values[3])
	d.snapshot.Advanced.LegacyKeyPresent = d.form.Values[4] != ""
	d.snapshot.Busy = message == "Применение настроек…"
	d.emitLocked()
}
func (d *reactDesktop) Alert(message string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.snapshot.Notice = message
	d.emitLocked()
}
func (d *reactDesktop) SyncStatus(s SyncStatus) {
	d.mu.Lock()
	defer d.mu.Unlock()
	state := "degraded"
	if s.State == SyncActive {
		state = "active"
	} else if s.State == SyncDisabled {
		state = "disabled"
	}
	d.snapshot.Sync = reactSync{State: state, Message: s.Message, Reason: s.Reason}
	d.emitLocked()
}
func (d *reactDesktop) Discovered(lines, message string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	rows := discoveredRows(lines)
	// Discovery UI rows preserve index order so eventConnect selects the same
	// element as the original discovery result kept by the Go loop.
	d.discovered = make([]reactDiscovered, 0, len(rows))
	for i, row := range rows {
		d.discovered = append(d.discovered, reactDiscovered{ID: strconv.Itoa(i), Name: row.name, Address: row.detail, Platform: "unknown"})
	}
	d.snapshot.Discovered, d.snapshot.DiscoveryMessage = d.discovered, message
	d.emitLocked()
}
func (d *reactDesktop) Pair(title, code string, mode int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	direction := "outgoing"
	name := strings.TrimPrefix(title, "Проверьте код на ")
	if mode == 1 {
		direction, name = "incoming", strings.TrimSuffix(title, " хочет подключиться")
	}
	d.snapshot.Pairing = &reactPairing{Direction: direction, DeviceName: name, Code: code, AwaitingApproval: true, NeedsLocalConfirmation: mode == 2}
	d.view = ViewPairing
	d.emitLocked()
}
func (d *reactDesktop) PairClose() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.snapshot.Pairing = nil
	d.view = ViewHome
	d.emitLocked()
}
func (d *reactDesktop) UpdateSettings(version string, enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.snapshot.Version, d.snapshot.Settings.AutoCheckUpdates = version, enabled
	d.emitLocked()
}
func (d *reactDesktop) UpdateEnabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.snapshot.Settings.AutoCheckUpdates
}
func (d *reactDesktop) UpdatePrompt(p updatePrompt) {
	d.mu.Lock()
	defer d.mu.Unlock()
	switch {
	case p.Busy && strings.Contains(p.Text, "Проверка"):
		d.snapshot.Update = reactUpdate{Kind: "checking"}
	case p.Busy && strings.Contains(p.Text, "Обновление готово"):
		d.snapshot.Update = reactUpdate{Kind: "installing", Version: d.snapshot.Update.Version}
	case p.Busy:
		d.snapshot.Update = reactUpdate{Kind: "downloading", Version: d.snapshot.Update.Version}
	case p.Primary == "Повторить":
		retry := "check"
		if p.Action == eventInstallUpdate {
			retry = "install"
		}
		d.snapshot.Update = reactUpdate{Kind: "error", Message: p.Text, Retry: retry}
	case p.Action == eventInstallUpdate:
		d.snapshot.Update = reactUpdate{Kind: "available", Version: extractUpdateVersion(p.Text)}
	default:
		d.snapshot.Update = reactUpdate{Kind: "current"}
	}
	d.view = ViewUpdate
	d.emitLocked()
}
func extractUpdateVersion(message string) string {
	for _, line := range strings.Split(message, "\n") {
		if strings.HasPrefix(line, "Доступна: ") {
			version := strings.TrimPrefix(strings.TrimPrefix(line, "Доступна: "), "v")
			return strings.TrimSuffix(version, " (бета)")
		}
	}
	return ""
}
func (d *reactDesktop) UpdateClose() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.snapshot.Update = reactUpdate{Kind: "idle"}
	d.view = ViewHome
	d.emitLocked()
}
func (d *reactDesktop) Preferences(auto, updates bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.form.Autostart = auto
	d.snapshot.Settings.Autostart, d.snapshot.Settings.AutoCheckUpdates = auto, updates
	d.emitLocked()
}
func (d *reactDesktop) UpdateChannelSettings(channel string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.snapshot.Settings.UpdateChannel = channel
	d.emitLocked()
}
func (d *reactDesktop) UpdateChannel() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.snapshot.Settings.UpdateChannel
}
func (d *reactDesktop) HistoryEnabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.snapshot.Settings.HistoryEnabled
}
func (d *reactDesktop) HistoryPrefs(c config.History) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.snapshot.Settings.HistoryEnabled, d.snapshot.Settings.HistoryRetention, d.snapshot.Settings.HistoryLimit = c.Enabled, c.Retention, c.MaxItems
	d.form.Values[11], d.form.Values[12] = strconv.Itoa(c.MaxItems), RetentionToLabel(c.Retention)
	d.emitLocked()
}
func (d *reactDesktop) ShowHistory(rows []historyRow, keyLost bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	entries := make([]reactHistoryEntry, 0, len(rows))
	d.historyIDs = make([]string, len(rows))
	group := ""
	for i, row := range rows {
		if row.group {
			group = row.header
			continue
		}
		d.historyIDs[i] = row.id
		source, clock, _ := strings.Cut(row.header, " · ")
		link := row.link
		if row.sensitive && !row.revealed {
			link = ""
		}
		entries = append(entries, reactHistoryEntry{ID: row.id, Group: group, Source: source, Time: clock, VisibleText: row.preview, Sensitive: row.sensitive, Revealed: row.revealed, Expanded: row.expanded, Truncated: row.truncated, URL: link})
	}
	d.snapshot.History = entries
	if keyLost {
		d.snapshot.Notice = historyKeyLostText
	}
	d.view = ViewHistory
	d.emitLocked()
}
func (d *reactDesktop) Platforms(platforms map[string]string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.platforms = make(map[string]string, len(platforms))
	for id, value := range platforms {
		if value == "darwin" {
			d.platforms[id] = "macos"
		} else if value == "windows" {
			d.platforms[id] = "windows"
		}
	}
	for i := range d.snapshot.Devices {
		if value, ok := d.platforms[d.snapshot.Devices[i].ID]; ok {
			d.snapshot.Devices[i].Platform = value
		}
	}
	d.emitLocked()
}
