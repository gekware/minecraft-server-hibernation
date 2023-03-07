package minequery

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var pingBeta18PingPacket = []byte{0xfe}

const (
	pingBeta18ResponsePacketID       byte = 0xff
	pingBeta18ResponseFieldSeparator      = "§"
)

// StatusBeta18 holds status response returned by Beta 1.8 to Release 1.4 (exclusively) Minecraft servers.
type StatusBeta18 struct {
	MOTD          string
	OnlinePlayers int
	MaxPlayers    int
}

// String returns a user-friendly representation of a server status response.
// It contains presumed (Beta 1.8+) Minecraft Server version, online count and naturalized MOTD.
func (s *StatusBeta18) String() string {
	return fmt.Sprintf("Minecraft Server (Beta 1.8+), %d/%d players online, MOTD: %s",
		s.OnlinePlayers, s.MaxPlayers, naturalizeMOTD(s.MOTD))
}

// PingBeta18 pings Beta 1.8 to Release 1.4 (exclusively) Minecraft servers (Notchian servers of more late versions
// also respond to this ping packet.)
//
//goland:noinspection GoUnusedExportedFunction
func PingBeta18(host string, port int) (*StatusBeta18, error) {
	return defaultPinger.PingBeta18(host, port)
}

// PingBeta18 pings Beta 1.8 to Release 1.4 (exclusively) Minecraft servers (Notchian servers of more late versions
// also respond to this ping packet.)
func (p *Pinger) PingBeta18(host string, port int) (*StatusBeta18, error) {
	conn, err := p.openTCPConn(host, port)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	// Send ping packet
	if err = p.pingBeta18WritePingPacket(conn); err != nil {
		return nil, fmt.Errorf("could not write ping packet: %w", err)
	}

	// Read status response (note: uses the same packet reading approach as 1.4)
	payload, err := p.pingBeta18ReadResponsePacket(conn)
	if err != nil {
		return nil, fmt.Errorf("could not read response packet: %w", err)
	}

	// Parse response data from status packet
	res, err := p.pingBeta18ParseResponsePayload(payload)
	if err != nil {
		return nil, fmt.Errorf("could not parse status from response packet: %w", err)
	}

	return res, nil
}

// Communication

func (p *Pinger) pingBeta18WritePingPacket(writer io.Writer) error {
	// Write single-byte FE ping packet
	_, err := writer.Write(pingBeta18PingPacket)
	return err
}

func (p *Pinger) pingBeta18ReadResponsePacket(reader io.Reader) ([]byte, error) {
	// Read first three bytes (packet ID as byte + packet length as short)
	// and create a reader over this buffer for sequential reading.
	b := make([]byte, 3)
	bn, err := reader.Read(b)
	if err != nil {
		return nil, err
	} else if bn < 3 {
		return nil, io.EOF
	}
	br := bytes.NewReader(b)

	// Read packet type, return error if it isn't FF kick packet
	id, err := br.ReadByte()
	if err != nil {
		return nil, err
	} else if id != pingBeta18ResponsePacketID {
		return nil, fmt.Errorf("expected packet ID %#x, but instead got %#x", ping16ResponsePacketID, id)
	}

	// Read packet length, return error if it isn't readable as unsigned short
	// Worth noting that this needs to be multiplied by two further on (for encoding reasons, most probably)
	var length uint16
	if err = binary.Read(br, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	// Read remainder of the status packet as raw bytes
	// This is a UTF-16BE string separated by § (paragraph sign)
	payload := bytes.NewBuffer(make([]byte, 0, length*2))
	if _, err = io.CopyN(payload, reader, int64(length*2)); err != nil {
		return nil, err
	}

	// Decode UTF-16BE string
	decoded, err := utf16BEDecoder.Bytes(payload.Bytes())
	if err != nil {
		return nil, err
	}

	return decoded, nil
}

// Response processing

func (p *Pinger) pingBeta18ParseResponsePayload(payload []byte) (*StatusBeta18, error) {
	// Split status string, parse and map to struct returning errors if conversions fail
	fields := strings.Split(string(payload), pingBeta18ResponseFieldSeparator)
	if len(fields) != 3 {
		return nil, fmt.Errorf("%w: expected 3 status fields, got %d", ErrInvalidStatus, len(fields))
	}
	motd, onlineString, maxString := fields[0], fields[1], fields[2]

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

	return &StatusBeta18{
		MOTD:          motd,
		OnlinePlayers: int(online),
		MaxPlayers:    int(max),
	}, nil
}
