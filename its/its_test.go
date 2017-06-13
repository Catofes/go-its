package its

import (
	"testing"
	"github.com/Catofes/go-its/config"
)

func TestAccountInfo_Connect(t *testing.T) {
	config.GetInstance("./test.json")
	account := AccountInfo{}
	account.Init("111111", "111111")
	//account.Connect()
}

func TestManager_Init(t *testing.T) {
	config.GetInstance("./test.json")
	m := (&Manager{}).Init()
	log.Debug("%v", m.Accounts)
}
