package cache

import (
	"errors"
	"github.com/garyburd/redigo/redis"
	"github.com/luoshanzhi/monster-go/core"
	"math/rand"
	"strconv"
	"time"
)

var pools []*redis.Pool

var pick = func(pls []*redis.Pool) (*redis.Pool, error) {
	if len(pls) == 0 {
		return nil, errors.New("pls为空")
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

func Open(connMaxLifetime time.Duration, maxOpenConns int, maxIdleConns int) {
	rdArr := core.CurEnvConfig.Redis
	if len(rdArr) == 0 {
		panic("缓存配置异常")
	}
	var pls []*redis.Pool
	for _, item := range rdArr {
		host := item.Host
		password := item.Password
		port := item.Port
		pl := &redis.Pool{
			MaxConnLifetime: connMaxLifetime,
			MaxActive:       maxOpenConns,
			MaxIdle:         maxIdleConns,
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
	conn := Get()
	defer conn.Close()
	err := conn.Err()
	if err != nil {
		panic(err)
	}
	core.CommonLog.Info("缓存: 启动成功")
}

func Close() {
	if pools == nil || len(pools) == 0 {
		return
	}
	for _, pl := range pools {
		pl.Close()
	}
}

func Get() redis.Conn {
	var conn redis.Conn
	pl, err := pick(pools)
	if err == nil && pl != nil {
		conn = pl.Get()
	}
	return conn
}
