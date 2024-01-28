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
	str    string   `val:"怪兽"`                                   //也支持 *string
	num    int      `val:"123"`                                  //也支持 *int
	bl     bool     `val:"true"`                                 //也支持 *bool
	numArr []int    `val:"[12,34,56,78,90]"`                     //也支持 *[]int
	strArr []string `val:"[\"aa\",\"bb\",\"cc\",\"dd\",\"ee\"]"` //也支持 *[]string
	ob     struct {
		Age  int
		Name string
		Man  bool
	} `val:"{\"age\":18,\"man\":true,\"name\":\"怪兽\"}"` //也支持 *struct
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
