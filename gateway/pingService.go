package gateway

import (
	"net"
	"sync"
	"time"

	"github.com/Workiva/go-datastructures/queue"
)

type pingService struct {
	address     *net.UDPAddr
	every       time.Duration
	latency     int64
	packageLost float32
	requestID   int64
	cap         int64
	responseIDs *queue.Queue
	udp         *udpService
	mutex       sync.Mutex
}

func (s *pingService) init() {
	s.cap = 100
	s.responseIDs = queue.New(2 * s.cap)
}

func (s *pingService) run(udp *udpService) {
	s.udp = udp
	s.loop()
}

func (s *pingService) loop() {
	for {
		time.Sleep(s.every)
		s.sendPingPackage()
	}
}

func (s *pingService) sendPingPackage() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	data := (&pingRequestPackage{
		ID:            s.requestID + 1,
		EchoTimestamp: time.Now().UnixNano(),
	}).toData()
	_, err := s.udp.conn.WriteToUDP(data, s.address)
	if err != nil {
		log.Warning("Send ping request to %s failed. %s", s.address.String(), err)
		return
	}
	s.Calculate()
	s.requestID++
}

func (s *pingService) Calculate() {
	outQueueID := s.requestID - s.cap + 1
	for {
		if s.responseIDs.Len() <= 0 {
			break
		}
		if id, err := s.responseIDs.Peek(); err != nil {
			if id.(int64) < outQueueID {
				s.responseIDs.Get(1)
			}
		}
	}
	s.packageLost = float32(s.responseIDs.Len()) / float32(s.cap)
}

func (s *pingService) handleResponsePackage(conn *net.UDPConn, addr *net.UDPAddr, n int, data []byte) {
	if n != pingPackageLength {
		log.Warning("Wrong ping package size from %s", addr.String())
		return
	}
	p := loadFromData(data)
	s.responseIDs.Put(p.ID)
	latency := time.Now().UnixNano() - p.EchoTimestamp
	s.latency = (s.latency*(s.cap-1) + latency) / s.cap
	s.Calculate()
}