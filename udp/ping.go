package udp

import (
	"net"
	"github.com/emirpasic/gods/maps/treemap"
	"sync"
	"time"
	"fmt"
)

type ICMPStack struct {
	data                   treemap.Map
	latency                int64
	received_package_count int64
	package_lost           float32
	mutex                  sync.Mutex
}

func (s*ICMPStack) Init() {
	s.data = *treemap.NewWithIntComparator()
}

func (s*ICMPStack) Get() *EchoPackage {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	tmp, _ := s.data.Max()
	id := tmp.(int)
	echoPackage := EchoPackage{}
	echoPackage.Id = id
	echoPackage.EchoTimestamp = time.Now().UnixNano()
	s.data.Put(id, echoPackage)
	if s.data.Size() > 100 {
		k, v := s.data.Min()
		p := v.(EchoPackage)
		if p.ReplyTimestamp > 0 {
			s.received_package_count--
		}
		s.data.Remove(k)
	}
	return &echoPackage
}

func (s*ICMPStack) Put(reply *EchoPackage) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	id := reply.Id
	v, ok := s.data.Get(id)
	if !ok {
		return
	}
	request := v.(EchoPackage)
	if request.ReplyTimestamp > 0 {
		return
	}
	request.ReplyTimestamp = reply.ReplyTimestamp
	request.Relay = time.Now().UnixNano() - request.EchoTimestamp

	s.latency = (s.latency*s.received_package_count + request.Relay) / int64(s.data.Size())
	s.received_package_count++
	if s.data.Size() > 100 {
		k, v := s.data.Min()
		p := v.(EchoPackage)
		if p.ReplyTimestamp > 0 {
			s.received_package_count--
		}
		s.data.Remove(k)
	}
	s.package_lost = float32(s.received_package_count) / float32(s.data.Size())
}

type RemoteServer struct {
	Address        string
	Port           int
	LastOnline     time.Time
	PackageReceive ICMPStack
}

type Ping struct {
	Servers map[string]RemoteServer
	every   time.Duration
	mutex   sync.Mutex
}

func (s *Ping) Init() {
	s.Servers = make(map[string]RemoteServer)
	s.every = 1000 * time.Millisecond
	Server.AddHandler(byte(1), s.EchoReplyHandler)
}

func (s*Ping) Loop() {
	go s.pingLoop()
	go s.syncLoop()
}

func (s*Ping) pingLoop() {
	for {
		time.Sleep(s.every)
		s.mutex.Lock()
		for _, v := range s.Servers {
			echoPackage := v.PackageReceive.Get()
			address, err := net.ResolveUDPAddr("udp", v.Address+":"+string(v.Port))
			if err != nil {
				continue
			}
			Server.connection.WriteToUDP(echoPackage.ToData(), address)
		}
		s.mutex.Unlock()
	}
}

func (s*Ping) syncLoop() {
	for {
		time.Sleep(s.every)
		for _, v := range s.Servers {
			fmt.Println(v.Address, " ", v.LastOnline, v.PackageReceive.latency, v.PackageReceive.package_lost)
		}
	}
}

func (s *Ping) EchoReplyHandler(conn *net.UDPConn, addr *net.UDPAddr, n int, data []byte) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	packageType := byte(1)
	if n != 21 {
		log.Info("Wrong package size at package type %d.", packageType)
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
