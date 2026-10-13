package ui

import (
	"clipare"
	"clipare/internal/config"
	"clipare/internal/update"
	"encoding/json"
	"errors"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// ReactService is the only Go service exported to the WebView. It does not
// expose arbitrary filesystem, process, network, or clipboard operations.
type ReactService struct{ d *reactDesktop }

type reactCommand struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Name  string `json:"name"`
	Code  string `json:"code"`
	View  string `json:"view"`
	Value any    `json:"value"`
}
type reactDevice struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Online   bool   `json:"online"`
	Address  string `json:"address,omitempty"`
}
type reactHistoryEntry struct {
	ID          string `json:"id"`
	Group       string `json:"group"`
	Source      string `json:"source"`
	Time        string `json:"time"`
	VisibleText string `json:"visibleText"`
	Sensitive   bool   `json:"sensitive"`
	Revealed    bool   `json:"revealed"`
	Expanded    bool   `json:"expanded"`
	Truncated   bool   `json:"truncated"`
	URL         string `json:"url,omitempty"`
}
type reactDiscovered struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	Platform string `json:"platform"`
}
type reactPairing struct {
	Direction              string `json:"direction"`
	DeviceName             string `json:"deviceName"`
	Code                   string `json:"code"`
	AwaitingApproval       bool   `json:"awaitingApproval"`
	NeedsLocalConfirmation bool   `json:"needsLocalConfirmation"`
}
type reactUpdate struct {
	Kind    string `json:"kind"`
	Version string `json:"version,omitempty"`
	Message string `json:"message,omitempty"`
	Retry   string `json:"retry,omitempty"`
}
type reactSettings struct {
	Autostart        bool   `json:"autostart"`
	AutoCheckUpdates bool   `json:"autoCheckUpdates"`
	UpdateChannel    string `json:"updateChannel"`
	HistoryEnabled   bool   `json:"historyEnabled"`
	HistoryRetention string `json:"historyRetention"`
	HistoryLimit     int    `json:"historyLimit"`
}
type reactAdvanced struct {
	DeviceID             string   `json:"deviceId"`
	ListenAddress        string   `json:"listenAddress"`
	Addresses            []string `json:"addresses"`
	Port                 int      `json:"port"`
	PublicKeyFingerprint string   `json:"publicKeyFingerprint"`
	LegacyKeyPresent     bool     `json:"legacyKeyPresent"`
}
type reactSync struct {
	State   string `json:"state"`
	Message string `json:"message"`
	Reason  string `json:"reason,omitempty"`
}
type reactSnapshot struct {
	Platform         string              `json:"platform"`
	Theme            string              `json:"theme"`
	Accent           string              `json:"accent"`
	Version          string              `json:"version"`
	DeviceName       string              `json:"deviceName"`
	Sync             reactSync           `json:"sync"`
	Devices          []reactDevice       `json:"devices"`
	History          []reactHistoryEntry `json:"history"`
	Discovered       []reactDiscovered   `json:"discovered"`
	DiscoveryMessage string              `json:"discoveryMessage,omitempty"`
	Pairing          *reactPairing       `json:"pairing,omitempty"`
	Update           reactUpdate         `json:"update"`
	Settings         reactSettings       `json:"settings"`
	Advanced         reactAdvanced       `json:"advanced"`
	Busy             bool                `json:"busy"`
	Notice           string              `json:"notice,omitempty"`
}
type queuedReactAction struct {
	event                                            int
	selectedPeer, selectedHistory, selectedDiscovery string
	mutate                                           func(*reactDesktop)
	view                                             ViewState
	setView                                          bool
}
type reactDesktop struct {
	mu                          sync.RWMutex
	snapshot                    reactSnapshot
	form                        form
	peers                       []config.Peer
	platforms                   map[string]string
	discovered                  []reactDiscovered
	historyIDs                  []string
	selected, discoverySelected int
	queue                       []queuedReactAction
	view                        ViewState
	publish                     func(reactSnapshot)
	openExternal                func(string) error
	lastPublished               string
}

