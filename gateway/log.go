package gateway

import (
	"os"

	"github.com/op/go-logging"
)

var log *logging.Logger

func logInit(debug bool) {
	log = logging.MustGetLogger("example")
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	format := logging.MustStringFormatter(
		`%{color}%{time:0102 15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	backendLeveled := logging.AddModuleLevel(backendFormatter)
	if debug {
		backendLeveled.SetLevel(logging.DEBUG, "")
	} else {
		backendLeveled.SetLevel(logging.WARNING, "")
	}
	logging.SetBackend(backendLeveled)
	
}
