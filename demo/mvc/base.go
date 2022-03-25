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
