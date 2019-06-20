package udp

import (
	"github.com/Catofes/go-its/config"
	"net"
	"github.com/op/go-logging"
	"sync"
	"strconv"
)

type handler func(*net.UDPConn, *net.UDPAddr, int, []byte)

type udpService struct {
	ListenAddress string
	ListenPort    int
	mutex         sync.Mutex
	handler       map[byte]Handler
	connection    *net.UDPConn
	isServer      bool
}

func (s *udpService) init() *udpService {
	s.handler = make(map[byte]Handler)
	return s
}

func (s *udpService) listen() {
	address, err := net.ResolveUDPAddr("udp", s.ListenAddress+":"+strconv.Itoa(s.ListenPort))
	if err != nil {
		log.Fatal("Can't resolve address: ", err)
	}
	connection, err := net.ListenUDP("udp", address)
	if err != nil {
		log.Fatal("Can't listen udp on", address, err)
	}
	s.connection = connection
	defer s.connection.Close()
	for {
		s.handleClient(s.connection)
	}
}

func (s *UdpService) handleClient(connection *net.UDPConn) {
	n, remoteAddress, err := connection.ReadFromUDP(s.buffer)
	if err != nil {
		log.Warning("Error read connection. %s", err.Error())
		return
	}
	//log.Debug("Get connection from %s, size %d.", remoteAddress.String(), n)
	if n <= 0 {
		return
	}
	packageType := s.buffer[0]
	s.mutex.Lock()
	if handler, ok := s.handler[packageType]; ok {
		s.mutex.Unlock()
		handler(connection, remoteAddress, n, s.buffer)
	} else {
		s.mutex.Unlock()
		log.Warning("Receive unknown package.")
	}
}

func (s *UdpService) AddHandler(packageType byte, handler Handler) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.handler[packageType] = handler
}

func (s *UdpService) DeleteHandler(packageType byte) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.handler, packageType)
}

func Run(isServer bool) {
	udpService = (&UdpService{}).Init()
	udpService.isServer = isServer
	udpService.AddHandler(byte(0), EchoRequestHandler)
	mainWaitGroup.Add(1)
	go udpService.Loop()
	service = (&MainService{}).Init()
	service.Loop()
	mainWaitGroup.Wait()
}
