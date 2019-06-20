package gateway

import (
	"net"
	"time"
)

type remoteServer struct {
	config
	pingService
	group      uint64
	infos      map[string]serverInfo
	udp        *udpService
	online     bool
	lastOnline time.Time
}

func (s *remoteServer) sameGroup(r *remoteServer) bool {
	if s.group&r.group > 0 {
		return true
	}
	return false
}

func (s *remoteServer) init() *remoteServer {
	s.pingService.init()
	s.infos = make(map[string]serverInfo)
	return s
}

type syncService struct {
	config
	address *net.UDPAddr
	group   uint64
	servers map[string]remoteServer
	udp     *udpService
}

func (s *syncService) init() *syncService {
	s.servers = make(map[string]remoteServer)
	return s
}

func (s *syncService) run(udp *udpService) {
	s.udp = udp
	udp.addHandler(0, pingResponseHandler)
	udp.addHandler(1, s.pingPackageHandler)
	udp.addHandler(2, s.syncPackageHandler)
	s.loop()
}

func (s *syncService) loop() {
	s.sendSyncPackages()
}

func (s *syncService) sendSyncPackages() {
	for {
		time.Sleep(time.Duration(s.SyncEvery) * time.Microsecond)
		for k := range s.servers {
			s.sendSyncPackageTo(k)
		}
	}
}

func (s *syncService) sendSyncPackageTo(address string) {
	remote, ok := s.servers[address]
	if !ok {
		return
	}
	remote.mutex.Lock()
	defer remote.mutex.Unlock()
	p := (&syncPackage{
		Self: serverInfo{
			IP:   s.address.IP,
			Port: uint16(s.address.Port),
		},
		Token: s.config.Token,
	}).init()
	for _, v := range s.servers {
		if v.online && v.sameGroup(&remote) {
			info := serverInfo{
				IP:          v.pingService.address.IP,
				Port:        uint16(s.address.Port),
				Latency:     uint64(v.pingService.latency),
				PackageLost: v.pingService.packageLost,
				LastOnline:  uint64(v.lastOnline.UnixNano()),
			}
			p.Servers = append(p.Servers, info)
		}
	}
	d := p.toData()
	for _, v := range d {
		s.udp.conn.WriteToUDP(v, remote.address)
	}
}

func (s *syncService) pingPackageHandler(conn *net.UDPConn, addr *net.UDPAddr, n int, data []byte) {
	remote, ok := s.servers[addr.String()]
	if !ok {
		return
	}
	remote.handleResponsePackage(conn, addr, n, data)
}

func (s *syncService) syncPackageHandler(conn *net.UDPConn, addr *net.UDPAddr, n int, data []byte) {
	data = data[:n]
	p := syncPackage{}
	p.loadFromData(data, n)
	r, ok := s.servers[addr.String()]
	if !ok {
		return
	}
	for _, v := range p.Servers {
		address := net.UDPAddr{
			IP:   v.IP,
			Port: int(v.Port),
		}
		r.infos[address.String()] = v
	}
}
