package minequery

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

var (
	queryRequestHeader            = []byte{0xfe, 0xfd}
	queryResponseStringTerminator = []byte{0x0}
	queryFullStatPadding          = []byte{0xff, 0xff, 0xff, 0x01}
	queryKVSectionPadding         = []byte{0x73, 0x70, 0x6c, 0x69, 0x74, 0x6e, 0x75, 0x6d, 0x00, 0x80, 0x00}
	queryPlayerSectionPadding     = []byte{0x01, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x5f, 0x00, 0x00}
)

const (
	queryPacketTypeHandshake byte = 9
	queryPacketTypeStat      byte = 0
)

const (
	querySessionIDMask int32 = 0x0f0f0f0f
)

const (
	queryGameType = "SMP"
	queryGameID   = "MINECRAFT"
)

// BasicQueryStatus holds basic, simplified query status response returned Minecraft servers via Query protocol.
type BasicQueryStatus struct {
	MOTD          string
	GameType      string
	Map           string
	OnlinePlayers int
	MaxPlayers    int
	Port          int
	Host          string
}

// FullQueryPluginEntry holds plugin entry info (name and version) of plugin sent via Query protocol.
type FullQueryPluginEntry struct {
	Name    string
	Version string
}

// FullQueryStatus holds full query status response returned Minecraft servers via Query protocol.
type FullQueryStatus struct {
	MOTD          string
	GameType      string
	GameID        string
	Version       string
	ServerVersion string
	Plugins       []FullQueryPluginEntry
	Map           string
	OnlinePlayers int
	MaxPlayers    int
	SamplePlayers []string
	Port          int
	Host          string
	Data          map[string]string
}

// QueryBasic queries Minecraft servers and returns simplified query response.
//
//goland:noinspection GoUnusedExportedFunction
func QueryBasic(host string, port int) (*BasicQueryStatus, error) {
	return defaultPinger.QueryBasic(host, port)
}

