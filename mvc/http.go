package mvc

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/luoshanzhi/monster-go"
	"github.com/luoshanzhi/monster-go/graceful"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func Serve(servers ...*Server) {
	type Option struct {
		listener net.Listener
		server   *http.Server
		vaild    bool
	}
	optionMap := make(map[string]*Option)
	gracefulStr := strings.TrimSpace(monster.Args.Graceful)
	if gracefulStr != "" {
		addrs := strings.Split(gracefulStr, ",")
		for i, addr := range addrs {
			// 在linux中，值为0、1、2的fd，分别代表标准输入、标准输出、标准错误输出，因为 0 1 2已经被linux使用了
			// 返回具有给定文件描述符和名称的新文件。如果fd不是有效的文件描述符，则返回值为nil。
			// 3是什么？3其实就是从父进程继承过来的socket fd。虽然子进程可以默认继承父进程绝大多数的文件描述符（除了文件锁之类的），但是golang的标准库os/exec只默认继承stdin stdout stderr这三个。
			// 需要让子进程继承的fd需要在fork之前手动放到ExtraFiles中。由于有了stdin 0 stdout 1 stderr 2，因此其它fd的序号从3开始。
			if listener, err := net.FileListener(os.NewFile(3+uintptr(i), "")); err == nil {
				optionMap[strings.TrimSpace(addr)] = &Option{
					listener: listener,
				}
			}
		}
	}
	for i, server := range servers {
		addr := strings.TrimSpace(server.Addr)
		handle := server.Handler
		certFile := strings.TrimSpace(server.CertFile)
		keyFile := strings.TrimSpace(server.KeyFile)
		if addr == "" {
			monster.CommonLog.Fatal("第" + strconv.Itoa(i) + "个server还未设置Addr")
		}
		if handle == nil {
			monster.CommonLog.Fatal(addr + " 还未设置Handler")
		}
		var handlerFunc http.HandlerFunc = func(w http.ResponseWriter, req *http.Request) {
			routeHandle(server, w, req)
		}
		httpServer := &http.Server{Addr: addr, Handler: handlerFunc}
		if server.Prepare != nil {
			server.Prepare(server, httpServer)
		}
		var listener net.Listener
		if option, ok := optionMap[addr]; ok {
			option.server = httpServer
			option.vaild = true
			listener = option.listener
		}
		if listener == nil {
			//多个listener(tcp或文件描述符)监听同一个端口不会同时生效，只有一个失效下一个才自动生效
			listen, err := net.Listen("tcp", addr)
			if err != nil {
				monster.CommonLog.Fatal("mvc("+addr+"): 启动失败", err)
			}
			optionMap[addr] = &Option{
				listener: listen,
				server:   httpServer,
				vaild:    true,
			}
			listener = listen
		}
		if listener != nil {
			go func() {
				monster.CommonLog.Info("mvc(" + addr + "): 启动成功")
				//不要把 server.Serve() 放在主协程里
				if certFile != "" && keyFile != "" {
					httpServer.ServeTLS(listener, certFile, keyFile)
				} else {
					httpServer.Serve(listener)
				}
			}()
		}
	}
	var files []graceful.File
	var httpServer []*http.Server
	for addr, item := range optionMap {
		//没有用的 listener 就取消
		if !item.vaild {
			item.listener.Close()
			monster.CommonLog.Info("mvc(" + addr + "): 关闭成功")
			continue
		}
		file, fileErr := item.listener.(*net.TCPListener).File()
		if fileErr != nil {
			monster.CommonLog.Fatal(fileErr)
		}
		files = append(files, graceful.File{
			File: file,
			Addr: addr,
		})
		httpServer = append(httpServer, item.server)
	}
	graceful.SignalHandler(files, func(ctx context.Context) {
		for _, server := range httpServer {
			server.Shutdown(ctx)
		}
	})
}

