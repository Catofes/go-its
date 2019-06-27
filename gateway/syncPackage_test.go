package gateway

import (
	"net"
	"testing"
	"time"
)

func TestSyncPackageParser(t *testing.T) {
	p := (&syncPackage{}).init()
	p.Self.Address,_ = net.ResolveUDPAddr("udp4","222.29.47.158:555")
	p.Token = 123
	s := serverInfo{
		Group:3,
		Latency:184932,
		PackageLost:0.8,
		LastOnline:uint64(time.Now().UnixNano())}
	s.Address, _ = net.ResolveUDPAddr("udp4","1.2.3.4:333")
	p.Servers = append(p.Servers, s)

	d := p.toData()
	r := (&syncPackage{}).init()
	err := r.loadFromData(d[0])
	if err != nil {
		log.Fatal(err)
	}
	if p.Self.Address.String()!=r.Self.Address.String() {
		log.Fatalf("Error self ip %s @ %s.", p.Self.Address.String(), r.Self.Address.String())
	}
	rs := r.Servers[0]
	if s.Address.String() != rs.Address.String() {
		log.Fatalf("Error server ip %s @ %s.", s.Address.String(), rs.Address.String())
	}
}
