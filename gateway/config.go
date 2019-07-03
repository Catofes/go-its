package gateway

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type config struct {
	Listen       string
	LocalAddress string
	ConnectTo    []string
	WebServer    string
	Token        uint64
	PingEvery    uint64
	SyncEvery    uint64
	CheckEvery   uint64
	DeleteEvery  uint64
	OfflineTime  uint64
	Account      []interface{}
	ItsURL       string
	Group        string
	GroupFilter  string
	TestMode     bool
	Debug        bool
	IsServer     bool
}

func (s *config) load(path string) *config {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Print(err)
		os.Exit(-1)
	}
	s.Listen = "0.0.0.0:4432"
	s.WebServer = "0.0.0.0:4432"
	s.PingEvery = 500
	s.SyncEvery = 6000
	s.CheckEvery = 12000
	s.DeleteEvery = 600000
	s.OfflineTime = 6000
	s.Group = "0"
	s.GroupFilter = "0"
	err = json.Unmarshal(d, s)
	if err != nil {
		fmt.Print(err)
		os.Exit(-1)
	}
	return s
}
