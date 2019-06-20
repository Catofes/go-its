package gateway

import (
	"encoding/binary"
	"errors"
	"math"
	"net"
)

const serverInfoLength = 26
const syncPackageHeader = 23

type serverInfo struct {
	IP          net.IP
	Port        uint16
	Latency     uint64
	PackageLost float32
	LastOnline  uint64
}

type syncPackage struct {
	Self    serverInfo
	Token   uint64
	Group   uint64
	Servers []serverInfo
}

func (s *syncPackage) init() *syncPackage {
	s.Servers = make([]serverInfo, 0)
	return s
}

func (s *syncPackage) makeHeader() []byte {
	data := make([]byte, syncPackageHeader)
	data[0] = 2
	copy(data[1:5], s.Self.IP.To4())
	binary.BigEndian.PutUint16(data[5:7], s.Self.Port)
	binary.BigEndian.PutUint64(data[7:15], s.Token)
	binary.BigEndian.PutUint64(data[15:23], s.Group)
	return data
}

func (s *syncPackage) loadHeader(data []byte) *syncPackage {
	copy(s.Self.IP, data[1:5])
	s.Token = binary.BigEndian.Uint64(data[7:15])
	s.Group = binary.BigEndian.Uint64(data[15:23])
	return s
}

func (s *serverInfo) toData() []byte {
	data := make([]byte, serverInfoLength)
	copy(data[0:4], s.IP.To4())
	binary.BigEndian.PutUint16(data[4:6], s.Port)
	binary.BigEndian.PutUint64(data[6:14], s.Latency)
	binary.BigEndian.PutUint32(data[14:18], math.Float32bits(s.PackageLost))
	binary.BigEndian.PutUint64(data[18:26], s.LastOnline)
	return data
}

func (s *serverInfo) fromData(data []byte) *serverInfo {
	copy(s.IP, data[0:4])
	s.Port = binary.BigEndian.Uint16(data[4:6])
	s.Latency = binary.BigEndian.Uint64(data[6:14])
	s.PackageLost = math.Float32frombits(binary.BigEndian.Uint32(data[14:18]))
	s.LastOnline = binary.BigEndian.Uint64(data[18:26])
	return s
}

func (s *syncPackage) toData() (data [][]byte) {
	data = make([][]byte, 0)
	for {
		d := make([]byte, 0)
		d = append(d, s.makeHeader()...)
		for len(d)+serverInfoLength < 1024 {
			if len(s.Servers) > 0 {
				server := s.Servers[0]
				s.Servers = s.Servers[1:]
				d = append(d, server.toData()...)
			} else {
				break
			}
		}
		data = append(data, d)
		if len(s.Servers) <= 0 {
			break
		}
	}
	return data
}

func (s *syncPackage) loadFromData(data []byte, n int) error {
	count := (n - syncPackageHeader) / serverInfoLength
	if count*serverInfoLength+syncPackageHeader != n {
		log.Warning("Wrong package received. Type 2.")
		return errors.New("wrong package size")
	}
	s.Servers = make([]serverInfo, 0)
	s.loadHeader(data[:syncPackageHeader])
	for i := 0; i < count; i++ {
		start := syncPackageHeader + i*serverInfoLength
		server := (&serverInfo{}).fromData(data[start:])
		s.Servers = append(s.Servers, *server)
	}
	return nil
}