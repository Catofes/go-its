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
	ListenPort          uint16
	CenterServerAddress string
	CenterServerPort    uint16
	WebServerAddress    string
	Token               uint64
	PingEvery           uint64
	SyncEvery           uint64
	CheckEvery          uint64
	DeleteEvery         uint64
	OfflineTime         uint64
	Account             []interface{}
	ItsUrl              string
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
	if s.PingEvery <= 0 {
		s.PingEvery = 1000
	}
	if s.SyncEvery <= 0 {
		s.SyncEvery = 6000
	}
	if s.CheckEvery <= 0 {
		s.CheckEvery = 6000
	}
	if s.OfflineTime <= 0 {
		s.OfflineTime = 15000
	}
	if s.DeleteEvery <= s.OfflineTime {
		s.DeleteEvery = 7 * 24 * 3600 * 1000
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

func init() {
	log = Log.GetInstance()
}
