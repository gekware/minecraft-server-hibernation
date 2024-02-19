# 🚀 Migrating MineQuery

MineQuery is in active development since its very release, and occasionally, newer
versions of it may introduce changes that might be breaking to existing codebase.

This page will help you migrate your code in order to adapt to changes.

## From v2.4.x

Version 2.5.0 enables SRV records support which is (to adhere to expected behavior) 
enabled *by default*. If you need to stick to v2.4.x behavior, disable it with
`WithPreferSRVRecords` option when creating new `Pinger` instance.

## From v2.2.x

Version 2.3.0 enables query session caching by default. If you need to stick to 
pre-2.3.x behavior, use `WithQueryCacheDisabled` option when creating new `Pinger`
instance.

Version 2.2.1 introduces `EnforcesSecureChat` field in `Status17`.

## From v2.0.x

Version 2.1.0 has moved from value parameters, receivers and return values to pointers.

### Receivers

All `Ping*` functions now take `*Pinger` pointer as receiver.

`DescriptionText()` now takes `*Status17` pointer as receiver.

`IsIncompatible()` now takes `*Status16` pointer as receiver.

### Parameters

`WithDialer` option now takes `*net.Dialer` as parameter.

### Fields

`Pinger` struct now has `Dialer *net.Dialer` field.

## From v1

Version 2 of MineQuery has several breaking changes from version 1. This section
will help you migrate your existing codebase to v2.

### Package renaming

MineQuery v2 has its package named `minequery/v2` instead of `ping`. Import path has
also changed:

| v1 import                                         | v2 import                                       |
|---------------------------------------------------|-------------------------------------------------|
| `import "github.com/dreamscached/minequery/ping"` | `import "github.com/dreamscached/minequery/v2"` |

### New ping function signatures

To remove own names (Legacy, Ancient) of Minecraft versions, it has been decided to
rename `Ping*` functions per `PingVERSION` scheme. See table below for reference.

| v1 signature                                                             | v2 signature                                                                                                                                                                                                                                                                              |
|--------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ping.Ping(host string, port int) (*ping.Response, error)`               | `minequery.Ping17(host string, port int) (*minequery.Status17, error)`                                                                                                                                                                                                                    |
| `ping.PingLegacy(host string, port int) (*ping.LegacyResponse, error)`   | `minequery.Ping16(host string, port int) (*minequery.Status16, error)`<br><br>⚠️ **Note!** MineQuery v1 does not differentiate 1.4 and 1.6 pings and the above example pings 1.6 servers. Use `minequery.Ping14(host string, port int) (*minequery.Status14, error)` to ping 1.4 servers. |
| `ping.PingAncient(host string, port int) (*ping.AncientResponse, error)` | `minequery.PingBeta18(host string, port int) (*minequery.StatusBeta18, error)`                                                                                                                                                                                                            |

### New response structure naming and signatures

Per same reasoning as ping function renaming, response structs also have been renamed.
See table below for reference.

Bear in mind that package name has also changed, see [Package renaming][1] section.

| v1 name                | v2 name                                                                                                                                                                                     |
|------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ping.Response`        | `minequery.Status17`                                                                                                                                                                        |
| `ping.LegacyResponse`  | `minequery.Status16`<br><br>⚠️ **Note!** MineQuery v1 does not differentiate 1.4 and 1.6 responses and the above example is related to 1.6 ping function. Use `minequery.Status14` for 1.4. |
| `ping.AncientResponse` | `minequery.StatusBeta18`                                                                                                                                                                    |

`ping.LegacyResponse` and `ping.AncientResponse` have been changed with new field names.
See table below for reference (applies both to `ping.LegacyResponse` and `ping.AncientResponse`.)

| v1 field name     | v2 field name   |
|-------------------|-----------------|
| `Version`         | `ServerVersion` |
| `MessageOfTheDay` | `MOTD`          |
| `PlayerCount`     | `OnlinePlayers` |

`ping.Response` has been heavily reworked (mostly, due to flattening) with fields renamed,
nested structs flattened and new fields added. See table below for reference.

| v1 field name       | v2 field name                                                                                                                                                |
|---------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `Version.Name`      | `VersionName`                                                                                                                                                |
| `Version.Protocol`  | `ProtocolVersion`                                                                                                                                            |
| `Players.Online`    | `OnlinePlayers`                                                                                                                                              |
| `Players.Max`       | `MaxPlayers`                                                                                                                                                 |
| `Description`       | `Description` <br><br>⚠️ **Note!** MineQuery v1 used `ping.Chat` type, v2 uses `minequery.Chat17` type, as well as adds `DescriptionText() string` function. |
| `Players.Sample`    | `SamplePlayers`<br><br>⚠️ **Note!** MineQuery v1 used anonymous struct, v2 uses `PlayerEntry17` named struct.                                                |
| `Players.Sample.ID` | `SamplePlayers.UUID`<br><br>⚠️ **Note!** MineQuery v1 did not parse UUIDs, v2 parses them to `uuid.UUID`.                                                    |
| `Favicon`           | `Icon`<br><br>⚠️ **Note!** MineQuery v1 did not process icon in any way, v2 decodes it into `image.Image` instance.                                          |
| *New in v2*         | `PreviewsChat`                                                                                                                                               |
| *New in v2*         | `EnforcesSecureChat`                                                                                                                                         |

[1]: #package-renaming