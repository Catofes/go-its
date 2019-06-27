package gateway

import (
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var ws *webServer

type webServer struct {
	config
	e       *echo.Echo
	address string
}

func (s *webServer) init() *webServer {
	s.e = echo.New()
	s.e.Use(middleware.Recover())
	return s
}

func (s *webServer) run(udp *udpService) {
	s.e.Logger.Fatal(s.e.Start(s.config.WebServer))
}

func (s *webServer) getStatus(c echo.Context) {
	response := make(map[string]interface{})
	response["check_status"] = its.Status
	response["last_check_time"] = its.LastCheckTime.Format("2006-01-02 15:04:05.999999999 -0700 MST")
	response["last_connect_time"] = its.LastConnectTime.Format("2006-01-02 15:04:05.999999999 -0700 MST")
	response["last_connect_response"] = its.LastText
	response["lost_count"] = its.LostCount
	response["lost_limit"] = its.LostLimit
	response["debug"] = ss
	c.JSON(200, response)
}

func (s *webServer) connect(c echo.Context) {
	its.connect()
	c.HTML(200, "")
}
