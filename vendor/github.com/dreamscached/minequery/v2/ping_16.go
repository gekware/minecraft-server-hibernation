package minequery

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var (
	ping16PingPacketHeader = []byte{
		// Packet ID (FE)
		0xfe,

		// Ping payload (01)
		0x01,

		// Packet identifier for plugin message (FA)
		0xfa,

		// Length of 'MC|PingHost' string (11) as unsigned short
		0x00, 0x0b,

		// 'MC|PingHost' string as UTF-16BE
		0x00, 0x4d, 0x00, 0x43, 0x00, 0x7c, 0x00, 0x50, 0x00, 0x69, 0x00, 0x6e, 0x00, 0x67, 0x00, 0x48, 0x00, 0x6f, 0x00, 0x73, 0x00, 0x74,
	}
	ping16ResponsePacketID byte = 0xff
)

const ping16ResponseFieldSeparator = "\x00"

var ping16ResponsePrefix = []byte{0xc2, 0xa7, 0x31, 0x0}

// Ping16ProtocolVersionIncompatible holds a special value (=127) returned in response that indicates incompatible
// Minecraft version (1.7+).
const Ping16ProtocolVersionIncompatible byte = 127

// Full version/snapshot to protocol version mapping list
// Extracted from https://wiki.vg/Protocol_version_numbers#Versions_before_the_Netty_rewrite
//
//goland:noinspection GoUnusedConst
const (
	// Ping16ProtocolVersion13w39b holds protocol version (=80) for Minecraft 13w39b.
	Ping16ProtocolVersion13w39b byte = 80

	// Ping16ProtocolVersion13w39a holds protocol version (=80) for Minecraft 13w39a.
	Ping16ProtocolVersion13w39a byte = 80

	// Ping16ProtocolVersion13w38c holds protocol version (=79) for Minecraft 13w38c.
	Ping16ProtocolVersion13w38c byte = 79

	// Ping16ProtocolVersion13w38b holds protocol version (=79) for Minecraft 13w38b.
	Ping16ProtocolVersion13w38b byte = 79

	// Ping16ProtocolVersion13w38a holds protocol version (=79) for Minecraft 13w38a.
	Ping16ProtocolVersion13w38a byte = 79

	// Ping16ProtocolVersion164 holds protocol version (=78) for Minecraft 1.6.4.
	Ping16ProtocolVersion164 byte = 78

	// Ping16ProtocolVersion163pre holds protocol version (=77) for Minecraft 1.6.3-pre.
	Ping16ProtocolVersion163pre byte = 77

	// Ping16ProtocolVersion13w37b holds protocol version (=76) for Minecraft 13w37b.
	Ping16ProtocolVersion13w37b byte = 76

	// Ping16ProtocolVersion13w37a holds protocol version (=76) for Minecraft 13w37a.
	Ping16ProtocolVersion13w37a byte = 76

	// Ping16ProtocolVersion13w36b holds protocol version (=75) for Minecraft 13w36b.
	Ping16ProtocolVersion13w36b byte = 75

	// Ping16ProtocolVersion13w36a holds protocol version (=75) for Minecraft 13w36a.
	Ping16ProtocolVersion13w36a byte = 75

	// Ping16ProtocolVersion162 holds protocol version (=74) for Minecraft 1.6.2.
	// It is the default value for protocol version sent by Ping16.
	Ping16ProtocolVersion162 byte = 74

	// Ping16ProtocolVersion161 holds protocol version (=73) for Minecraft 1.6.1.
	Ping16ProtocolVersion161 byte = 73

	// Ping16ProtocolVersion16pre holds protocol version (=72) for Minecraft 1.6-pre.
	Ping16ProtocolVersion16pre byte = 72

	// Ping16ProtocolVersion13w26a holds protocol version (=72) for Minecraft 13w26a.
	Ping16ProtocolVersion13w26a byte = 72

	// Ping16ProtocolVersion13w25c holds protocol version (=71) for Minecraft 13w25c.
	Ping16ProtocolVersion13w25c byte = 71

	// Ping16ProtocolVersion13w25b holds protocol version (=71) for Minecraft 13w25b.
	Ping16ProtocolVersion13w25b byte = 71

	// Ping16ProtocolVersion13w25a holds protocol version (=71) for Minecraft 13w25a.
	Ping16ProtocolVersion13w25a byte = 71

	// Ping16ProtocolVersion13w24b holds protocol version (=70) for Minecraft 13w24b.
	Ping16ProtocolVersion13w24b byte = 70

	// Ping16ProtocolVersion13w24a holds protocol version (=69) for Minecraft 13w24a.
	Ping16ProtocolVersion13w24a byte = 69

	// Ping16ProtocolVersion13w23b holds protocol version (=68) for Minecraft 13w23b.
	Ping16ProtocolVersion13w23b byte = 68

	// Ping16ProtocolVersion13w23a holds protocol version (=67) for Minecraft 13w23a.
	Ping16ProtocolVersion13w23a byte = 67

	// Ping16ProtocolVersion13w22a holds protocol version (=67) for Minecraft 13w22a.
	Ping16ProtocolVersion13w22a byte = 67

	// Ping16ProtocolVersion13w21b holds protocol version (=67) for Minecraft 13w21b.
	Ping16ProtocolVersion13w21b byte = 67

	// Ping16ProtocolVersion13w21a holds protocol version (=67) for Minecraft 13w21a.
	Ping16ProtocolVersion13w21a byte = 67

	// Ping16ProtocolVersion13w19a holds protocol version (=66) for Minecraft 13w19a.
	Ping16ProtocolVersion13w19a byte = 66

	// Ping16ProtocolVersion13w18c holds protocol version (=65) for Minecraft 13w18c.
	Ping16ProtocolVersion13w18c byte = 65

	// Ping16ProtocolVersion13w18b holds protocol version (=65) for Minecraft 13w18b.
	Ping16ProtocolVersion13w18b byte = 65

	// Ping16ProtocolVersion13w18a holds protocol version (=65) for Minecraft 13w18a.
	Ping16ProtocolVersion13w18a byte = 65

	// Ping16ProtocolVersion13w17a holds protocol version (=64) for Minecraft 13w17a.
	Ping16ProtocolVersion13w17a byte = 64

	// Ping16ProtocolVersion13w16b holds protocol version (=63) for Minecraft 13w16b.
	Ping16ProtocolVersion13w16b byte = 63

	// Ping16ProtocolVersion13w16a holds protocol version (=62) for Minecraft 13w16a.
	Ping16ProtocolVersion13w16a byte = 62

	// Ping16ProtocolVersion152 holds protocol version (=61) for Minecraft 1.5.2.
	Ping16ProtocolVersion152 byte = 61

	// Ping16ProtocolVersion20Purple holds protocol version (=92) for Minecraft 2.0 (Purple).
	Ping16ProtocolVersion20Purple byte = 92

	// Ping16ProtocolVersion20Red holds protocol version (=91) for Minecraft 2.0 (Red).
	Ping16ProtocolVersion20Red byte = 91

	// Ping16ProtocolVersion20Blue holds protocol version (=90) for Minecraft 2.0 (Blue).
	Ping16ProtocolVersion20Blue byte = 90

	// Ping16ProtocolVersion151 holds protocol version (=60) for Minecraft 1.5.1.
	Ping16ProtocolVersion151 byte = 60

	// Ping16ProtocolVersion13w12t holds protocol version (=60) for Minecraft 13w12~.
	Ping16ProtocolVersion13w12t byte = 60

	// Ping16ProtocolVersion13w11a holds protocol version (=60) for Minecraft 13w11a.
	Ping16ProtocolVersion13w11a byte = 60

	// Ping16ProtocolVersion15 holds protocol version (=60) for Minecraft 1.5.
	Ping16ProtocolVersion15 byte = 60

	// Ping16ProtocolVersion13w10b holds protocol version (=60) for Minecraft 13w10b.
	Ping16ProtocolVersion13w10b byte = 60

	// Ping16ProtocolVersion13w10a holds protocol version (=60) for Minecraft 13w10a.
	Ping16ProtocolVersion13w10a byte = 60

	// Ping16ProtocolVersion13w09c holds protocol version (=60) for Minecraft 13w09c.
	Ping16ProtocolVersion13w09c byte = 60

	// Ping16ProtocolVersion13w09b holds protocol version (=59) for Minecraft 13w09b.
	Ping16ProtocolVersion13w09b byte = 59

	// Ping16ProtocolVersion13w09a holds protocol version (=59) for Minecraft 13w09a.
	Ping16ProtocolVersion13w09a byte = 59

	// Ping16ProtocolVersion13w07a holds protocol version (=58) for Minecraft 13w07a.
	Ping16ProtocolVersion13w07a byte = 58

	// Ping16ProtocolVersion13w06a holds protocol version (=58) for Minecraft 13w06a.
	Ping16ProtocolVersion13w06a byte = 58

	// Ping16ProtocolVersion13w05b holds protocol version (=56) for Minecraft 13w05b.
	Ping16ProtocolVersion13w05b byte = 56

	// Ping16ProtocolVersion13w05a holds protocol version (=56) for Minecraft 13w05a.
	Ping16ProtocolVersion13w05a byte = 56

	// Ping16ProtocolVersion13w04a holds protocol version (=55) for Minecraft 13w04a.
	Ping16ProtocolVersion13w04a byte = 55

	// Ping16ProtocolVersion13w03a holds protocol version (=54) for Minecraft 13w03a.
	Ping16ProtocolVersion13w03a byte = 54

	// Ping16ProtocolVersion13w02b holds protocol version (=53) for Minecraft 13w02b.
	Ping16ProtocolVersion13w02b byte = 53

	// Ping16ProtocolVersion13w02a holds protocol version (=53) for Minecraft 13w02a.
	Ping16ProtocolVersion13w02a byte = 53

	// Ping16ProtocolVersion13w01b holds protocol version (=52) for Minecraft 13w01b.
	Ping16ProtocolVersion13w01b byte = 52

	// Ping16ProtocolVersion13w01a holds protocol version (=52) for Minecraft 13w01a.
	Ping16ProtocolVersion13w01a byte = 52

	// Ping16ProtocolVersion147 holds protocol version (=51) for Minecraft 1.4.7.
	Ping16ProtocolVersion147 byte = 51

	// Ping16ProtocolVersion146 holds protocol version (=51) for Minecraft 1.4.6.
	Ping16ProtocolVersion146 byte = 51

	// Ping16ProtocolVersion12w50b holds protocol version (=51) for Minecraft 12w50b.
	Ping16ProtocolVersion12w50b byte = 51

	// Ping16ProtocolVersion12w50a holds protocol version (=51) for Minecraft 12w50a.
	Ping16ProtocolVersion12w50a byte = 51

	// Ping16ProtocolVersion12w49a holds protocol version (=50) for Minecraft 12w49a.
	Ping16ProtocolVersion12w49a byte = 50

	// Ping16ProtocolVersion145 holds protocol version (=49) for Minecraft 1.4.5.
	Ping16ProtocolVersion145 byte = 49

	// Ping16ProtocolVersion144 holds protocol version (=49) for Minecraft 1.4.4.
	Ping16ProtocolVersion144 byte = 49

	// Ping16ProtocolVersion143pre holds protocol version (=48) for Minecraft 1.4.3-pre.
	Ping16ProtocolVersion143pre byte = 48

	// Ping16ProtocolVersion142 holds protocol version (=47) for Minecraft 1.4.2.
	Ping16ProtocolVersion142 byte = 47

	// Ping16ProtocolVersion141 holds protocol version (=47) for Minecraft 1.4.1.
	Ping16ProtocolVersion141 byte = 47

	// Ping16ProtocolVersion14 holds protocol version (=47) for Minecraft 1.4.
	Ping16ProtocolVersion14 byte = 47

	// Ping16ProtocolVersion12w42b holds protocol version (=47) for Minecraft 12w42b.
	Ping16ProtocolVersion12w42b byte = 47

	// Ping16ProtocolVersion12w42a holds protocol version (=46) for Minecraft 12w42a.
	Ping16ProtocolVersion12w42a byte = 46

	// Ping16ProtocolVersion12w41b holds protocol version (=46) for Minecraft 12w41b.
	Ping16ProtocolVersion12w41b byte = 46

	// Ping16ProtocolVersion12w41a holds protocol version (=46) for Minecraft 12w41a.
	Ping16ProtocolVersion12w41a byte = 46

	// Ping16ProtocolVersion12w40b holds protocol version (=45) for Minecraft 12w40b.
	Ping16ProtocolVersion12w40b byte = 45

	// Ping16ProtocolVersion12w40a holds protocol version (=44) for Minecraft 12w40a.
	Ping16ProtocolVersion12w40a byte = 44

	// Ping16ProtocolVersion12w39b holds protocol version (=43) for Minecraft 12w39b.
	Ping16ProtocolVersion12w39b byte = 43

	// Ping16ProtocolVersion12w39a holds protocol version (=43) for Minecraft 12w39a.
	Ping16ProtocolVersion12w39a byte = 43

	// Ping16ProtocolVersion12w38b holds protocol version (=43) for Minecraft 12w38b.
	Ping16ProtocolVersion12w38b byte = 43

	// Ping16ProtocolVersion12w38a holds protocol version (=43) for Minecraft 12w38a.
	Ping16ProtocolVersion12w38a byte = 43

	// Ping16ProtocolVersion12w37a holds protocol version (=42) for Minecraft 12w37a.
	Ping16ProtocolVersion12w37a byte = 42

	// Ping16ProtocolVersion12w36a holds protocol version (=42) for Minecraft 12w36a.
	Ping16ProtocolVersion12w36a byte = 42

	// Ping16ProtocolVersion12w34b holds protocol version (=42) for Minecraft 12w34b.
	Ping16ProtocolVersion12w34b byte = 42

	// Ping16ProtocolVersion12w34a holds protocol version (=41) for Minecraft 12w34a.
	Ping16ProtocolVersion12w34a byte = 41

	// Ping16ProtocolVersion12w32a holds protocol version (=40) for Minecraft 12w32a.
	Ping16ProtocolVersion12w32a byte = 40

	// Ping16ProtocolVersion132 holds protocol version (=39) for Minecraft 1.3.2.
	Ping16ProtocolVersion132 byte = 39

	// Ping16ProtocolVersion131 holds protocol version (=39) for Minecraft 1.3.1.
	Ping16ProtocolVersion131 byte = 39

	// Ping16ProtocolVersion12w30e holds protocol version (=39) for Minecraft 12w30e.
	Ping16ProtocolVersion12w30e byte = 39

	// Ping16ProtocolVersion12w30d holds protocol version (=39) for Minecraft 12w30d.
	Ping16ProtocolVersion12w30d byte = 39

	// Ping16ProtocolVersion12w30c holds protocol version (=39) for Minecraft 12w30c.
	Ping16ProtocolVersion12w30c byte = 39

	// Ping16ProtocolVersion12w30b holds protocol version (=38) for Minecraft 12w30b.
	Ping16ProtocolVersion12w30b byte = 38

	// Ping16ProtocolVersion12w30a holds protocol version (=38) for Minecraft 12w30a.
	Ping16ProtocolVersion12w30a byte = 38

	// Ping16ProtocolVersion12w27a holds protocol version (=38) for Minecraft 12w27a.
	Ping16ProtocolVersion12w27a byte = 38

	// Ping16ProtocolVersion12w26a holds protocol version (=37) for Minecraft 12w26a.
	Ping16ProtocolVersion12w26a byte = 37

	// Ping16ProtocolVersion12w25a holds protocol version (=37) for Minecraft 12w25a.
	Ping16ProtocolVersion12w25a byte = 37

	// Ping16ProtocolVersion12w24a holds protocol version (=36) for Minecraft 12w24a.
	Ping16ProtocolVersion12w24a byte = 36

	// Ping16ProtocolVersion12w23b holds protocol version (=35) for Minecraft 12w23b.
	Ping16ProtocolVersion12w23b byte = 35

	// Ping16ProtocolVersion12w23a holds protocol version (=35) for Minecraft 12w23a.
	Ping16ProtocolVersion12w23a byte = 35

	// Ping16ProtocolVersion12w22a holds protocol version (=34) for Minecraft 12w22a.
	Ping16ProtocolVersion12w22a byte = 34

	// Ping16ProtocolVersion12w21b holds protocol version (=33) for Minecraft 12w21b.
	Ping16ProtocolVersion12w21b byte = 33

	// Ping16ProtocolVersion12w21a holds protocol version (=33) for Minecraft 12w21a.
	Ping16ProtocolVersion12w21a byte = 33

	// Ping16ProtocolVersion12w19a holds protocol version (=32) for Minecraft 12w19a.
	Ping16ProtocolVersion12w19a byte = 32

	// Ping16ProtocolVersion12w18a holds protocol version (=32) for Minecraft 12w18a.
	Ping16ProtocolVersion12w18a byte = 32

	// Ping16ProtocolVersion12w17a holds protocol version (=31) for Minecraft 12w17a.
	Ping16ProtocolVersion12w17a byte = 31

	// Ping16ProtocolVersion12w16a holds protocol version (=30) for Minecraft 12w16a.
	Ping16ProtocolVersion12w16a byte = 30

	// Ping16ProtocolVersion12w15a holds protocol version (=29) for Minecraft 12w15a.
	Ping16ProtocolVersion12w15a byte = 29

	// Ping16ProtocolVersion125 holds protocol version (=29) for Minecraft 1.2.5.
	Ping16ProtocolVersion125 byte = 29

	// Ping16ProtocolVersion124 holds protocol version (=29) for Minecraft 1.2.4.
	Ping16ProtocolVersion124 byte = 29

	// Ping16ProtocolVersion123 holds protocol version (=28) for Minecraft 1.2.3.
	Ping16ProtocolVersion123 byte = 28

	// Ping16ProtocolVersion122 holds protocol version (=28) for Minecraft 1.2.2.
	Ping16ProtocolVersion122 byte = 28

	// Ping16ProtocolVersion121 holds protocol version (=28) for Minecraft 1.2.1.
	Ping16ProtocolVersion121 byte = 28

	// Ping16ProtocolVersion12w08a holds protocol version (=28) for Minecraft 12w08a.
	Ping16ProtocolVersion12w08a byte = 28

	// Ping16ProtocolVersion12w07b holds protocol version (=27) for Minecraft 12w07b.
	Ping16ProtocolVersion12w07b byte = 27

	// Ping16ProtocolVersion12w07a holds protocol version (=27) for Minecraft 12w07a.
	Ping16ProtocolVersion12w07a byte = 27

	// Ping16ProtocolVersion12w06a holds protocol version (=25) for Minecraft 12w06a.
	Ping16ProtocolVersion12w06a byte = 25

	// Ping16ProtocolVersion12w05b holds protocol version (=24) for Minecraft 12w05b.
	Ping16ProtocolVersion12w05b byte = 24

	// Ping16ProtocolVersion12w05a holds protocol version (=24) for Minecraft 12w05a.
	Ping16ProtocolVersion12w05a byte = 24

	// Ping16ProtocolVersion12w04a holds protocol version (=24) for Minecraft 12w04a.
	Ping16ProtocolVersion12w04a byte = 24

	// Ping16ProtocolVersion12w03a holds protocol version (=24) for Minecraft 12w03a.
	Ping16ProtocolVersion12w03a byte = 24

	// Ping16ProtocolVersion11 holds protocol version (=23) for Minecraft 1.1.
	Ping16ProtocolVersion11 byte = 23

	// Ping16ProtocolVersion12w01a holds protocol version (=23) for Minecraft 12w01a.
	Ping16ProtocolVersion12w01a byte = 23

	// Ping16ProtocolVersion11w50a holds protocol version (=22) for Minecraft 11w50a.
	Ping16ProtocolVersion11w50a byte = 22

	// Ping16ProtocolVersion11w49a holds protocol version (=22) for Minecraft 11w49a.
	Ping16ProtocolVersion11w49a byte = 22

	// Ping16ProtocolVersion11w48a holds protocol version (=22) for Minecraft 11w48a.
	Ping16ProtocolVersion11w48a byte = 22

	// Ping16ProtocolVersion11w47a holds protocol version (=22) for Minecraft 11w47a.
	Ping16ProtocolVersion11w47a byte = 22

	// Ping16ProtocolVersion101 holds protocol version (=22) for Minecraft 1.0.1.
	Ping16ProtocolVersion101 byte = 22

	// Ping16ProtocolVersion100 holds protocol version (=22) for Minecraft 1.0.0.
	Ping16ProtocolVersion100 byte = 22
)

