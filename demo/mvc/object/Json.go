package object

import (
	"github.com/luoshanzhi/monster-go/mvc"
	"net/http"
	"time"
)

type Json struct {
	Code   int         `json:"code"`
	Msg    string      `json:"msg"`
	Data   interface{} `json:"data"`
	Time   int         `json:"time"`
	common *Common
}

func (the *Json) Init() {
	the.Code = -1
	the.Time = int(time.Now().Unix())
}

// 实现 Multiton 方法的模块都是多例，工厂每次取出来都是不同实例
func (the *Json) Multiton() {

}

// 实现 Out 方法的 mvc 框架输出时会调用
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
