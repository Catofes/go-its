package udp

import (
	"testing"
	"net"
	"time"
)

func TestSyncPackageParser(t *testing.T) {
	p := (&SyncPackage{}).Init()
	p.Self.Ip = net.ParseIP("222.29.47.158")
	p.Self.Port = 555
	p.Token = 123
	s := ServerInfo{
		net.ParseIP("10.3.5.6"),
		333,
		184932,
		0.8,
		uint64(time.Now().UnixNano())}
	p.Servers.Push(&s)

	d, _ := p.ToData()
	r := (&SyncPackage{}).Init()
	r.LoadFromData(d[0], 41)
	if !p.Self.Ip.Equal(r.Self.Ip) {
		log.Fatal("Error self ip.", p.Self.Ip, r.Self.Ip)
	}
	if p.Self.Port != r.Self.Port {
		log.Fatal("Error self port.", p.Self.Port, r.Self.Port)
	}
	v, _ := r.Servers.Pop()
	rs := v.(*ServerInfo)
	if !s.Ip.Equal(rs.Ip) {
		log.Fatal("Error server ip.", s.Ip, rs.Ip)
	}
}
