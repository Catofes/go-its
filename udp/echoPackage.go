package udp

import (
	"encoding/binary"
	"net"
	"time"
)

const EchoPackageLength = 23

type EchoPackage struct {
	Id             int
	EchoTimestamp  int64
	ReplyTimestamp int64
	Relay          int64
}

func (s *EchoPackage) ToData() (data []byte) {
	data = make([]byte, EchoPackageLength)
	data[0] = 0
	binary.BigEndian.PutUint32(data[1:5], uint32(s.Id))
	binary.BigEndian.PutUint64(data[5:13], uint64(s.EchoTimestamp))
	binary.BigEndian.PutUint64(data[13:21], uint64(s.ReplyTimestamp))
	return data
}

func (s *EchoPackage) LoadFromData(data []byte) {
	s.Id = int(binary.BigEndian.Uint32(data[1:5]))
	s.EchoTimestamp = int64(binary.BigEndian.Uint64(data[5:13]))
	s.ReplyTimestamp = int64(binary.BigEndian.Uint64(data[13:21]))
}

func EchoRequestHandler(conn *net.UDPConn, addr *net.UDPAddr, n int, data []byte) {
	packageType := byte(0)
	if n != EchoPackageLength {
		log.Info("Wrong package size at package type %d.", packageType)
	}
	echoPackage := EchoPackage{}
	echoPackage.LoadFromData(data)
	echoPackage.ReplyTimestamp = time.Now().UnixNano()
	data = echoPackage.ToData()
	data[0] = 1
	n, err := conn.WriteToUDP(data, addr)
	if err != nil || n != EchoPackageLength {
		log.Info("Write package to %s wrong.", addr.String())
	}
}
