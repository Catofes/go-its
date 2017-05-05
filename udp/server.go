package udp

import (
	"github.com/Catofes/go-its/config"
	"net"
	"github.com/op/go-logging"
	Log "github.com/Catofes/go-its/log"
	"sync"
)

var log *logging.Logger
var Server *MainServer
var MainWaitGroup sync.WaitGroup

func init() {
	log = Log.GetInstance()
}

type Handler func(*net.UDPConn, *net.UDPAddr, int, []byte)
type MainServer struct {
	ListenAddress string
	ListenPort    int
	buffer        []byte
	mutex         sync.Mutex
	handler       map[byte]Handler
	connection    *net.UDPConn
}

func (s *MainServer) loadConfig() {
	c := config.GetInstance("")
	s.ListenAddress = c.ListenAddress
	s.ListenPort = c.ListenPort
}

func (s *MainServer) Init() {
	s.loadConfig()
	s.buffer = make([]byte, 1024)
	s.handler = make(map[byte]Handler, 5)
}

func (s *MainServer) Loop() {
	address, err := net.ResolveUDPAddr("udp", s.ListenAddress+":"+string(s.ListenPort))
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

func (s *MainServer) handleClient(connection *net.UDPConn) {
	n, remoteAddress, err := connection.ReadFromUDP(s.buffer)
	if err != nil {
		log.Warning("Error read connection. %s", err.Error())
		return
	}
	log.Debug("Get connection from %s, size %d.", remoteAddress.String(), n)
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

func (s *MainServer) AddHandler(package_type byte, handler Handler) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.handler[package_type] = handler
}

func (s *MainServer) DeleteHandler(package_type byte) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.handler, package_type)
}

func Run() {
	Server = &MainServer{}
	Server.Init()
	Server.AddHandler(byte(0), EchoRequestHandler)
	MainWaitGroup.Add(1)
	go Server.Loop()
}
