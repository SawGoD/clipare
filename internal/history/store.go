package history

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Store persists clipboard history in a single append-only file that contains
// nothing but ciphertext: every record is independently sealed with AES-256-GCM
// (unique nonce per record, record ID bound as AAD). No plaintext ever reaches
// disk, so there is no WAL/journal/temporary-file leak surface.
//
// Records that cannot be authenticated with the current key (for example after
// an OS profile reset replaced the master key) are preserved on disk as locked
// records: the service reports ErrKeyLost and only an explicit clear may drop
// them. The API is deliberately storage-agnostic so a SQLite backend can be
// added later for attachment metadata without changing callers.
type Store struct {
	mu     sync.Mutex
	path   string
	box    *box
	file   *os.File
	index  map[string]recordRef
	hashes map[string]string // content hash -> newest record ID
}

type recordRef struct {
	offset, length int64
	created        time.Time
	hash           string
	locked         bool
}

const (
	recordHeader = 4 // u32 length prefix
	idLimit      = 128
)

// OpenStore opens (or creates) the encrypted history file. A torn trailing
// record — for example after a crash mid-append — is discarded and compacted.
func OpenStore(path string, key []byte) (*Store, error) {
	b, e := newBox(key)
	if e != nil {
		return nil, e
	}
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return nil, errors.New("cannot create history directory")
	}
	s := &Store{path: path, box: b, index: map[string]recordRef{}, hashes: map[string]string{}}
	torn, e := s.load()
	if e != nil {
		return nil, e
	}
	f, e := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if e != nil {
		return nil, errors.New("cannot open history file")
	}
	if torn {
		if e = s.rewriteWith(f); e != nil {
			f.Close()
			return nil, e
		}
		return s, nil
	}
	s.file = f
	if _, e = f.Seek(0, 2); e != nil {
		f.Close()
		return nil, errors.New("cannot seek history file")
	}
	return s, nil
}

