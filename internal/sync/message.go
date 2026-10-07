package sync

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxText = 1 << 20
const MaxBody = 6*MaxText + 1024 // JSON escaping can expand every byte sixfold.
type Message struct {
	Version   int    `json:"version"`
	ID        string `json:"id"`
	Source    string `json:"source"`
	Timestamp int64  `json:"timestamp"`
	MIME      string `json:"mime"`
	Data      string `json:"data"`
}

func Hash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func NewMessage(source, data string) (Message, error) {
	return newMessageAfter(source, data, 0, "")
}

// UUIDv7 preserves local copy order even when several copies occur in one second.
func newMessageAfter(source, data string, previousTime int64, previousID string) (Message, error) {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		return Message{}, e
	}
	ms := time.Now().UnixMilli()
	if previousTime*1000 > ms {
		ms = previousTime * 1000
	}
	if len(previousID) == 36 && previousID[14] == '7' {
		prefix, err := hex.DecodeString(strings.ReplaceAll(previousID[:13], "-", ""))
		if err == nil && len(prefix) == 6 {
			var old int64
			for _, v := range prefix {
				old = old<<8 | int64(v)
			}
			if old >= ms {
				ms = old + 1
			}
		}
	} else if previousTime == ms/1000 && previousID != "" {
		ms = (previousTime + 1) * 1000
	}
	for i := 5; i >= 0; i-- {
		b[i] = byte(ms >> (8 * (5 - i)))
	}
	b[6] = (b[6] & 15) | 112
	b[8] = (b[8] & 63) | 128
	return Message{1, fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:]), source, ms / 1000, "text/plain", data}, nil
}
func (m Message) Validate() error {
	id := strings.ReplaceAll(m.ID, "-", "")
	b, e := hex.DecodeString(id)
	if len(m.ID) != 36 || m.ID[8] != '-' || m.ID[13] != '-' || m.ID[18] != '-' || m.ID[23] != '-' || e != nil || len(b) != 16 || b[8]&192 != 128 || (b[6]>>4 != 4 && b[6]>>4 != 7) {
		return errors.New("invalid message id")
	}
	if m.Version != 1 || m.MIME != "text/plain" || m.Source == "" || len(m.Source) > 128 || m.Timestamp <= 0 {
		return errors.New("unsupported message")
	}
	if len(m.Data) > MaxText || !utf8.ValidString(m.Data) || strings.ContainsRune(m.Data, 0) {
		return errors.New("invalid text or size")
	}
	return nil
}