func newReactDesktop(publish func(reactSnapshot)) *reactDesktop {
	platform := "windows"
	if runtime.GOOS == "darwin" {
		platform = "macos"
	}
	return &reactDesktop{selected: -1, discoverySelected: -1, publish: publish,
		snapshot: reactSnapshot{Platform: platform, Theme: "light", Accent: "#0f6cbd", Version: clipare.Version(),
			Sync:    reactSync{State: "degraded", Message: "Запуск Clipare…"},
			Devices: []reactDevice{}, History: []reactHistoryEntry{}, Discovered: []reactDiscovered{},
			Update: reactUpdate{Kind: "idle"}, Advanced: reactAdvanced{Addresses: []string{}}}}
}
func (s *ReactService) Snapshot() reactSnapshot {
	s.d.mu.RLock()
	defer s.d.mu.RUnlock()
	return cloneReactSnapshot(s.d.snapshot)
}
func cloneReactSnapshot(source reactSnapshot) reactSnapshot {
	encoded, err := json.Marshal(source)
	if err != nil {
		return reactSnapshot{}
	}
	var copy reactSnapshot
	if json.Unmarshal(encoded, &copy) != nil {
		return reactSnapshot{}
	}
	return copy
}
func (d *reactDesktop) emitLocked() {
	if d.publish == nil {
		return
	}
	encoded, err := json.Marshal(d.snapshot)
	if err != nil || string(encoded) == d.lastPublished {
		return
	}
	d.lastPublished = string(encoded)
	var copy reactSnapshot
	if json.Unmarshal(encoded, &copy) == nil {
		d.publish(copy)
	}
}
func (d *reactDesktop) hasPeer(id string) bool {
	for _, p := range d.peers {
		if p.ID == id {
			return true
		}
	}
	return false
}
func (d *reactDesktop) hasHistory(id string) bool {
	for _, known := range d.historyIDs {
		if known == id {
			return true
		}
	}
	return false
}

