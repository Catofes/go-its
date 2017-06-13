package main

import (
	"github.com/Catofes/go-its/config"
	"github.com/Catofes/go-its/udp"
	"flag"
)

func init() {
	configPath := flag.String("conf", "./test.json", "Path to config file.")
	flag.Parse()
	config.GetInstance(*configPath)
}

func main() {
	udp.Run(true)
}
