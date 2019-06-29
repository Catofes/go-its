package gateway

import "net"

var us *udpService

type handler func(*net.UDPConn, *net.UDPAddr, int, []byte)

type service interface {
	run(*udpService)
}

type udpService struct {
	config
	handlers map[byte]handler
	services map[string]service
	conn     *net.UDPConn
}

func (s *udpService) init() *udpService {
	s.handlers = make(map[byte]handler)
	s.services = make(map[string]service)
	address, err := net.ResolveUDPAddr("udp", s.Listen)
	if err != nil {
		log.Fatal("Can't resolve address: ", err)
	}
	s.conn, err = net.ListenUDP("udp", address)
	if err != nil {
		log.Fatal("Can't listen udp on ", address, err)
	}
	return s
}

func (s *udpService) run() {
	for _, v := range s.services {
		go v.run(s)
	}
	for {
		buffer := make([]byte, 1500)
		n, remoteAddress, err := s.conn.ReadFromUDP(buffer)
		if err != nil {
			log.Warning("Error read connection. %s", err.Error())
			continue
		}
		//log.Debug("Get connection from %s, size %d.", remoteAddress.String(), n)
		if n <= 1 {
			log.Warning("Error package length from %s.", remoteAddress.String())
			continue
		}
		packageType := buffer[0]
		if h, ok := s.handlers[packageType]; ok {
			h(s.conn, remoteAddress, n, buffer[:n])
		} else {
			log.Warning("Error package type &d from &s.", packageType, remoteAddress.String())
			continue
		}
	}
}

func (s *udpService) addHandler(packageType byte, h handler) {
	s.handlers[packageType] = h
}

func (s *udpService) deleteHandler(packageType byte) {
	if _, ok := s.handlers[packageType]; ok {
		delete(s.handlers, packageType)
	}
}

func (s *udpService) addService(name string, sv service) {
	s.services[name] = sv
}

func (s *udpService) deleteService(name string) {
	if _, ok := s.services[name]; ok {
		delete(s.services, name)
	}
}