// Status16 holds status response returned by 1.6 to 1.7 (exclusively) Minecraft servers.
type Status16 struct {
	ProtocolVersion int
	ServerVersion   string
	MOTD            string
	OnlinePlayers   int
	MaxPlayers      int
}

// String returns a user-friendly representation of a server status response.
// It contains Minecraft Server version, protocol version number, online count and naturalized MOTD.
func (s *Status16) String() string {
	return fmt.Sprintf("Minecraft Server (1.6+ or older, %s, protocol version %d), %d/%d players online, MOTD: %s",
		s.ServerVersion, s.ProtocolVersion, s.OnlinePlayers, s.MaxPlayers, naturalizeMOTD(s.MOTD))
}

// IsIncompatible checks if response returned an incompatible protocol version (=127), meaning
// this server cannot be joined unless client version is 1.7+.
func (s *Status16) IsIncompatible() bool {
	return s.ProtocolVersion == int(Ping16ProtocolVersionIncompatible)
}

// Ping16 pings 1.6 to 1.7 (exclusively) Minecraft servers (Notchian servers of more late versions also respond
// to this ping packet.)
//
//goland:noinspection GoUnusedExportedFunction
func Ping16(host string, port int) (*Status16, error) {
	return defaultPinger.Ping16(host, port)
}