func routeHandle(server *Server, w http.ResponseWriter, req *http.Request) {
	if monster.CurEnv == "release" {
		defer func() {
			if err := recover(); err != nil {
				monster.CommonLog.Error(req.URL.Path+":", err)
				ResponseOut(w, http.StatusInternalServerError, nil, "请求异常")
			}
		}()
	}
	interceptors := server.Interceptors
	interceptorsLength := len(interceptors)
	var throughoutBefore = make([]reflect.Value, interceptorsLength)
	for i, interceptor := range interceptors {
		ret := interceptor.Before(w, req, throughoutBefore, i)
		if ret != nil {
			fitOut(w, req, ret)
			return
		}
	}
	route, err := server.Handler(req)
	if err != nil {
		ResponseOut(w, http.StatusInternalServerError, nil, err.Error())
		return
	}
	controllerName := route.ControllerName
	methodName := route.MethodName
	controller := monster.Factory(controllerName)
	if controller == nil {
		ResponseOut(w, http.StatusInternalServerError, nil, "错误的路由")
		return
	}
	if methodName == "Init" || methodName == "Use" {
		ResponseOut(w, http.StatusInternalServerError, nil, "错误的路由函数")
		return
	}
	controllerType := reflect.TypeOf(controller)
	if _, ok := controllerType.MethodByName(methodName); !ok {
		ResponseOut(w, http.StatusInternalServerError, nil, "错误的路由函数")
		return
	}
	controllerValue := reflect.ValueOf(controller)
	controllerFunc := controllerValue.MethodByName(methodName)
	controllerFuncType := controllerFunc.Type()
	if !vaildMethod(req, controllerFuncType) {
		ResponseOut(w, http.StatusInternalServerError, nil, "错误的路由请求类型")
		return
	}
	paramNum := controllerFuncType.NumIn()
	paramList := make([]reflect.Value, paramNum)
	wType := reflect.TypeOf(w)
	reqType := reflect.TypeOf(req)
	for i := 0; i < paramNum; i++ {
		in := controllerFuncType.In(i)
		if wType.AssignableTo(in) {
			paramList[i] = reflect.ValueOf(w)
		} else if reqType.AssignableTo(in) {
			paramList[i] = reflect.ValueOf(req)
		} else if in.Kind() == reflect.Struct {
			paramList[i] = parseParam(req, in)
		} else {
			paramList[i] = reflect.New(in).Elem()
		}
	}
	var throughoutInvoke = make([]reflect.Value, interceptorsLength)
	for i, interceptor := range interceptors {
		ret := interceptor.Invoke(w, req, controllerFunc, paramList, throughoutInvoke, i)
		if ret != nil {
			fitOut(w, req, ret)
			return
		}
	}
	returns := controllerFunc.Call(paramList)
	returnLength := len(returns)
	if returnLength == 1 {
		var throughoutAfter = make([]reflect.Value, interceptorsLength)
		for i, interceptor := range interceptors {
			ret := interceptor.After(w, req, returns[0], throughoutAfter, i)
			if ret != nil {
				fitOut(w, req, ret)
				return
			}
		}
		fitOut(w, req, returns[0].Interface())
	} else if returnLength > 1 {
		ResponseOut(w, http.StatusInternalServerError, nil, "路由函数只能返回1个返回值")
	}
}

func vaildMethod(req *http.Request, funcType reflect.Type) bool {
	method := strings.ToUpper(req.Method)
	mMap := make(map[string]bool)
	for i, paramNum := 0, funcType.NumIn(); i < paramNum; i++ {
		in := funcType.In(i)
		if reflect.TypeOf((*OPTIONS)(nil)).Elem() == in {
			mMap["OPTIONS"] = true
		} else if reflect.TypeOf((*GET)(nil)).Elem() == in {
			mMap["GET"] = true
		} else if reflect.TypeOf((*HEAD)(nil)).Elem() == in {
			mMap["HEAD"] = true
		} else if reflect.TypeOf((*POST)(nil)).Elem() == in {
			mMap["POST"] = true
		} else if reflect.TypeOf((*PUT)(nil)).Elem() == in {
			mMap["PUT"] = true
		} else if reflect.TypeOf((*DELETE)(nil)).Elem() == in {
			mMap["DELETE"] = true
		} else if reflect.TypeOf((*TRACE)(nil)).Elem() == in {
			mMap["TRACE"] = true
		} else if reflect.TypeOf((*CONNECT)(nil)).Elem() == in {
			mMap["CONNECT"] = true
		}
	}
	if len(mMap) == 0 {
		return true
	} else {
		_, ok := mMap[method]
		return ok
	}
}

