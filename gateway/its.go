package gateway

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"math"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
)

var its *itsManager

type accountInfo struct {
	config
	AccountName     string
	AccountPassword string
	ConnectLimit    bool
	mutex           sync.Mutex
}

func (s *accountInfo) Connect() (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	resp, err := http.PostForm(s.config.ItsURL, url.Values{
		"uid":       {s.AccountName},
		"password":  {s.AccountPassword},
		"range":     {"1"},
		"operation": {"connect"},
		"timeout":   {"1"}})
	if err != nil {
		log.Warning("Request connection %s failed. Err: %s.", s.AccountName, err.Error())
		return "", errors.New("requset connection failed")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	decoder := simplifiedchinese.GBK.NewDecoder()
	data := make([]byte, len(body)*2)
	decoder.Transform(data, body, false)
	str := string(data)
	if strings.Contains(str, "当前连接数超过预定值") {
		log.Warning("%s connection over limit.", s.AccountName)
		s.Disconnect()
		return "", errors.New("connection over limit")
	}
	if strings.Contains(str, "今天不能再使用客户端") {
		log.Warning("%s api limit reach.", s.AccountName)
		s.ConnectLimit = true
		return "", errors.New("api limit")
	}
	log.Debug("Connect %s Sent.", s.AccountName)
	return str, nil
}

func (s *accountInfo) Disconnect() error {
	_, err := http.PostForm(s.config.ItsURL, url.Values{
		"uid":       {s.AccountName},
		"password":  {s.AccountPassword},
		"range":     {"4"},
		"operation": {"disconnectall"},
		"timeout":   {"1"}})
	if err != nil {
		log.Warning("Request disconnection %s failed.", s.AccountName)
		return errors.New("requset disconnect failed")
	}
	log.Warning("Disconnect %s sent.", s.AccountName)
	return nil
}

type itsManager struct {
	config
	Accounts        []*accountInfo
	Status          bool
	LastText        string
	LastConnectTime time.Time
	LastCheckTime   time.Time
	LostCount       int
	LostLimit       int
	Day             int
	mutex           sync.Mutex
}

func (s *itsManager) init() *itsManager {
	s.Accounts = make([]*accountInfo, 0)
	for _, v := range s.config.Account {
		a := v.(map[string]interface{})
		u := a["Username"].(string)
		p := a["Password"].(string)
		s.Accounts = append(s.Accounts, &accountInfo{config: s.config, AccountName: u, AccountPassword: p})
	}
	s.LostLimit = 1
	return s
}

func (s *itsManager) linkDown() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	log.Warning("LinkDown. Connection info: count/limit %d/%d.", s.LostCount, s.LostLimit)
	s.LostCount++
	s.Status = false
	if s.LostCount > s.LostLimit {
		s.LostCount = 0
		if s.LostLimit > 64 {
			s.LostLimit = int(math.Floor(math.Sqrt(256 * float64(s.LostLimit))))
		} else {
			s.LostLimit *= 2
		}
		if s.LostLimit < 1 {
			s.LostLimit = 1
		}
		if s.LostLimit >= 256 {
			s.LostLimit = 256
		}
		s.connect()
	}
}

func (s *itsManager) linkUp() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.LostCount = 0
	s.LostLimit = s.LostLimit/2 + 1
	s.Status = true
	s.LastCheckTime = time.Now()
}

func (s *itsManager) connect() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var account *accountInfo
	for _, v := range s.Accounts {
		if !v.ConnectLimit {
			account = v
			break
		}
	}
	if account != nil {
		if s.TestMode {
			return
		}
		str, err := account.Connect()
		if err == nil {
			s.LastConnectTime = time.Now()
			s.LastText = str
		}
	} else {
		log.Warning("all account disable")
	}
}

func (s *itsManager) loop() {
	for {
		time.Sleep(1 * time.Hour)
		s.mutex.Lock()
		now := time.Now()
		if now.Day() != s.Day {
			s.Day = now.Day()
			for _, v := range s.Accounts {
				v.ConnectLimit = false
			}
		}
		s.mutex.Unlock()
	}
}
