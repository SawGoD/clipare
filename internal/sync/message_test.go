package sync

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMessage(t *testing.T) {
	m, e := NewMessage("mac", "hello\n世界")
	if e != nil || m.Validate() != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(m)
	var got Message
	if e = json.Unmarshal(b, &got); e != nil || got != m {
		t.Fatal(e)
	}
	m.Data = strings.Repeat("a", MaxText+1)
	if m.Validate() == nil {
		t.Fatal("size")
	}
	if Hash("abc") != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Fatal("hash")
	}
}
func TestCache(t *testing.T) {
	c := NewCache(2, time.Minute)
	now := time.Now()
	c.Add("a", now)
	if !c.Seen("a", now) || c.Seen("a", now.Add(time.Minute)) {
		t.Fatal("ttl")
	}
	c.Add("b", now.Add(time.Second))
	c.Add("c", now.Add(2*time.Second))
	if c.Seen("a", now) || !c.Seen("c", now) {
		t.Fatal("capacity")
	}
}
