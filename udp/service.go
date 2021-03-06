package udp

import (
	"net"
	"github.com/emirpasic/gods/maps/treemap"
	"sync"
	"time"
	"github.com/Catofes/go-its/config"
	"github.com/Catofes/go-its/its"
)

type ICMPStack struct {
	data                 treemap.Map
	Latency              int64
	ReceivedPackageCount int64
	PackageLost          float32
	mutex                sync.Mutex
}

func (s *ICMPStack) Init() *ICMPStack {
	s.data = *treemap.NewWithIntComparator()
	return s
}

func (s *ICMPStack) Get() *EchoPackage {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	tmp, _ := s.data.Max()
	var id = 0
	if tmp == nil {
		id = 1
	} else {
		id = tmp.(int) + 1
	}
	echoPackage := EchoPackage{}
	echoPackage.Id = id
	echoPackage.EchoTimestamp = time.Now().UnixNano()
	s.data.Put(id, &echoPackage)
	if s.data.Size() > 100 {
		k, v := s.data.Min()
		p := v.(*EchoPackage)
		if p.ReplyTimestamp > 0 {
			s.ReceivedPackageCount--
		}
		s.data.Remove(k)
	}
	s.PackageLost = 1 - float32(s.ReceivedPackageCount)/float32(s.data.Size()-1)
	return &echoPackage
}

func (s *ICMPStack) Put(reply *EchoPackage) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	id := reply.Id
	v, ok := s.data.Get(id)
	if !ok {
		return
	}
	request := v.(*EchoPackage)
	if request.ReplyTimestamp > 0 {
		return
	}
	request.ReplyTimestamp = reply.ReplyTimestamp
	request.Relay = time.Now().UnixNano() - request.EchoTimestamp

	s.Latency = (s.Latency*s.ReceivedPackageCount + request.Relay) / int64(s.ReceivedPackageCount+1)
	s.ReceivedPackageCount++
	if s.data.Size() > 100 {
		k, v := s.data.Min()
		p := v.(*EchoPackage)
		if p.ReplyTimestamp > 0 {
			s.ReceivedPackageCount--
		}
		s.data.Remove(k)
	}
	s.PackageLost = 1 - float32(s.ReceivedPackageCount)/float32(s.data.Size())
}

type RemoteServer struct {
	Ip             net.IP
	Port           uint16
	LastOnline     time.Time
	OffLine        bool
	LinkDown       bool
	PackageReceive *ICMPStack
	ServerInfo     map[string]*ServerInfo
}

func (s *RemoteServer) Init(ip net.IP, port uint16) *RemoteServer {
	s.Ip = ip
	s.Port = port
	s.LastOnline = time.Time{}
	s.LinkDown = false
	s.OffLine = false
	s.PackageReceive = (&ICMPStack{}).Init()
	s.ServerInfo = make(map[string]*ServerInfo)
	return s
}

type MainService struct {
	Servers     map[string]*RemoteServer
	ip          net.IP
	pingEvery   time.Duration
	syncEvery   time.Duration
	offlineTime time.Duration
	checkEvery  time.Duration
	deleteEvery time.Duration
	Mutex       sync.Mutex
}

func (s *MainService) Init() *MainService {
	c := config.GetInstance("")
	s.Servers = make(map[string]*RemoteServer)
	s.pingEvery = time.Duration(c.PingEvery) * time.Millisecond
	s.syncEvery = time.Duration(c.SyncEvery) * time.Millisecond
	s.offlineTime = time.Duration(c.OfflineTime) * time.Millisecond
	s.checkEvery = time.Duration(c.CheckEvery) * time.Millisecond
	s.deleteEvery = time.Duration(c.DeleteEvery) * time.Millisecond
	udpService.AddHandler(byte(1), s.echoReplyHandler)
	udpService.AddHandler(byte(2), s.syncHandler)
	if !udpService.isServer {
		Center := (&RemoteServer{}).Init(net.ParseIP(c.CenterServerAddress), c.CenterServerPort)
		s.Servers[c.CenterServerAddress] = Center
	} else {
		s.ip = net.ParseIP(c.CenterServerAddress)
	}
	return s
}

func (s *MainService) Loop() {
	go s.pingLoop()
	go s.syncLoop()
	go s.deleteLoop()
	if udpService.isServer {
		(&its.Manager{}).Init()
		go (&WebServer{}).Init().Run()
		go its.ItsManager.Loop()
		go s.checkLoop()
	}
}

func (s *MainService) pingLoop() {
	for {
		time.Sleep(s.pingEvery)
		s.Mutex.Lock()
		for _, v := range s.Servers {
			if v.Ip.Equal(s.ip) {
				continue
			}
			echoPackage := v.PackageReceive.Get()
			address := &net.UDPAddr{}
			address.IP = v.Ip
			address.Port = int(v.Port)
			udpService.connection.WriteToUDP(echoPackage.ToData(), address)
		}
		s.Mutex.Unlock()
	}
}

func (s *MainService) syncLoop() {
	for {
		time.Sleep(s.syncEvery)
		s.Mutex.Lock()
		if udpService.isServer {
			for _, v := range s.Servers {
				s.syncTo(v)
			}
		} else {
			v, ok := s.Servers[config.GetInstance("").CenterServerAddress]
			if ok {
				s.syncTo(v)
			}
		}
		for _, v := range s.Servers {
			log.Debug("udpService %s, LastOnline %s, Latency %d, PackageLost %f",
				v.Ip.String(), v.LastOnline.String(), v.PackageReceive.Latency, v.PackageReceive.PackageLost)
			for _, w := range v.ServerInfo {
				log.Debug("	SubServer %s, LastOnline %s, Latency %d, PackageLost %f",
					w.Ip.String(), time.Unix(int64(w.LastOnline)/1e9, int64(w.LastOnline)%1e9),
					w.Latency, w.PackageLost)
			}
		}
		s.Mutex.Unlock()
	}
}

