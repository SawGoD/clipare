package ui

import (
	"clipare"
	"clipare/internal/app"
	"clipare/internal/autostart"
	"clipare/internal/clipboard"
	"clipare/internal/config"
	"clipare/internal/discovery"
	"clipare/internal/history"
	"clipare/internal/pairing"
	"clipare/internal/peers"
	"clipare/internal/update"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
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
	eventCheckUpdate
	eventInstallUpdate
	eventLaterUpdate
	eventUpdatePreference
	eventAutostartPreference
	eventFormChanged
	eventDeviceName

	// Values 26-32 are reserved: native adapters already use those tags for
	// the advanced gearshape (30), back/notice (31) and "Дополнительно" (32).
	eventReserved26
	eventReserved27
	eventReserved28
	eventReserved29
	eventReserved30
	eventReserved31
	eventReserved32

	eventHistoryOpen
	eventHistoryCopy
	eventHistoryEye
	eventHistoryDelete
	eventHistoryClear
	eventHistoryEnabled
	eventHistorySettings

	// Values 40-44 are reserved: native adapters use those tags for help
	// buttons, so the expand action starts at 45.
	eventReserved40
	eventReserved41
	eventReserved42
	eventReserved43
	eventReserved44

	eventHistoryExpand
	eventUpdateChannel
)

type updateChannelDesktop interface {
	UpdateChannelSettings(string)
	UpdateChannel() string
}

func renderUpdateChannel(d desktop, c config.Config) {
	if v, ok := d.(updateChannelDesktop); ok {
		v.UpdateChannelSettings(update.ResolveChannel(c.Updates.Channel, clipare.Version()))
	}
}

type updateDesktop interface {
	UpdateSettings(string, bool)
	UpdateEnabled() bool
	UpdatePrompt(updatePrompt)
	UpdateClose()
}

// Restore native toggles after rejected or failed persistence, without replacing
// technical drafts or changing the current navigation view.
type preferenceDesktop interface{ Preferences(bool, bool) }

func renderPreferences(d desktop, autostart, updates bool) {
	if p, ok := d.(preferenceDesktop); ok {
		p.Preferences(autostart, updates)
	}
}

type updateResult struct {
	release *update.Release
	plan    *update.Plan
	err     error
	manual  bool
	work    string
	helper  bool
	channel string
}

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
	Values    [13]string
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
	maxItems, retention := historyFormValues(c)
	return form{Values: [13]string{c.Device.Name, c.Device.ID, c.Listen.Address, strconv.Itoa(c.Listen.Port), c.Security.Secret, "Протокол 1 · Public key SHA-256: " + c.Identity.Fingerprint(), "", "", "", "45873", "", maxItems, retention}, Autostart: c.Autostart}
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
	// Immediate history preferences are persisted separately. A technical
	// form save must not consume half-edited or unrepresented history values.
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
	return RunWithReady(parent, path, log, nil, false)
}
func RunWithReady(parent context.Context, path string, log *slog.Logger, ready func() error, restored bool) error {
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
	return runDesktopReady(parent, path, log, d, b, nil, loopTiming{30 * time.Millisecond, 2 * time.Second, 5 * time.Second}, ready, restored)
}

