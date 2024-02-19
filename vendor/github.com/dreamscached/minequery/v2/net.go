package minequery

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

func (p *Pinger) openTCPConn(host string, port int) (net.Conn, error) {
	conn, err := p.Dialer.Dial("tcp", toAddrString(host, port))
	if err != nil {
		return nil, err
	}
	if p.Timeout != 0 {
		if err = conn.SetDeadline(time.Now().Add(p.Timeout)); err != nil {
			return nil, err
		}
	}
	return conn, nil
}

func (p *Pinger) openUDPConn(host string, port int) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", toAddrString(host, port))
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}
	if p.Timeout != 0 {
		if err = conn.SetDeadline(time.Now().Add(p.Timeout)); err != nil {
			return nil, err
		}
	}
	return conn, nil
}

func (p *Pinger) openUDPConnWithLocalAddr(host string, remotePort int, localAddr string) (*net.UDPConn, error) {
	lAddrObj, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		return nil, err
	}
	addr, err := net.ResolveUDPAddr("udp", toAddrString(host, remotePort))
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", lAddrObj, addr)
	if err != nil {
		return nil, err
	}
	if p.Timeout != 0 {
		if err = conn.SetDeadline(time.Now().Add(p.Timeout)); err != nil {
			return nil, err
		}
	}
	return conn, nil
}

// resolveSRV performs SRV lookup of a Minecraft server hostname.
//
// In case there are no records found, an empty string, a zero port and nil error are returned.
//
// In case when there is more than one record, the hostname and port of the first record with
// the least weight is returned.
func (p *Pinger) resolveSRV(host string) (string, uint16, error) {
	_, records, err := net.LookupSRV("minecraft", "tcp", host)
	if err != nil {
		var dnsError *net.DNSError
		if errors.As(err, &dnsError) && dnsError.IsNotFound {
			return "", 0, nil
		}

		return "", 0, err
	}

	if len(records) == 0 {
		return "", 0, nil
	}
	target := records[0]
	return target.Target, target.Port, nil
}

func shouldWrapIPv6(host string) bool {
	return len(host) >= 2 && !(host[0] == '[' && host[1] == ']') && strings.Count(host, ":") >= 2
}

func toAddrString(host string, port int) string {
	if shouldWrapIPv6(host) {
		return fmt.Sprintf(`[%s]:%d`, host, port)
	}
	return fmt.Sprintf(`%s:%d`, host, port)
}
