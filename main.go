package monster

import (
	"encoding/json"
	"errors"
	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	"io"
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
	separator       = string(os.PathSeparator)
	factoryMap      map[string]interface{}
	factoryMapGuard sync.RWMutex
	typeMap         map[reflect.Type]string
	SettingConfig   Setting
	CurEnv          string //dev,beta,release
	CurEnvConfig    EnvConfig
	StatisticsLog   = logrus.New()
	AccessLog       = logrus.New()
	CommonLog       = logrus.New()
	ErrorLog        = logrus.New()
	logLevel        string
	logPath         string
	settingFile     string
)

func init() {
	CurEnv = "dev"
	logLevel = "trace"
	logPath = "." + separator + "log"
}

func Recover() {
	if err := recover(); err != nil {
		ErrorLog.Info(err)
	}
}

func Init(fm map[string]interface{}) {
	if err := setSetting(); err != nil {
		panic(err)
	}
	if err := setLog(); err != nil {
		panic(err)
	}
	factoryMap = fm
	typeMap = make(map[reflect.Type]string)
	for key, val := range factoryMap {
		typeMap[reflect.TypeOf(val)] = key
	}
}

func setSetting() error {
	if settingFile != "" {
		//初始化配置
		var settingConfig Setting
		if err := loadJson(settingFile, &settingConfig); err != nil {
			return err
		}
		SettingConfig = settingConfig
		CurEnv = settingConfig.Env
		CurEnvConfig = settingConfig.EnvConfig[CurEnv]
		logLevel = CurEnvConfig.LogLevel
	}
	return nil
}

func setLog() error {
	if !exists(logPath) {
		if err := os.Mkdir(logPath, os.ModePerm); err != nil {
			return err
		}
	}
	level, levelErr := logrus.ParseLevel(logLevel)
	if levelErr != nil {
		return levelErr
	}
	StatisticsLog.SetLevel(level)
	AccessLog.SetLevel(level)
	CommonLog.SetLevel(level)
	ErrorLog.SetLevel(level)
	statisticsLogFile := logPath + separator + "statistics.log"
	accessLogFile := logPath + separator + "access.log"
	commonLogFile := logPath + separator + "common.log"
	errorLogFile := logPath + separator + "error.log"
	/* 日志轮转相关函数
	`WithLinkName` 为最新的日志建立软连接
	`WithRotationTime` 设置日志分割的时间，隔多久分割一次
	WithMaxAge 和 WithRotationCount二者只能设置一个
	`WithMaxAge` 设置文件清理前的最长保存时间
	`WithRotationCount` 设置文件清理前最多保存的个数
	*/
	// 下面配置日志每隔2小时轮转一个新文件，保留最近12个日志文件，多余的自动清理掉。
	set := func(logger *logrus.Logger, path string, formatter logrus.Formatter) {
		writer, _ := rotatelogs.New(
			path+".%Y%m%d%H%M",
			rotatelogs.WithLinkName(path),
			rotatelogs.WithRotationCount(12),
			rotatelogs.WithRotationTime(time.Duration(120)*time.Minute),
		)
		if formatter == nil {
			formatter = &logrus.TextFormatter{
				TimestampFormat: "2006-01-02 15:04:05",
			}
		}
		logger.SetFormatter(formatter)
		var writers []io.Writer
		writers = append(writers, writer)
		if CurEnv == "dev" {
			//如果是开发环境，同时输出控制台
			writers = append(writers, os.Stdout)
		}
		multiWriter := io.MultiWriter(writers...)
		logger.SetOutput(multiWriter)
	}
	set(StatisticsLog, statisticsLogFile, &logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})
	set(AccessLog, accessLogFile, nil)
	set(CommonLog, commonLogFile, nil)
	set(ErrorLog, errorLogFile, nil)
	return nil
}

func SetLogPath(path string) {
	reg := regexp.MustCompile(separator + `$`)
	logPath = reg.ReplaceAllString(strings.TrimSpace(path), "")
}

func SetSettingFile(file string) {
	settingFile = strings.TrimSpace(file)
}