// Ping16 pings 1.6 to 1.7 (exclusively) Minecraft servers (Notchian servers of more late versions also respond
// to this ping packet.)
func (p *Pinger) Ping16(host string, port int) (*Status16, error) {
	status, err := p.pingGeneric(p.ping16, host, port)
	if err != nil {
		return nil, err
	}
	return status.(*Status16), nil
}

func (p *Pinger) ping16(host string, port int) (interface{}, error) {
	conn, err := p.openTCPConn(host, port)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	// Send ping packet
	protocolVersion := p.ProtocolVersion16
	if protocolVersion == 0 {
		protocolVersion = Ping16ProtocolVersion162
	}
	if err = p.ping16WritePingPacket(conn, protocolVersion, host, port); err != nil {
		return nil, fmt.Errorf("could not write ping packet: %w", err)
	}

	// Read status response (note: uses the same packet reading approach as 1.4)
	payload, err := p.pingBeta18ReadResponsePacket(conn)
	if err != nil {
		return nil, fmt.Errorf("could not read response packet: %w", err)
	}

	// Parse response data from status packet
	res, err := p.ping16ParseResponsePayload(payload)
	if err != nil {
		return nil, fmt.Errorf("could not parse status from response packet: %w", err)
	}

	return res, nil
}

// Communication

