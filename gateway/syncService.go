package gateway

import (
	"net"
	"strconv"
	"sync"
	"time"
)

var ss *syncService

type remoteServer struct {
	config
	*pingService
	group    uint64
	infos    map[string]serverInfo
	udp      *udpService
	offline  bool
	linkDown bool
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
	address       *net.UDPAddr
	group         uint64
	groupFilter   uint64
	servers       map[string]*remoteServer
	udp           *udpService
	mutex         *sync.Mutex
	linkDownCount int
	offLineCount  int
	serverCount   int
	checkResult   bool
	lastCheckTime time.Time
}

func (s *syncService) init() *syncService {
	if s.config.LocalAddress != "" {
		s.address, _ = net.ResolveUDPAddr("udp", s.config.LocalAddress)
	} else {
		s.address, _ = net.ResolveUDPAddr("udp", s.config.Listen)
	}
	var err error
	s.group, err = strconv.ParseUint(s.config.Group, 2, 64)
	if err != nil {
		log.Fatal(err)
	}
	s.groupFilter, err = strconv.ParseUint(s.config.GroupFilter, 2, 64)
	if err != nil {
		log.Fatal(err)
	}
	s.servers = make(map[string]*remoteServer)
	s.mutex = &sync.Mutex{}
	return s
}

func (s *syncService) run(udp *udpService) {
	s.udp = udp
	for _, v := range s.config.ConnectTo {
		if v != s.Listen {
			a, err := net.ResolveUDPAddr("udp", v)
			if err != nil {
				log.Warningf("Add server failed: %s", err)
			}
			s.addServer(serverInfo{
				Address: a,
				Group:   0,
			})
		}
	}
	udp.addHandler(0, pingResponseHandler)
	udp.addHandler(1, s.pingPackageHandler)
	udp.addHandler(2, s.syncPackageHandler)
	s.loop()
}

func (s *syncService) loop() {
	go s.sendSyncPackages()
	go s.deleteServer()
	if s.IsServer {
		go func() {
			for {
				time.Sleep(int2Time(s.config.CheckEvery))
				s.checkServer()
			}
		}()
	}
}

func (s *syncService) sendSyncPackages() {
	for {
		time.Sleep(int2Time(s.config.SyncEvery))
		s.mutex.Lock()
		log.Debugf("Send sync package to %v.\n", s.servers)
		for _, v := range s.servers {
			s.sendSyncPackageTo(v)
		}
		s.mutex.Unlock()
	}
}

func (s *syncService) sendSyncPackageTo(remote *remoteServer) {
	if remote.address.String() == s.address.String() {
		return
	}
	remote.mutex.Lock()
	defer remote.mutex.Unlock()
	p := (&syncPackage{
		Self: serverInfo{
			Address: s.address,
			Group:   s.group,
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
				Group:       v.group,
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
	p := (&syncPackage{}).init()
	p.loadFromData(data)
	if p.Token != s.config.Token {
		log.Info("wrong token package from %s", addr.String())
		return
	}
	log.Debugf("Handle sync package from %s, remote servers %v .\n", addr.String(), p)
	r, ok := s.servers[addr.String()]
	if !ok {
		r = s.addServer(p.Self)
	}
	if r == nil {
		return
	}
	r.group = p.Self.Group
	s.updateGroup(r)
	s.updateServer(r, p)
}

func (s *syncService) addServer(p serverInfo) *remoteServer {
	if p.Address.String() == s.address.String() {
		return nil
	}
	if _, ok := s.servers[p.Address.String()]; ok {
		return nil
	}
	log.Debugf("Add server %s with group %d.\n", p.Address.String(), p.Group)
	r := (&remoteServer{
		config: s.config,
	}).init(*p.Address)

	if p.Group != 0 && (p.Group&(^s.groupFilter) == 0) {
		log.Debug("filter server of %s, not running", p.Address.String())
	} else {
		r.run(s.udp)
	}
	s.servers[p.Address.String()] = r
	s.updateGroup(r)
	return r
}

func (s *syncService) updateGroup(r *remoteServer) *remoteServer {
	t := s.group
	s.group = s.group | (r.group & (^s.groupFilter))
	if s.group != t {
		log.Debugf("Add group [%d->%d].\n", t, s.group)
	}
	return r
}

func (s *syncService) updateServer(r *remoteServer, p *syncPackage) *remoteServer {
	if r.group != 0 && (r.group&(^s.groupFilter) == 0) {
		log.Info("delete server %s. different group [%d:%d]", r.address.String(), s.group, r.group)
		r.cancel()
		return nil
	}
	for _, v := range p.Servers {
		if _, ok := r.infos[v.Address.String()]; ok {
			r.infos[v.Address.String()] = v
		} else {
			s.addServer(v)
			r.infos[v.Address.String()] = v
		}
	}
	return r
}

func (s *syncService) deleteServer() {
	for {
		time.Sleep(4 * int2Time(s.CheckEvery))
		s.mutex.Lock()
		for k, v := range s.servers {
			ignore := false
			if !s.IsServer {
				for _, v := range s.ConnectTo {
					if k == v {
						ignore = true
						break
					}
				}
			}
			if ignore {
				continue
			}
			select {
			case <-v.ctx.Done():
				delete(s.servers, k)
			default:
				if v.lastOnline.Add(int2Time(s.config.DeleteEvery)).Before(time.Now()) {
					log.Warning("Delete server %s.", k)
					v.cancel()
					delete(s.servers, k)
				}
			}
		}
		s.mutex.Unlock()
	}
}

func (s *syncService) checkServer() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.linkDownCount = 0
	s.offLineCount = 0
	s.serverCount = 0
	s.checkResult = false
	s.lastCheckTime = time.Now()
	for k, v := range s.servers {
		//Ignore self
		if k == s.address.String() {
			continue
		}
		//Ignore blank server
		if v.lastOnline.Equal(time.Time{}) {
			continue
		}

		//Ignore not same group server
		if s.group&v.group == 0 {
			continue
		}

		s.serverCount++
		//Time out
		if v.lastOnline.Add(2 * int2Time(s.config.OfflineTime)).Before(time.Now()) {
			timeoutCount := 0
			totalServer := 0
			for n, u := range s.servers {
				//Ignore self
				if k == n {
					continue
				}

				//Ignore not same group server
				if s.group&v.group == 0 {
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
				s.offLineCount++
			} else {
				v.linkDown = true
				v.offline = false
				s.linkDownCount++
			}
		} else {
			v.linkDown = false
			v.offline = false
		}
	}
	log.Debug("Check Result: Offline/LinkDown: %d/%d", s.offLineCount, s.linkDownCount)
	if float64(s.offLineCount)/float64(s.serverCount) > 0.6 {
		log.Warning("%d/%d OffLine!", s.offLineCount, s.serverCount)
		s.checkResult = true
	}
	if s.linkDownCount > 0 {
		log.Warning("%d/%d Link Down!", s.linkDownCount, s.serverCount)
		s.checkResult = true
	}
	if s.checkResult {
		its.linkDown()
	} else {
		its.linkUp()
	}
}
