package main

import (
	"fmt"
	"github.com/luoshanzhi/monster-go/core"
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

var factoryMap = map[string]interface{}{
	"Bird":   (*Bird)(nil),
	"Dog":    (*Dog)(nil),
	"Common": (*Common)(nil),
}

func main() {
	core.Init(factoryMap) //初始化工厂
	bird := core.Factory("Bird").(*Bird)
	dog := core.Factory("Dog").(*Dog)
	common := core.Factory("Common").(*Common)
	bird.Fly()
	dog.Run()
	common.Print()
}