func (p *Pinger) ping16WritePingPacket(writer io.Writer, protocol byte, host string, port int) error {
	// Allocate buffer with initial capacity of 64 which should be enough for most packets.
	packet := bytes.NewBuffer(make([]byte, 0, 64))

	// Write hardcoded (it doesn't change ever) packet header
	packet.Write(ping16PingPacketHeader)

	// Encode hostname to UTF16BE and store in buffer to calculate length further on
	hb := &bytes.Buffer{}
	if _, err := utf16BEEncoder.Writer(hb).Write([]byte(host)); err != nil {
		return err
	}

	// Write packet length (7 + length of hostname string)
	_ = binary.Write(packet, binary.BigEndian, uint16(7+hb.Len()))

	// Get preferred protocol version and fallback to Ping16ProtocolVersion162 if not set
	// and write it to packet
	packet.WriteByte(protocol)

	// Write hostname string length
	_ = binary.Write(packet, binary.BigEndian, uint16(len(host)))

	// Write hostname string
	_, _ = hb.WriteTo(packet)

	// Write target server port
	_ = binary.Write(packet, binary.BigEndian, uint32(port))

	_, err := packet.WriteTo(writer)
	return err
}

// Response processing

func (p *Pinger) ping16ParseResponsePayload(payload []byte) (*Status16, error) {
	// Check if data string begins with '§1\x00' (00 a7 00 31 00 00) and strip it
	if bytes.HasPrefix(payload, ping16ResponsePrefix) {
		payload = payload[len(ping16ResponsePrefix):]
	} else if p.UseStrict {
		return nil, fmt.Errorf("%w: status string is missing necessary prefix", ErrInvalidStatus)
	}

	// Split status string, parse and map to struct returning errors if conversions fail
	fields := strings.Split(string(payload), ping16ResponseFieldSeparator)
	if len(fields) != 5 {
		return nil, fmt.Errorf("%w: expected 5 status fields, got %d", ErrInvalidStatus, len(fields))
	}
	serverProtocolVersionString, serverVersion, motd, onlineString, maxString := fields[0], fields[1], fields[2], fields[3], fields[4]

	// Parse protocol version
	serverProtocolVersion, err := strconv.ParseInt(serverProtocolVersionString, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("%w: could not parse protocol version: %s", ErrInvalidStatus, err)
	}

	// Parse online players
	online, err := strconv.ParseInt(onlineString, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("%w: could not parse online players count: %s", ErrInvalidStatus, err)
	}

	// Parse max players
	max, err := strconv.ParseInt(maxString, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("%w: could not parse max players count: %s", ErrInvalidStatus, err)
	}

	return &Status16{
		ProtocolVersion: int(serverProtocolVersion),
		ServerVersion:   serverVersion,
		MOTD:            motd,
		OnlinePlayers:   int(online),
		MaxPlayers:      int(max),
	}, nil
}
