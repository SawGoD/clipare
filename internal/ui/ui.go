package ui

import (
	"clipare/internal/app"
	"clipare/internal/autostart"
	"clipare/internal/clipboard"
	"clipare/internal/config"
	"clipare/internal/discovery"
	"clipare/internal/pairing"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	eventNone = iota
	eventSave
	eventSettings
	eventPause
	eventQuit
	eventGenerate
	eventCopy
	eventImport
	eventUpsert
	eventRemove
	eventSelect
	eventAdd
	eventRefresh
	eventConnect
	eventApprove
	eventReject
	eventCancelPair
	eventCloseDiscovery
	eventConfirmLocal
)

type pairingDesktop interface {
	Discovered(string, string)
	DiscoveredSelected() int
	Pair(string, string, int)
	PairClose()
}
type scanResult struct {
	devices []discovery.Device
	err     error
}

type form struct {
	Values    [11]string
	Autostart bool
}
type desktop interface {
	Init() error
	Poll() int
	Show(form, []config.Peer, []string)
	Read() form
	Selected() int
	SetPeer(config.Peer)
	Update(string, bool, []config.Peer, map[string]bool)
	Alert(string)
	Close()
}

func toForm(c config.Config) form {
	return form{Values: [11]string{c.Device.Name, c.Device.ID, c.Listen.Address, strconv.Itoa(c.Listen.Port), c.Security.Secret, "Протокол 1 · Public key SHA-256: " + c.Identity.Fingerprint(), "", "", "", "45873", ""}, Autostart: c.Autostart}
}
func fromForm(f form, c config.Config) (config.Config, error) {
	c.Device.Name = f.Values[0]
	c.Listen.Address = f.Values[2]
	p, e := strconv.Atoi(f.Values[3])
	if e != nil {
		return c, errors.New("Порт должен быть числом от 1 до 65535")
	}
	c.Listen.Port = p
	if c.Security.Secret != f.Values[4] {
		old := c.Security.Secret
		c.Peers = append([]config.Peer(nil), c.Peers...)
		for j := range c.Peers {
			if c.Peers[j].Legacy && (c.Peers[j].LegacySecret == old || c.Peers[j].LegacySecret == "") {
				c.Peers[j].LegacySecret = f.Values[4]
			}
		}
	}
	c.Security.Secret = f.Values[4]
	c.Autostart = f.Autostart
	return c, nil
}

type result struct {
	c       config.Config
	s       *app.Session
	err     error
	persist bool
}
type peerStatus struct {
	id     string
	online bool
}

type sessionStarter func(context.Context, config.Config, clipboard.Backend, *slog.Logger, func(string, bool)) (*app.Session, error)
type loopTiming struct{ poll, refresh, retry time.Duration }

func Run(parent context.Context, path string, log *slog.Logger) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	d, e := newDesktop()
	if e != nil {
		return e
	}
	b, e := clipboard.New()
	if e != nil {
		return e
	}
	return runDesktop(parent, path, log, d, b, nil, loopTiming{30 * time.Millisecond, 2 * time.Second, 5 * time.Second})
}

