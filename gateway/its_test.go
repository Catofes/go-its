package gateway

import (
	"testing"
)

func TestAccountInfo_Connect(t *testing.T) {
	c:= (&config{}).load("./test.json")
	account:= accountInfo{config:*c,AccountName:"111111",AccountPassword:"111111"}
	account.Connect()
}