// load reads the journal. Structurally valid frames install refs, even when
// they fail authentication (locked). It returns true when unreadable trailing
// garbage must be compacted away.
func (s *Store) load() (bool, error) {
	if _, e := os.Stat(s.path); errors.Is(e, fs.ErrNotExist) {
		return false, nil
	}
	f, e := os.Open(s.path)
	if e != nil {
		return false, errors.New("cannot read history file")
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	torn := false
	var offset int64
	for {
		var prefix [recordHeader]byte
		if _, e := readFull(reader, prefix[:]); e != nil {
			break // clean end of journal
		}
		length := binary.BigEndian.Uint32(prefix[:])
		if length < 2+nonceSize || length > recordLimit+idLimit+recordHeader {
			torn = true
			break
		}
		frame := make([]byte, length)
		if _, e := readFull(reader, frame); e != nil {
			torn = true
			break
		}
		s.admit(frame, offset, recordHeader+int64(length))
		offset += recordHeader + int64(length)
	}
	return torn, nil
}

func readFull(reader *bufio.Reader, buffer []byte) (int, error) {
	total := 0
	for total < len(buffer) {
		n, e := reader.Read(buffer[total:])
		total += n
		if e != nil {
			return total, e
		}
	}
	return total, nil
}

// admit decodes one length-prefixed frame and installs it in the index.
// Records that fail authentication are kept as locked refs, never dropped.
func (s *Store) admit(frame []byte, offset, length int64) {
	if len(frame) < 2 {
		return
	}
	idLength := int(binary.BigEndian.Uint16(frame[:2]))
	if idLength == 0 || idLength > idLimit || len(frame) < 2+idLength+nonceSize {
		return
	}
	id := string(frame[2 : 2+idLength])
	if _, exists := s.index[id]; exists {
		return // duplicated frame after a crash between write and sync
	}
	plaintext, e := s.box.open(id, frame[2+idLength:])
	if e != nil {
		s.index[id] = recordRef{offset: offset, length: length, locked: true}
		return
	}
	var r record
	if json.Unmarshal(plaintext, &r) != nil || r.Hash == "" || r.Created <= 0 {
		s.index[id] = recordRef{offset: offset, length: length, locked: true}
		return
	}
	s.install(id, recordRef{offset: offset, length: length, created: time.Unix(r.Created, 0), hash: r.Hash})
}

func (s *Store) install(id string, ref recordRef) {
	if previous, ok := s.hashes[ref.hash]; ok && previous != id {
		if old, exists := s.index[previous]; exists && !old.created.Before(ref.created) {
			// A newer row for identical content already exists.
			s.index[id] = ref
			return
		}
		delete(s.index, previous)
	}
	s.index[id] = ref
	s.hashes[ref.hash] = id
}

// Add records one clipboard event. Duplicate message IDs are ignored; a
// repeated copy of identical content refreshes the existing row instead of
// creating a duplicate. It reports whether a new row was created.
func (s *Store) Add(item Item, now time.Time) (bool, error) {
	if item.ID == "" || len(item.ID) > idLimit {
		return false, errors.New("invalid history item id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ready() {
		return false, errors.New("history file unavailable")
	}
	if _, exists := s.index[item.ID]; exists {
		return false, nil
	}
	r := record{
		OriginID: item.OriginID, OriginName: item.OriginName, Type: item.Type,
		Hash: hashText(item.Text), Created: now.Unix(), Expires: now.Add(72 * 24 * time.Hour).Unix(),
		Text: item.Text, Sensitive: item.Sensitive, Files: item.Files,
	}
	if existing, ok := s.hashes[r.Hash]; ok {
		// Same content copied again: refresh the existing row instead of
		// creating a duplicate history entry.
		return false, s.replaceLocked(existing, r, now)
	}
	return true, s.appendLocked(item.ID, r)
}

// replaceLocked re-seals the existing row (same ID and origin) with a fresh
// timestamp, so repeated copies move the row to the top without duplicating it.
func (s *Store) replaceLocked(id string, fresh record, now time.Time) error {
	ref, ok := s.index[id]
	if !ok || ref.locked {
		return nil
	}
	fresh.Created = now.Unix()
	fresh.Expires = now.Add(72 * 24 * time.Hour).Unix()
	return s.appendLocked(id, fresh)
}

func (s *Store) appendLocked(id string, r record) error {
	plaintext, e := json.Marshal(r)
	if e != nil {
		return errors.New("cannot encode history record")
	}
	blob, e := s.box.seal(id, plaintext)
	if e != nil {
		return e
	}
	if len(blob) > recordLimit {
		return errors.New("history record too large")
	}
	frame := make([]byte, recordHeader+2+len(id)+len(blob))
	binary.BigEndian.PutUint32(frame, uint32(2+len(id)+len(blob)))
	binary.BigEndian.PutUint16(frame[recordHeader:], uint16(len(id)))
	copy(frame[recordHeader+2:], id)
	copy(frame[recordHeader+2+len(id):], blob)
	if _, e = s.file.Write(frame); e != nil {
		return errors.New("cannot write history record")
	}
	if e = s.file.Sync(); e != nil {
		return errors.New("cannot flush history record")
	}
	offset, _ := s.file.Seek(0, 1)
	s.install(id, recordRef{offset: offset - int64(len(frame)), length: int64(len(frame)), created: time.Unix(r.Created, 0), hash: r.Hash})
	return nil
}

// List returns all rows, newest first.
func (s *Store) List() ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]Item, 0, len(s.index))
	ids := make([]string, 0, len(s.index))
	for id, ref := range s.index {
		if ref.locked {
			return nil, ErrKeyLost
		}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := s.index[ids[i]], s.index[ids[j]]
		if !a.created.Equal(b.created) {
			return a.created.After(b.created)
		}
		return ids[i] > ids[j]
	})
	for _, id := range ids {
		r, e := s.readLocked(id)
		if e != nil {
			return nil, e
		}
		items = append(items, Item{
			ID: id, OriginID: r.OriginID, OriginName: r.OriginName, Timestamp: r.Created,
			Type: r.Type, Text: r.Text, Sensitive: r.Sensitive, Files: r.Files,
		})
	}
	return items, nil
}

func (s *Store) readLocked(id string) (record, error) {
	ref, ok := s.index[id]
	if !ok {
		return record{}, errors.New("history record is missing")
	}
	if ref.locked {
		return record{}, ErrKeyLost
	}
	frame := make([]byte, ref.length)
	if _, e := s.file.ReadAt(frame, ref.offset); e != nil {
		return record{}, errors.New("cannot read history record")
	}
	idLength := int(binary.BigEndian.Uint16(frame[recordHeader:]))
	blob := frame[recordHeader+2+idLength:]
	plaintext, e := s.box.open(id, blob)
	if e != nil {
		return record{}, e
	}
	var r record
	if json.Unmarshal(plaintext, &r) != nil {
		return record{}, ErrKeyLost
	}
	return r, nil
}

