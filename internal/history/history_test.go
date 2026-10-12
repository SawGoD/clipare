package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testItem(id, text string) Item {
	return Item{ID: id, OriginID: "mac", OriginName: "MacBook", Timestamp: 100, Type: Classify(text), Text: text}
}

func openTestStore(t *testing.T, key []byte) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), HistoryFile)
	s, e := OpenStore(path, key)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	return s, path
}

var storeKey = []byte("0123456789abcdef0123456789abcdef")

func TestStoreAddAndList(t *testing.T) {
	s, _ := openTestStore(t, storeKey)
	now := time.Unix(1700000000, 0)
	if added, e := s.Add(testItem("id-1", "hello"), now); e != nil || !added {
		t.Fatal(added, e)
	}
	if added, e := s.Add(testItem("id-2", "https://github.com/SawGoD/clipare"), now.Add(time.Minute)); e != nil || !added {
		t.Fatal(added, e)
	}
	items, e := s.List()
	if e != nil || len(items) != 2 {
		t.Fatal(items, e)
	}
	if items[0].Text != "https://github.com/SawGoD/clipare" || items[1].Text != "hello" {
		t.Fatal("unexpected order", items)
	}
	if items[0].OriginName != "MacBook" || items[0].Type != ContentURL {
		t.Fatal("metadata lost", items[0])
	}
}

func TestStoreDuplicateMessageIgnored(t *testing.T) {
	s, _ := openTestStore(t, storeKey)
	now := time.Unix(1700000000, 0)
	if _, e := s.Add(testItem("id-1", "hello"), now); e != nil {
		t.Fatal(e)
	}
	added, e := s.Add(testItem("id-1", "hello"), now.Add(time.Hour))
	if e != nil || added {
		t.Fatal("duplicate message id created a row", added, e)
	}
	items, _ := s.List()
	if len(items) != 1 || items[0].Timestamp != now.Unix() {
		t.Fatal("duplicate changed the stored row", items)
	}
}

func TestStoreContentDedupBumpsExistingRow(t *testing.T) {
	s, _ := openTestStore(t, storeKey)
	now := time.Unix(1700000000, 0)
	_, e := s.Add(testItem("id-1", "hello"), now)
	if e != nil {
		t.Fatal(e)
	}
	// Copy from history produces a fresh message id for identical content.
	added, e := s.Add(testItem("id-2", "hello"), now.Add(time.Minute))
	if e != nil || added {
		t.Fatal("identical content created a duplicate row", added, e)
	}
	items, _ := s.List()
	if len(items) != 1 || items[0].ID != "id-1" || items[0].Timestamp != now.Add(time.Minute).Unix() {
		t.Fatal("existing row was not refreshed", items)
	}
}

func TestStoreRefreshedRowSurvivesRestart(t *testing.T) {
	s, path := openTestStore(t, storeKey)
	now := time.Unix(1700000000, 0)
	s.Add(testItem("first", "same"), now)
	s.Add(testItem("second", "same"), now.Add(time.Hour))
	s.Close()
	reopened, err := OpenStore(path, storeKey)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	items, err := reopened.List()
	if err != nil || len(items) != 1 || items[0].Timestamp != now.Add(time.Hour).Unix() {
		t.Fatal("refreshed row lost after restart", items, err)
	}
}

func TestStorePartialPrefixRecoveryAllowsFutureAppends(t *testing.T) {
	for size := 1; size <= 3; size++ {
		t.Run(string(rune('0'+size)), func(t *testing.T) {
			s, path := openTestStore(t, storeKey)
			now := time.Unix(1700000000, 0)
			s.Add(testItem("first", "first"), now)
			s.Close()
			f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = f.Write(make([]byte, size)); err != nil {
				t.Fatal(err)
			}
			f.Close()
			s, err = OpenStore(path, storeKey)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Add(testItem("second", "second"), now.Add(time.Minute)); err != nil {
				t.Fatal(err)
			}
			s.Close()
			s, err = OpenStore(path, storeKey)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			items, err := s.List()
			if err != nil || len(items) != 2 {
				t.Fatal("append lost after recovery", items, err)
			}
		})
	}
}

