package gateway

import (
	"time"

	"github.com/labstack/echo"
)

var ws *webServer

type webServer struct {
	config
	e       *echo.Echo
	address string
}

func (s *webServer) init() *webServer {
	s.e = echo.New()
	//s.e.Use(middleware.Recover())

	s.e.GET("/", s.getStatus)
	s.e.PUT("/", s.connect)
	return s
}

func (s *webServer) run(udp *udpService) {
	s.e.Logger.Fatal(s.e.Start(s.config.WebServer))
}

func (s *webServer) getStatus(c echo.Context) error {
	response := make(map[string]interface{})
	if its != nil {
		response["check_status"] = its.Status
		response["last_check_time"] = its.LastCheckTime.Format("2006-01-02 15:04:05.999999999 -0700 MST")
		response["last_connect_time"] = its.LastConnectTime.Format("2006-01-02 15:04:05.999999999 -0700 MST")
		response["last_connect_response"] = its.LastText
		response["lost_count"] = its.LostCount
		response["lost_limit"] = its.LostLimit
	}
	type server struct {
		Name       string
		Infos      map[string]serverInfo
		Group      uint64
		Offline    bool
		LinkDown   bool
		LastOnline time.Time
	}
	response["config"] = ss
	d := make(map[string]server)
	for k, v := range ss.servers {
		t := server{
			Name:       k,
			Infos:      v.infos,
			Group:      v.group,
			Offline:    v.offline,
			LinkDown:   v.linkDown,
			LastOnline: v.lastOnline,
		}
		d[k] = t
	}
	response["debug"] = d
	c.JSON(200, response)
	return nil
}

func (s *webServer) connect(c echo.Context) error {
	if s.IsServer {
		its.connect()
	}
	c.HTML(200, "")
	return nil
}
