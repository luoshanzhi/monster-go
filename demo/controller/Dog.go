package controller

import (
	"fmt"
	"github.com/luoshanzhi/monster-go/core"
	"github.com/luoshanzhi/monster-go/demo/module"
	"github.com/luoshanzhi/monster-go/demo/object/param"
	"github.com/luoshanzhi/monster-go/demo/view"
	"strconv"
	"strings"
	"time"
)

type Dog struct {
	//框架自动注入 module.Common 模块, 带不带*都可以，私有属性也可以注入
	common *module.Common
}

func (the *Dog) Init() {
	fmt.Println("Dog.Init: 创建工厂模块才调用")
}

func (the *Dog) Use() {
	fmt.Println("Dog.Use: 每次引用工厂模块都调用")
}

func (the *Dog) Run(param param.RunParam) *view.Json {
	jsonView := core.Factory("view/Json").(*view.Json)
	name := strings.TrimSpace(param.Name)
	data := "正在跑" + the.common.Md5(strconv.Itoa(int(time.Now().Unix())))
	if name != "" {
		data = `"` + name + `"` + data
	}
	jsonView.Data = data
	jsonView.Code = 0
	jsonView.Msg = "成功"
	return jsonView
}
