package object

import (
	"fmt"
	"github.com/luoshanzhi/monster-go"
	"github.com/luoshanzhi/monster-go/mvc"
	"net/http"
)

type Dog struct {
	common *Common
}

// 传一个普通结构(非指针，非*http.Request和http.ResponseWriter), 框架会按参数名 首字母大写 注入结构里面
// 参考url: http://127.0.0.1:9022/dog/run?str=a&num=1&bl=true&strArr1=bbb&strArr1=ccc&numArr1=333&numArr1=444&StrArr2=eee&StrArr2=fff&StrArr2=hhh&numArr2=123&&numArr2=456
func (the *Dog) Run(req *http.Request, param Param) *Json { //也可以传多个一样的param(非指针),效果是一样的，也可以把参数分开放在两个不同的param里
	fmt.Println("running: " + the.common.Now())
	jsonView := monster.Factory("Json").(*Json)
	data := "running(" + req.Host + ")" + the.common.Now()
	fmt.Println(param.Str, param.Num, param.Bl, param.StrArr1, param.NumArr1, param.StrArr2, param.NumArr2)
	jsonView.Data = data
	jsonView.Code = 0
	jsonView.Msg = "成功"
	return jsonView
}

// json太长，只支持POST传json 同时 Content-Type: application/json , 下面是参考:
// url: http://127.0.0.1:9022/dog/json
// json: {"str":"a","num":1,"bl":true,"strArr1":["bbb","ccc"],"numArr1":[111,222],"StrArr2":["eee","hhh"],"numArr2":[123,456]}
func (the *Dog) Json(req *http.Request, param Param, _ mvc.POST) *Json {
	fmt.Println("running: " + the.common.Now())
	jsonView := monster.Factory("Json").(*Json)
	data := "running(" + req.Host + ")" + the.common.Now()
	fmt.Println(param.Str, param.Num, param.Bl, param.StrArr1, param.NumArr1, param.StrArr2, param.NumArr2)
	jsonView.Data = data
	jsonView.Code = 0
	jsonView.Msg = "成功"
	return jsonView
}
