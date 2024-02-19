<h1 align="center">📡 MineQuery</h1>
<h4 align="center">Minecraft Server List Ping library written in Go</h4>
<p align="center">
    <a href="https://github.com/dreamscached/minequery/blob/v2/go.mod">
        <img alt="Go version badge" src="https://img.shields.io/github/go-mod/go-version/dreamscached/minequery">
    </a>
    <a href="https://github.com/dreamscached/minequery/releases/latest">
        <img alt="Latest release badge" src="https://img.shields.io/github/v/release/dreamscached/minequery">
    </a>
    <a href="https://pkg.go.dev/github.com/dreamscached/minequery/v2">
        <img alt="Go reference badge" src="https://pkg.go.dev/badge/github.com/dreamscached/minequery.svg">
    </a>
    <a href="https://github.com/dreamscached/minequery/blob/v2/LICENSE">
        <img alt="License badge" src="https://img.shields.io/github/license/dreamscached/minequery">
    </a>
    <br/>
    <a href="https://github.com/dreamscached/minequery#readme">
        <img alt="Minecraft version support badge" src="https://img.shields.io/badge/minecraft%20version-Beta%201.8%20to%201.3%20%7C%201.4%20to%201.5%20%7C%201.6%20%7C%201.7%2B-brightgreen">
    </a>
</p>

# 🚀 Migrating from v2 or v1

If you're new to MineQuery, you can skip this part. If you have used it before, you
might want to give it a read if you're planning to switch from v1, or want to know
about breaking changes in v2.x.x version and how to adapt your codebase.

See [MIGRATING.md][1] for help with migrating from MineQuery.

## ✨ Features

### ⛏ Minecraft Version Support

MineQuery supports pinging of all versions of Minecraft.

| [Beta 1.8 to 1.3][2] | [1.4 to 1.5][3] | [1.6][4]    | [1.7+][5]   |
|----------------------|-----------------|-------------|-------------|
| ✅ Supported          | ✅ Supported     | ✅ Supported | ✅ Supported |

### 📡 Query Protocol Support

MineQuery v2.1.0+ fully supports [Query][9] protocol.

### 🏷 SRV Record Support

MineQuery v2.5.0+ fully supports SRV records.

## 📚 How to use

### Basic usage

For simple pinging with default parameters, use package-global `Ping*` functions
(where `*` is your respective Minecraft server version.)

If you're unsure about version, it is known that Notchian servers respond to
all previous version pings (e.g. 1.7+ server will respond to 1.6 ping, and so on.)

Here's a quick example how to:

#### Pinging (1.7+ servers)

```go
import "github.com/dreamscached/minequery/v2"

res, err := minequery.Ping17("localhost", 25565)
if err != nil { panic(err) }
fmt.Println(res)
```

#### Querying

```go
import "github.com/dreamscached/minequery/v2"

res, err := minequery.QueryBasic("localhost", 25565)
// ... or ...
res, err := minequery.QueryFull("localhost", 25565)
if err != nil { panic(err) }
fmt.Println(res)
```

For full info on response object structure, see [documentation][7].

### Advanced usage

#### Pinger

For more advanced usage, such as setting custom timeout or enabling more strict
response validation, you can use `Pinger` struct with `PingerOption` passed to it:

```go
import "github.com/dreamscached/minequery/v2"

pinger := minequery.NewPinger(
minequery.WithTimeout(5 * time.Second),
minequery.WithUseStrict(true),
minequery.WithProtocolVersion16(minequery.Ping16ProtocolVersion162),
minequery.WithProtocolVersion17(minequery.Ping17ProtocolVersion172),
)
```

Then, use `Ping*` functions on it the same way as described in [Basic usage][8] section:

```go
import "github.com/dreamscached/minequery/v2"

// Ping Beta 1.8+
pinger.PingBeta18("localhost", 25565)
// Ping 1.4+
pinger.Ping14("localhost", 25565)
// Ping 1.6+
pinger.Ping16("localhost", 25565)
// Ping 1.7+
pinger.Ping17("localhost", 25565)
```

Or `Query*`:

```go
import "github.com/dreamscached/minequery/v2"

// Query basic stats
res, err := pinger.QueryBasic("localhost", 25565)
// Query full stats
res, err := pinger.QueryFull("localhost", 25565)
```

#### WithTimeout

By default, `Pinger` has 15-second timeout before connection aborts. If you need
to customize this duration, you can use `WithTimeout` option.

#### WithUseStrict

By default, `Pinger` does not validate response data it receives and silently
omits erroneous values it processes (incorrect favicon or bad player UUID).
If you need it to return an error in case of invalid response, you can use
`WithUseStrict` option.

#### WithQueryCacheExpiry

By default, `Pinger` stores query sessions in cache for 30 seconds and flushes expired
entries every 5 minutes. If you want to override these defaults, use `WithQueryCacheExpiry`
option.

#### WithQueryCacheDisabled

By default, `Pinger` stores query sessions in cache, reusing sessions and security tokens
and saving bandwidth. If you don't want to use session cache, use `WithQueryCacheDisabled`
option.

#### WithQueryCache

By default, `Pinger` stores query sessions using `patrickmn/go-cache` library
(and `WithQueryCacheExpiry`/`WithQueryCacheDisabled` only affect this implementation of
cache for Go). If you wish to use other cache implementation, you can use any that
implements `Cache` interface.

#### WithProtocolVersion16

By default, `Pinger` sends protocol version 74 in 1.6 ping packets. If you need
to customize protocol version sent, use `WithProtocolVersion16`. MineQuery provides
a convenient set of constants you can use &mdash; see `Ping16ProtocolVersion*` constants.

#### WithProtocolVersion17

By default, `Pinger` sends protocol version -1 in 1.7 ping packets. If you need
to customize protocol version sent, use `WithProtocolVersion17`. MineQuery provides
a convenient set of constants you can use &mdash; see `Ping17ProtocolVersion*` constants.

#### WithUnmarshaller

By default, `Pinger` uses standard Go `json.Unmarshal` function to unmarshal JSON which can
be slow compared to alternatives. If you need to use another unmarshaller library, you can
use this option to provide an `Unmarshaller` implementation that will be used instead.

#### WithImageDecoder

By default, `Pinger` uses standard Go `png.Decode` function to decode PNG from binary stream.
If you need to use another decoding library, you can use this option to provide
`png.Decode`-compatible function that will be used instead.

#### WithImageEncoding

By default, `Pinger` uses standard Go `base64.StdEncoding` encoding to decode Base64 string
returned in 1.7+ responses. If you need to use another encoding, you can use this option to
provide a compatible implementation that will be used instead.

[1]: MIGRATING.md

[2]: https://wiki.vg/Server_List_Ping#Beta_1.8_to_1.3

[3]: https://wiki.vg/Server_List_Ping#1.4_to_1.5

[4]: https://wiki.vg/Server_List_Ping#1.6

[5]: https://wiki.vg/Server_List_Ping#Current

[6]: https://github.com/dreamscached/minequery/issues/25

[7]: https://pkg.go.dev/github.com/dreamscached/minequery/v2

[8]: #basic-usage

[9]: https://wiki.vg/Query
