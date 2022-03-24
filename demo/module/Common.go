package module

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"
)

type Common struct {
	str    string   `val:"怪兽"`
	num    int      `val:"123"`
	numArr []int    `val:"[12,34,56,78,90]"`
	strArr []string `val:"[\"aa\",\"bb\",\"cc\",\"dd\",\"ee\"]"`
	ob     struct {
		Age  int
		Name string
	} `val:"{\"age\":18,\"name\":\"怪兽\"}"`
}

func (the *Common) Print() {
	fmt.Println(the.str, the.num, the.numArr, the.strArr, the.ob)
}

func (the *Common) Init() {
	fmt.Println("Common.Init: 创建工厂模块才调用")
}

func (the *Common) Use() {
	fmt.Println("Common.Use: 每次引用工厂模块都调用")
}

func (the *Common) Md5(str string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.TrimSpace(str))))
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
