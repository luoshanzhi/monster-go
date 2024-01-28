package mvc

import (
	"context"
	"github.com/luoshanzhi/monster-go"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

func Serve(servers ...*Server) {
	var httpServers []*http.Server
	for i, server := range servers {
		addr := strings.TrimSpace(server.Addr)
		handle := server.Handler
		certFile := strings.TrimSpace(server.CertFile)
		keyFile := strings.TrimSpace(server.KeyFile)
		if addr == "" {
			monster.CommonLog.Fatal("第" + strconv.Itoa(i) + "个server还未设置Addr")
		}
		if handle == nil {
			monster.CommonLog.Fatal(addr + " 还未设置Handler")
		}
		var handlerFunc = func(server *Server) http.HandlerFunc {
			return func(w http.ResponseWriter, req *http.Request) {
				routeHandle(server, w, req)
			}
		}
		httpServer := &http.Server{Addr: addr, Handler: handlerFunc(server), ErrorLog: log.New(monster.ErrorLog.Out, "", 0)}
		if server.Prepare != nil {
			server.Prepare(server, httpServer)
		}
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			monster.CommonLog.Fatal("mvc("+addr+"): 启动失败", err)
		}
		if listener != nil {
			go func() {
				monster.CommonLog.Info("mvc(" + addr + "): 启动成功")
				//不要把 server.Serve() 放在主协程里
				if certFile != "" && keyFile != "" {
					httpServer.ServeTLS(listener, certFile, keyFile)
				} else {
					httpServer.Serve(listener)
				}
			}()
		}
		httpServers = append(httpServers, httpServer)
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	for {
		sig := <-ch
		// timeout context for shutdown
		//ctx, _ := context.WithTimeout(context.Background(), 20*time.Second)
		ctx := context.Background()
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			signal.Stop(ch)
			for _, server := range httpServers {
				server.Shutdown(ctx)
			}
			monster.CommonLog.Info("程序退出成功")
			return
		}
	}
}
