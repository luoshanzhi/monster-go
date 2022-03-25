package mvc

import (
	"net/http"
	"os"
)

type Server struct {
	Addr         string
	Handler      func(req *http.Request) (Route, error)
	Interceptors []Interceptor
	CertFile     string
	KeyFile      string
	Prepare      func(server *Server, httpServer *http.Server) //允许修改原始 http.Server 信息
}

type File struct {
	File *os.File
	Addr string
}

type Route struct {
	ControllerName string
	MethodName     string
}
