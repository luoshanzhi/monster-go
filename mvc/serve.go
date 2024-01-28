//go:build !windows

package mvc

import (
	"context"
	"github.com/luoshanzhi/monster-go"
	"github.com/luoshanzhi/monster-go/graceful"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func Serve(servers ...*Server) {
	type Option struct {
		listener net.Listener
		server   *http.Server
		vaild    bool
	}
	optionMap := make(map[string]*Option)
	gracefulStr := commandLineGraceful()
	if gracefulStr != "" {
		addrs := strings.Split(gracefulStr, ",")
		for i, addr := range addrs {
			// 在linux中，值为0、1、2的fd，分别代表标准输入、标准输出、标准错误输出，因为 0 1 2已经被linux使用了
			// 返回具有给定文件描述符和名称的新文件。如果fd不是有效的文件描述符，则返回值为nil。
			// 3是什么？3其实就是从父进程继承过来的socket fd。虽然子进程可以默认继承父进程绝大多数的文件描述符（除了文件锁之类的），但是golang的标准库os/exec只默认继承stdin stdout stderr这三个。
			// 需要让子进程继承的fd需要在fork之前手动放到ExtraFiles中。由于有了stdin 0 stdout 1 stderr 2，因此其它fd的序号从3开始。
			if listener, err := net.FileListener(os.NewFile(3+uintptr(i), "")); err == nil {
				optionMap[strings.TrimSpace(addr)] = &Option{
					listener: listener,
				}
			}
		}
	}
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
		var listener net.Listener
		if option, ok := optionMap[addr]; ok {
			option.server = httpServer
			option.vaild = true
			listener = option.listener
		}
		if listener == nil {
			//多个listener(tcp或文件描述符)监听同一个端口不会同时生效，只有一个失效下一个才自动生效
			listen, err := net.Listen("tcp", addr)
			if err != nil {
				monster.CommonLog.Fatal("mvc("+addr+"): 启动失败", err)
			}
			optionMap[addr] = &Option{
				listener: listen,
				server:   httpServer,
				vaild:    true,
			}
			listener = listen
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
	}
	var files []graceful.File
	var httpServer []*http.Server
	for addr, item := range optionMap {
		//没有用的 listener 就取消
		if !item.vaild {
			item.listener.Close()
			monster.CommonLog.Info("mvc(" + addr + "): 关闭成功")
			continue
		}
		file, fileErr := item.listener.(*net.TCPListener).File()
		if fileErr != nil {
			monster.CommonLog.Fatal(fileErr)
		}
		files = append(files, graceful.File{
			File: file,
			Addr: addr,
		})
		httpServer = append(httpServer, item.server)
	}
	graceful.SignalHandler(files, func(ctx context.Context) {
		for _, server := range httpServer {
			server.Shutdown(ctx)
		}
	})
}
