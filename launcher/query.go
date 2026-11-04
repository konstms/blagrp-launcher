package main

import (
	"encoding/binary"
	"errors"
	"net"
	"strconv"
	"time"
)

type ServerInfo struct {
	Players int
	Max     int
	Name    string
}

// querySamp — стандартный запрос SA-MP 'i' (информация о сервере).
func querySamp(addr string) (*ServerInfo, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	port, _ := strconv.Atoi(portStr)
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return nil, errors.New("адрес не найден")
	}
	ip := ips[0].To4()
	if ip == nil {
		return nil, errors.New("нужен IPv4")
	}
	conn, err := net.DialTimeout("udp", addr, 3*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	pkt := []byte("SAMP")
	pkt = append(pkt, ip...)
	pkt = append(pkt, byte(port), byte(port>>8), 'i')
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := conn.Write(pkt); err != nil {
		return nil, err
	}
	buf := make([]byte, 2048)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	b := buf[:n]
	if len(b) < 11+1+2+2+4 || string(b[:4]) != "SAMP" || b[10] != 'i' {
		return nil, errors.New("неверный ответ")
	}
	p := 11
	p++ // пароль
	info := &ServerInfo{}
	info.Players = int(binary.LittleEndian.Uint16(b[p:]))
	info.Max = int(binary.LittleEndian.Uint16(b[p+2:]))
	p += 4
	l := int(binary.LittleEndian.Uint32(b[p:]))
	p += 4
	if l >= 0 && p+l <= len(b) {
		info.Name = string(b[p : p+l])
	}
	return info, nil
}
