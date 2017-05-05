package main

import (
	"github.com/Catofes/go-its/config"
	"github.com/Catofes/go-its/udp"
	"flag"
)

func init() {
	configPath := flag.String("conf", "./client.json", "Path to config file.")
	flag.Parse()
	config.GetInstance(*configPath)
}

func main() {
	c := config.GetInstance("")
	udp.Run()
	ping := udp.Ping{}
	ping.Init()
	center := udp.RemoteServer{}
	center.Address = c.CenterServerAddress
	center.Port = c.CenterServerPort
	ping.Servers[c.CenterServerAddress] = center
	ping.Loop()
	udp.MainWaitGroup.Wait()
}
