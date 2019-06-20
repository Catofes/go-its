package udp

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/adaptors/httprouter"
	"github.com/Catofes/go-its/its"
	"github.com/Catofes/go-its/config"
)

type WebServer struct {
	app     *iris.Framework
	address string
}

func (s *WebServer) Init() *WebServer {
	s.app = iris.New()
	s.app.Adapt(httprouter.New())
	s.address = config.GetInstance("").WebServerAddress
	s.bind()
	return s
}

func (s *WebServer) Run() {
	s.app.Listen(s.address)
}

func (s *WebServer) bind() {
	s.app.Get("/", s.get_status)
	s.app.Post("/", s.connect)
}

func (s *WebServer) get_status(ctx *iris.Context) {
	response := make(map[string]interface{})
	response["check_status"] = its.ItsManager.Status
	response["last_check_time"] = its.ItsManager.LastCheckTime.Format("2006-01-02 15:04:05.999999999 -0700 MST")
	response["last_connect_time"] = its.ItsManager.LastConnectTime.Format("2006-01-02 15:04:05.999999999 -0700 MST")
	response["last_connect_response"] = its.ItsManager.LastText
	response["lost_count"] = its.ItsManager.LostCount
	response["lost_limit"] = its.ItsManager.LostLimit
	response["debug"] = service.Servers
	ctx.JSON(iris.StatusOK, response)
}

func (s *WebServer) connect(ctx *iris.Context) {
	its.ItsManager.Connect()
	ctx.SetStatusCode(200)
}