// Perform validates the command and queues it for the existing serialized Go
// GUI loop. The frontend cannot invoke backend services directly.
func (s *ReactService) Perform(c reactCommand) error {
	d := s.d
	if c.Type == "history.openLink" {
		d.mu.RLock()
		var link string
		for _, entry := range d.snapshot.History {
			if entry.ID == c.ID {
				link = entry.URL
				break
			}
		}
		open := d.openExternal
		d.mu.RUnlock()
		parsed, err := url.Parse(link)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Hostname() == "" || parsed.User != nil {
			return errors.New("Ссылка недоступна")
		}
		if open == nil {
			return errors.New("Не удалось открыть ссылку")
		}
		return open(parsed.String())
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	a, err := d.action(c)
	if err != nil {
		return err
	}
	if a.event == eventNone && !a.setView {
		return nil
	}
	if len(d.queue) >= 32 {
		return errors.New("Интерфейс занят. Повторите действие")
	}
	d.queue = append(d.queue, a)
	if c.Type == "device.importCode" || c.Type == "settings.manualPeer" || c.Type == "settings.generateLegacyKey" {
		d.queue = append(d.queue, queuedReactAction{event: eventSave})
	}
	return nil
}

func (d *reactDesktop) action(c reactCommand) (queuedReactAction, error) {
	var a queuedReactAction
	switch c.Type {
	case "sync.toggle":
		a.event = eventPause
	case "device.rename":
		if strings.TrimSpace(c.Name) == "" || len(c.Name) > 256 {
			return a, errors.New("Некорректное имя устройства")
		}
		a.event = eventDeviceName
		a.mutate = func(d *reactDesktop) { d.form.Values[0] = c.Name }
	case "device.remove":
		if !d.hasPeer(c.ID) {
			return a, errors.New("Устройство больше не найдено")
		}
		a.event, a.selectedPeer = eventRemove, c.ID
	case "device.discover":
		a.event, a.view, a.setView = eventAdd, ViewDiscovery, true
	case "device.connect":
		found := false
		for _, p := range d.discovered {
			if p.ID == c.ID {
				found = true
				break
			}
		}
		if !found {
			return a, errors.New("Устройство больше не найдено")
		}
		a.event, a.selectedDiscovery, a.view, a.setView = eventConnect, c.ID, ViewPairing, true
	case "device.importCode":
		if strings.TrimSpace(c.Code) == "" || len(c.Code) > 8192 {
			return a, errors.New("Некорректный код подключения")
		}
		a.event = eventImport
		a.mutate = func(d *reactDesktop) { d.form.Values[10] = c.Code }
	case "pair.approve":
		a.event = eventApprove
	case "pair.reject":
		a.event = eventReject
	case "pair.cancel":
		a.event = eventCancelPair
	case "pair.confirmLocal":
		a.event = eventConfirmLocal
	case "history.open":
		a.event, a.view, a.setView = eventHistoryOpen, ViewHistory, true
	case "history.clear":
		a.event = eventHistoryClear
	case "settings.autostart", "settings.autoCheckUpdates", "settings.historyEnabled":
		value, ok := c.Value.(bool)
		if !ok {
			return a, errors.New("Некорректное значение переключателя")
		}
		switch c.Type {
		case "settings.autostart":
			a.event = eventAutostartPreference
			a.mutate = func(d *reactDesktop) { d.form.Autostart = value }
		case "settings.autoCheckUpdates":
			a.event = eventUpdatePreference
			a.mutate = func(d *reactDesktop) { d.snapshot.Settings.AutoCheckUpdates = value }
		case "settings.historyEnabled":
			a.event = eventHistoryEnabled
			a.mutate = func(d *reactDesktop) { d.snapshot.Settings.HistoryEnabled = value }
		}
	case "settings.updateChannel":
		value, ok := c.Value.(string)
		if !ok || value != update.ChannelStable && value != update.ChannelBeta {
			return a, errors.New("Некорректный канал обновлений")
		}
		a.event = eventUpdateChannel
		a.mutate = func(d *reactDesktop) { d.snapshot.Settings.UpdateChannel = value }
	case "settings.historyRetention":
		value, ok := c.Value.(string)
		if !ok || RetentionToLabel(value) == "" {
			return a, errors.New("Некорректный срок хранения")
		}
		a.event = eventHistorySettings
		a.mutate = func(d *reactDesktop) { d.form.Values[12] = RetentionToLabel(value) }
	case "settings.historyLimit":
		value, ok := c.Value.(float64)
		if !ok || value < config.HistoryMinItems || value > config.HistoryMaxItems || value != float64(int(value)) {
			return a, errors.New("Некорректный лимит истории")
		}
		a.event = eventHistorySettings
		a.mutate = func(d *reactDesktop) { d.form.Values[11] = strconv.Itoa(int(value)) }
	case "settings.advanced":
		value, ok := c.Value.(map[string]any)
		if !ok {
			return a, errors.New("Некорректные параметры")
		}
		address, aok := value["listenAddress"].(string)
		port, pok := value["port"].(float64)
		if !aok || !pok || port < 1 || port > 65535 || port != float64(int(port)) {
			return a, errors.New("Некорректный адрес или порт")
		}
		name, hasName := value["deviceName"].(string)
		if hasName && (strings.TrimSpace(name) == "" || len(name) > 256) {
			return a, errors.New("Некорректное имя устройства")
		}
		a.event = eventSave
		a.mutate = func(d *reactDesktop) {
			if hasName {
				d.form.Values[0] = strings.TrimSpace(name)
			}
			d.form.Values[2], d.form.Values[3] = address, strconv.Itoa(int(port))
		}
	case "settings.generateLegacyKey":
		a.event = eventGenerate
	case "settings.copyLegacyCode":
		a.event = eventCopy
	case "settings.manualPeer":
		value, ok := c.Value.(map[string]any)
		if !ok {
			return a, errors.New("Некорректные данные устройства")
		}
		name, nok := value["name"].(string)
		id, iok := value["id"].(string)
		address, aok := value["address"].(string)
		port, pok := value["port"].(float64)
		if !nok || !iok || !aok || !pok || name == "" || id == "" || address == "" || port < 1 || port > 65535 || port != float64(int(port)) {
			return a, errors.New("Заполните данные устройства")
		}
		a.event = eventUpsert
		a.mutate = func(d *reactDesktop) {
			d.selected = -1
			d.form.Values[6], d.form.Values[7], d.form.Values[8], d.form.Values[9] = name, id, address, strconv.Itoa(int(port))
		}
	case "history.copy", "history.delete", "history.reveal", "history.expand":
		if !d.hasHistory(c.ID) {
			return a, errors.New("Запись истории больше не найдена")
		}
		a.selectedHistory = c.ID
		switch c.Type {
		case "history.copy":
			a.event = eventHistoryCopy
		case "history.delete":
			a.event = eventHistoryDelete
		case "history.reveal":
			a.event = eventHistoryEye
		case "history.expand":
			a.event = eventHistoryExpand
		}
	case "update.check":
		a.event, a.view, a.setView = eventCheckUpdate, ViewUpdate, true
	case "update.install":
		a.event = eventInstallUpdate
	case "update.dismiss":
		a.event = eventLaterUpdate
	case "view.open":
		switch c.View {
		case "home":
			a.view, a.setView = ViewHome, true
			if d.view == ViewDiscovery {
				a.event = eventCloseDiscovery
			} else if d.view == ViewPairing {
				if d.snapshot.Pairing != nil && d.snapshot.Pairing.Direction == "incoming" {
					a.event = eventReject
				} else {
					a.event = eventCancelPair
				}
			}
		case "discovery":
			a.view, a.setView = ViewDiscovery, true
		case "pairing":
			a.view, a.setView = ViewPairing, true
		case "history":
			a.event, a.view, a.setView = eventHistoryOpen, ViewHistory, true
		case "updates":
			a.view, a.setView = ViewUpdate, true
		default:
			return a, errors.New("Неизвестный экран")
		}
	case "notice.dismiss":
		d.snapshot.Notice = ""
		d.emitLocked()
	default:
		return a, errors.New("Недоступное действие")
	}
	return a, nil
}
