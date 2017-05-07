package main

import (
	"github.com/Catofes/go-its/udp"
	"github.com/Catofes/go-its/config"
	"flag"
)

func init() {
	configPath := flag.String("conf", "./client.json", "Path to config file.")
	flag.Parse()
	config.GetInstance(*configPath)
}

func main() {
	udp.Run(false)
}
