package main

import (
	"errors"
	"fmt"
	"github.com/luoshanzhi/monster-go"
	"github.com/luoshanzhi/monster-go/demo/mvc/object"
	"github.com/luoshanzhi/monster-go/mvc"
	"net/http"
	"regexp"
)

var factoryMap = map[string]interface{}{
	"Bird":            (*object.Bird)(nil),
	"Dog":             (*object.Dog)(nil),
	"Common":          (*object.Common)(nil),
	"Json":            (*object.Json)(nil),
	"InterceptorName": (*object.InterceptorName)(nil),
}

func handler(req *http.Request) (mvc.Route, error) {
	var route mvc.Route
	path := req.URL.Path
	pathReg := regexp.MustCompile(`^/(\w+)/(\w+)`)
	pathRes := pathReg.FindStringSubmatch(path)
	if len(pathRes) != 3 {
		return route, errors.New("错误的路由")
	}
	// 通过url解析，到相关的控制器模块，控制器模块要先在工厂注册
	route.ControllerName = monster.FirstUpper(pathRes[1])
	//控制器模块对应的方法
	route.MethodName = monster.FirstUpper(pathRes[2])
	return route, nil
}

func prepare(server *mvc.Server, httpServer *http.Server) {
	//这里可以更改 http.Server 实例信息
	fmt.Println("prepare(" + server.Addr + ")")
}

// 测试连接:
// http://127.0.0.1:9022/bird/fly
// http://127.0.0.1:9022/dog/run?str=a&num=1&bl=true&strArr1=bbb&strArr1=ccc&numArr1=333&numArr1=444&StrArr2=eee&StrArr2=fff&StrArr2=hhh&numArr2=123&&numArr2=456
// http://127.0.0.1:9022/bird/upload 单个上传图片, 参数file
// http://127.0.0.1:9022/bird/uploads 多个上传图片, 参数files
// http://127.0.0.1:9021/bigBird/fly 只有9021端口可以调用
func main() {
	monster.Init(factoryMap) //初始化工厂
	//9021端口添加一个拦截器
	var interceptors9021 = []mvc.Interceptor{
		monster.Factory("InterceptorName").(mvc.Interceptor),
	}
	mvc.Serve(
		&mvc.Server{Addr: ":9020", Handler: handler, Prepare: prepare},
		&mvc.Server{Addr: ":9021", Handler: handler, Prepare: prepare, Interceptors: interceptors9021},
		&mvc.Server{Addr: ":9022", Handler: handler, Prepare: prepare},
		//CertFile 和 KeyFile 同时不为空就是 https
		&mvc.Server{Addr: ":9023", Handler: handler, Prepare: prepare, CertFile: "server.crt", KeyFile: "server.key"},
	)
}