func parseParam(req *http.Request, in reflect.Type) reflect.Value {
	objAddr := reflect.New(in)
	objValue := objAddr.Elem()
	if parseJson(req, objAddr.Interface()) {
		return objValue
	}
	objType := objValue.Type()
	numField := objType.NumField()
	req.ParseForm()
	get := req.URL.Query() //get区取数据
	post := req.PostForm   //post区取数据
	file := parseFile(req) //post区上传取数据
	setValue := func(value reflect.Value, item interface{}) {
		switch value.Interface().(type) {
		case string:
			value.SetString(item.(string))
		case int64, int32, int16, int8, int:
			val, err := strconv.ParseInt(item.(string), 10, 64)
			if err == nil {
				value.SetInt(val)
			}
		case bool:
			val := strings.ToLower(item.(string))
			if val == "false" {
				value.SetBool(false)
			} else if val == "true" {
				value.SetBool(true)
			}
		case float64, float32:
			val, err := strconv.ParseFloat(item.(string), 64)
			if err == nil {
				value.SetFloat(val)
			}
		default:
			value.Set(reflect.ValueOf(item))
		}
	}
	for i := 0; i < numField; i++ {
		field := objType.Field(i)
		fieldName := field.Name
		fieldNameFL := monster.FirstLower(fieldName)
		valueField := objValue.Field(i)
		var list []interface{}
		if strings.Index(field.Type.String(), "*multipart.FileHeader") != -1 {
			if files, ok := file[fieldNameFL]; ok && len(files) > 0 {
				for _, f := range files {
					list = append(list, f)
				}
			}
		} else {
			if vals, ok := post[fieldNameFL]; ok && len(vals) > 0 {
				for _, v := range vals {
					list = append(list, v)
				}
			} else if vals, ok := get[fieldNameFL]; ok && len(vals) > 0 {
				for _, v := range vals {
					list = append(list, v)
				}
			}
		}
		if len(list) == 0 {
			continue
		}
		switch valueField.Kind() {
		case reflect.Slice, reflect.Array:
			valueType := valueField.Type()
			valueItemType := valueType.Elem()
			for i, item := range list {
				newValue := reflect.New(valueItemType).Elem()
				setValue(newValue, item)
				if valueField.Kind() == reflect.Slice {
					valueField.Set(reflect.Append(valueField, newValue))
				} else if valueField.Kind() == reflect.Array {
					valueField.Index(i).Set(newValue)
				}
			}
		default:
			setValue(valueField, list[0])
		}
	}
	return objValue
}

func parseFile(req *http.Request) map[string][]*multipart.FileHeader {
	file := make(map[string][]*multipart.FileHeader)
	contentType := strings.ToLower(req.Header.Get("Content-Type"))
	if strings.Index(contentType, "multipart/form-data") == -1 {
		return file
	}
	//http默认上传内存
	var defaultMaxMemory int64 = 32 << 20 // 32 MB
	err := req.ParseMultipartForm(defaultMaxMemory)
	if err != nil {
		return file
	}
	if req.MultipartForm != nil && req.MultipartForm.File != nil {
		for key, fhs := range req.MultipartForm.File {
			if len(fhs) > 0 {
				if _, ok := file[key]; !ok {
					file[key] = make([]*multipart.FileHeader, 0)
				}
				for _, item := range fhs {
					file[key] = append(file[key], item)
				}
			}
		}
	}
	return file
}

func parseJson(req *http.Request, obj interface{}) bool {
	contentType := strings.ToLower(req.Header.Get("Content-Type"))
	if strings.Index(contentType, "application/json") == -1 {
		return false
	}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return false
	}
	jsonErr := json.Unmarshal(body, obj)
	return jsonErr == nil
}

func fitOut(w http.ResponseWriter, req *http.Request, val interface{}) {
	var err error
	if view, ok := val.(View); ok {
		if outErr := view.Out(w, req); outErr != nil {
			err = errors.New("路由函数输出错误")
		}
	} else {
		err = errors.New("错误的路由函数输出")
	}
	if err != nil {
		ResponseOut(w, http.StatusInternalServerError, nil, err.Error())
	}
}

func ResponseOut(w http.ResponseWriter, status int, header map[string]string, content string) {
	if header != nil {
		for k, val := range header {
			w.Header().Set(k, val)
		}
	}
	//必须让 w.WriteHeader 在所有的 w.Header 之后，因为 w.WriteHeader 后 Set Header 是无效的
	w.WriteHeader(status)
	w.Write([]byte(content))
}
