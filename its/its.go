package its

import (
	"time"
	"sync"
	"net/http"
	"net/url"
	"io/ioutil"
	"errors"
	Log "github.com/Catofes/go-its/log"
	"github.com/op/go-logging"
	"golang.org/x/text/encoding/simplifiedchinese"
	"strings"
	"github.com/Catofes/go-its/config"
	"github.com/emirpasic/gods/lists/arraylist"
	"math"
)

var log *logging.Logger
var ItsManager *Manager

func init() {
	log = Log.GetInstance()
}

type AccountInfo struct {
	AccountName     string
	AccountPassword string
	ConnectLimit    bool
	mutex           sync.Mutex
}

func (s *AccountInfo) Init(name string, password string) *AccountInfo {
	s.AccountName = name
	s.AccountPassword = password
	return s
}

func (s *AccountInfo) Connect() (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	resp, err := http.PostForm(config.GetInstance("").ItsUrl, url.Values{
		"uid":       {s.AccountName},
		"password":  {s.AccountPassword},
		"range":     {"1"},
		"operation": {"connect"},
		"timeout":   {"1"}})
	if err != nil {
		log.Warning("Request connection %s failed.", s.AccountName)
		return "", errors.New("Requset Connection Failed.")
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
		return "", errors.New("Connection over limit.")
	}
	if strings.Contains(str, "今天不能再使用客户端") {
		log.Warning("%s api limit reach.", s.AccountName)
		s.ConnectLimit = true
		return "", errors.New("Api limit.")
	}
	log.Debug("Connect %s Sent.", s.AccountName)
	return str, nil
}

func (s *AccountInfo) Disconnect() error {
	_, err := http.PostForm(config.GetInstance("").ItsUrl, url.Values{
		"uid":       {s.AccountName},
		"password":  {s.AccountPassword},
		"range":     {"4"},
		"operation": {"disconnectall"},
		"timeout":   {"1"}})
	if err != nil {
		log.Warning("Request disconnection %s failed.", s.AccountName)
		return errors.New("Requset Disconnect Failed.")
	}
	log.Warning("Disconnect %s sent.", s.AccountName)
	return nil
}

type Manager struct {
	Accounts        *arraylist.List
	Status          bool
	LastText        string
	LastConnectTime time.Time
	LastCheckTime   time.Time
	LostCount       int
	LostLimit       int
	Day             int
	mutex           sync.Mutex
}

func (s *Manager) Init() *Manager {
	c := config.GetInstance("")
	s.Accounts = arraylist.New()
	for _, v := range c.Account {
		a := v.(map[string]interface{})
		u := a["Username"].(string)
		p := a["Password"].(string)
		s.Accounts.Add((&AccountInfo{}).Init(u, p))
	}
	s.LostLimit = 1
	ItsManager = s
	return s
}

func (s *Manager) LinkDown() {
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
		s.Connect()
	}
}

func (s *Manager) LinkUp() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.LostCount = 0
	s.LostLimit = s.LostLimit/2 + 1
	s.Status = true
	s.LastCheckTime = time.Now()
}

func (s *Manager) Connect() {
	var account *AccountInfo = nil
	for i := 0; i < s.Accounts.Size(); i++ {
		v, _ := s.Accounts.Get(i)
		if !v.(*AccountInfo).ConnectLimit {
			account = v.(*AccountInfo)
		}
	}
	if account != nil {
		_, err := account.Connect()
		if err == nil {
			s.LastConnectTime = time.Now()
		}
	}
}

func (s *Manager) Loop() {
	for {
		time.Sleep(1 * time.Hour)
		s.mutex.Lock()
		now := time.Now()
		if now.Day() != s.Day {
			s.Day = now.Day()
			for i := 0; i < s.Accounts.Size(); i++ {
				v, _ := s.Accounts.Get(i)
				v.(*AccountInfo).ConnectLimit = false
			}
		}
		s.mutex.Unlock()
	}
}
