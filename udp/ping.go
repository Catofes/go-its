package udp

import (
	"github.com/emirpasic/gods/stacks/arraystack"
	"net"
	"github.com/emirpasic/gods/maps/treemap"
	"sync"
	"time"
	"encoding/binary"
)

type ICMPStack struct {
	data                   treemap.Map
	latency                uint64
	received_package_count int
	package_lost           float32
	mutex                  sync.Mutex
}

func (s*ICMPStack) Init() {
	s.data = *treemap.NewWithIntComparator()
}

func (s*ICMPStack) Get() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	id, _ := s.data.Max()
	id = int(id) + 1

	send_data := make([]byte, 21)
	send_data[0] = byte(0)
	binary.BigEndian.PutUint64(data[1:13])
	binary.BigEndian.PutUint64(data[13:21], uint64(time.Now().UnixNano()))
	s.data.Put(id, )
}

type RemoteServer struct {
	Address        string
	Port           int
	LastOnline     int64
	PackageReceive arraystack.Stack
}

type Ping struct {
	Servers map[string]RemoteServer
	every   int
}

func (s *Ping) Init() {
	Server
}

func (s *Ping) EchoReplyHandler(conn *net.UDPConn, addr *net.UDPAddr, n int, data []byte) {

}