// The GUI loop is also exercised with a synthetic desktop and clipboard, so
// recovery tests never modify the user's pasteboard, settings or login items.
func runDesktop(parent context.Context, path string, log *slog.Logger, d desktop, b clipboard.Backend, start sessionStarter, timing loopTiming) error {
	return runDesktopReady(parent, path, log, d, b, start, timing, nil, false)
}
func runDesktopReady(parent context.Context, path string, log *slog.Logger, d desktop, b clipboard.Backend, start sessionStarter, timing loopTiming, ready func() error, restored bool) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	if local, ok := d.(interface{ Instance(string) }); ok {
		local.Instance(path)
	}
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
	// History keeps only local, encrypted data; open it lazily so disabled
	// installations never create keys or files.
	var hist *history.Service
	var historyOpenError error
	openHistory := func(next config.Config, inspect bool) {
		if hist != nil {
			return
		}
		if !next.History.Enabled {
			if !inspect {
				return
			}
			if _, err := os.Stat(history.DefaultPath(path)); err != nil {
				historyOpenError = nil
				if !os.IsNotExist(err) {
					historyOpenError = err
				}
				return
			}
		}
		h, e := history.Open(history.DefaultPath(path), next, log)
		historyOpenError = e
		if e != nil {
			log.Warn("clipboard history unavailable")
			return
		}
		hist = h
	}
	openHistory(c, false)
	defer func() {
		if hist != nil {
			hist.Close()
		}
	}()
	// Pairing can expose home before settings have ever been opened.
	if prepared, ok := d.(interface {
		Prepare(form, []config.Peer, []string)
	}); ok {
		prepared.Prepare(toForm(c), c.Peers, config.Addresses())
	}
	if ready != nil {
		if e = ready(); e != nil {
			return e
		}
	}
	if restored {
		d.Alert("Обновление не удалось установить. Предыдущая версия восстановлена")
	}
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
			var controlHandler http.Handler
			if service != nil {
				controlHandler = service
			}
			var opts []app.Option
			if hist != nil {
				opts = append(opts, app.WithHistory(hist))
			}
			return app.StartWithControl(ctx, next, b, log, health, controlHandler, opts...)
		}
	}
	ud, hasUpdateUI := d.(updateDesktop)
	updateDone := make(chan updateResult, 1)
	var updateWG sync.WaitGroup
	checking, installing := false, false
	var available *update.Release
	deferredUpdateOffer := false
	cache, cacheErr := update.CacheDir()
	if environment, ok := d.(interface{ UpdateCacheDir() (string, error) }); ok {
		cache, cacheErr = environment.UpdateCacheDir()
	}
	checker := update.Checker{Current: clipare.Version(), StatePath: filepath.Join(cache, "state.json")}
	defer func() { cancel(); updateWG.Wait() }()
	checkUpdate := func(manual bool) {
		if !hasUpdateUI || checking || installing || cacheErr != nil {
			return
		}
		if outgoing || shownIncoming != "" {
			if manual {
				d.Alert("Завершите подключение устройства перед проверкой обновлений")
			}
			return
		}
		checking = true
		enabled := c.Updates.Enabled
		currentChecker := checker
		currentChecker.Channel = update.ResolveChannel(c.Updates.Channel, clipare.Version())
		if manual {
			available = nil
			ud.UpdatePrompt(updateProgress("Проверка обновлений…"))
		}
		updateWG.Add(1)
		go func() {
			defer updateWG.Done()
			update.Cleanup(cache, time.Now())
			if manual || enabled && currentChecker.Due(time.Now()) {
				log.Info("update check started")
			}
			r, err := currentChecker.Check(ctx, enabled, manual)
			select {
			case updateDone <- updateResult{release: r, err: err, manual: manual, channel: currentChecker.Channel}:
			case <-ctx.Done():
			}
		}()
	}
	if hasUpdateUI {
		ud.UpdateSettings(clipare.Version(), c.Updates.Enabled)
		renderUpdateChannel(d, c)
		checkUpdate(false)
	}
	nextUpdatePoll := time.Now().Add(time.Minute)
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
	historyAttached := false
	// Clipboard history rendering state; row IDs map list selections to items.
	// Reveal and expansion are runtime-only and reset when the view is opened.
	historyItems := map[string]history.Item{}
	historyRowIDs := []string{}
	revealed := map[string]bool{}
	expanded := map[string]bool{}
	var renderHistory func()
	renderHistory = func() {
		hd, ok := d.(historyDesktop)
		if !ok {
			return
		}
		if !busy {
			openHistory(c, true)
		}
		if hist == nil && historyOpenError != nil {
			d.Alert("История недоступна. Проверьте доступ к защищённому хранилищу ключей и файлу истории")
			return
		}
		var items []history.Item
		keyLost := false
		if hist != nil {
			var e error
			items, e = hist.Items()
			keyLost = errors.Is(e, history.ErrKeyLost)
			if e != nil {
				items = nil
			}
		}
		historyItems = map[string]history.Item{}
		for _, item := range items {
			historyItems[item.ID] = item
		}
		rows := historyRows(items, time.Now(), revealed, expanded)
		historyRowIDs = historyRowIDs[:0]
		for _, row := range rows {
			historyRowIDs = append(historyRowIDs, row.id)
		}
		hd.ShowHistory(rows, keyLost)
	}
	preferenceAuto, preferenceUpdates := c.Autostart, c.Updates.Enabled
	status := "Настройте подключение"
	var retryAt time.Time
	exe, e := os.Executable()
	if e != nil {
		return e
	}
	apply := func(next config.Config, persist bool) {
		preferenceAuto, preferenceUpdates = next.Autostart, next.Updates.Enabled
		renderPreferences(d, preferenceAuto, preferenceUpdates)
		openHistory(next, false)
		historyAttached = hist != nil
		if hist != nil {
			hist.UpdateSettings(next)
		}
		renderHistoryPrefs(d, next.History)
		busy = true
		retryAt = time.Time{}
		status = "Применение настроек…"
		states = map[string]bool{}
		d.Update(status, false, c.Peers, states)
		renderStatus(d, status, next.Mode() != "disabled", false, next.Listen.Address == "127.0.0.1", true)
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
	// History preferences do not change networking; save them independently
	// without consuming technical drafts or restarting a working sync session.
	saveHistory := func(settings config.History) {
		if settings == c.History {
			return
		}
		next := c
		next.History = settings
		if err := next.Validate(); err != nil {
			d.Alert("Проверьте настройки истории: " + err.Error())
			renderHistoryPrefs(d, c.History)
			return
		}
		if !historyAttached && settings.Enabled {
			apply(next, true)
			return
		}
		var err error
		if service != nil {
			next, err = service.SaveSettings(next)
		} else {
			err = config.Save(path, next)
		}
		if err != nil {
			d.Alert("Не удалось сохранить настройки истории")
			renderHistoryPrefs(d, c.History)
			return
		}
		c.History, draft.History = next.History, next.History
		if hist != nil {
			hist.UpdateSettings(c)
		}
		renderHistoryPrefs(d, c.History)
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
	renderStatus(d, status, c.Mode() != "disabled", false, c.Listen.Address == "127.0.0.1", busy)
	renderApply(d, d.Read(), toForm(c), needsSetup || !reflect.DeepEqual(draft.Peers, c.Peers))
	for {
		select {
		case <-ctx.Done():
			return nil
		case ur := <-updateDone:
			if ur.helper {
				if ur.err == nil {
					if e := ur.plan.Arm(); e == nil {
						log.Info("update staged", "version", ur.plan.Version)
						return nil
					} else {
						ur.err = e
					}
				}
				installing = false
				if service != nil {
					service.ResumePairing()
				}
				ud.UpdatePrompt(updateFailure("Не удалось запустить установку. Текущая версия продолжает работать", clipare.Version(), eventInstallUpdate))
				continue
			}
			if ur.plan != nil && ur.err == nil {
				if busy {
					installing = false
					if service != nil {
						service.ResumePairing()
					}
					os.RemoveAll(ur.work)
					ud.UpdatePrompt(updateFailure("Дождитесь применения настроек и попробуйте обновление снова", clipare.Version(), eventInstallUpdate))
					continue
				}
				ud.UpdatePrompt(updateProgress("Обновление готово\n\nClipare перезапустится для установки…"))
				updateWG.Add(1)
				go func() {
					defer updateWG.Done()
					err := ur.plan.LaunchHelper(ctx)
					select {
					case updateDone <- updateResult{plan: ur.plan, err: err, helper: true, work: ur.work}:
					case <-ctx.Done():
					}
				}()
				continue
			}
			if installing {
				installing = false
				if service != nil {
					service.ResumePairing()
				}
				if ur.work != "" {
					os.RemoveAll(ur.work)
				}
				log.Warn("update failed")
				message := "Не удалось подготовить обновление. Текущая версия продолжает работать"
				if errors.Is(ur.err, update.ErrInstall) || errors.Is(ur.err, update.ErrVerify) || errors.Is(ur.err, update.ErrPackage) || errors.Is(ur.err, update.ErrDownload) || errors.Is(ur.err, update.ErrPlatform) {
					message = ur.err.Error()
				}
				ud.UpdatePrompt(updateFailure(message, clipare.Version(), eventInstallUpdate))
				continue
			}
			checking = false
			if ur.channel != "" && ur.channel != update.ResolveChannel(c.Updates.Channel, clipare.Version()) {
				checkUpdate(ur.manual)
				continue
			}
			if ur.err != nil {
				log.Debug("update check failed")
				if ur.manual {
					ud.UpdatePrompt(updateFailure(update.ErrUnavailable.Error(), clipare.Version(), eventCheckUpdate))
				}
				continue
			}
			if ur.release == nil {
				if ur.manual {
					message := "Установлена последняя версия Clipare"
					if update.DevelopmentVersion(clipare.Version()) {
						message = "Автообновления недоступны для dev-сборок"
					}
					ud.UpdatePrompt(updateNotice(message, clipare.Version()))
				}
				continue
			}
			if !ur.manual && !c.Updates.Enabled {
				continue
			}
			if _, e := ur.release.Platform(runtime.GOOS, runtime.GOARCH); e != nil {
				if ur.manual {
					ud.UpdatePrompt(updateNotice(update.ErrPlatform.Error(), clipare.Version()))
				}
				continue
			}
			available = ur.release
			log.Info("update available", "version", available.Version)
			if outgoing || shownIncoming != "" {
				deferredUpdateOffer = true
			} else {
				ud.UpdatePrompt(updateOffer(clipare.Version(), available.Version))
			}
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
			draft.MembershipVersions = peers.CloneVersions(c.MembershipVersions)
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
				message = discoveryEmptyMessage
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
			renderStatus(d, status, c.Mode() != "disabled", session != nil, c.Listen.Address == "127.0.0.1", busy)
			preferenceAuto, preferenceUpdates = c.Autostart, c.Updates.Enabled
			renderPreferences(d, preferenceAuto, preferenceUpdates)
			renderUpdateChannel(d, c)
			if hist != nil {
				hist.UpdateSettings(c)
			}
			renderHistoryPrefs(d, c.History)
			renderApply(d, d.Read(), toForm(c), needsSetup || !reflect.DeepEqual(draft.Peers, c.Peers))
		case st := <-statusUpdates:
			states[st.id] = st.online
			if session != nil {
				renderPlatforms(d, session)
			}
		case <-refresh.C:
			if session != nil {
				renderPlatforms(d, session)
			}
			// An open History section refreshes itself as events are recorded.
			if v, ok := d.(viewDesktop); ok && v.CurrentView() == ViewHistory {
				renderHistory()
			}
			if deferredUpdateOffer && !outgoing && shownIncoming == "" && available != nil {
				deferredUpdateOffer = false
				ud.UpdatePrompt(updateOffer(clipare.Version(), available.Version))
			}
			if hasUpdateUI && !time.Now().Before(nextUpdatePoll) {
				nextUpdatePoll = time.Now().Add(time.Minute)
				checkUpdate(false)
			}
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
			renderStatus(d, status, c.Mode() != "disabled", session != nil, c.Listen.Address == "127.0.0.1", busy)
		case <-tick.C:
			action := d.Poll()
			if action == eventQuit {
				return nil
			}
			if action == eventNone {
				continue
			}
			changes := renderApply(d, d.Read(), toForm(c), needsSetup || !reflect.DeepEqual(draft.Peers, c.Peers))
			if (action == eventSave || action == eventDeviceName) && changes&(1<<action) == 0 {
				continue
			}
			if action == eventFormChanged {
				renderActions(d, d.Read())
				continue
			}
			if (action == eventSave || action == eventImport || action == eventUpsert || action == eventCopy || action == eventDeviceName) && !actionReady(action, d.Read()) {
				continue
			}
			if installing && action != eventSettings && action != eventLaterUpdate {
				continue
			}
			if busy {
				if action != eventSettings {
					if action == eventAutostartPreference || action == eventUpdatePreference {
						renderPreferences(d, preferenceAuto, preferenceUpdates)
					}
					if action == eventUpdateChannel {
						renderUpdateChannel(d, c)
					}
					continue
				}
			}
			if (outgoing || shownIncoming != "") && (action == eventSave || action == eventPause || action == eventGenerate || action == eventImport || action == eventUpsert || action == eventRemove || action == eventAutostartPreference || action == eventUpdatePreference) {
				renderPreferences(d, c.Autostart, c.Updates.Enabled)
				d.Alert("Завершите или отмените подключение, прежде чем изменять настройки")
				continue
			}
			switch action {
			case eventUpdateChannel:
				v, ok := d.(updateChannelDesktop)
				if !ok {
					continue
				}
				channel := v.UpdateChannel()
				if channel != update.ChannelStable && channel != update.ChannelBeta {
					renderUpdateChannel(d, c)
					continue
				}
				if channel == c.Updates.Channel {
					continue
				}
				next := c
				next.Updates.Channel = channel
				var err error
				if service != nil {
					next, err = service.SaveSettings(next)
				} else {
					err = config.Save(path, next)
				}
				if err != nil {
					d.Alert("Не удалось сохранить канал обновлений")
					renderUpdateChannel(d, c)
					continue
				}
				c.Updates.Channel, draft.Updates.Channel = channel, channel
				available = nil
				deferredUpdateOffer = false
				ud.UpdateClose()
				renderUpdateChannel(d, c)
				checkUpdate(false)
			case eventDeviceName:
				if outgoing || shownIncoming != "" {
					d.Alert("Завершите подключение перед изменением имени")
					continue
				}
				next := c
				next.Device.Name = strings.TrimSpace(d.Read().Values[0])
				if err := next.Validate(); err != nil {
					d.Alert(err.Error())
					continue
				}
				draft = next
				apply(next, true)
			case eventHistoryOpen:
				// Secrets and expansion never survive a fresh view switch.
				for id := range revealed {
					delete(revealed, id)
				}
				for id := range expanded {
					delete(expanded, id)
				}
				renderHistory()
			case eventHistoryCopy:
				item, ok := selectedHistoryItem(historyRowIDs, historyItems, d.Selected())
				if !ok {
					continue
				}
				if e := b.Write(item.Text); e != nil {
					d.Alert("Не удалось скопировать запись")
					continue
				}
				// The watcher re-records the copy; the row moves to the top on
				// the next refresh instead of duplicating.
			case eventHistoryEye:
				id, ok := selectedHistoryID(historyRowIDs, d.Selected())
				if !ok {
					continue
				}
				if revealed[id] {
					delete(revealed, id)
				} else {
					revealed[id] = true
				}
				renderHistory()
			case eventHistoryExpand:
				id, ok := selectedHistoryID(historyRowIDs, d.Selected())
				if !ok {
					continue
				}
				if expanded[id] {
					delete(expanded, id)
				} else {
					expanded[id] = true
				}
				renderHistory()
			case eventHistoryDelete:
				if item, ok := selectedHistoryItem(historyRowIDs, historyItems, d.Selected()); ok && hist != nil {
					if e := hist.Delete(item.ID); e != nil {
						d.Alert("Не удалось удалить запись")
						continue
					}
				}
				renderHistory()
			case eventHistoryClear:
				openHistory(c, true)
				if hist == nil && historyOpenError != nil {
					d.Alert("Не удалось открыть историю для очистки. Проверьте доступ к хранилищу ключей и файлу истории")
					continue
				}
				if hist != nil {
					if e := hist.Clear(); e != nil {
						d.Alert("Не удалось очистить историю")
						continue
					}
				}
				if v, ok := d.(viewDesktop); ok && v.CurrentView() == ViewHistory {
					renderHistory()
				}
			case eventHistoryEnabled:
				hd, ok := d.(historyDesktop)
				if !ok {
					continue
				}
				settings := c.History
				settings.Enabled = hd.HistoryEnabled()
				saveHistory(settings)
			case eventHistorySettings:
				settings, err := historySettingsFromForm(d.Read(), c.History)
				if err != nil {
					d.Alert("Проверьте настройки истории: " + err.Error())
					renderHistoryPrefs(d, c.History)
					continue
				}
				saveHistory(settings)
			case eventCheckUpdate:
				checkUpdate(true)
			case eventLaterUpdate:
				if hasUpdateUI {
					ud.UpdateClose()
				}
			case eventUpdatePreference:
				if !hasUpdateUI || needsSetup {
					renderPreferences(d, c.Autostart, c.Updates.Enabled)
					continue
				}
				next := c
				next.Updates.Enabled = ud.UpdateEnabled()
				draft = next
				apply(next, true)
			case eventAutostartPreference:
				if needsSetup {
					renderPreferences(d, c.Autostart, c.Updates.Enabled)
					d.Alert("Сначала настройте подключение в расширенных параметрах")
					continue
				}
				next := c
				next.Autostart = d.Read().Autostart
				draft = next
				apply(next, true)
			case eventInstallUpdate:
				if !hasUpdateUI || available == nil || checking || busy {
					continue
				}
				if outgoing || shownIncoming != "" || service != nil && !service.SuspendPairing() {
					d.Alert("Завершите текущее подключение устройства перед обновлением")
					continue
				}
				installing = true
				r := *available
				ud.UpdatePrompt(updateProgress("Загрузка Clipare " + update.DisplayVersion(r.Version) + "…"))
				updateWG.Add(1)
				go func() {
					defer updateWG.Done()
					log.Info("download started", "version", r.Version)
					s, err := (update.Downloader{}).Stage(ctx, r, cache)
					result := updateResult{err: err}
					if err == nil {
						log.Info("download completed", "version", r.Version)
						log.Info("checksum verified", "version", r.Version)
						result.work = s.Work
						p, err := s.Prepare(ctx, exe, path)
						result.err = err
						if err == nil {
							result.plan = &p
						}
					}
					select {
					case updateDone <- result:
					case <-ctx.Done():
						if result.work != "" {
							os.RemoveAll(result.work)
						}
					}
				}()
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
				if hasUpdateUI {
					ud.UpdateSettings(clipare.Version(), c.Updates.Enabled)
					renderUpdateChannel(d, c)
				}
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
				if hasUpdateUI {
					next.Updates.Enabled = ud.UpdateEnabled()
				}
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
					d.Alert("Код скопирован. Добавьте его в настройках другого устройства. Код содержит общий ключ — передавайте его приватно")
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
				d.Alert("Устройство добавлено. Сохраните настройки, затем скопируйте код этого устройства и добавьте его на другом")
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
					next := removeDevice(c, draft.Peers[index].ID)
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
			renderApply(d, d.Read(), toForm(c), needsSetup || !reflect.DeepEqual(draft.Peers, c.Peers))
		}
	}
}

// Deleting a trusted peer is not a settings-form submission. Ignore incomplete
// technical drafts and remove the peer's credential and membership atomically.
func removeDevice(c config.Config, id string) config.Config {
	next := c
	next.Peers = make([]config.Peer, 0, len(c.Peers))
	for _, peer := range c.Peers {
		if peer.ID != id {
			next.Peers = append(next.Peers, peer)
			continue
		}
		if !peer.Legacy {
			next = peers.Remove(next, id)
		}
	}
	return next
}

func shouldRetry(now, due time.Time, busy, running, needsSetup bool) bool {
	return !due.IsZero() && !now.Before(due) && !busy && !running && !needsSetup
}
