package gateway

import (
	"net"
	"sync"
	"time"
)

var ss *syncService

type remoteServer struct {
	config
	*pingService
	group      uint64
	infos      map[string]serverInfo
	udp        *udpService
	offline    bool
	linkDown   bool
	lastOnline time.Time
}

func (s *remoteServer) sameGroup(r *remoteServer) bool {
	if s.group&r.group > 0 {
		return true
	}
	return false
}

func (s *remoteServer) init(address net.UDPAddr) *remoteServer {
	s.pingService = (&pingService{
		address: &address,
		every:   int2Time(s.config.PingEvery),
	}).init()
	s.infos = make(map[string]serverInfo)
	return s
}

func (s *remoteServer) run(udp *udpService) {
	s.udp = udp
	s.pingService.run(s.udp)
}

type syncService struct {
	config
	address *net.UDPAddr
	group   uint64
	servers map[string]*remoteServer
	udp     *udpService
	mutex   *sync.Mutex
}

func (s *syncService) init() *syncService {
	s.servers = make(map[string]*remoteServer)
	s.mutex = &sync.Mutex{}
	if !s.IsServer {
		a, _ := net.ResolveUDPAddr("udp", s.config.CenterServer)
		s.addServer(serverInfo{
			Address: a,
			Group:   0,
		})
	}
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
	go s.sendSyncPackages()
	go s.deleteServer()
	if s.IsServer {
		go s.checkServer()
	}
}

func (s *syncService) sendSyncPackages() {
	for {
		time.Sleep(int2Time(s.config.SyncEvery))
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
	s.mutex.Lock()
	defer s.mutex.Unlock()
	p := (&syncPackage{
		Self: serverInfo{
			Address: s.address,
		},
		Token: s.config.Token,
	}).init()
	for _, v := range s.servers {
		if !v.offline {
			info := serverInfo{
				Address:     v.pingService.address,
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
	s.mutex.Lock()
	defer s.mutex.Unlock()
	data = data[:n]
	p := syncPackage{}
	p.loadFromData(data)
	if p.Token != s.config.Token {
		log.Info("wrong token package from %s", addr.String())
		return
	}
	r, ok := s.servers[addr.String()]
	if !ok {
		r = s.addServer(p.Self)
	}
	if r == nil {
		return
	}
	for _, v := range p.Servers {
		if _, ok := r.infos[v.Address.String()]; ok {
			r.infos[v.Address.String()] = v
		} else {
			s.addServer(v)
		}
	}

}

func (s *syncService) addServer(p serverInfo) *remoteServer {
	if p.Group != 0 && (p.Group&(^s.config.GroupFilter) == 0) {
		log.Info("filter server of %s", p.Address.String())
		return nil
	}
	r := (&remoteServer{
		config: s.config,
	}).init(*p.Address)
	s.servers[p.Address.String()] = r
	r.run(s.udp)
	s.updateGroup(r)
	return r
}

func (s *syncService) updateGroup(r *remoteServer) {
	s.group = s.group | (r.group & (^s.config.GroupFilter))
}

func (s *syncService) deleteServer() {
	for {
		time.Sleep(100 * int2Time(s.CheckEvery))
		s.mutex.Lock()
		for k, v := range s.servers {
			if v.lastOnline.Add(int2Time(s.config.DeleteEvery)).Before(time.Now()) {
				log.Warning("Delete server %s.", k)
				delete(s.servers, k)
			}
		}
		s.mutex.Unlock()
	}
}

func (s *syncService) checkServer() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	linkDownCount := 0
	offLineCount := 0
	serverCount := 0
	checkResult := false
	for k, v := range s.servers {
		//Ignore blank server
		if v.lastOnline.Equal(time.Time{}) {
			continue
		}

		//Ignore not same group server
		if s.group&v.group == 0 {
			continue
		}

		serverCount++
		//Time out
		if v.lastOnline.Add(2 * int2Time(s.config.OfflineTime)).Before(time.Now()) {
			timeoutCount := 0
			totalServer := 0
			for n, u := range s.servers {
				if k == n {
					continue
				}
				if rs, ok := u.infos[k]; ok {
					totalServer++
					t := time.Unix(int64(rs.LastOnline)/1e9, int64(rs.LastOnline)%1e9)
					if t.Add(int2Time(s.OfflineTime)).Before(time.Now()) {
						timeoutCount++
					}
				}
			}
			if float64(timeoutCount)/float64(totalServer) > 0.6 {
				v.offline = true
				v.linkDown = false
				offLineCount++
			} else {
				v.linkDown = true
				v.offline = false
				linkDownCount++
			}
		} else {
			v.offline = false
		}
		log.Debug("Check Result: Offline/LinkDown: %d/%d", offLineCount, linkDownCount)
		if float64(offLineCount)/float64(len(s.servers)) > 0.6 {
			log.Warning("%d/%d OffLine!", offLineCount, serverCount)
			checkResult = true
		}
		if linkDownCount > 0 {
			log.Warning("%d/%d Link Down!", linkDownCount, serverCount)
			checkResult = true
		}
		if checkResult {
			its.linkDown()
		} else {
			its.linkUp()
		}
	}
}
