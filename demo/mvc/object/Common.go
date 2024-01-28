package object

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"
)

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
