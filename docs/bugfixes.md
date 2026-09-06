# bugfix notes

Defects found while fixing the client packet short read (msh reading only the first TCP segment
of a client handshake and answering `client request unknown`). None of them caused that bug, but
they were all uncovered while tracing that code path.

Each entry has a stable anchor (`B1`, `B2`, ...) that the commit messages refer to.

Entries are grouped by whether they sit on the same code path as the short read.

---

## Same code path as the short read

### B1 — the client connection is leaked on early returns

**Where:** `lib/conn/conn.go`, `HandlerClientConn()` and `openProxy()`

**What:** four early returns abandoned `clientConn` without closing it. Every other path in
`HandlerClientConn` registers a `defer` that logs `closing connection for: <address>` and closes
the socket; these four did not.

| return | context |
| --- | --- |
| `HandlerClientConn`, after `getReqType` | the request could not be understood |
| `HandlerClientConn`, `servctrl.WarmMS()` failed | only in the "ms online" branch — the `defer` of the sibling branch does not cover it |
| `HandlerClientConn`, `default:` case | unknown request type; falls off the end of the function |
| `openProxy`, `net.Dial` to the minecraft server failed | `forwardTCP` is never launched, so nobody else closes it |

**Trigger:** the first one was reached by *every* fragmented handshake, which is exactly what the
short read bug produced — so before that fix this leaked on a large share of real connections. The
other three need a failing warm, an unknown request, or an unreachable minecraft server.

**Impact:** a file descriptor and a goroutine's worth of socket state kept per failed connection,
until the OS eventually reclaims it. The client is left hanging instead of seeing the connection
close, so it waits for its own timeout rather than failing fast.

**Fix:** close the connection (with the same log line used elsewhere) on all four returns.

A single `defer clientConn.Close()` at the top of `HandlerClientConn` would be wrong: on the happy
path the connection is handed over to the `forwardTCP` goroutines, which close it themselves — a
top level defer would tear the proxy down.

**Test:** `Test_HandlerClientConn_closesOnBadRequest` in `lib/conn/conn_test.go` sends a well framed
packet that is not a handshake and asserts the client read returns `io.EOF`. Before the fix the read
blocks until the client's own deadline.

### B2 — errors while answering the client were discarded, under a deadline meant for reading

**Where:** `lib/conn/conn.go` (8 call sites), `lib/conn/conn-prot.go` (`getPing`, `getClientPacket`)

**What:** two defects on the same spot.

1. Every answer to the client was `clientConn.Write(mes)` with both return values dropped — the
   same two lines repeated verbatim in nine places. If the write failed, the client got nothing
   and the log said nothing.
2. `getClientPacket` used `SetDeadline`, which applies to reads *and* writes, and that deadline
   was still in force when msh answered. A request that took most of the budget to read left the
   response write with an expired deadline — and the resulting error was exactly the one being
   discarded. `forwardTCP` resets the deadlines to 60s, but the path where msh answers on its own
   (hibernating server) never gets there.

There was also one write with no log at all: the request packet forwarded to the minecraft server
in `openProxy`. It is the only socket write in the project without a `TYPE_BYT` line, and that kind
of line is what made the short read diagnosable in the first place.

**Trigger:** any client that disconnects between its request and msh's answer, plus the narrow
window where reading the request consumes the whole timeout.

**Impact:** silent failures. The operator sees a client that "just doesn't connect" with nothing in
the log, which is the same symptom the short read produced.

**Fix:** a single `answerClient()` helper that sets its own write deadline, writes, reports the
error as `ERROR_CONN_WRITE` (a code that already existed, used by `forwardTCP`) and logs the bytes
sent with the same `TYPE_BYT` line as before. `getClientPacket` now sets only a **read** deadline,
so the request timeout no longer bleeds into the response. The missing `msh --> server` log was
added, along with an error check that closes both connections when forwarding the request fails.

