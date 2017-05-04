package udp

import (
	"net"
	"encoding/binary"
	"time"
)

func EchoRequestHandler(conn *net.UDPConn, addr *net.UDPAddr, n int, data []byte) {
	package_type := byte(0)
	if n != 21 {
		log.Info("Wrong package size at package type %d.", package_type)
	}
	binary.BigEndian.PutUint64(data[13:21], uint64(time.Now().UnixNano()))
	data[0] = byte(1)
	data = data[0:21]
	n, err := conn.WriteToUDP(data, addr)
	if err != nil || n != 21 {
		log.Info("Write package to %s wront.", addr.String())
	}
}