// QueryBasic queries Minecraft servers and returns simplified query response.
//
//goland:noinspection GoUnusedExportedFunction
func (p *Pinger) QueryBasic(host string, port int) (*BasicQueryStatus, error) {
	// Try to use cache first.
	sessionData, hit := p.getCachedSession(host, port)
	if hit {
		// Open UDP connection with predefined local address from cache.
		conn, err := p.openUDPConnWithLocalAddr(host, port, sessionData.Address)
		if err != nil {
			return nil, err
		}

		// Request basic query info with cached session.
		res, err := p.requestBasicStat(conn, sessionData)
		if err == nil {
			_ = conn.Close()
			return res, nil
		}

		_ = conn.Close()
		// On error, fall back to creating a new session.
	}

	// Open UDP connection.
	conn, err := p.openUDPConn(host, port)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	// Create a new session and obtain challenge token.
	sessionData, err = p.createAndCacheSession(port, host, conn)
	if err != nil {
		return nil, err
	}

	// Request basic query info with newly created session.
	res, err := p.requestBasicStat(conn, sessionData)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// QueryFull queries Minecraft servers and returns full query response.
//
//goland:noinspection GoUnusedExportedFunction
func QueryFull(host string, port int) (*FullQueryStatus, error) {
	return defaultPinger.QueryFull(host, port)
}

// QueryFull queries Minecraft servers and returns full query response.
//
//goland:noinspection GoUnusedExportedFunction
func (p *Pinger) QueryFull(host string, port int) (*FullQueryStatus, error) {
	// Try to use cache first.
	sessionData, hit := p.getCachedSession(host, port)
	if hit {
		// Open UDP connection with predefined local address from cache.
		conn, err := p.openUDPConnWithLocalAddr(host, port, sessionData.Address)
		if err != nil {
			return nil, err
		}

		// Request full query info with cached session.
		res, err := p.requestFullStat(conn, sessionData)
		if err == nil {
			_ = conn.Close()
			return res, nil
		}

		_ = conn.Close()
		// On error, fall back to creating a new session.
	}

	// Open UDP connection.
	conn, err := p.openUDPConn(host, port)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	// Create a new session and obtain challenge token.
	sessionData, err = p.createAndCacheSession(port, host, conn)
	if err != nil {
		return nil, err
	}

	// Request full query info with newly created session.
	res, err := p.requestFullStat(conn, sessionData)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (p *Pinger) requestBasicStat(conn *net.UDPConn, session session) (*BasicQueryStatus, error) {
	if err := p.writeQueryBasicStatPacket(conn, session.SessionID, session.Token); err != nil {
		return nil, err
	}

	content, err := p.readQueryStatResponsePacket(conn, session.SessionID)
	if err != nil {
		return nil, err
	}

	return p.parseQueryBasicStatResponse(content)
}

func (p *Pinger) requestFullStat(conn *net.UDPConn, session session) (*FullQueryStatus, error) {
	if err := p.writeQueryFullStatPacket(conn, session.SessionID, session.Token); err != nil {
		return nil, err
	}

	content, err := p.readQueryStatResponsePacket(conn, session.SessionID)
	if err != nil {
		return nil, err
	}

	return p.parseQueryFullStatResponse(content)
}

// Session management

type session struct {
	SessionID, Token int32
	Address          string
}

func getSessionCacheKey(host string, port int) string { return fmt.Sprintf("%s:%d", host, port) }
func generateSessionID() int32                        { return int32(time.Now().Unix()) & querySessionIDMask }

func (p *Pinger) createAndCacheSession(port int, host string, conn *net.UDPConn) (session, error) {
	// Generate new time-based session ID and write a handshake packet
	sessionID := generateSessionID()
	if err := p.writeQueryHandshakePacket(conn, sessionID); err != nil {
		return session{}, err
	}

	// Read response packet and get data stream
	content, err := p.readQueryHandshakeResponsePacket(conn, sessionID)
	if err != nil {
		return session{}, err
	}

	// Parse response and obtain challenge token
	token, err := p.parseQueryHandshakeResponse(content)
	if err != nil {
		return session{}, err
	}

	sessionData := session{sessionID, token, conn.LocalAddr().String()}
	if p.SessionCache != nil {
		p.SessionCache.SetDefault(getSessionCacheKey(host, port), sessionData)
	}
	return sessionData, nil
}

func (p *Pinger) getCachedSession(host string, port int) (session, bool) {
	if p.SessionCache == nil {
		return session{}, false
	}

	key := getSessionCacheKey(host, port)
	data, hit := p.SessionCache.Get(key)
	if !hit {
		return session{}, false
	}

	sessionData := data.(session)
	return sessionData, true
}

// Communication

func (p *Pinger) writeQueryHandshakePacket(conn *net.UDPConn, sessionID int32) error {
	var packet bytes.Buffer

	// Write request packet header
	_, _ = packet.Write(queryRequestHeader)

	// Write packet type
	_ = packet.WriteByte(queryPacketTypeHandshake)

	// Write session ID
	_ = binary.Write(&packet, binary.BigEndian, sessionID)

	_, err := packet.WriteTo(conn)
	return err
}

func (p *Pinger) readQueryHandshakeResponsePacket(conn *net.UDPConn, sessionID int32) (io.Reader, error) {
	// Read UDP packet into 1024 byte buffer and create a reader
	// of it to read data sequentially.
	b := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(b)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(b[:n])

	// Read packet type, which must be handshake
	id, err := reader.ReadByte()
	if err != nil {
		return nil, err
	} else if id != queryPacketTypeHandshake {
		return nil, fmt.Errorf("expected packet ID %#x, but instead got %#x", queryPacketTypeHandshake, id)
	}

	// Read session ID from response, return an error if it's
	// not the one in request.
	var resSessionID int32
	if err = binary.Read(reader, binary.BigEndian, &resSessionID); err != nil {
		return nil, err
	} else if resSessionID != sessionID {
		return nil, fmt.Errorf("expected session ID %#x, but instead got %#x", sessionID, resSessionID)
	}

	return reader, nil
}

func (p *Pinger) writeQueryBasicStatPacket(conn *net.UDPConn, sessionID int32, token int32) error {
	var packet bytes.Buffer

	// Write request packet header
	_, _ = packet.Write(queryRequestHeader)

	// Write packet type
	_ = packet.WriteByte(queryPacketTypeStat)

	// Write session ID
	_ = binary.Write(&packet, binary.BigEndian, sessionID)

	// Write token
	_ = binary.Write(&packet, binary.BigEndian, token)

	_, err := packet.WriteTo(conn)
	return err
}

func (p *Pinger) readQueryStatResponsePacket(conn *net.UDPConn, sessionID int32) (io.Reader, error) {
	// Read UDP packet into 1024 byte buffer and create a reader
	// of it to read data sequentially.
	b := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(b)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(b[:n])

	// Read packet type, which must be stat
	id, err := reader.ReadByte()
	if err != nil {
		return nil, err
	} else if id != queryPacketTypeStat {
		return nil, fmt.Errorf("expected packet ID %#x, but instead got %#x", queryPacketTypeStat, id)
	}

	// Read session ID from response, return an error if it's
	// not the one in request.
	var resSessionID int32
	if err = binary.Read(reader, binary.BigEndian, &resSessionID); err != nil {
		return nil, err
	} else if resSessionID != sessionID {
		return nil, fmt.Errorf("expected session ID %#x, but instead got %#x", sessionID, resSessionID)
	}

	return reader, nil
}

func (p *Pinger) writeQueryFullStatPacket(conn *net.UDPConn, sessionID int32, token int32) error {
	var packet bytes.Buffer

	// Write request packet header
	_, _ = packet.Write(queryRequestHeader)

	// Write packet type
	_ = packet.WriteByte(queryPacketTypeStat)

	// Write session ID
	_ = binary.Write(&packet, binary.BigEndian, sessionID)

	// Write token
	_ = binary.Write(&packet, binary.BigEndian, token)

	// Write padding
	_, _ = packet.Write(queryFullStatPadding)

	_, err := packet.WriteTo(conn)
	return err
}

// Response processing

func (p *Pinger) parseQueryHandshakeResponse(reader io.Reader) (int32, error) {
	// Read all the remaining data in packet and ensure it has NUL terminator (if UseStrict)
	token, _ := readAll(reader)
	if len(token) == 0 {
		return 0, fmt.Errorf("challenge token is empty")
	}
	if bytes.HasSuffix(token, queryResponseStringTerminator) {
		token = token[:len(token)-1]
	} else if p.UseStrict {
		return 0, fmt.Errorf("challenge token did not end with NUL byte")
	}

	// Parse token string to int
	tokenInt, err := strconv.ParseInt(string(token), 10, 32)
	if err != nil {
		return 0, err
	}

	return int32(tokenInt), nil
}

func (p *Pinger) parseQueryBasicStatResponse(reader io.Reader) (*BasicQueryStatus, error) {
	// Read all the remaining data and ensure it has NUL terminator (if UseStrict)
	data, _ := readAll(reader)
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: empty response body", ErrInvalidStatus)
	}
	if bytes.HasSuffix(data, queryResponseStringTerminator) {
		data = data[:len(data)-len(queryResponseStringTerminator)]
	} else if p.UseStrict {
		return nil, fmt.Errorf("%w: response body is not NUL-termianted", ErrInvalidStatus)
	}

	// Split response string by NUL bytes (into 6 substrings, because 6th also contains port and hostname
	// that need to be parsed specially).
	fields := strings.SplitN(string(data), string(queryResponseStringTerminator), 6)
	if len(fields) != 6 {
		return nil, fmt.Errorf("%w: expected 5 first string fields in response body, got %#v", ErrInvalidStatus, len(fields)-1)
	}
	motd, gameType, mapName, onlinePlayersStr, maxPlayerStr := fields[0], fields[1], fields[2], fields[3], fields[4]

	// Ensure gametype is indeed a hardcoded SMP string
	if gameType != queryGameType && p.UseStrict {
		return nil, fmt.Errorf("%w: expected gametype field to be %#v, got %#v", ErrInvalidStatus, queryGameType, gameType)
	}

	// Parse online players integer
	onlinePlayers, err := strconv.ParseInt(onlinePlayersStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("%w: could not parse online players count: %s", ErrInvalidStatus, err)
	}

	// Parse max players integer
	maxPlayers, err := strconv.ParseInt(maxPlayerStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("%w: could not parse max players count: %s", ErrInvalidStatus, err)
	}

	// Create a reader for remaining data to read it sequentially (port + hostname)
	remReader := bytes.NewReader([]byte(fields[5]))

	// Unpack port as short integer (little-endian)
	var port int16
	if err = binary.Read(remReader, binary.LittleEndian, &port); err != nil {
		return nil, err
	}

	// Read host as byte sequence
	hostBytes, err := readAll(remReader)
	if err != nil {
		return nil, err
	}
	return &BasicQueryStatus{
		MOTD:          motd,
		GameType:      gameType,
		Map:           mapName,
		OnlinePlayers: int(onlinePlayers),
		MaxPlayers:    int(maxPlayers),
		Port:          int(port),
		Host:          string(hostBytes),
	}, nil
}

func (p *Pinger) parseQueryFullStatResponse(reader io.Reader) (*FullQueryStatus, error) {
	// Read all the remaining data and ensure it has NUL terminator (if UseStrict)
	data, _ := readAll(reader)
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: empty response body", ErrInvalidStatus)
	}
	if bytes.HasSuffix(data, queryResponseStringTerminator) {
		data = data[:len(data)-len(queryResponseStringTerminator)]
	} else if p.UseStrict {
		return nil, fmt.Errorf("%w: response body is not NUL-termianted", ErrInvalidStatus)
	}
	dataReader := bytes.NewReader(data)

	// Read padding for Key-Value section and ensure it is equal to hardcoded value (if UseStrict)
	pb := make([]byte, len(queryKVSectionPadding))
	if _, err := dataReader.Read(pb); err != nil {
		return nil, err
	} else if !bytes.Equal(pb, queryKVSectionPadding) && p.UseStrict {
		return nil, fmt.Errorf("%w: key-value section padding is invalid", ErrInvalidStatus)
	}

	// Read and parse KV map
	fields, err := queryReadFullStatFieldMap(dataReader)
	if err != nil {
		return nil, err
	}

	// Read padding for players section and ensure it is also hardcoded (if UseStrict)
	pb = make([]byte, len(queryPlayerSectionPadding))
	if _, err = dataReader.Read(pb); err != nil {
		return nil, err
	} else if !bytes.Equal(pb, queryPlayerSectionPadding) && p.UseStrict {
		return nil, fmt.Errorf("%w: player section padding is invalid", ErrInvalidStatus)
	}

	// Read player list
	players, err := queryReadFullStatPlayerList(dataReader)
	if err != nil {
		return nil, err
	}

	// Read hostname field (MOTD)
	motd, err := queryGetFullStatField(fields, "hostname")
	if err != nil {
		return nil, err
	}

	// Read gametype field and ensure it is a hardcoded SMP value (if UseStrict)
	gameType, err := queryGetFullStatField(fields, "gametype")
	if err != nil {
		return nil, err
	} else if gameType != queryGameType && p.UseStrict {
		return nil, fmt.Errorf("%w: expected gametype field to be %#v, got %#v", ErrInvalidStatus, queryGameType, gameType)
	}

	// Read game_id field and ensure it is a hardcoded MINECRAFT value (if UseStrict)
	gameID, err := queryGetFullStatField(fields, "game_id")
	if err != nil {
		return nil, err
	} else if gameID != queryGameID && p.UseStrict {
		return nil, fmt.Errorf("%w: expected game_id field to be %#v, got %#v", ErrInvalidStatus, queryGameID, gameID)
	}

	// Read version field
	version, err := queryGetFullStatField(fields, "version")
	if err != nil {
		return nil, err
	}

	// Read server version and plugins field (still present on vanilla too) and parse it
	serverVersionStr, err := queryGetFullStatField(fields, "plugins")
	if err != nil {
		return nil, err
	}
	serverVersion, plugins, err := queryParseFullStatPluginsList(serverVersionStr)
	if err != nil {
		return nil, fmt.Errorf("%w: could not parse plugins field: %s", ErrInvalidStatus, err)
	}

	// Read map field
	mapName, err := queryGetFullStatField(fields, "map")
	if err != nil {
		return nil, err
	}

	// Read numplayers field (online players) and parse int from it
	onlinePlayersStr, err := queryGetFullStatField(fields, "numplayers")
	if err != nil {
		return nil, err
	}
	onlinePlayers, err := strconv.ParseInt(onlinePlayersStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("%w: could not parse numplayers field: %s", ErrInvalidStatus, err)
	}

	// Read maxplayers field (max players) and parse int from it
	maxPlayersStr, err := queryGetFullStatField(fields, "maxplayers")
	if err != nil {
		return nil, err
	}
	maxPlayers, err := strconv.ParseInt(maxPlayersStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("%w: could not parse maxplayers field: %s", ErrInvalidStatus, err)
	}

	// Read hostport field (port) and parse int from it
	portStr, err := queryGetFullStatField(fields, "hostport")
	if err != nil {
		return nil, err
	}
	port, err := strconv.ParseInt(portStr, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("%w: could not parse hostport field: %s", ErrInvalidStatus, err)
	}

	// Read hostname field
	hostname, err := queryGetFullStatField(fields, "hostip")
	if err != nil {
		return nil, err
	}

	return &FullQueryStatus{
		MOTD:          motd,
		GameType:      gameType,
		GameID:        gameID,
		Version:       version,
		ServerVersion: serverVersion,
		Plugins:       plugins,
		Map:           mapName,
		OnlinePlayers: int(onlinePlayers),
		MaxPlayers:    int(maxPlayers),
		SamplePlayers: players,
		Port:          int(port),
		Host:          hostname,
		Data:          fields,
	}, nil
}