**Test:** `Test_answerClient` in `lib/conn/conn-prot_test.go` — answering on a closed connection
must return a log. Before the fix the error was dropped and the function reported success.

---

## Different subsystem

These two were found while tracing the short read, but they sit on other code paths and neither
contributed to it. They share one root mistake with it: treating a VarInt as a fixed size field.

### B3 — the answer header overflows above 16383 bytes

**Where:** `lib/conn/conn-prot.go`, `buildMessage()` / `mountHeader()`

**What:** the length fields of msh's own answers were encoded with a hand rolled base-128 helper
that always emitted exactly 2 bytes:

```go
firstByte := len(message)%128 + 128
secondByte := float64(len(message) / 128)
```

They are VarInts. From `len >= 16384` the second byte gets the continuation bit set, so the client
waits for a third byte that is never sent and the frame is misread. The code even documented its own
ceiling — `bytes used: [ 2 | 1 | 2 | 0 - 16379 ]` — without enforcing it.

**Trigger:** an INFO answer larger than 16383 bytes. The answer is dominated by the base64 server
icon: the default msh icon is 11056 base64 chars, so the response is around 11.3 KB and there is
roughly 5 KB of headroom. `loadIcon()` scales any user icon to 64x64 and recompresses it, so only an
unusually incompressible 64x64 PNG gets there. Latent and bounded — but reachable.

**Impact:** the client cannot parse the status response; the server list entry fails to load.

**Fix:** `mountHeader` uses `utility.EncodeVarInt`, which produces 1 to 5 bytes as the spec requires.
The pointless `float64` (integer division converted to float and back to byte) is gone with it.

**Wire change to be aware of:** messages under 128 bytes (the JOIN loadscreen text) now use a 1 byte
length field instead of the old overlong `[len|0x80, 0]`. Both are valid VarInts and both are
accepted by clients, but as with the short read itself, this can only be proven against a real
client over a real network.

**Test:** `Test_buildMessage` in `lib/conn/conn-prot_test.go` builds answers of 3 sizes and decodes
the full header back. Before the fix, the 20000 byte case reports a packet length of 3630 for a
packet holding 20013 bytes of data.

### B4 — the minecraft server info response header was assumed to be 5 bytes

**Where:** `lib/servctrl/servctrl-utils.go`, `getServInfo()`

**What:** msh emulates a status request against the real minecraft server to count players, and
stripped the response header with a constant:

```go
// remove first 5 bytes that are used as header to get only the json data
recInfoData = recInfoData[5:]
```

The header is `packet length (VarInt) | packet id | json length (VarInt)`, so its size follows the
size of the json:

| json size | real header | what `[5:]` does |
| --- | --- | --- |
| < 128 bytes | 3 bytes | eats 2 bytes of json → unmarshal fails |
| 128 – 16383 bytes | **5 bytes** | correct |
| >= 16384 bytes | 7 bytes | leaves 2 junk bytes → unmarshal fails |

The json was also never cut at its declared length, so anything the server sent after it went into
the unmarshal too.

**Trigger:** both ends are reachable but borderline. Under 128 bytes needs a short MOTD and a server
with no favicon (whether it fits depends on the minecraft version, which keeps adding fields); over
16384 bytes needs a heavy `server-icon.png`.

**Impact:** not fatal. `countPlayerSafe()` falls back to the `list` command and then to msh's
internal connection count, so player counting keeps working — but it logs an error every cycle and
spams the server console with `list` commands.

**Fix:** `extractServInfo()`, a pure function that decodes the two VarInts with
`utility.ParseVarInt`, checks the packet id and slices the json at its declared length.

**Test:** `Test_extractServInfo` in `lib/servctrl/servctrl-utils_test.go` covers all three header
sizes plus malformed responses. Before the fix the 3 byte and 7 byte cases return corrupted json,
and a response declaring more json than it carries is accepted silently.