// The GUI loop is also exercised with a synthetic desktop and clipboard, so
// recovery tests never modify the user's pasteboard, settings or login items.
func runDesktop(parent context.Context, path string, log *slog.Logger, d desktop, b clipboard.Backend, start sessionStarter, timing loopTiming) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	if e := d.Init(); e != nil {
		return e
	}
	defer d.Close()
	c, e := config.LoadMigrated(path)
	needsSetup := e != nil
	first := os.IsNotExist(e)
	if e != nil {
		if !first {
			return errors.New("Не удалось загрузить или обновить настройки. Исходный файл сохранён; проверьте конфигурацию и резервную копию")
		}
		c, e = config.Default()
		if e != nil {
			return e
		}
		if start == nil {
			if status, err := (discovery.CLI{}).Status(ctx); err == nil {
				if address := discovery.LocalAddress(status); address != "" {
					for _, local := range config.Addresses() {
						if local == address {
							c.Listen.Address = address
							break
						}
					}
				}
			}
		}
	}
	draft := c
	pd, hasPairUI := d.(pairingDesktop)
	var service *pairing.Service
	var memberUpdates <-chan config.Config
	var scanCancel, pairCancel context.CancelFunc
	var discovered []discovery.Device
	scanning, discoverOpen, outgoing := false, false, false
	scanDone := make(chan scanResult, 1)
	pairDone := make(chan error, 1)
	sasUpdates := make(chan string, 1)
	var shownIncoming string
	var outgoingName string
	var needsLocalConfirm bool
	var localVerifyReady bool
	localConfirmation := make(chan struct{}, 1)
	var nextScan time.Time
	var controlDone chan struct{}
	var controlWG sync.WaitGroup
	if hasPairUI {
		service = pairing.NewService(path, c)
		memberUpdates = service.Updates
		controlDone = make(chan struct{})
		go func() { defer close(controlDone); service.Propagate(ctx) }()
	}
	if start == nil {
		start = func(ctx context.Context, next config.Config, b clipboard.Backend, log *slog.Logger, health func(string, bool)) (*app.Session, error) {
			if service == nil {
				return app.Start(ctx, next, b, log, health)
			}
			return app.StartWithControl(ctx, next, b, log, health, service)
		}
	}
	defer func() {
		if scanCancel != nil {
			scanCancel()
		}
		if pairCancel != nil {
			pairCancel()
		}
		if controlDone != nil {
			cancel()
			<-controlDone
		}
		controlWG.Wait()
	}()
	renderDiscovery := func(message string) {
		if !hasPairUI {
			return
		}
		var lines []string
		for _, v := range discovered {
			lines = append(lines, v.DeviceName+" — "+v.IP)
		}
		pd.Discovered(strings.Join(lines, "\n"), message)
	}
	scan := func() {
		if scanning || !hasPairUI {
			return
		}
		scanning = true
		renderDiscovery("Поиск устройств…")
		scanCtx, stop := context.WithCancel(ctx)
		scanCancel = stop
		known := map[string]bool{c.Device.ID: true}
		for _, p := range c.Peers {
			known[p.ID] = true
		}
		controlWG.Add(1)
		go func() {
			defer controlWG.Done()
			p := discovery.NewProber()
			defer p.Close()
			v, e := discovery.Scan(scanCtx, discovery.CLI{}, p, known)
			select {
			case scanDone <- scanResult{v, e}:
			case <-ctx.Done():
			}
		}()
	}
	states := map[string]bool{}
	statusUpdates := make(chan peerStatus, 128)
	health := func(id string, online bool) {
		select {
		case statusUpdates <- peerStatus{id, online}:
		case <-ctx.Done():
		default:
		}
	}
	var session *app.Session
	done := make(chan result, 1)
	busy := false
	status := "Настройте подключение"
	var retryAt time.Time
	exe, e := os.Executable()
	if e != nil {
		return e
	}
	apply := func(next config.Config, persist bool) {
		busy = true
		retryAt = time.Time{}
		status = "Применение настроек…"
		states = map[string]bool{}
		d.Update(status, false, c.Peers, states)
		old := session
		previousConfig := c
		previousAutostart := c.Autostart
		session = nil
		go func() {
			if old != nil {
				old.Stop()
			}
			startedConfig := next
			s, err := start(ctx, next, b, log, health)
			if (err == nil || errors.Is(err, app.ErrAddressUnavailable)) && persist {
				var storageErr error
				if service != nil {
					next, storageErr = service.SaveSettings(next)
				} else {
					storageErr = config.Save(path, next)
				}
				if storageErr == nil && (next.Autostart || previousAutostart) {
					storageErr = autostart.Set(next.Autostart, exe, path)
				}
				if storageErr == nil && s != nil && !reflect.DeepEqual(startedConfig, next) {
					s.Stop()
					s, err = start(ctx, next, b, log, health)
				}
				if storageErr != nil {
					err = storageErr
					if s != nil {
						s.Stop()
					}
					s = nil
					if old != nil {
						if service != nil {
							_, _ = service.SaveSettings(previousConfig)
						} else {
							_ = config.Save(path, previousConfig)
						}
					}
				}
			}
			if err != nil && !errors.Is(err, app.ErrAddressUnavailable) && old != nil {
				s, _ = start(ctx, previousConfig, b, log, health)
			}
			done <- result{next, s, err, persist}
		}()
	}
	if needsSetup && hasPairUI && discovery.NetBirdAddress(c.Listen.Address) {
		d.Show(toForm(draft), draft.Peers, config.Addresses())
		apply(c, true)
	} else if needsSetup {
		d.Show(toForm(draft), draft.Peers, config.Addresses())
	} else {
		apply(c, false)
	}
	tick := time.NewTicker(timing.poll)
	defer tick.Stop()
	defer func() {
		cancel()
		if busy {
			r := <-done
			if r.s != nil {
				r.s.Stop()
			}
		}
		if session != nil {
			session.Stop()
		}
	}()
	refresh := time.NewTicker(timing.refresh)
	defer refresh.Stop()
	d.Update(status, false, c.Peers, states)
	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-memberUpdates:
			if reflect.DeepEqual(update, c) {
				continue
			}
			if busy {
				continue
			}
			f := d.Read()
			c = update
			draft.Peers = c.Peers
			draft.Group = c.Group
			draft.Removed = c.Removed
			d.Show(f, draft.Peers, config.Addresses())
			apply(c, false)
		case sr := <-scanDone:
			scanning = false
			nextScan = time.Now().Add(20 * time.Second)
			if !discoverOpen {
				continue
			}
			discovered = sr.devices
			message := "Выберите устройство и нажмите «Подключить»"
			if sr.err != nil {
				message = discovery.ErrUnavailable.Error()
			} else if len(discovered) == 0 {
				message = "Устройства не найдены. Запустите Clipare на другом компьютере или добавьте его по коду"
			}
			renderDiscovery(message)
		case sas := <-sasUpdates:
			localVerifyReady = true
			mode := 0
			if needsLocalConfirm {
				mode = 2
			}
			pd.Pair("Проверьте код на "+outgoingName, sas, mode)
		case err := <-pairDone:
			outgoing = false
			pairCancel = nil
			pd.PairClose()
			if err != nil && !errors.Is(err, context.Canceled) {
				d.Alert(err.Error())
			} else if err == nil {
				d.Alert("Устройство подключено")
			}
		case r := <-done:
			busy = false
			session = r.s
			if errors.Is(r.err, app.ErrAddressUnavailable) {
				c = r.c
				needsSetup = false
				if r.persist {
					draft = c
				}
				retryAt = time.Now().Add(timing.retry)
				status = "Ожидание NetBird: локальный IP ещё недоступен"
			} else if r.err != nil {
				status = r.err.Error()
				d.Alert(status)
				d.Show(toForm(draft), draft.Peers, config.Addresses())
			} else {
				c = r.c
				needsSetup = false
				if r.persist {
					draft = c
				}
				status = "Синхронизация включена"
				if c.Mode() == "disabled" {
					status = "Синхронизация приостановлена"
				}
				if c.Listen.Address == "127.0.0.1" {
					status = "Только локальный доступ — выберите NetBird IP"
				}
			}
			if service != nil {
				latest := service.Config()
				if r.err == nil && !reflect.DeepEqual(latest, c) {
					c = latest
					draft = c
					apply(c, false)
				}
			}
			d.Update(status, session != nil && c.Mode() != "disabled", c.Peers, states)
		case st := <-statusUpdates:
			states[st.id] = st.online
		case <-refresh.C:
			if service != nil {
				if snap, ok := service.Snapshot(); ok && snap.State == pairing.Pending && snap.Session != shownIncoming && !outgoing {
					shownIncoming = snap.Session
					pd.Pair(snap.Name+" хочет подключиться", snap.SAS, 1)
				} else if shownIncoming != "" && (!ok || snap.State != pairing.Pending) {
					shownIncoming = ""
					pd.PairClose()
				}
			}
			if discoverOpen && !outgoing && !scanning && !time.Now().Before(nextScan) {
				scan()
			}
			if session != nil {
				select {
				case <-session.Done():
					if err := session.Err(); err != nil {
						status = "Ошибка синхронизации: " + err.Error()
					}
					session = nil
				default:
				}
			}
			if shouldRetry(time.Now(), retryAt, busy, session != nil, needsSetup) {
				apply(c, false)
			}
			d.Update(status, session != nil && c.Mode() != "disabled", c.Peers, states)
		case <-tick.C:
			action := d.Poll()
			if action == eventQuit {
				return nil
			}
			if action == eventNone {
				continue
			}
			if busy {
				if action != eventSettings {
					continue
				}
			}
			if (outgoing || shownIncoming != "") && (action == eventSave || action == eventPause || action == eventGenerate || action == eventImport || action == eventUpsert || action == eventRemove) {
				d.Alert("Завершите или отмените подключение, прежде чем изменять настройки")
				continue
			}
			switch action {
			case eventAdd, eventRefresh:
				if !hasPairUI {
					continue
				}
				discoverOpen = true
				if !discovery.NetBirdAddress(c.Listen.Address) {
					address := ""
					for _, v := range config.Addresses() {
						if discovery.NetBirdAddress(v) {
							address = v
							break
						}
					}
					if address == "" {
						pd.Discovered("", "NetBird не запущен. Подключите NetBird и нажмите «Обновить»")
						continue
					}
					next := c
					next.Listen.Address = address
					draft = next
					apply(next, true)
				}
				scan()
			case eventCloseDiscovery:
				discoverOpen = false
				if scanCancel != nil {
					scanCancel()
				}
			case eventConnect:
				if !hasPairUI || service == nil || outgoing || shownIncoming != "" {
					continue
				}
				index := pd.DiscoveredSelected()
				if index < 0 || index >= len(discovered) {
					d.Alert("Выберите устройство из списка")
					continue
				}
				outgoing = true
				pairCtx, stop := context.WithCancel(ctx)
				pairCancel = stop
				target := discovered[index]
				outgoingName = target.DeviceName
				if outgoingName == "" {
					outgoingName = target.Name
				}
				needsLocalConfirm = len(c.Peers) > 0
				localVerifyReady = false
				select {
				case <-localConfirmation:
				default:
				}
				controlWG.Add(1)
				go func() {
					defer controlWG.Done()
					err := service.Connect(pairCtx, target, func(sas string) {
						select {
						case sasUpdates <- sas:
						case <-pairCtx.Done():
						}
					}, func(ctx context.Context) error {
						select {
						case <-localConfirmation:
							return nil
						case <-ctx.Done():
							return ctx.Err()
						}
					})
					select {
					case pairDone <- err:
					case <-ctx.Done():
					}
				}()
			case eventApprove, eventReject:
				if service == nil || shownIncoming == "" {
					continue
				}
				if err := service.Decide(shownIncoming, action == eventApprove); err != nil {
					d.Alert(err.Error())
				}
				shownIncoming = ""
				pd.PairClose()
			case eventConfirmLocal:
				if outgoing && needsLocalConfirm && localVerifyReady {
					select {
					case localConfirmation <- struct{}{}:
					default:
					}
				}
			case eventCancelPair:
				if pairCancel != nil {
					pairCancel()
				}
				if hasPairUI {
					pd.PairClose()
				}
			case eventSettings:
				draft = c
				d.Show(toForm(draft), draft.Peers, config.Addresses())
			case eventPause:
				if session == nil {
					d.Alert("Сначала сохраните настройки подключения")
					continue
				}
				next := c
				if next.Mode() == "disabled" {
					next.Sync.Enabled = true
					if next.Sync.Mode == "disabled" {
						next.Sync.Mode = "bidirectional"
					}
				} else {
					next.Sync.Enabled = false
				}
				apply(next, true)
			case eventSave:
				next, err := fromForm(d.Read(), draft)
				if err == nil {
					err = next.Validate()
				}
				if err != nil {
					d.Alert("Проверьте настройки: " + err.Error())
					continue
				}
				draft = next
				apply(next, true)
			case eventGenerate:
				f := d.Read()
				secret, err := config.NewSecret()
				if err != nil {
					d.Alert("Не удалось создать ключ")
					continue
				}
				f.Values[4] = secret
				d.Show(f, draft.Peers, config.Addresses())
			case eventCopy:
				next, err := fromForm(d.Read(), draft)
				if err == nil {
					err = next.Validate()
				}
				if err != nil {
					d.Alert("Сначала задайте корректный IP, порт и ключ")
					continue
				}
				if next.Listen.Address == "127.0.0.1" {
					d.Alert("Выберите NetBird IP перед копированием кода")
					continue
				}
				if err = b.Write(config.ConnectionCode(next)); err != nil {
					d.Alert("Не удалось скопировать код")
				} else {
					d.Alert("Код скопирован. Добавьте его в настройках другого компьютера. Код содержит общий ключ — передавайте его приватно")
				}
			case eventImport:
				f := d.Read()
				next, err := fromForm(f, draft)
				if err == nil {
					next, err = config.AddConnection(next, f.Values[10])
				}
				if err != nil {
					d.Alert(err.Error())
					continue
				}
				draft = next
				d.Show(toForm(draft), draft.Peers, config.Addresses())
				d.Alert("Устройство добавлено. Сохраните настройки, затем скопируйте код этого компьютера и добавьте его на другом")
			case eventUpsert:
				f := d.Read()
				p, err := strconv.Atoi(f.Values[9])
				if err != nil {
					d.Alert("Проверьте порт устройства")
					continue
				}
				next, err := fromForm(f, draft)
				if err != nil {
					d.Alert(err.Error())
					continue
				}
				peer := config.Peer{ID: f.Values[7], Name: f.Values[6], Address: f.Values[8], Port: p, Legacy: true, LegacySecret: draft.Security.Secret}
				next.Peers = append([]config.Peer(nil), draft.Peers...)
				index := d.Selected()
				if index >= 0 && index < len(next.Peers) {
					if !next.Peers[index].Legacy {
						d.Alert("Это устройство подключено безопасным pairing. Для изменения identity удалите его и подключите заново")
						continue
					}
					next.Peers[index] = peer
				} else {
					next.Peers = append(next.Peers, peer)
				}
				if err = next.Validate(); err != nil {
					d.Alert("Проверьте устройство: " + err.Error())
					continue
				}
				draft = next
				d.Show(toForm(draft), draft.Peers, config.Addresses())
			case eventRemove:
				index := d.Selected()
				if index >= 0 && index < len(draft.Peers) {
					next, err := fromForm(d.Read(), draft)
					if err != nil {
						d.Alert(err.Error())
						continue
					}
					next.Peers = append(append([]config.Peer(nil), draft.Peers[:index]...), draft.Peers[index+1:]...)
					if !draft.Peers[index].Legacy {
						next.Removed = append(append([]string(nil), next.Removed...), draft.Peers[index].ID)
					}
					draft = next
					d.Show(toForm(draft), draft.Peers, config.Addresses())
					if hasPairUI {
						apply(next, true)
					}
				}
			case eventSelect:
				index := d.Selected()
				if index >= 0 && index < len(draft.Peers) {
					d.SetPeer(draft.Peers[index])
				} else {
					d.SetPeer(config.Peer{Port: 45873})
				}
			default:
				d.Alert(fmt.Sprintf("Unknown action %d", action))
			}
		}
	}
}

func shouldRetry(now, due time.Time, busy, running, needsSetup bool) bool {
	return !due.IsZero() && !now.Before(due) && !busy && !running && !needsSetup
}