func callFunc(obj interface{}, name string, args ...interface{}) {
	objType := reflect.TypeOf(obj)
	if _, ok := objType.MethodByName(name); !ok {
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
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}
			switch fieldType.Kind() {
			case reflect.String:
				if unsafeFieldValue.Kind() == reflect.Ptr {
					unsafeFieldValue.Set(reflect.ValueOf(&val))
				} else {
					unsafeFieldValue.Set(reflect.ValueOf(val))
				}
			case reflect.Int:
				if val, err := strconv.ParseInt(val, 10, 64); err == nil {
					newVal := int(val)
					if unsafeFieldValue.Kind() == reflect.Ptr {
						unsafeFieldValue.Set(reflect.ValueOf(&newVal))
					} else {
						unsafeFieldValue.Set(reflect.ValueOf(newVal))
					}
				}
			case reflect.Int64:
				if val, err := strconv.ParseInt(val, 10, 64); err == nil {
					if unsafeFieldValue.Kind() == reflect.Ptr {
						unsafeFieldValue.Set(reflect.ValueOf(&val))
					} else {
						unsafeFieldValue.Set(reflect.ValueOf(val))
					}
				}
			case reflect.Int32:
				if val, err := strconv.ParseInt(val, 10, 32); err == nil {
					newVal := int32(val)
					if unsafeFieldValue.Kind() == reflect.Ptr {
						unsafeFieldValue.Set(reflect.ValueOf(&newVal))
					} else {
						unsafeFieldValue.Set(reflect.ValueOf(newVal))
					}
				}
			case reflect.Int16:
				if val, err := strconv.ParseInt(val, 10, 16); err == nil {
					newVal := int16(val)
					if unsafeFieldValue.Kind() == reflect.Ptr {
						unsafeFieldValue.Set(reflect.ValueOf(&newVal))
					} else {
						unsafeFieldValue.Set(reflect.ValueOf(newVal))
					}
				}
			case reflect.Int8:
				if val, err := strconv.ParseInt(val, 10, 8); err == nil {
					newVal := int8(val)
					if unsafeFieldValue.Kind() == reflect.Ptr {
						unsafeFieldValue.Set(reflect.ValueOf(&newVal))
					} else {
						unsafeFieldValue.Set(reflect.ValueOf(newVal))
					}
				}
			case reflect.Uint:
				if val, err := strconv.ParseUint(val, 10, 64); err == nil {
					newVal := uint(val)
					if unsafeFieldValue.Kind() == reflect.Ptr {
						unsafeFieldValue.Set(reflect.ValueOf(&newVal))
					} else {
						unsafeFieldValue.Set(reflect.ValueOf(newVal))
					}
				}
			case reflect.Uint64:
				if val, err := strconv.ParseUint(val, 10, 64); err == nil {
					if unsafeFieldValue.Kind() == reflect.Ptr {
						unsafeFieldValue.Set(reflect.ValueOf(&val))
					} else {
						unsafeFieldValue.Set(reflect.ValueOf(val))
					}
				}
			case reflect.Uint32:
				if val, err := strconv.ParseUint(val, 10, 32); err == nil {
					newVal := uint32(val)
					if unsafeFieldValue.Kind() == reflect.Ptr {
						unsafeFieldValue.Set(reflect.ValueOf(&newVal))
					} else {
						unsafeFieldValue.Set(reflect.ValueOf(newVal))
					}
				}
			case reflect.Uint16:
				if val, err := strconv.ParseUint(val, 10, 16); err == nil {
					newVal := uint16(val)
					if unsafeFieldValue.Kind() == reflect.Ptr {
						unsafeFieldValue.Set(reflect.ValueOf(&newVal))
					} else {
						unsafeFieldValue.Set(reflect.ValueOf(newVal))
					}
				}
			case reflect.Uint8:
				if val, err := strconv.ParseUint(val, 10, 8); err == nil {
					newVal := uint8(val)
					if unsafeFieldValue.Kind() == reflect.Ptr {
						unsafeFieldValue.Set(reflect.ValueOf(&newVal))
					} else {
						unsafeFieldValue.Set(reflect.ValueOf(newVal))
					}
				}
			case reflect.Float64:
				if val, err := strconv.ParseFloat(val, 64); err == nil {
					if unsafeFieldValue.Kind() == reflect.Ptr {
						unsafeFieldValue.Set(reflect.ValueOf(&val))
					} else {
						unsafeFieldValue.Set(reflect.ValueOf(val))
					}
				}
			case reflect.Float32:
				if val, err := strconv.ParseFloat(val, 32); err == nil {
					newVal := float32(val)
					if unsafeFieldValue.Kind() == reflect.Ptr {
						unsafeFieldValue.Set(reflect.ValueOf(&newVal))
					} else {
						unsafeFieldValue.Set(reflect.ValueOf(newVal))
					}
				}
			case reflect.Bool:
				if val, err := strconv.ParseBool(val); err == nil {
					newVal := val
					if unsafeFieldValue.Kind() == reflect.Ptr {
						unsafeFieldValue.Set(reflect.ValueOf(&newVal))
					} else {
						unsafeFieldValue.Set(reflect.ValueOf(newVal))
					}
				}
			case reflect.Complex128:
				if val, err := strconv.ParseComplex(val, 128); err == nil {
					if unsafeFieldValue.Kind() == reflect.Ptr {
						unsafeFieldValue.Set(reflect.ValueOf(&val))
					} else {
						unsafeFieldValue.Set(reflect.ValueOf(val))
					}
				}
			case reflect.Complex64:
				if val, err := strconv.ParseComplex(val, 64); err == nil {
					newVal := complex64(val)
					if unsafeFieldValue.Kind() == reflect.Ptr {
						unsafeFieldValue.Set(reflect.ValueOf(&newVal))
					} else {
						unsafeFieldValue.Set(reflect.ValueOf(newVal))
					}
				}
			default:
				obValue := reflect.New(fieldType)
				if err := json.Unmarshal([]byte(val), obValue.Interface()); err == nil {
					if unsafeFieldValue.Kind() == reflect.Ptr {
						unsafeFieldValue.Set(obValue)
					} else {
						unsafeFieldValue.Set(obValue.Elem())
					}
				}
			}
		}
	}
}

func loadJson(fileName string, data interface{}) error {
	fileName = strings.TrimSpace(fileName)
	if !exists(fileName) {
		return errors.New(`"` + fileName + `" IsNotExist`)
	}
	b, err := os.ReadFile(fileName)
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
	factoryMapGuard.RLock()
	_, ok := factoryMap[name]
	factoryMapGuard.RUnlock()
	return ok
}

func Factory(name string, args ...interface{}) interface{} {
	if !In(name) {
		panic(errors.New("工厂不存在" + name))
	}
	//防止并发写map异常
	factoryMapGuard.RLock()
	obj := factoryMap[name]
	factoryMapGuard.RUnlock()
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

func FirstLower(str string) string {
	return strings.ToLower(str[0:1]) + str[1:]
}

func FirstUpper(str string) string {
	return strings.ToUpper(str[0:1]) + str[1:]
}
