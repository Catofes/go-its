package udp

import (
	"encoding/binary"
	"net"
	"time"
)

//EchoPackageLength should always be 25
const echoPackageLength = 25

//EchoPackage Type 0, ReplyPackage Type 1
type echoPackage struct {
	Group				int
	ID					int
	EchoTimestamp		int64
	ReplyTimestamp  	int64
	Relay           	int64
}
type responsePackage = echoPackage

//ToData convert EchoPackage to binary.
func (s *echoPackage) toData() (data []byte) {
	data = make([]byte, echoPackageLength)
	data[0] = 0
	binary.BigEndian.PutUint32(data[1:5], uint32(s.Group))
	binary.BigEndian.PutUint32(data[5:9], uint32(s.ID))
	binary.BigEndian.PutUint64(data[9:17], uint64(s.EchoTimestamp))
	binary.BigEndian.PutUint64(data[17:25], uint64(s.ReplyTimestamp))
	return data
}

//LoadFromData convert data to EchoPackage.
func (s *echoPackage) loadFromData(data []byte) {
	s.Group = int(binary.BigEndian.Uint32(data[1:5]))
	s.ID = int(binary.BigEndian.Uint32(data[5:9]))
	s.EchoTimestamp = int64(binary.BigEndian.Uint64(data[9:17]))
	s.ReplyTimestamp = int64(binary.BigEndian.Uint64(data[17:25]))
}

func (s *echoPackage) response(*responsePackage){
	r := responsePackage{
		Group:			s.Group,
		ID:				s.ID,
		EchoTimestamp:	s.EchoTimestamp,
		ReplyTimestamp:	time.Now().UnixNano(),
	}
	return r
}

//EchoRequestHandler return EchoPackage.
func echoRequestHandler(conn *net.UDPConn, addr *net.UDPAddr, n int, data []byte) {
	packageType := byte(0)
	if n != echoPackageLength {
		log.Info("Wrong package size at package type %d.", packageType)
	}
	echoPackage := echoPackage{}
	echoPackage.LoadFromData(data)
	echoPackage.ReplyTimestamp = time.Now().UnixNano()
	data = echoPackage.ToData()
	data[0] = 1
	n, err := conn.WriteToUDP(data, addr)
	if err != nil || n != echoPackageLength {
		log.Info("Write package to %s wrong.", addr.String())
	}
}
