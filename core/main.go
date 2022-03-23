package core

import (
	"encoding/json"
	"errors"
	"flag"
	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

var (
	factoryMap      map[string]interface{}
	factoryMapGuard sync.Mutex
	RootPath        string
	Args            struct {
		Path     string
		Graceful bool
	}
	SettingConfig Setting
	CurEnv        string
	CurEnvConfig  EnvConfig
	AccessLog     = logrus.New()
	CommonLog     = logrus.New()
)

func init() {
	pathPtr := flag.String("path", "", "项目路径")
	gracefulPtr := flag.Bool("graceful", false, "从文件描述符打开listener")
	flag.Parse()
	path := strings.TrimSpace(*pathPtr)
	if path == "" {
		path, _ = os.Getwd()
	}
	Args.Path = path
	Args.Graceful = *gracefulPtr
	RootPath = path + "/"
}

func Init(settingFile string, logPath string, fm map[string]interface{}) {
	//初始化配置
	var settingConfig Setting
	if err := loadJson(settingFile, &settingConfig); err != nil {
		panic(err)
	}
	SettingConfig = settingConfig
	CurEnv = settingConfig.Env
	CurEnvConfig = settingConfig.EnvConfig[CurEnv]
	if err := setLog(logPath); err != nil {
		panic(err)
	}
	factoryMap = fm
}

func setLog(logPath string) error {
	reg := regexp.MustCompile(`/$`)
	logPath = reg.ReplaceAllString(strings.TrimSpace(logPath), "") + "/"
	if !exists(logPath) {
		return errors.New(`"` + logPath + `"路径不存在`)
	}
	logLevel, logLevelErr := logrus.ParseLevel(CurEnvConfig.LogLevel)
	if logLevelErr != nil {
		return logLevelErr
	}
	AccessLog.SetLevel(logLevel)
	CommonLog.SetLevel(logLevel)
	accessLogFile := logPath + "access.log"
	commonLogFile := logPath + "module.log"
	/* 日志轮转相关函数
	`WithLinkName` 为最新的日志建立软连接
	`WithRotationTime` 设置日志分割的时间，隔多久分割一次
	WithMaxAge 和 WithRotationCount二者只能设置一个
	`WithMaxAge` 设置文件清理前的最长保存时间
	`WithRotationCount` 设置文件清理前最多保存的个数
	*/
	// 下面配置日志每隔2小时轮转一个新文件，保留最近12个日志文件，多余的自动清理掉。
	set := func(logger *logrus.Logger, path string) {
		writer, _ := rotatelogs.New(
			path+".%Y%m%d%H%M",
			rotatelogs.WithLinkName(path),
			rotatelogs.WithRotationCount(12),
			rotatelogs.WithRotationTime(time.Duration(120)*time.Minute),
		)
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
		var writers []io.Writer
		writers = append(writers, writer)
		if CurEnv == "dev" {
			//如果是开发环境，同时输出控制台
			writers = append(writers, os.Stdout)
		}
		multiWriter := io.MultiWriter(writers...)
		logger.SetOutput(multiWriter)
	}
	set(AccessLog, accessLogFile)
	set(CommonLog, commonLogFile)
	return nil
}

func callFunc(obj interface{}, name string, args ...interface{}) {
	objType := reflect.TypeOf(obj)
	_, ok := objType.MethodByName(name)
	if !ok {
		return
	}
	objValue := reflect.ValueOf(obj)
	method := objValue.MethodByName(name)
	paramNum := len(args)
	paramList := make([]reflect.Value, paramNum)
	for i := 0; i < paramNum; i++ {
		paramList[i] = reflect.ValueOf(args[i])
	}
	method.Call(paramList)
}

func inject(obj interface{}) {
	objType := reflect.TypeOf(obj)
	objValue := reflect.ValueOf(obj)
	if objType.Kind() != reflect.Ptr {
		return
	}
	typeMap := make(map[reflect.Type]string)
	//防止并发遍历map异常
	factoryMapGuard.Lock()
	for key, val := range factoryMap {
		oType := reflect.TypeOf(val)
		typeMap[oType] = key
	}
	factoryMapGuard.Unlock()
	objType = objType.Elem()
	objValue = objValue.Elem()
	numField := objValue.NumField()
	for i := 0; i < numField; i++ {
		field := objType.Field(i)
		fieldType := field.Type
		fieldValue := objValue.Field(i)
		//反射设置私有字段
		unsafeFieldValue := reflect.NewAt(fieldValue.Type(), unsafe.Pointer(fieldValue.UnsafeAddr())).Elem()
		if fieldValue.Kind() != reflect.Ptr {
			fieldType = fieldValue.Addr().Type()
		}
		if key, ok := typeMap[fieldType]; ok {
			factory := Factory(key)
			if fieldValue.Kind() == reflect.Ptr {
				unsafeFieldValue.Set(reflect.ValueOf(factory))
			} else {
				unsafeFieldValue.Set(reflect.ValueOf(factory).Elem())
			}
		} else {
			val := field.Tag.Get("val")
			if val == "" {
				continue
			}
			switch fieldValue.Kind() {
			case reflect.String:
				unsafeFieldValue.Set(reflect.ValueOf(val))
			case reflect.Int, reflect.Int64:
				if val, err := strconv.ParseInt(val, 10, 64); err == nil {
					unsafeFieldValue.Set(reflect.ValueOf(int(val)))
				}
			case reflect.Int32:
				if val, err := strconv.ParseInt(val, 10, 32); err == nil {
					unsafeFieldValue.Set(reflect.ValueOf(int32(val)))
				}
			case reflect.Int16:
				if val, err := strconv.ParseInt(val, 10, 16); err == nil {
					unsafeFieldValue.Set(reflect.ValueOf(int16(val)))
				}
			case reflect.Int8:
				if val, err := strconv.ParseInt(val, 10, 8); err == nil {
					unsafeFieldValue.Set(reflect.ValueOf(int8(val)))
				}
			case reflect.Uint, reflect.Uint64:
				if val, err := strconv.ParseUint(val, 10, 64); err == nil {
					unsafeFieldValue.Set(reflect.ValueOf(uint(val)))
				}
			case reflect.Uint32:
				if val, err := strconv.ParseUint(val, 10, 32); err == nil {
					unsafeFieldValue.Set(reflect.ValueOf(uint32(val)))
				}
			case reflect.Uint16:
				if val, err := strconv.ParseUint(val, 10, 16); err == nil {
					unsafeFieldValue.Set(reflect.ValueOf(uint16(val)))
				}
			case reflect.Uint8:
				if val, err := strconv.ParseUint(val, 10, 8); err == nil {
					unsafeFieldValue.Set(reflect.ValueOf(uint8(val)))
				}
			case reflect.Float64:
				if val, err := strconv.ParseFloat(val, 64); err == nil {
					unsafeFieldValue.Set(reflect.ValueOf(float64(val)))
				}
			case reflect.Float32:
				if val, err := strconv.ParseFloat(val, 32); err == nil {
					unsafeFieldValue.Set(reflect.ValueOf(float32(val)))
				}
			case reflect.Bool:
				if val, err := strconv.ParseBool(val); err == nil {
					unsafeFieldValue.Set(reflect.ValueOf(val))
				}
			case reflect.Complex64:
				if val, err := strconv.ParseComplex(val, 64); err == nil {
					unsafeFieldValue.Set(reflect.ValueOf(complex64(val)))
				}
			case reflect.Complex128:
				if val, err := strconv.ParseComplex(val, 128); err == nil {
					unsafeFieldValue.Set(reflect.ValueOf(complex128(val)))
				}
			}
		}
	}
}

func loadJson(fileName string, data interface{}) error {
	fileName = strings.TrimSpace(fileName)
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, data)
	if err != nil {
		return err
	}
	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func In(name string) bool {
	//防止并发写map异常
	factoryMapGuard.Lock()
	_, ok := factoryMap[name]
	factoryMapGuard.Unlock()
	return ok
}

func Factory(name string, args ...interface{}) interface{} {
	if !In(name) {
		panic(errors.New("工厂不存在" + name))
	}
	//防止并发写map异常
	factoryMapGuard.Lock()
	obj := factoryMap[name]
	factoryMapGuard.Unlock()
	objType := reflect.TypeOf(obj)
	objValue := reflect.ValueOf(obj)
	create := func() interface{} {
		objValue = reflect.New(objType.Elem())
		obj := objValue.Interface()
		inject(obj)
		callFunc(obj, "Init", args...)
		return obj
	}
	if objValue.IsNil() {
		obj = create()
		//防止并发写map异常
		factoryMapGuard.Lock()
		factoryMap[name] = obj
		factoryMapGuard.Unlock()
	}
	_, ok := obj.(Multiton)
	if ok {
		obj = create()
	}
	callFunc(obj, "Use", args...)
	return obj
}
