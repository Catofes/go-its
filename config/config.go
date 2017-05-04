package config

import (
	"os"
	"encoding/json"
	"sync"
	"github.com/op/go-logging"
	Log "github.com/Catofes/go-its/log"
)

var log *logging.Logger

type MainConfig struct {
	ListenAddress       string
	ListenPort          int
	CenterServerAddress string
	CenterServerPort    int
}

func (s *MainConfig) Load(file_path string) {
	f, err := os.Open(file_path)
	if err != nil {
		log.Fatal("Load config file failed.", err)
	}
	decoder := json.NewDecoder(f)
	err = decoder.Decode(s)
	if err != nil {
		log.Fatal("Decode config file failed.", err)
	}
}

var instance *MainConfig
var once sync.Once

func GetInstance(path string) *MainConfig {
	if path != "" {
		once.Do(func() {
			instance = &MainConfig{}
			instance.Load(path)
		})
	}
	return instance
}

func init()  {
	log = Log.GetInstance()
}