func TestCleanupNeverDropsLockedRecordsAtItemLimit(t *testing.T) {
	now := time.Now()
	locked := recordRef{offset: 1, locked: true}
	refs := []recordRef{locked, {offset: 2, created: now}, {offset: 3, created: now.Add(-time.Minute)}}
	drop := survivorsToDrop(refs, now, time.Hour, 1)
	if drop[locked] || !drop[refs[2]] || drop[refs[1]] {
		t.Fatal("incorrect cleanup", drop)
	}
}

func TestRepeatedContentJournalIsBounded(t *testing.T) {
	s, path := openTestStore(t, storeKey)
	now := time.Now()
	text := strings.Repeat("sample", 2000)
	for i := 0; i < 30; i++ {
		if _, err := s.Add(testItem(string(rune('a'+i)), text), now.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatal(err)
		}
		if err := s.Cleanup(now, time.Hour, 10); err != nil {
			t.Fatal(err)
		}
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() > 100000 {
		t.Fatal("journal grew without bound", info, err)
	}
}

func TestStoreDeleteAndClear(t *testing.T) {
	s, _ := openTestStore(t, storeKey)
	now := time.Unix(1700000000, 0)
	if _, e := s.Add(testItem("id-1", "one"), now); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Add(testItem("id-2", "two"), now); e != nil {
		t.Fatal(e)
	}
	if e := s.Delete("id-1"); e != nil {
		t.Fatal(e)
	}
	items, _ := s.List()
	if len(items) != 1 || items[0].ID != "id-2" {
		t.Fatal("delete failed", items)
	}
	// Deleting the same content hash must not resurrect anything.
	if _, e := s.Add(testItem("id-3", "one"), now); e != nil {
		t.Fatal(e)
	}
	if e := s.Clear(); e != nil {
		t.Fatal(e)
	}
	items, _ = s.List()
	if len(items) != 0 {
		t.Fatal("clear failed", items)
	}
}

func TestStoreTTLCleanup(t *testing.T) {
	s, _ := openTestStore(t, storeKey)
	base := time.Unix(1700000000, 0)
	if _, e := s.Add(testItem("old", "one"), base); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Add(testItem("new", "two"), base.Add(time.Hour)); e != nil {
		t.Fatal(e)
	}
	if e := s.Cleanup(base.Add(time.Hour), 30*time.Minute, 50); e != nil {
		t.Fatal(e)
	}
	items, _ := s.List()
	if len(items) != 1 || items[0].ID != "new" {
		t.Fatal("TTL cleanup failed", items)
	}
}

func TestStoreMaxCountCleanup(t *testing.T) {
	s, _ := openTestStore(t, storeKey)
	base := time.Unix(1700000000, 0)
	for i := 0; i < 5; i++ {
		if _, e := s.Add(testItem(string(rune('a'+i)), "text"+string(rune('a'+i))), base.Add(time.Duration(i)*time.Minute)); e != nil {
			t.Fatal(e)
		}
	}
	if e := s.Cleanup(base.Add(10*time.Minute), 72*time.Hour, 3); e != nil {
		t.Fatal(e)
	}
	items, _ := s.List()
	if len(items) != 3 || items[0].ID != "e" || items[2].ID != "c" {
		t.Fatal("item limit cleanup failed", items)
	}
}

func TestStorePersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, HistoryFile)
	s, e := OpenStore(path, storeKey)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Add(testItem("id-1", "persist me"), time.Unix(1700000000, 0)); e != nil {
		t.Fatal(e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	reopened, e := OpenStore(path, storeKey)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	items, e := reopened.List()
	if e != nil || len(items) != 1 || items[0].Text != "persist me" {
		t.Fatal("restart lost history", items, e)
	}
}

func TestStoreRejectsWrongKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, HistoryFile)
	s, e := OpenStore(path, storeKey)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Add(testItem("id-1", "secret content"), time.Unix(1700000000, 0)); e != nil {
		t.Fatal(e)
	}
	s.Close()
	other := []byte("ffffffffffffffffffffffffffffffff")
	reopened, e := OpenStore(path, other)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	if _, e = reopened.List(); e == nil {
		t.Fatal("wrong key decrypted history")
	}
	if !strings.Contains(e.Error(), ErrKeyLost.Error()) {
		t.Fatal("expected key-lost error", e)
	}
}

