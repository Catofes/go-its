package main

import (
	"flag"

	"github.com/Catofes/go-its/gateway"
)

func main() {
	configPath := flag.String("conf", "./test.json", "Path to config file.")
	flag.Parse()
	gateway.Run(*configPath)
}
