package its

import (
	"github.com/emirpasic/gods/sets"
)

type AccountManager struct {
	AccountInfo sets.Set
}

type AccountInfo struct {
	accountName     string
	accountPassword string
	connectionCount int
}
