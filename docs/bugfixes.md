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
