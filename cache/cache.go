package cache

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"github.com/garyburd/redigo/redis"
	"github.com/luoshanzhi/monster-go"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	caMap      = map[string][]*redis.Pool{}
	caMapGuard sync.RWMutex
)

var pick = func(caKey string, pls []*redis.Pool) (*redis.Pool, error) {
	if len(pls) == 0 {
		return nil, errors.New("pls len is 0")
	}
	var maxCan int = -1
	var maxCanPl *redis.Pool
	uniMap := make(map[int]bool)
	for _, item := range pls {
		stats := item.Stats()
		idleCount := stats.IdleCount
		uniMap[idleCount] = true
		if maxCan == -1 || idleCount > maxCan {
			maxCan = idleCount
			maxCanPl = item
		}
	}
	if len(uniMap) == 1 {
		//所有都相等就随机选一个
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		return pls[r.Intn(len(pls))], nil
	} else {
		return maxCanPl, nil
	}
}

func SetPick(pk func(caKey string, pls []*redis.Pool) (*redis.Pool, error)) {
	if pk != nil {
		pick = pk
	}
}

func getCaKey(caKey ...string) string {
	caKey_ := "base"
	if len(caKey) > 0 {
		caKey[0] = strings.TrimSpace(caKey[0])
		if caKey[0] != "" {
			caKey_ = caKey[0]
		}
	}
	return caKey_
}

func Open(options Options, caKey ...string) {
	caKey_ := getCaKey(caKey...)
	rdArr, ok := monster.CurEnvConfig.Redis[caKey_]
	if !ok {
		panic("caKey error")
	}
	if len(rdArr) == 0 {
		panic("config error")
	}
	var pls []*redis.Pool
	for _, item := range rdArr {
		host := item.Host
		password := item.Password
		port := item.Port
		pl := &redis.Pool{
			Wait:            options.Wait,
			MaxConnLifetime: options.ConnMaxLifetime,
			MaxActive:       options.MaxOpenConns,
			MaxIdle:         options.MaxIdleConns,
			Dial: func() (redis.Conn, error) {
				conn, err := redis.Dial("tcp", host+":"+strconv.Itoa(port))
				if err != nil {
					return nil, err
				}
				if _, err := conn.Do("AUTH", password); err != nil {
					conn.Close()
					return nil, err
				}
				return conn, err
			},
		}
		pls = append(pls, pl)
	}
	//防止并发写map异常
	caMapGuard.Lock()
	caMap[caKey_] = pls
	caMapGuard.Unlock()
	conn := Conn(caKey...)
	defer conn.Close()
	err := conn.Err()
	if err != nil {
		panic(err)
	}
	monster.CommonLog.Info("缓存(" + caKey_ + "): 启动成功")
	if options.StatisticsLog {
		statisticsLogDuration := options.StatisticsLogDuration
		if statisticsLogDuration <= 0 {
			statisticsLogDuration = time.Second * 5
		}
		go func() {
			defer monster.Recover()
			for {
				stats, err := Stats(caKey...)
				if err != nil {
					return
				}
				monster.StatisticsLog.
					WithField("name", caKey_+"-cache").
					WithField("use", stats.Use).
					WithField("idle", stats.Idle).
					Info()
				time.Sleep(statisticsLogDuration)
			}
		}()
	}
}

func Close(caKey ...string) {
	caKey_ := getCaKey(caKey...)
	//防止并发写map异常
	caMapGuard.Lock()
	pools, ok := caMap[caKey_]
	if ok && pools != nil {
		for _, pl := range pools {
			pl.Close()
		}
		caMap[caKey_] = nil
	}
	caMapGuard.Unlock()
}

func Conn(caKey ...string) redis.Conn {
	return ConnContext(context.Background(), caKey...)
}

func ConnContext(ctx context.Context, caKey ...string) redis.Conn {
	caKey_ := getCaKey(caKey...)
	var conn redis.Conn
	//防止并发写map异常
	caMapGuard.RLock()
	pools := caMap[caKey_]
	caMapGuard.RUnlock()
	pl, err := pick(caKey_, pools)
	if err == nil && pl != nil {
		conn, _ = pl.GetContext(ctx)
	}
	return conn
}

func Stats(caKey ...string) (Statistics, error) {
	caKey_ := getCaKey(caKey...)
	var statistics Statistics
	//防止并发写map异常
	caMapGuard.RLock()
	pools := caMap[caKey_]
	caMapGuard.RUnlock()
	if len(pools) == 0 {
		return statistics, errors.New("pools len is 0")
	}
	for _, pl := range pools {
		stats := pl.Stats()
		statistics.Use += stats.ActiveCount - stats.IdleCount
		statistics.Idle += stats.IdleCount
	}
	return statistics, nil
}

func Get(conn redis.Conn, key string, data interface{}) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("key is empty")
	}
	if reflect.TypeOf(data).Kind() != reflect.Ptr {
		return errors.New("data is not ptr")
	}
	reply, replyErr := conn.Do("GET", key)
	if replyErr != nil {
		return replyErr
	}
	var str string
	_, scanErr := redis.Scan([]interface{}{reply}, &str)
	if scanErr != nil {
		return scanErr
	}
	//bytes.NewBuffer和bytes.Buffer类似，只不过可以传入一个初始的byte数组，返回一个指针
	dec := gob.NewDecoder(bytes.NewBuffer([]byte(str)))
	//调用Decode方法,传入结构体对象指针，会自动将buf.Bytes()里面的内容转换成结构体
	if err := dec.Decode(data); err != nil {
		return err
	}
	return nil
}

// timeout(单位:秒) <=0 不限制时间
func Set(conn redis.Conn, key string, data interface{}, timeout int) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("key is empty")
	}
	//创建缓存
	buf := new(bytes.Buffer)
	//把指针丢进去
	enc := gob.NewEncoder(buf)
	//调用Encode进行序列化
	if err := enc.Encode(data); err != nil {
		return err
	}
	if err := conn.Send("SET", key, buf.String()); err != nil {
		return err
	}
	if timeout > 0 {
		if err := conn.Send("EXPIRE", key, timeout); err != nil {
			return err
		}
	}
	return conn.Flush()
}

func Exists(conn redis.Conn, key string) (bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return false, errors.New("key is empty")
	}
	exists, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return false, err
	}
	return exists, nil
}

func Del(conn redis.Conn, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("key is empty")
	}
	_, err := conn.Do("DEL", key)
	if err != nil {
		return err
	}
	return nil
}

// key: * 代表查询所有
func Keys(conn redis.Conn, key string) ([]string, error) {
	var keys []string
	key = strings.TrimSpace(key)
	if key == "" {
		return keys, errors.New("key is empty")
	}
	reply, doErr := redis.Values(conn.Do("KEYS", key))
	if doErr != nil {
		return keys, doErr
	}
	arr := make([]string, len(reply))
	arrAddr := make([]interface{}, len(arr))
	for i, _ := range arr {
		arrAddr[i] = &arr[i]
	}
	_, scanErr := redis.Scan(reply, arrAddr...)
	if scanErr != nil {
		return keys, scanErr
	}
	keys = arr
	return keys, nil
}

func Ttl(conn redis.Conn, key string) (int, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return 0, errors.New("key is empty")
	}
	ttl, err := redis.Int(conn.Do("TTL", key))
	if err != nil {
		return 0, err
	}
	return ttl, nil
}
