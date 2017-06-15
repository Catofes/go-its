package udp

import (
	"github.com/emirpasic/gods/stacks/arraystack"
	"encoding/binary"
	"errors"
	"net"
	"math"
)

const ServerInfoLength = 26
const SyncPackageHeader = 15

type ServerInfo struct {
	Ip          net.IP
	Port        uint16
	Latency     uint64
	PackageLost float32
	LastOnline  uint64
}

type SyncPackage struct {
	Self    ServerInfo
	Token   uint64
	Servers *arraystack.Stack
}

func (s *SyncPackage) Init() *SyncPackage {
	s.Servers = arraystack.New()
	return s
}

func (s *SyncPackage) ToData() (all_data map[int][]byte, n int) {
	all_data = make(map[int]([]byte))
	n = 0
	for {
		i := 0
		data := make([]byte, 1024)
		for i < (1024-SyncPackageHeader)/ServerInfoLength {
			start := i*ServerInfoLength + SyncPackageHeader
			v, ok := s.Servers.Pop()
			if ok {
				server := v.(*ServerInfo)
				copy(data[start:start+4], server.Ip.To4())
				binary.BigEndian.PutUint16(data[start+4:start+6], server.Port)
				binary.BigEndian.PutUint64(data[start+6:start+14], server.Latency)
				binary.BigEndian.PutUint32(data[start+14:start+18], math.Float32bits(server.PackageLost))
				binary.BigEndian.PutUint64(data[start+18:start+26], server.LastOnline)
				i++
			} else {
				break
			}
		}
		if i > 0 {
			data[0] = 2
			copy(data[1:5], s.Self.Ip.To4())
			binary.BigEndian.PutUint16(data[5:7], s.Self.Port)
			binary.BigEndian.PutUint64(data[7:15], s.Token)
			all_data[n] = data[0:i*ServerInfoLength+SyncPackageHeader]
			n++
		} else {
			break
		}
	}
	return all_data, n
}

func (s *SyncPackage) LoadFromData(data []byte, n int) error {
	server_count := (n - SyncPackageHeader) / ServerInfoLength
	if server_count*ServerInfoLength+SyncPackageHeader != n {
		log.Warning("Wrong package received. Type 2.")
		return errors.New("Wrong package size.")
	}
	s.Init()
	s.Self = ServerInfo{make(net.IP, 4), binary.BigEndian.Uint16(data[5:7]), 0, 0, 0}
	copy(s.Self.Ip, data[1:5])
	s.Token = binary.BigEndian.Uint64(data[7:15])
	for i := 0; i < server_count; i++ {
		start := SyncPackageHeader + i*ServerInfoLength
		server := ServerInfo{make(net.IP, 4),
							 binary.BigEndian.Uint16(data[start+4:start+6]),
							 binary.BigEndian.Uint64(data[start+6:start+14]),
							 math.Float32frombits(binary.BigEndian.Uint32(data[start+14:start+18])),
							 binary.BigEndian.Uint64(data[start+18:start+26])}
		copy(server.Ip, data[start:start+4])
		s.Servers.Push(&server)
	}
	return nil
}
