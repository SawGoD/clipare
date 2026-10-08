package security

import (
	"clipare/internal/config"
	"clipare/internal/identity"
	"errors"
)

const SourceHeader = "X-Clipare-Source"

type PeerKeyProvider interface{ KeyForPeer(string) ([]byte, error) }
type PeerKeys struct{ config config.Config }

func (*PeerKeys) String() string   { return "[redacted peer credentials]" }
func (*PeerKeys) GoString() string { return "[redacted peer credentials]" }

func NewPeerKeys(c config.Config) *PeerKeys { return &PeerKeys{c} }
func (p *PeerKeys) Secret() []byte          { return []byte(p.config.Security.Secret) }
func (p *PeerKeys) KeyForPeer(id string) ([]byte, error) {
	for _, peer := range p.config.Peers {
		if peer.ID == id {
			if peer.Legacy || p.config.SchemaVersion == 0 {
				if peer.LegacySecret != "" {
					return []byte(peer.LegacySecret), nil
				}
				return p.Secret(), nil
			}
			return identity.PairKey(p.config.Identity, peer.PublicKey, p.config.Group.ID, p.config.Device.ID, id)
		}
	}
	return nil, errors.New("untrusted peer")
}
func (p *PeerKeys) Legacy(id string) bool {
	for _, peer := range p.config.Peers {
		if peer.ID == id {
			return peer.Legacy || p.config.SchemaVersion == 0
		}
	}
	return false
}
func AuthenticatedData(method, path, source string, body []byte) []byte {
	b := []byte(method + "\n" + path + "\n" + source + "\n")
	return append(b, body...)
}