func TestStoreRawFileHasNoPlaintext(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, HistoryFile)
	payload := "super-secret-clipboard-payload-12345"
	s, e := OpenStore(path, storeKey)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Add(testItem("id-1", payload), time.Unix(1700000000, 0)); e != nil {
		t.Fatal(e)
	}
	if e = s.Cleanup(time.Now().Add(time.Hour), 24*time.Hour, 10); e != nil {
		t.Fatal(e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	entries, e := os.ReadDir(dir)
	if e != nil {
		t.Fatal(e)
	}
	if len(entries) == 0 {
		t.Fatal("no history files written")
	}
	for _, entry := range entries {
		data, e := os.ReadFile(filepath.Join(dir, entry.Name()))
		if e != nil {
			t.Fatal(e)
		}
		if strings.Contains(string(data), payload) {
			t.Fatalf("plaintext found in %s", entry.Name())
		}
		if strings.Contains(string(data), "MacBook") {
			t.Fatalf("origin metadata found in plaintext in %s", entry.Name())
		}
	}
}

func TestStoreModifiedCiphertextFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, HistoryFile)
	s, e := OpenStore(path, storeKey)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Add(testItem("id-1", "payload"), time.Unix(1700000000, 0)); e != nil {
		t.Fatal(e)
	}
	s.Close()
	raw, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	// Flip one ciphertext byte near the end (inside the GCM tag).
	raw[len(raw)-2] ^= 0x40
	if e = os.WriteFile(path, raw, 0600); e != nil {
		t.Fatal(e)
	}
	reopened, e := OpenStore(path, storeKey)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	if _, e = reopened.List(); e == nil {
		t.Fatal("tampered ciphertext accepted")
	}
}

func TestStoreTornTailRecovered(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, HistoryFile)
	s, e := OpenStore(path, storeKey)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Add(testItem("id-1", "complete"), time.Unix(1700000000, 0)); e != nil {
		t.Fatal(e)
	}
	s.Close()
	raw, _ := os.ReadFile(path)
	raw = append(raw, 0, 0, 0, 200, 1, 2, 3) // truncated length-prefixed frame
	if e = os.WriteFile(path, raw, 0600); e != nil {
		t.Fatal(e)
	}
	reopened, e := OpenStore(path, storeKey)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	items, e := reopened.List()
	if e != nil || len(items) != 1 || items[0].Text != "complete" {
		t.Fatal("torn tail not recovered", items, e)
	}
}

func TestCryptoNonceUniqueAndBoundToID(t *testing.T) {
	b, e := newBox(storeKey)
	if e != nil {
		t.Fatal(e)
	}
	first, e := b.seal("id-1", []byte("text"))
	if e != nil {
		t.Fatal(e)
	}
	second, e := b.seal("id-1", []byte("text"))
	if e != nil {
		t.Fatal(e)
	}
	if string(first) == string(second) {
		t.Fatal("nonce reuse detected")
	}
	plain, e := b.open("id-1", second)
	if e != nil || string(plain) != "text" {
		t.Fatal("roundtrip failed", e)
	}
	// A blob sealed for one record must not open under another ID (AAD).
	if _, e = b.open("id-2", second); e == nil {
		t.Fatal("record blob opened under wrong id")
	}
}

func TestCryptoRejectsWrongKeySize(t *testing.T) {
	if _, e := newBox([]byte("short")); e == nil {
		t.Fatal("short key accepted")
	}
}