func queryReadFullStatFieldMap(reader io.ByteReader) (map[string]string, error) {
	fields := make(map[string]string)
	for {
		key, err := readAllUntilZero(reader)
		if err != nil {
			return nil, err
		} else if len(key) == 0 {
			break
		}
		value, err := readAllUntilZero(reader)
		if err != nil {
			return nil, err
		}
		fields[string(key)] = string(value)
	}
	return fields, nil
}

func queryReadFullStatPlayerList(reader io.ByteReader) ([]string, error) {
	players := make([]string, 0, 10)
	for {
		nickname, err := readAllUntilZero(reader)
		if err != nil {
			return nil, err
		} else if len(nickname) == 0 {
			break
		}
		players = append(players, string(nickname))
	}
	return players, nil
}

func queryParseFullStatPluginsList(str string) (string, []FullQueryPluginEntry, error) {
	// Split version string by color; left part is server version and brand, right part is plugins list
	parts := strings.SplitN(str, ":", 2)
	if len(parts) < 2 {
		return parts[0], make([]FullQueryPluginEntry, 0), nil
	}
	ver, rem := parts[0], parts[1]

	// Split plugins part by semicolon and process
	pluginNames := strings.Split(rem, ";")
	plugins := make([]FullQueryPluginEntry, len(pluginNames))
	for i, name := range pluginNames {
		// Split plugin name by space; left part is name, right is version (be sure to trim spaces)
		// Also ensure plugin entry has two parts.
		nameParts := strings.SplitN(strings.TrimSpace(name), " ", 2)
		if len(nameParts) < 2 {
			return "", nil, fmt.Errorf("invalid plugin field syntax")
		}
		plugins[i] = FullQueryPluginEntry{nameParts[0], nameParts[1]}
	}

	return ver, plugins, nil
}

func queryGetFullStatField(fields map[string]string, key string) (string, error) {
	value, ok := fields[key]
	if !ok {
		return "", fmt.Errorf("%w: response body does not contain %s field", ErrInvalidStatus, key)
	}
	delete(fields, key)
	return value, nil
}
