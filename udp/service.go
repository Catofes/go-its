package udp

import (
	"net"
	"github.com/emirpasic/gods/maps/treemap"
	"sync"
	"time"
	"github.com/Catofes/go-its/config"
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
	s.PackageLost = float32(s.ReceivedPackageCount) / float32(s.data.Size())
}

type RemoteServer struct {
	Ip             net.IP
	Port           uint16
	LastOnline     time.Time
	PackageReceive *ICMPStack
	ServerInfo     map[string]*ServerInfo
}

type MainService struct {
	Servers   map[string]*RemoteServer
	ip        net.IP
	pingEvery time.Duration
	syncEvery time.Duration
	Mutex     sync.Mutex
}

func (s *MainService) Init() *MainService {
	s.Servers = make(map[string]*RemoteServer)
	s.pingEvery = 100 * time.Millisecond
	s.syncEvery = 6 * time.Second
	Server.AddHandler(byte(1), s.echoReplyHandler)
	Server.AddHandler(byte(2), s.syncHandler)
	c := config.GetInstance("")
	if !Server.isServer {
		Center := RemoteServer{net.ParseIP(c.CenterServerAddress), c.CenterServerPort, time.Time{},
							   (&ICMPStack{}).Init(), make(map[string]*ServerInfo)}
		s.Servers[c.CenterServerAddress] = &Center
	} else {
		s.ip = net.ParseIP(c.CenterServerAddress)
	}
	return s
}

func (s *MainService) Loop() {
	go s.pingLoop()
	go s.syncLoop()
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
			//log.Debug("Send ping to address: %s.", address.String())
			Server.connection.WriteToUDP(echoPackage.ToData(), address)
		}
		s.Mutex.Unlock()
	}
}

func (s *MainService) syncLoop() {
	for {
		time.Sleep(s.syncEvery)
		s.Mutex.Lock()
		if Server.isServer {
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
			log.Debug("Server %s, LastOnline %s, Latency %d, PackageLost %f",
				v.Ip.String(), v.LastOnline.String(), v.PackageReceive.Latency, v.PackageReceive.PackageLost)
			for _, vv := range v.ServerInfo {
				log.Debug("	SubServer %s, LastOnline %s, Latency %d, PackageLost %f",
					vv.Ip.String(), time.Unix(int64(vv.LastOnline)/1e9, int64(vv.Latency)%1e9),
					vv.Latency, vv.PackageLost)
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
		Server.connection.WriteToUDP(d[i], address)
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
	if Server.isServer {
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
			s.Servers[addr.IP.String()] = &RemoteServer{
				addr.IP, uint16(addr.Port), time.Now(),
				(&ICMPStack{}).Init(), make(map[string]*ServerInfo)}
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
				log.Debug("Add reomte server %s", serverInfo.Ip.String())
				s.Servers[serverInfo.Ip.String()] = &RemoteServer{
					serverInfo.Ip, serverInfo.Port, time.Time{},
					(&ICMPStack{}).Init(), make(map[string]*ServerInfo)}
			}

		}
	}
}