// Delete removes a single row locally; nothing is ever propagated to peers.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ready() {
		return errors.New("history file unavailable")
	}
	if _, ok := s.index[id]; !ok {
		return nil
	}
	delete(s.index, id)
	s.reindexHashesLocked()
	return s.rewriteLocked()
}

// Clear removes every row, including locked ones. Configuration, peers,
// identity and pairing keys are untouched; only history data lives here.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ready() {
		return errors.New("history file unavailable")
	}
	s.index = map[string]recordRef{}
	s.hashes = map[string]string{}
	return s.rewriteLocked()
}

// Cleanup drops rows past the retention window and enforces the item limit.
func (s *Store) Cleanup(now time.Time, retention time.Duration, max int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ready() || len(s.index) == 0 {
		return nil
	}
	refs := make([]recordRef, 0, len(s.index))
	for _, ref := range s.index {
		refs = append(refs, ref)
	}
	drop := survivorsToDrop(refs, now, retention, max)
	if len(drop) == 0 {
		return nil
	}
	for id, ref := range s.index {
		if drop[ref] {
			delete(s.index, id)
		}
	}
	s.reindexHashesLocked()
	return s.rewriteLocked()
}

func (s *Store) reindexHashesLocked() {
	s.hashes = map[string]string{}
	for id, ref := range s.index {
		if !ref.locked {
			s.hashes[ref.hash] = id
		}
	}
}

// rewriteLocked compacts the journal through a ciphertext-only temp file.
// Locked records are copied verbatim: a wrong key must never destroy data.
func (s *Store) rewriteLocked() error {
	return s.rewriteWith(s.file)
}

func (s *Store) rewriteWith(current *os.File) error {
	ids := make([]string, 0, len(s.index))
	for id := range s.index {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := s.index[ids[i]], s.index[ids[j]]
		if !a.created.Equal(b.created) {
			return a.created.Before(b.created)
		}
		return ids[i] < ids[j]
	})
	frames := make(map[string][]byte, len(ids))
	for _, id := range ids {
		ref := s.index[id]
		frame := make([]byte, ref.length)
		if _, e := current.ReadAt(frame, ref.offset); e != nil {
			return errors.New("cannot read history record")
		}
		frames[id] = frame
	}
	temp, e := os.CreateTemp(filepath.Dir(s.path), ".clipare-history-*.tmp")
	if e != nil {
		return errors.New("cannot prepare history rewrite")
	}
	name := temp.Name()
	defer os.Remove(name)
	if e = temp.Chmod(0600); e != nil {
		temp.Close()
		return errors.New("cannot prepare history rewrite")
	}
	offsets := make(map[string]recordRef, len(ids))
	var position int64
	for _, id := range ids {
		frame := frames[id]
		if _, e = temp.Write(frame); e != nil {
			temp.Close()
			return errors.New("cannot rewrite history file")
		}
		ref := s.index[id]
		ref.offset, ref.length = position, int64(len(frame))
		offsets[id] = ref
		position += int64(len(frame))
	}
	if e = temp.Sync(); e != nil {
		temp.Close()
		return errors.New("cannot finish history rewrite")
	}
	if e = temp.Close(); e != nil {
		return errors.New("cannot finish history rewrite")
	}
	// Close the old handle before renaming: appends after the swap must go to
	// the new inode, not to the unlinked previous file.
	if e = current.Close(); e != nil {
		return errors.New("cannot finish history rewrite")
	}
	if s.file == current {
		s.file = nil
	}
	if e = os.Rename(name, s.path); e != nil {
		return errors.New("cannot replace history file")
	}
	f, e := os.OpenFile(s.path, os.O_RDWR, 0600)
	if e != nil {
		return errors.New("cannot reopen history file")
	}
	if _, e = f.Seek(0, 2); e != nil {
		f.Close()
		return errors.New("cannot seek history file")
	}
	s.file = f
	s.index = offsets
	s.reindexHashesLocked()
	return nil
}

func (s *Store) ready() bool { return s.file != nil }

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file == nil {
		return nil
	}
	e := s.file.Close()
	s.file = nil
	return e
}

// hashText keys deduplication. It is stored in the record header and only
// confirms equality of content, never reveals it.
func hashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
