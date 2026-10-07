package ui

import (
	"clipare/internal/app"
	"clipare/internal/autostart"
	"clipare/internal/clipboard"
	"clipare/internal/config"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strconv"
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
)

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
	return form{Values: [11]string{c.Device.Name, c.Device.ID, c.Listen.Address, strconv.Itoa(c.Listen.Port), c.Security.Secret, "", "", "", "", "45873", ""}, Autostart: c.Autostart}
}
func fromForm(f form, c config.Config) (config.Config, error) {
	c.Device.Name = f.Values[0]
	c.Listen.Address = f.Values[2]
	p, e := strconv.Atoi(f.Values[3])
	if e != nil {
		return c, errors.New("Порт должен быть числом от 1 до 65535")
	}
	c.Listen.Port = p
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
	return runDesktop(parent, path, log, d, b, app.Start, loopTiming{30 * time.Millisecond, 2 * time.Second, 5 * time.Second})
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
	c, e := config.Load(path)
	needsSetup := e != nil
	first := os.IsNotExist(e)
	if e != nil {
		c, e = config.Default()
		if e != nil {
			return e
		}
		if !first {
			d.Alert("Не удалось прочитать настройки. Проверьте и сохраните их заново")
		}
	}
	draft := c
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
			s, err := start(ctx, next, b, log, health)
			if (err == nil || errors.Is(err, app.ErrAddressUnavailable)) && persist {
				storageErr := config.Save(path, next)
				if storageErr == nil && (next.Autostart || previousAutostart) {
					storageErr = autostart.Set(next.Autostart, exe, path)
				}
				if storageErr != nil {
					err = storageErr
					if s != nil {
						s.Stop()
					}
					s = nil
					if old != nil {
						_ = config.Save(path, previousConfig)
					}
				}
			}
			if err != nil && !errors.Is(err, app.ErrAddressUnavailable) && old != nil {
				s, _ = start(ctx, previousConfig, b, log, health)
			}
			done <- result{next, s, err, persist}
		}()
	}
	if needsSetup {
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
			d.Update(status, session != nil && c.Mode() != "disabled", c.Peers, states)
		case st := <-statusUpdates:
			states[st.id] = st.online
		case <-refresh.C:
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
			switch action {
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
				peer := config.Peer{ID: f.Values[7], Name: f.Values[6], Address: f.Values[8], Port: p}
				next.Peers = append([]config.Peer(nil), draft.Peers...)
				index := d.Selected()
				if index >= 0 && index < len(next.Peers) {
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
					draft = next
					d.Show(toForm(draft), draft.Peers, config.Addresses())
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
