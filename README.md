# monster

    一个简单的go语言web开发框架; 
    简单的 工厂模式；
    简单的 MVC模式,支持https,支持热更新,优雅关闭服务器,
    简单的 数据库(mysql主从)；
    简单的 缓存(redis)

1. 工厂使用

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

//struct 实现空 Multiton 方法就代表多例，每次工厂取出都是全新的实例

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
flying: 2022-03-25 16:25:45.588542 +0800 CST m = +0.000633474
running: 2022-03-25 16:25:45.588668 +0800 CST m =+0.000759914
怪兽 123 true [12 34 56 78 90] [aa bb cc dd ee] {18 怪兽 true}

```

2. MVC使用

        支持https,优雅关闭服务器,即使热更新时存在端口不同也会优雅关闭服务
        优雅关闭服务器: kill pid
        重启(支持热更新): kill -USR2 pid

```go
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/luoshanzhi/monster-go"
	"github.com/luoshanzhi/monster-go/mvc"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Bird struct {
	//框架自动注入 Common 工厂实例, 带不带*都可以，私有属性也可以注入
	common *Common
}

func (the *Bird) Fly(req *http.Request) *Json {
	jsonView := monster.Factory("Json").(*Json)
	data := "flying(" + req.Host + ")" + the.common.Now()
	jsonView.Data = data
	jsonView.Code = 0
	jsonView.Msg = "成功"
	return jsonView
}

type Dog struct {
	common *Common
}

func (the *Dog) Run(req *http.Request) *Json {
	fmt.Println("running: " + the.common.Now())
	jsonView := monster.Factory("Json").(*Json)
	data := "running(" + req.Host + ")" + the.common.Now()
	jsonView.Data = data
	jsonView.Code = 0
	jsonView.Msg = "成功"
	return jsonView
}

type Common struct {
}

func (the *Common) Now() string {
	return time.Now().String()
}

func (the *Common) JsonEncode(data interface{}) (string, error) {
	byteBuf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(byteBuf)
	encoder.SetEscapeHTML(false) //不转译html字符
	err := encoder.Encode(data)
	if err != nil {
		return "", err
	}
	return byteBuf.String(), nil
}

func (the *Common) JsonDecode(data string, ob interface{}) error {
	data = strings.TrimSpace(data)
	return json.Unmarshal([]byte(data), ob)
}

type Json struct {
	Code int `json:"code"`
	//描述
	Msg string `json:"msg"`
	//返回数据
	Data interface{} `json:"data"`
	//当前时间戳
	Time   int `json:"time"`
	common *Common
}

func (the *Json) Init() {
	the.Code = -1
	the.Time = int(time.Now().Unix())
}

//实现 Multiton 方法的模块都是多例，工厂每次取出来都是不同实例
func (the *Json) Multiton() {

}

//实现 Out 方法的 mvc 框架输出时会调用
func (the *Json) Out(w http.ResponseWriter, req *http.Request) error {
	status := http.StatusOK
	header := map[string]string{
		"Content-Type": "application/json; charset=utf-8",
	}
	the.Code = 0
	content, err := the.common.JsonEncode(the)
	if err != nil {
		return err
	}
	mvc.ResponseOut(w, status, header, content)
	return nil
}

var factoryMap = map[string]interface{}{
	"Bird":   (*Bird)(nil),
	"Dog":    (*Dog)(nil),
	"Common": (*Common)(nil),
	"Json":   (*Json)(nil),
}

func handler(req *http.Request) (mvc.Route, error) {
	var route mvc.Route
	path := req.URL.Path
	pathReg := regexp.MustCompile(`^/(\w+)/(\w+)`)
	pathRes := pathReg.FindStringSubmatch(path)
	if len(pathRes) != 3 {
		return route, errors.New("错误的路由")
	}
	route.ControllerName = monster.FirstUpper(pathRes[1])
	route.MethodName = monster.FirstUpper(pathRes[2])
	return route, nil
}

func prepare(server *mvc.Server, httpServer *http.Server) {
	//这里可以更改 http.Server 实例信息
	fmt.Println("prepare(" + server.Addr + ")")
}

func main() {
	monster.Init(factoryMap) //初始化工厂
	mvc.Serve(
		&mvc.Server{Addr: ":9020", Handler: handler, Prepare: prepare},
		&mvc.Server{Addr: ":9021", Handler: handler, Prepare: prepare},
		&mvc.Server{Addr: ":9022", Handler: handler, Prepare: prepare},
		//CertFile 和 KeyFile 同时不为空就是 https
		&mvc.Server{Addr: ":9023", Handler: handler, Prepare: prepare, CertFile: "server.crt", KeyFile: "server.key"},
	)
}


```
