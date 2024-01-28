package object

import (
	"fmt"
	"github.com/luoshanzhi/monster-go/mvc"
	"net/http"
	"reflect"
	"regexp"
)

type InterceptorName struct {
}

func (the *InterceptorName) Init() {

}

//throughout是贯穿对象，每个拦截器都有它，目的是拦截器和拦截器之前传值，throughout长度等于所有总拦截器数量,
//throughoutIndex是当前是第几个拦截器0开始，
//throughout[throughoutIndex]=当前拦截器赋值给后续拦截器用，
//throughout[throughoutIndex-1] 取出上层拦截器的传值

// 前置拦截器，请求最外层，可以修改Path等信息原始信息
func (the *InterceptorName) Before(w http.ResponseWriter, req *http.Request, throughout []reflect.Value, throughoutIndex int) mvc.View {
	//把url里的控制器 bigBird 改成 bird
	req.URL.Path = regexp.MustCompile(`(?i)bigBird`).ReplaceAllString(req.URL.Path, "bird")
	//返回 nil 代表拦截器成功通过
	//返回 mvc.View 代表成功被拦截，框架输出返回视图
	return nil
}

// 成功通过前置拦截器，在 调用控制器方法前 调用此方法
// method 是准备调用的 调用控制器方法，此阶段可以用来记录 控制器方法调用 情况,log等等
// in 是准备调用的 调用控制器方法 传入的参数
func (the *InterceptorName) Invoke(w http.ResponseWriter, req *http.Request, method reflect.Value, in []reflect.Value, throughout []reflect.Value, throughoutIndex int) mvc.View {
	fmt.Println("控制器方法被调用")
	return nil
}

// 控制器方法成功调用，拿到控制器方法返回值，调用此方法
// ret 是控制器方法的返回值，此阶段可以用来修改 控制器方法的返回值，比如修改json对象，加密返回值
func (the *InterceptorName) After(w http.ResponseWriter, req *http.Request, ret reflect.Value, throughout []reflect.Value, throughoutIndex int) mvc.View {
	if jsonView, ok := ret.Interface().(*Json); ok {
		jsonView.Msg = "已经成功修改"
	}
	return nil
}
