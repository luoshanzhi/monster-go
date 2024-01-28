package object

import (
	"errors"
	"github.com/luoshanzhi/monster-go"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strings"
)

type Bird struct {
	//框架自动注入 Common 工厂实例, 带不带*都可以，私有属性也可以注入
	common *Common
}

// 框架会直接注入这些参数 w http.ResponseWriter, req *http.Request
// 参数顺序不限制, 也可以少写或者不写参数
// 可以写出下面几种格式:
// func (the *Bird) Fly(req *http.Request, w http.ResponseWriter) *Json {
// func (the *Bird) Fly(req *http.Request, _ mvc.GET) *Json { //只支持GET方式
// func (the *Bird) Fly(req *http.Request, _ mvc.POST) *Json { //只支持POST方式
// func (the *Bird) Fly(req *http.Request, _ mvc.GET, _ mvc.POST) *Json { //只支持GET+POST方式
// 接受所有Method: OPTIONS,GET,HEAD,POST,PUT,DELETE,TRACE,CONNECT
func (the *Bird) Fly(req *http.Request) *Json { //不指定请求Method，默认接受所有Method方式参数*
	jsonView := monster.Factory("Json").(*Json)
	data := "flying(" + req.Host + ")" + the.common.Now()
	jsonView.Data = data
	jsonView.Code = 0
	jsonView.Msg = "成功"
	return jsonView
}

// 上传单个文件
func (the *Bird) Upload(param UploadFile) *Json {
	jsonView := monster.Factory("Json").(*Json)
	file := param.File
	if file == nil {
		jsonView.Msg = "上传文件不能为空"
		return jsonView
	}
	err := the.saveUploadFile(file)
	if err != nil {
		jsonView.Msg = "上传文件异常"
		return jsonView
	}
	jsonView.Code = 0
	jsonView.Msg = "成功"
	return jsonView
}

// 上传多个文件
func (the *Bird) Uploads(param UploadFiles) *Json {
	jsonView := monster.Factory("Json").(*Json)
	files := param.Files
	if len(files) == 0 {
		jsonView.Msg = "上传文件不能为空"
		return jsonView
	}
	for _, file := range files {
		err := the.saveUploadFile(file)
		if err != nil {
			jsonView.Msg = "上传文件异常"
			return jsonView
		}
	}
	jsonView.Code = 0
	jsonView.Msg = "成功"
	return jsonView
}

func (the *Bird) saveUploadFile(fileHeader *multipart.FileHeader) error {
	fileType := strings.ToLower(strings.TrimSpace(fileHeader.Header.Get("Content-Type")))
	name := strings.TrimSpace(fileHeader.Filename)
	size := int(fileHeader.Size)
	maxSize := 2 * 1024 * 1024 //2MB
	if name == "" || size <= 0 || size > maxSize || (fileType != "image/pjpeg" && fileType != "image/jpeg" && fileType != "image/png" && fileType != "image/x-png" && fileType != "image/gif") {
		return errors.New("错误的文件")
	}
	fileName := path.Base(name)
	destDir := "./demo/mvc/upload/"
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		if err := os.Mkdir(destDir, os.ModePerm); err != nil {
			return err
		}
	}
	file, fileErr := fileHeader.Open()
	if fileErr != nil {
		return fileErr
	}
	defer file.Close()
	newFile, newFileErr := os.Create(destDir + fileName)
	if newFileErr != nil {
		return newFileErr
	}
	defer newFile.Close()
	// 复制文件到目标目录
	_, copyErr := io.Copy(newFile, file)
	if copyErr != nil {
		return copyErr
	}
	return nil
}
