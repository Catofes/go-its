package udp

import (
	"github.com/Catofes/go-its/config"
	"net"
	"github.com/op/go-logging"
	Log "github.com/Catofes/go-its/log"
	"sync"
	"strconv"
)

var log *logging.Logger
var Server *MainUdpService
var mainWaitGroup sync.WaitGroup

func init() {
	log = Log.GetInstance()
}

type Handler func(*net.UDPConn, *net.UDPAddr, int, []byte)
type MainUdpService struct {
	ListenAddress string
	ListenPort    int
	buffer        []byte
	mutex         sync.Mutex
	handler       map[byte]Handler
	connection    *net.UDPConn
	isServer      bool
}

func (s *MainUdpService) loadConfig() {
	c := config.GetInstance("")
	s.ListenAddress = c.ListenAddress
	s.ListenPort = int(c.ListenPort)
}

func (s *MainUdpService) Init() *MainUdpService {
	s.loadConfig()
	s.buffer = make([]byte, 1024)
	s.handler = make(map[byte]Handler, 5)
	return s
}

func (s *MainUdpService) Loop() {
	address, err := net.ResolveUDPAddr("udp", s.ListenAddress+":"+strconv.Itoa(s.ListenPort))
	if err != nil {
		log.Fatal("Can't resolve address: ", err)
	}
	connection, err := net.ListenUDP("udp", address)
	s.connection = connection
	if err != nil {
		log.Fatal("Can't listen udp on", address, err)
	}
	defer s.connection.Close()
	for {
		s.handleClient(s.connection)
	}
}

func (s *MainUdpService) handleClient(connection *net.UDPConn) {
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
		log.Warning("Receive unknow package.")
	}
}

func (s *MainUdpService) AddHandler(package_type byte, handler Handler) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.handler[package_type] = handler
}

func (s *MainUdpService) DeleteHandler(package_type byte) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.handler, package_type)
}

func Run(is_server bool) {
	Server = (&MainUdpService{}).Init()
	Server.isServer = is_server
	Server.AddHandler(byte(0), EchoRequestHandler)
	mainWaitGroup.Add(1)
	go Server.Loop()
	(&MainService{}).Init().Loop()
	mainWaitGroup.Wait()
}
