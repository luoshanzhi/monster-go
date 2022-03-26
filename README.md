# monster

```text
一个简单的go语言web开发框架; 
简单的 工厂模式；
简单的 MVC模式,支持https,请求拦截器,支持热更新,优雅关闭服务器,
简单的 数据库(mysql主从)；
简单的 缓存(redis)
```

> ### monster实战作品, 用微信扫码访问, php和go都是一套前端代码
> php(之前版本)：![飞鲸体育][phpQrcode]　go(monster版本)：![飞鲸体育][goQrcode]

> ### 1. 工厂使用 [查看demo, base.go是入口文件][demoFactory]

```go
package main

import (
	"fmt"
	"github.com/luoshanzhi/monster-go"
	"time"
)

type Bird struct {
	//框架自动注入 Common 工厂实例, 带不带*都可以，私有属性也可以注入
	common *Common
}

func (the *Bird) Fly() {
	fmt.Println("flying: " + the.common.Now())
}

type Dog struct {
	common *Common
}

func (the *Dog) Init() {
	//工厂模块初始化时，Init方法被调用
	fmt.Println("Init Dog")
}

func (the *Dog) Use() {
	//工厂模块每次使用时, Use方法被调用
	fmt.Println("Use Dog")
}

func (the *Dog) Run() {
	fmt.Println("running: " + the.common.Now())
}

type Common struct {
	//下面是各种自动注入值
	str    string   `val:"怪兽"`
	num    int      `val:"123"`
	bl     bool     `val:"true"`
	numArr []int    `val:"[12,34,56,78,90]"`
	strArr []string `val:"[\"aa\",\"bb\",\"cc\",\"dd\",\"ee\"]"`
	ob     struct {
		Age  int
		Name string
		Man  bool
	} `val:"{\"age\":18,\"man\":true,\"name\":\"怪兽\"}"`
}

func (the *Common) Now() string {
	return time.Now().String()
}

func (the *Common) Print() {
	fmt.Println(the.str, the.num, the.bl, the.numArr, the.strArr, the.ob)
}

//object 实现空 Multiton 方法就代表多例，每次工厂取出都是全新的实例

var factoryMap = map[string]interface{}{
	"Bird":   (*Bird)(nil),
	"Dog":    (*Dog)(nil),
	"Common": (*Common)(nil),
}

func main() {
	monster.Init(factoryMap) //初始化工厂
	bird := monster.Factory("Bird").(*Bird)
	dog := monster.Factory("Dog").(*Dog)
	common := monster.Factory("Common").(*Common)
	bird.Fly()
	dog.Run()
	common.Print()
}


输出:
Init Dog
Use Dog
flying: 2022-03-26 16:37:07.403121 +0800 CST m = +0.001406368
running: 2022-03-26 16:37:07.403492 +0800 CST m = +0.001777576
怪兽 123 true [12 34 56 78 90] [aa bb cc dd ee] {18 怪兽 true}
```

> ### 2. MVC使用 [查看demo, base.go是入口文件][demoMvc]

```text
支持https,优雅关闭服务器,即使热更新时存在端口不同也会优雅关闭服务
优雅关闭服务器: kill pid
重启(支持热更新): kill -USR2 pid
```

```go
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

//测试连接:
//http://127.0.0.1:9022/bird/fly
//http://127.0.0.1:9022/dog/run?str=a&num=1&bl=true&strArr1=bbb&strArr1=ccc&numArr1=333&numArr1=444&StrArr2=eee&StrArr2=fff&StrArr2=hhh&numArr2=123&&numArr2=456
//http://127.0.0.1:9022/bird/upload 单个上传图片, 参数file
//http://127.0.0.1:9022/bird/uploads 多个上传图片, 参数files
//http://127.0.0.1:9021/bigBird/fly 只有9021端口可以调用
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
```

[demoFactory]: https://github.com/luoshanzhi/monster-go/tree/main/demo/factory

[demoMvc]: https://github.com/luoshanzhi/monster-go/tree/main/demo/mvc

[phpQrcode]: https://oss.tranhn.com/phpQrcode.png

[goQrcode]: https://oss.tranhn.com/goQrcode.png