func (s *MainService) checkLoop() {
	for {
		time.Sleep(s.checkEvery)
		s.Mutex.Lock()
		linkDown := 0
		offLine := 0
		checkResult := false
		for _, v := range s.Servers {
			if v.LastOnline.Equal(time.Time{}) {
				continue
			}
			//Time out
			if v.LastOnline.Add(2 * s.offlineTime).Before(time.Now()) {
				timeoutCount := 0
				totalServer := 0
				for _, u := range s.Servers {
					if u.Ip.Equal(v.Ip) {
						continue
					}
					remoteServer, ok := u.ServerInfo[v.Ip.String()]
					if ok {
						t_ := remoteServer.LastOnline
						t := time.Unix(int64(t_)/1e9, int64(t_)%1e9)
						if t.Add(s.offlineTime).Before(time.Now()) {
							timeoutCount++
						}
						totalServer++
					}
				}
				if float64(timeoutCount)/float64(totalServer) > 0.6 {
					v.OffLine = true
					offLine++
				} else {
					v.LinkDown = true
					linkDown++
				}
			} else {
				v.OffLine = false
			}
		}
		log.Debug("Check Result: Offline/LinkDown: %d/%d", offLine, linkDown)
		if float64(offLine)/float64(len(s.Servers)) > 0.6 {
			log.Warning("%d/%d OffLine!", offLine, len(s.Servers))
			checkResult = true
		}
		if linkDown > 0 {
			log.Warning("%d/%d Link Down!", linkDown, len(s.Servers))
			checkResult = true
		}
		if checkResult {
			its.ItsManager.LinkDown()
		} else {
			its.ItsManager.LinkUp()
		}
		s.Mutex.Unlock()
	}
}

func (s *MainService) deleteLoop() {
	for {
		time.Sleep(100 * s.checkEvery)
		s.Mutex.Lock()
		for k, v := range s.Servers {
			if v.LastOnline.Add(s.deleteEvery).Before(time.Now()) {
				log.Warning("Delete server %s.", )
				delete(s.Servers, k)
			}
		}
		s.Mutex.Unlock()
	}
}

func (s *MainService) syncTo(remoteServer *RemoteServer) {
	p := (&SyncPackage{}).Init()
	p.Self.Ip = remoteServer.Ip
	p.Self.Port = remoteServer.Port
	p.Token = config.GetInstance("").Token
	for _, v := range s.Servers {
		if v.OffLine {
			continue
		}
		p.Servers.Push(&ServerInfo{v.Ip, v.Port,
								   uint64(v.PackageReceive.Latency),
								   v.PackageReceive.PackageLost,
								   uint64(v.LastOnline.UnixNano())})
	}
	d, n := p.ToData()
	address := &net.UDPAddr{}
	address.IP = remoteServer.Ip
	address.Port = int(remoteServer.Port)
	for i := 0; i < n; i++ {
		udpService.connection.WriteToUDP(d[i], address)
	}
}

func (s *MainService) echoReplyHandler(conn *net.UDPConn, addr *net.UDPAddr, n int, data []byte) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	if n != EchoPackageLength {
		log.Info("Wrong package size at package type 1.")
	}
	v, ok := s.Servers[addr.IP.String()]
	if !ok {
		return
	}
	replyPackage := EchoPackage{}
	replyPackage.LoadFromData(data)
	v.PackageReceive.Put(&replyPackage)
	v.LastOnline = time.Now()
}

func (s *MainService) syncHandler(conn *net.UDPConn, addr *net.UDPAddr, n int, data []byte) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	replyPackage := (&SyncPackage{}).Init()
	err := replyPackage.LoadFromData(data, n)
	if err != nil {
		return
	}
	if replyPackage.Token != config.GetInstance("").Token {
		log.Debug("Receive wrong token package.")
		return
	}
	if udpService.isServer {
		remoteServer, alreadyIn := s.Servers[addr.IP.String()]
		if alreadyIn {
			for {
				v, ok := replyPackage.Servers.Pop()
				if ! ok {
					break
				}
				serverInfo := v.(*ServerInfo)
				if remoteServer.Ip.Equal(serverInfo.Ip) {
					continue
				}
				remoteServer.ServerInfo[serverInfo.Ip.String()] = serverInfo
			}
		} else {
			s.Servers[addr.IP.String()] = (&RemoteServer{}).Init(addr.IP, uint16(addr.Port))
		}

	} else {
		if !s.ip.Equal(replyPackage.Self.Ip) {
			s.ip = replyPackage.Self.Ip
		}
		for {
			v, ok := replyPackage.Servers.Pop()
			if ! ok {
				break
			}
			serverInfo := v.(*ServerInfo)
			if s.ip.Equal(serverInfo.Ip) {
				continue
			}
			_, alreadyIn := s.Servers[serverInfo.Ip.String()]
			if !alreadyIn {
				log.Warning("Add reomte server %s", serverInfo.Ip.String())
				s.Servers[serverInfo.Ip.String()] = (&RemoteServer{}).Init(serverInfo.Ip, serverInfo.Port)
			}

		}
	}
}
