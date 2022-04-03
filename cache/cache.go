package cache

import (
	"errors"
	"github.com/garyburd/redigo/redis"
	"github.com/luoshanzhi/monster-go"
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

func Open(options Options) {
	rdArr := monster.CurEnvConfig.Redis
	if len(rdArr) == 0 {
		panic("缓存配置异常")
	}
	var pls []*redis.Pool
	for _, item := range rdArr {
		host := item.Host
		password := item.Password
		port := item.Port
		pl := &redis.Pool{
			Wait:        options.Wait,
			IdleTimeout: options.ConnMaxLifetime,
			MaxActive:   options.MaxOpenConns,
			MaxIdle:     options.MaxIdleConns,
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

func Get() redis.Conn {
	var conn redis.Conn
	pl, err := pick(pools)
	if err == nil && pl != nil {
		conn = pl.Get()
	}
	return conn
}

func Stats() (Statistics, error) {
	var statistics Statistics
	if len(pools) == 0 {
		return statistics, errors.New("缓存不存在")
	}
	for _, pl := range pools {
		stats := pl.Stats()
		statistics.Use += stats.ActiveCount - stats.IdleCount
		statistics.Idle += stats.IdleCount
	}
	return statistics, nil
}
