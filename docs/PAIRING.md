# Clipare pairing and group trust

The clipboard JSON protocol stays at version 1. Configuration schema 2 adds a
persistent X25519 identity, group ID, pinned public peer keys and removal IDs.
There are no cloud services, accounts, NetBird tokens or additional ports.

## Discovery

The local, injectable command runner executes `netbird status --json` with a
3-second deadline and a 1 MiB stdout cap. The parser uses `peers.details[]`,
`netbirdIp`, `fqdn` and `status`, matching the
[NetBird v0.80.0 status structures](https://github.com/netbirdio/netbird/blob/v0.80.0/client/status/status.go).
Only connected IPv4 CGNAT peers are probed, through a four-worker pool with
750 ms deadlines. Proxies and redirects are disabled. Scans occur only while
the Add Device window is open, on entry/manual refresh and every 20 seconds.

`GET /api/v1/discovery` exposes only public application/identity metadata.
Unknown senders cannot fetch clipboard, config, secrets or membership. Pairing
and discovery reuse the explicitly bound clipboard HTTP server, never an
automatically selected wildcard address. Nonstandard ports use manual fallback;
custom private ranges remain supported for explicitly configured legacy traffic,
not automatic pairing. FQDN requests cannot resolve to public addresses.

## Committed handshake

1. A generates a random 256-bit session ID, ephemeral X25519 key, nonce and an
   expiring Hello. It sends a SHA-256 commitment of the canonical full Hello.
2. B accepts at most one incoming session and returns its ephemeral/static public
   keys, nonce, device metadata, chosen group and expiration (at most 120 seconds).
   If either participant already has members, its established group is selected;
   conflicting established groups fail before a user dialog or any trust change.
3. A reveals the committed Hello and proves knowledge of the transcript key.
   B verifies the commitment and proof before presenting a notification/dialog.
4. Both combine static-static, ephemeral-ephemeral and the two cross X25519 DH
   results. HKDF-SHA256 binds them to the ordered, role-specific transcript,
   session, group and expiry. Separate domains derive session key and six-digit
   SAS. B proves possession of the same key; A verifies it before displaying SAS.
5. The user compares both displays. B's local GUI must approve/reject; no
   remotely callable approve endpoint exists. If A is already established, its
   local GUI also requires an explicit “Code matches” acknowledgement. A then
   proves confirmation through a separate authenticated confirm endpoint.
   A polls authenticated session
   status. Approved metadata is MAC-bound to the complete status response.
6. B saves the peer only after local approval and initiator confirmation. A saves the matching group/keys
   only after verified approval. Finish erases ephemeral state; its loss does
   not undo already persisted, explicitly approved trust. Cancellation has a
   separate authenticated endpoint. Rejected status releases the session slot.

Canonical Hello and transcript fields include permanent public identities,
ephemeral keys, nonces, IDs, names, IP/port, session/group IDs and expiration.
Changing committed fields or replaying old key proofs invalidates the handshake.
Old requests have bounded expiration; used session IDs are cached for five
minutes (maximum 256). Process restart invalidates ephemeral sessions: captured
proofs cannot authenticate a newly generated responder transcript. Public-key
validation rejects malformed and low-order X25519 inputs.

SAS is only a visual MITM/error check (approximately one-in-a-million accidental
match per attempt), never a permanent secret or a substitute for cryptographic
keys. Users must compare both screens, not simply approve every dialog. This is
a documented custom handshake with automated tests, not a claim of independent
cryptographic audit or formal verification.

Control routes:

```text
POST /api/v1/pair/request
POST /api/v1/pair/exchange
POST /api/v1/pair/confirm
POST /api/v1/pair/status
POST /api/v1/pair/finish
POST /api/v1/pair/cancel
POST /api/v1/group/membership
POST /api/v1/group/upgrade
```

Pairing POST bodies are capped at 32 KiB; membership bodies and pairing responses
at 128 KiB to accommodate the bounded member/removal lists. A bounded per-source-IP limiter allows one
request per second; the source must be a NetBird IPv4 address. No timers or
goroutines are allocated per incoming session/source. HTTP header/read/write
deadlines and application cancellation bound requests and shutdown.

## Pairwise trust and mesh

Permanent clipboard keys use static identity X25519 ECDH, HKDF-SHA256, group ID
and a canonical ordered pair of device IDs. They are derived when needed, not
distributed or stored as a group master secret. Compromising A's private key
exposes relationships involving A, not B–C's key. X25519 identities are pinned
and unique within a configuration. Rotating identities/pair relationships is
not implemented in this release.

An authenticated member may introduce new public member metadata to its group.
The pairing sponsor supplies the current membership to the new device and
publishes its addition to existing devices. Each pair derives its own key from
the public metadata and local private identity. Updates happen immediately on
change and every 20 seconds for eventual reconciliation with offline devices.
They contain no clipboard history or secrets. Known public keys cannot be
replaced by membership metadata. Removal tombstones prevent stale snapshots
from reintroducing removed IDs. Since 0.5.0-beta.8, each removal/restoration has
a per-device membership revision. Legacy tombstones normalize to revision 1;
equal revisions favor removal. A new, cryptographically verified and explicitly
approved pairing advances the removed peer's revision before persisting it.
Rejection, cancellation and ordinary unversioned metadata cannot clear removal.
Higher revisions from authenticated existing members propagate the restored
mesh; stale removal cannot undo restoration, and stale restoration cannot undo
a subsequent removal. Removed devices cannot authenticate membership updates
using their former pairwise key to restore themselves. Known active public keys
remain pinned; this is not an identity rotation mechanism.

All group members must run beta.8 or later before using reinstatement. The
optional `versions` wire field and persisted `membership_versions` config field
are not understood by older strict decoders; downgrading after saving this
state is unsupported. Bounds remain 256 revision entries and 64 remote peers.
No automatic merging of already established groups is performed.

Clipboard/health HMAC includes sender ID, method and route, in addition to raw
body and timestamp. The source selects exactly that peer's key; source/header
mismatch and unknown/removed IDs fail. Local health diagnostics use the local
config secret only from the local listen IP. Legacy remote health remains a
shared-secret compatibility exception while legacy relationships exist.

## Legacy migration

Config migration validates before writing, creates an exact owner-only backup
without overwriting an existing backup, then atomically replaces the config.
Failures leave the original file intact. Unix files are 0600; Windows files
inherit the user profile ACL. Private-key access is behind `identity.Provider`;
per-peer credentials are behind `PeerKeyProvider`, allowing future secure-store
backends without changing the wire protocol.

Migrated, already-connected legacy installations derive a domain-separated
group identifier from their existing high-entropy group secret so independent
migrations agree. Fresh groups get random IDs. `clipare1:` remains legacy-only
and never exports the identity private key. It cannot downgrade a pinned paired
relationship. Legacy and new credentials can coexist.

Updated trusted legacy peers use `/group/upgrade`, authenticated with the
existing legacy credential, to exchange public identities. The response is
MAC-bound to target ID, request timestamp and random request nonce. Retries
cannot substitute an already pinned key. Legacy credentials are retained for
this upgrade retry path, not used to authenticate paired clipboard traffic.
After the first authenticated pairwise membership request, the corresponding
legacy upgrade retry credential is erased. Legacy retries cannot change pinned
metadata, even before that confirmation. Clients without this API keep working as legacy. Full automatic mesh requires
updated members; the old group-secret trust limitations apply during bootstrap.

Headless clients reconcile authenticated membership but disable incoming pairing
because they have no user approval UI. Clipboard payloads, config/keys and raw
pairing payloads are never logged.
