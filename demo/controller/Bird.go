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

type Bird struct {
	//框架自动注入 module.Common 模块, 带不带*都可以，私有属性也可以注入
	common *module.Common
}

func (the *Bird) Init() {
	fmt.Println("Bird.Init: 创建工厂模块才调用")
}

func (the *Bird) Use() {
	fmt.Println("Bird.Use: 每次引用工厂模块都调用")
}

func (the *Bird) Fly(param param.FlyParam) *view.Json {
	jsonView := core.Factory("view/Json").(*view.Json)
	name := strings.TrimSpace(param.Name)
	data := "正在飞" + the.common.Md5(strconv.Itoa(int(time.Now().Unix())))
	if name != "" {
		data = `"` + name + `"` + data
	}
	jsonView.Data = data
	jsonView.Code = 0
	jsonView.Msg = "成功"
	return jsonView
}
