package minequery

import (
	"encoding/base64"
	"encoding/json"
	"image/png"
	"net"
	"time"

	"github.com/patrickmn/go-cache"
)

// PingerOption is a configuring function that applies certain changes to Pinger.
type PingerOption func(*Pinger)

// WithDialer sets Pinger Dialer used on Ping* function calls.
//
//goland:noinspection GoUnusedExportedFunction
func WithDialer(dialer *net.Dialer) PingerOption {
	return func(p *Pinger) {
		p.Dialer = dialer
	}
}

// WithTimeout sets Pinger Dialer timeout to the provided value.
//
//goland:noinspection GoUnusedExportedFunction
func WithTimeout(timeout time.Duration) PingerOption {
	return func(p *Pinger) {
		p.Timeout = timeout
		p.Dialer.Timeout = timeout
	}
}

// WithUseStrict sets Pinger UseStrict to the provided value.
//
//goland:noinspection GoUnusedExportedFunction
func WithUseStrict(useStrict bool) PingerOption {
	return func(p *Pinger) { p.UseStrict = useStrict }
}

// WithPreferSRVRecord sets Pinger PreferSRVRecord to the provided value.
//
//goland:noinspection GoUnusedExportedFunction
func WithPreferSRVRecord(preferSRV bool) PingerOption {
	return func(p *Pinger) {
		p.PreferSRVRecord = preferSRV
	}
}

// WithProtocolVersion16 sets Pinger ProtocolVersion16 value.
//
//goland:noinspection GoUnusedExportedFunction
func WithProtocolVersion16(version byte) PingerOption {
	return func(p *Pinger) {
		p.ProtocolVersion16 = version
	}
}

// WithProtocolVersion17 sets Pinger ProtocolVersion17 value.
//
//goland:noinspection GoUnusedExportedFunction
func WithProtocolVersion17(version int32) PingerOption {
	return func(p *Pinger) {
		p.ProtocolVersion17 = version
	}
}

// WithQueryCacheExpiry sets Pinger cache expiry and purge duration values.
// This function uses go-cache library; consider using WithQueryCache for
// custom implementations that implement Cache interface.
//
//goland:noinspection GoUnusedExportedFunction
func WithQueryCacheExpiry(expire, purge time.Duration) PingerOption {
	return func(p *Pinger) {
		p.SessionCache = cache.New(expire, purge)
	}
}

// WithQueryCache sets Pinger cache instance that will be used for server query.
//
//goland:noinspection GoUnusedExportedFunction
func WithQueryCache(cache Cache) PingerOption {
	return func(p *Pinger) {
		p.SessionCache = cache
	}
}

// WithQueryCacheDisabled disables Pinger cache used for server query.
//
//goland:noinspection GoUnusedExportedFunction
func WithQueryCacheDisabled() PingerOption {
	return func(p *Pinger) {
		p.SessionCache = nil
	}
}

// WithUnmarshaller sets JSON unmarshalling function used for unmarshalling 1.7+ responses.
//
//goland:noinspection GoUnusedExportedFunction
func WithUnmarshaller(fn UnmarshalFunc) PingerOption {
	return func(p *Pinger) {
		p.UnmarshalFunc = fn
	}
}

// WithImageDecoder sets PNG decoding function used for decoding 1.7+ response favicons.
//
//goland:noinspection GoUnusedExportedFunction
func WithImageDecoder(fn ImageDecodeFunc) PingerOption {
	return func(p *Pinger) {
		p.ImageDecodeFunc = fn
	}
}

// WithImageEncoding sets Base64 encoding used for decoding 1.7+ response favicon Base64-encoded data.
//
//goland:noinspection GoUnusedExportedFunction
func WithImageEncoding(coding *base64.Encoding) PingerOption {
	return func(p *Pinger) {
		p.ImageEncoding = coding
	}
}

// defaultPinger is a default (zero-value) Pinger used in functions
// that don't have Pinger as receiver. The default Pinger has timeout set to 15 seconds.
var defaultPinger = NewPinger()

// Pinger contains options to ping and query Minecraft servers.
type Pinger struct {
	// Dialer used to establish and maintain connection with servers.
	Dialer *net.Dialer

	// Timeout is used to set TCP/UDP connection timeout on call of Ping* and Query* functions.
	Timeout time.Duration

	// SessionCache holds query protocol sessions in order to reuse them instead of creating new each time.
	SessionCache Cache

	// UseStrict is a configuration value that defines if tolerable errors (in server ping/query responses)
	// that are by default silently ignored should be actually returned as errors.
	UseStrict bool

	// PreferSRVRecord is a configuration value that defines if Pinger will prefer SRV records, which is the
	// default behavior of Minecraft clients.
	PreferSRVRecord bool

	// UnmarshalFunc is the function used to unmarshal JSON (used by Ping17 for responses from 1.7+ servers).
	// By default, it uses json.Unmarshal function.
	UnmarshalFunc UnmarshalFunc

	// ImageDecodeFunc is the function used to decode PNG. It is provided with binary stream decoded from
	// Base64-encoded string in server status.
	ImageDecodeFunc ImageDecodeFunc

	// ImageEncoding is the encoding used in PNG favicon decoding process.
	ImageEncoding *base64.Encoding

	// ProtocolVersion16 is protocol version to use when pinging with Ping16 function.
	// By default, Ping16ProtocolVersion162 (=74) will be used.
	// See ping_16.go for full list of built-in constants.
	ProtocolVersion16 byte

	// ProtocolVersion17 is protocol version to use when pinging with Ping17 function.
	// By default, Ping17ProtocolVersionUndefined (=-1) will be used.
	// See ping_17.go for full list of built-in constants.
	ProtocolVersion17 int32
}

func newDefaultPinger() *Pinger {
	// Create struct with default parameters.
	p := &Pinger{}

	// Apply default configuration
	WithDialer(&net.Dialer{})(p)
	WithQueryCacheExpiry(30*time.Second, 5*time.Minute)(p)
	WithTimeout(15 * time.Second)(p)
	WithPreferSRVRecord(true)(p)
	WithUnmarshaller(json.Unmarshal)(p)
	WithImageEncoding(base64.StdEncoding)(p)
	WithImageDecoder(png.Decode)(p)

	return p
}

// NewPinger constructs new Pinger instance optionally with additional options.
func NewPinger(options ...PingerOption) *Pinger {
	pinger := newDefaultPinger()
	for _, configure := range options {
		configure(pinger)
	}
	return pinger
}
