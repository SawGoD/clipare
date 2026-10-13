package ui

import (
	"clipare/internal/config"
	"encoding/json"
	"strings"
	"testing"
)

func TestReactBridgeNeverPublishesLegacyKey(t *testing.T) {
	d := newReactDesktop(nil)
	f := form{}
	f.Values[0], f.Values[1], f.Values[2], f.Values[3] = "Laptop", "id", "127.0.0.1", "45873"
	f.Values[4] = "private-legacy-key"
	d.Prepare(f, nil, []string{"127.0.0.1"})
	b, err := json.Marshal((&ReactService{d}).Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), f.Values[4]) {
		t.Fatal("legacy key leaked to frontend")
	}
	if !strings.Contains(string(b), "legacyKeyPresent") {
		t.Fatal("key presence missing")
	}
}

func TestReactBridgeValidatesAndSelectsPeer(t *testing.T) {
	d := newReactDesktop(nil)
	service := &ReactService{d}
	d.Show(form{}, []config.Peer{{ID: "first", Name: "One"}, {ID: "second", Name: "Two"}}, nil)
	if err := service.Perform(reactCommand{Type: "device.remove", ID: "missing"}); err == nil {
		t.Fatal("unknown peer accepted")
	}
	if err := service.Perform(reactCommand{Type: "device.remove", ID: "second"}); err != nil {
		t.Fatal(err)
	}
	if action := d.Poll(); action != eventRemove {
		t.Fatalf("got event %d", action)
	}
	if selected := d.Selected(); selected != 1 {
		t.Fatalf("selected %d", selected)
	}
	if err := service.Perform(reactCommand{Type: "shell.exec"}); err == nil {
		t.Fatal("arbitrary command accepted")
	}
}

func TestReactBridgeAdvancedSettingsPreserveDraftUntilApply(t *testing.T) {
	d := newReactDesktop(nil)
	service := &ReactService{d}
	f := form{}
	f.Values[0], f.Values[2], f.Values[3], f.Values[4] = "Old", "127.0.0.1", "45873", "legacy-secret"
	d.Prepare(f, nil, nil)
	if err := service.Perform(reactCommand{Type: "settings.advanced", Value: map[string]any{"deviceName": "New", "listenAddress": "127.0.0.1", "port": float64(45874)}}); err != nil {
		t.Fatal(err)
	}
	if got := d.Read().Values[0]; got != "Old" {
		t.Fatalf("draft changed before GUI event: %s", got)
	}
	if action := d.Poll(); action != eventSave {
		t.Fatalf("got %d", action)
	}
	if got := d.Read().Values[0]; got != "New" {
		t.Fatalf("new name: %s", got)
	}
	if got := d.Read().Values[4]; got != "legacy-secret" {
		t.Fatal("legacy key discarded")
	}
}

func TestReactBridgeOpensOnlyVisibleHTTPHistoryLink(t *testing.T) {
	d := newReactDesktop(nil)
	var opened string
	d.openExternal = func(link string) error { opened = link; return nil }
	d.snapshot.History = []reactHistoryEntry{{ID: "safe", URL: "https://example.com/path"}, {ID: "unsafe", URL: "file:///tmp/private"}, {ID: "masked"}}
	service := &ReactService{d}
	if err := service.Perform(reactCommand{Type: "history.openLink", ID: "safe"}); err != nil || opened != "https://example.com/path" {
		t.Fatalf("safe link: %q, %v", opened, err)
	}
	for _, id := range []string{"unsafe", "masked", "missing"} {
		if err := service.Perform(reactCommand{Type: "history.openLink", ID: id}); err == nil {
			t.Fatalf("accepted unavailable link %q", id)
		}
	}
	if opened != "https://example.com/path" {
		t.Fatal("invalid link was opened")
	}
}

func TestReactBridgeBackCancelsPairing(t *testing.T) {
	d := newReactDesktop(nil)
	d.view = ViewPairing
	if err := (&ReactService{d}).Perform(reactCommand{Type: "view.open", View: "home"}); err != nil {
		t.Fatal(err)
	}
	if got := d.Poll(); got != eventCancelPair {
		t.Fatalf("back event = %d", got)
	}
	d.view = ViewPairing
	d.snapshot.Pairing = &reactPairing{Direction: "incoming"}
	if err := (&ReactService{d}).Perform(reactCommand{Type: "view.open", View: "home"}); err != nil {
		t.Fatal(err)
	}
	if got := d.Poll(); got != eventReject {
		t.Fatalf("incoming back event = %d", got)
	}
}

func TestReactUpdatePromptPreservesInstallRetryAndProgress(t *testing.T) {
	d := newReactDesktop(nil)
	d.UpdatePrompt(updateOffer("0.6.0-beta.4", "0.6.0-beta.5"))
	if got := d.snapshot.Update; got.Kind != "available" || got.Version != "0.6.0-beta.5" {
		t.Fatalf("offer: %+v", got)
	}
	d.UpdatePrompt(updateProgress("Загрузка Clipare 0.6.0-beta.5…"))
	if d.snapshot.Update.Kind != "downloading" {
		t.Fatalf("download: %+v", d.snapshot.Update)
	}
	d.UpdatePrompt(updateProgress("Обновление готово\n\nClipare перезапустится для установки…"))
	if d.snapshot.Update.Kind != "installing" {
		t.Fatalf("install: %+v", d.snapshot.Update)
	}
	d.UpdatePrompt(updateFailure("Ошибка подготовки", "0.6.0-beta.4", eventInstallUpdate))
	if d.snapshot.Update.Kind != "error" || d.snapshot.Update.Retry != "install" {
		t.Fatalf("retry: %+v", d.snapshot.Update)
	}
}

func TestReactLeavingHistoryDiscardsVisibleSecret(t *testing.T) {
	d := newReactDesktop(nil)
	d.ShowHistory([]historyRow{{id: "secret", header: "Device · 10:45", preview: "sensitive-value", sensitive: true, revealed: true}}, false)
	if len(d.snapshot.History) != 1 || d.snapshot.History[0].VisibleText != "sensitive-value" {
		t.Fatal("test history not shown")
	}
	if err := (&ReactService{d}).Perform(reactCommand{Type: "view.open", View: "home"}); err != nil {
		t.Fatal(err)
	}
	if got := d.Poll(); got != eventHistoryClose {
		t.Fatalf("close event = %d", got)
	}
	if len((&ReactService{d}).Snapshot().History) != 0 {
		t.Fatal("secret retained in React snapshot")
	}
	d.ShowHistory([]historyRow{{id: "secret", header: "Device · 10:45", preview: "sensitive-value", sensitive: true, revealed: true}}, false)
	d.UpdatePrompt(updateOffer("0.6.0-beta.4", "0.6.0-beta.5"))
	if len((&ReactService{d}).Snapshot().History) != 0 {
		t.Fatal("secret retained after update interrupted history")
	}
}
