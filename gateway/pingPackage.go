package gateway

import (
	"encoding/binary"
	"net"
	"time"
)

//EchoPackageLength should always be 25
const pingPackageLength = 26

//EchoPackage Type 0, ReplyPackage Type 1
type pingRequestPackage struct {
	ID             int64
	EchoTimestamp  int64
	ReplyTimestamp int64
}
type pingResponsePackage = pingRequestPackage

//ToData convert EchoPackage to binary.
func (s *pingRequestPackage) toData() (data []byte) {
	data = make([]byte, pingPackageLength)
	data[0] = 0
	binary.BigEndian.PutUint64(data[1:9], uint64(s.ID))
	binary.BigEndian.PutUint64(data[9:17], uint64(s.EchoTimestamp))
	binary.BigEndian.PutUint64(data[17:25], uint64(s.ReplyTimestamp))
	return data
}

//LoadFromData convert data to EchoPackage.
func loadFromData(data []byte) *pingRequestPackage {
	s := &pingRequestPackage{}
	s.ID = int64(binary.BigEndian.Uint64(data[1:9]))
	s.EchoTimestamp = int64(binary.BigEndian.Uint64(data[9:17]))
	s.ReplyTimestamp = int64(binary.BigEndian.Uint64(data[17:25]))
	return s
}

func (s *pingRequestPackage) response() *pingResponsePackage {
	r := &pingResponsePackage{
		ID:             s.ID,
		EchoTimestamp:  s.EchoTimestamp,
		ReplyTimestamp: time.Now().UnixNano(),
	}
	return r
}

func pingResponseHandler(conn *net.UDPConn, addr *net.UDPAddr, n int, data []byte) {
	if n != pingPackageLength {
		log.Warning("Wrong ping package size from %s", addr.String())
		return
	}
	r := loadFromData(data).response().toData()
	r[0] = 1
	n, err := conn.WriteToUDP(r, addr)
	if err != nil || n != pingPackageLength {
		log.Info("Write package to %s wrong.", addr.String())
	}
}
