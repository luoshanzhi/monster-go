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
	"time"
)

var pools []*redis.Pool

var pick = func(pls []*redis.Pool) (*redis.Pool, error) {
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
		rand.Seed(time.Now().UnixNano())
		return pls[rand.Intn(len(pls))], nil
	} else {
		return maxCanPl, nil
	}
}

func SetPick(pk func(pls []*redis.Pool) (*redis.Pool, error)) {
	if pk != nil {
		pick = pk
	}
}

func Open(options Options) {
	rdArr := monster.CurEnvConfig.Redis
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
	pools = pls
	conn := Conn()
	defer conn.Close()
	err := conn.Err()
	if err != nil {
		panic(err)
	}
	monster.CommonLog.Info("缓存: 启动成功")
	if options.StatisticsLog {
		statisticsLogDuration := options.StatisticsLogDuration
		if statisticsLogDuration <= 0 {
			statisticsLogDuration = time.Second * 5
		}
		go func() {
			for {
				stats, err := Stats()
				if err != nil {
					return
				}
				monster.StatisticsLog.
					WithField("name", "cache").
					WithField("use", stats.Use).
					WithField("idle", stats.Idle).
					Info()
				time.Sleep(statisticsLogDuration)
			}
		}()
	}
}

func Close() {
	if pools == nil || len(pools) == 0 {
		return
	}
	for _, pl := range pools {
		pl.Close()
	}
	pools = nil
}

func Conn() redis.Conn {
	return ConnContext(context.Background())
}

func ConnContext(ctx context.Context) redis.Conn {
	var conn redis.Conn
	pl, err := pick(pools)
	if err == nil && pl != nil {
		conn, _ = pl.GetContext(ctx)
	}
	return conn
}

func Stats() (Statistics, error) {
	var statistics Statistics
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

//timeout(单位:秒) <=0 不限制时间
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

//key: * 代表查询所有
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
