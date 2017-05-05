package udp

import (
	"net"
	"time"
)

func EchoRequestHandler(conn *net.UDPConn, addr *net.UDPAddr, n int, data []byte) {
	packageType := byte(0)
	if n != 21 {
		log.Info("Wrong package size at package type %d.", packageType)
	}
	echoPackage := EchoPackage{}
	echoPackage.LoadFromData(data)
	echoPackage.ReplyTimestamp = time.Now().UnixNano()
	data = echoPackage.ToData()
	data[0] = 1
	n, err := conn.WriteToUDP(data, addr)
	if err != nil || n != 21 {
		log.Info("Write package to %s wront.", addr.String())
	}
}
