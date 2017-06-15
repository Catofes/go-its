package web

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"github.com/Catofes/go-its/its"
	"github.com/Catofes/go-its/config"
)

var WebServer *Server

type Server struct {
	app     *iris.Framework
	address string
}

func (s *Server) Init() *Server {
	s.app = iris.New()
	s.app.Adapt(httprouter.New())
	s.address = config.GetInstance("").WebServerAddress
	s.bind()
	return s
}

func (s *Server) Run() {
	s.app.Listen(s.address)
}

func (s *Server) bind() {
	s.app.Get("/", s.get_status)
}

func (s *Server) get_status(ctx *iris.Context) {
	response := make(map[string]interface{})
	response["check_status"] = its.ItsManager.Status
	response["last_check_time"] = its.ItsManager.LastCheckTime.String()
	response["last_connect_time"] = its.ItsManager.LastConnectTime.String()
	response["last_connect_response"] = its.ItsManager.LastText
	response["lost_count"] = its.ItsManager.LostCount
	response["lost_limit"] = its.ItsManager.LostLimit
	ctx.JSON(iris.StatusOK, response)
}
