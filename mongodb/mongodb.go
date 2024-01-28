package mongodb

import (
	"context"
	"errors"
	"github.com/luoshanzhi/monster-go"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	dbMap      = map[string]*dbStore{}
	dbMapGuard sync.RWMutex
)

var pick = func(dbKey string, dbType string, dbs []*Pool) (*Pool, error) {
	if dbType != "master" && dbType != "slave" {
		return nil, errors.New("dbType error")
	}
	if len(dbs) == 0 {
		return nil, errors.New("dbs len is 0")
	}
	var maxCan = -1
	var maxCanDB *Pool
	uniMap := make(map[int]bool)
	for _, item := range dbs {
		stats := item.Stats
		maxOpenConnections := item.MaxOpenConns
		inUse := stats.Use
		can := maxOpenConnections - inUse
		uniMap[can] = true
		if maxCan == -1 || can > maxCan {
			maxCan = can
			maxCanDB = item
		}
	}
	if len(uniMap) == 1 {
		//所有都相等就随机选一个
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		return dbs[r.Intn(len(dbs))], nil
	} else {
		return maxCanDB, nil
	}
}

func SetPick(pk func(dbKey string, dbType string, dbs []*Pool) (*Pool, error)) {
	if pk != nil {
		pick = pk
	}
}

func getDbKey(dbKey ...string) string {
	dbKey_ := "base"
	if len(dbKey) > 0 {
		dbKey[0] = strings.TrimSpace(dbKey[0])
		if dbKey[0] != "" {
			dbKey_ = dbKey[0]
		}
	}
	return dbKey_
}

func Open(options_ Options, dbKey ...string) {
	OpenMaster(options_, dbKey...)
	OpenSlave(options_, dbKey...)
}

func OpenMaster(options_ Options, dbKey ...string) {
	BaseOpen("master", options_, dbKey...)
}

func OpenSlave(options_ Options, dbKey ...string) {
	BaseOpen("slave", options_, dbKey...)
}

func Close(dbKey ...string) {
	CloseMaster(dbKey...)
	CloseSlave(dbKey...)
}

func CloseMaster(dbKey ...string) {
	BaseClose("master", dbKey...)
}

func CloseSlave(dbKey ...string) {
	BaseClose("slave", dbKey...)
}

func BaseOpen(dbType string, options_ Options, dbKey ...string) {
	if dbType != "master" && dbType != "slave" {
		panic("dbType error")
	}
	dbKey_ := getDbKey(dbKey...)
	mongodbConfig, ok := monster.CurEnvConfig.Mongodb[dbKey_]
	if !ok {
		panic("dbKey error")
	}
	var dbArr []monster.MongodbSettingItem
	if dbType == "master" {
		dbArr = mongodbConfig.Master
	} else if dbType == "slave" {
		dbArr = mongodbConfig.Slave
	}
	if len(dbArr) == 0 {
		panic("config error")
	}
	var dbs []*Pool
	ctx := context.Background()
	for _, item := range dbArr {
		host := item.Host
		user := item.User
		password := item.Password
		port := item.Port
		dBase := item.Database
		stats := &Statistics{
			Idle: options_.MaxOpenConns,
		}
		db := &Pool{
			MaxOpenConns: options_.MaxOpenConns,
			DatabaseName: dBase,
			Stats:        stats,
			ctx:          ctx,
		}
		poolMonitor := func(stats *Statistics) *event.PoolMonitor {
			return &event.PoolMonitor{
				Event: func(poolEvent *event.PoolEvent) {
					/*
						PoolCreated        = "ConnectionPoolCreated"
						PoolReady          = "ConnectionPoolReady"
						PoolCleared        = "ConnectionPoolCleared"
						PoolClosedEvent    = "ConnectionPoolClosed"
						ConnectionCreated  = "ConnectionCreated"
						ConnectionReady    = "ConnectionReady"
						ConnectionClosed   = "ConnectionClosed"
						GetStarted         = "ConnectionCheckOutStarted"
						GetFailed          = "ConnectionCheckOutFailed"
						GetSucceeded       = "ConnectionCheckedOut"
						ConnectionReturned = "ConnectionCheckedIn"
					*/
					//fmt.Println(poolEvent.Type)
					switch poolEvent.Type {
					case event.GetSucceeded:
						stats.Use++
						stats.Idle--
					case event.ConnectionReturned:
						stats.Idle++
						stats.Use--
					}
				},
			}
		}(stats)
		clientOptions := options.Client().ApplyURI("mongodb://" + user + ":" + password + "@" + host + ":" + strconv.Itoa(port) + "/" + dBase)
		clientOptions.SetMaxPoolSize(uint64(options_.MaxOpenConns))
		clientOptions.SetMaxConnecting(uint64(options_.MaxOpenConns))
		clientOptions.SetMinPoolSize(uint64(options_.MaxIdleConns))
		clientOptions.SetMaxConnIdleTime(options_.ConnMaxLifetime)
		clientOptions.SetPoolMonitor(poolMonitor)
		client, err := mongo.Connect(ctx, clientOptions)
		if err != nil {
			panic(err)
		}
		db.Client = client
		dbs = append(dbs, db)
	}
	//防止并发写map异常
	dbMapGuard.Lock()
	if _, ok := dbMap[dbKey_]; !ok {
		dbMap[dbKey_] = &dbStore{}
	}
	dbSt := dbMap[dbKey_]
	if dbType == "master" {
		dbSt.masters = dbs
		monster.CommonLog.Info("mongodb(" + dbKey_ + "): 主库启动成功")
	} else if dbType == "slave" {
		dbSt.slaves = dbs
		monster.CommonLog.Info("mongodb(" + dbKey_ + "): 从库启动成功")
	}
	dbMapGuard.Unlock()
	if options_.StatisticsLog {
		statisticsLogDuration := options_.StatisticsLogDuration
		if statisticsLogDuration <= 0 {
			statisticsLogDuration = time.Second * 5
		}
		go func() {
			defer monster.Recover()
			for {
				stats, err := BaseStats(dbType, dbKey_)
				if err != nil {
					return
				}
				monster.StatisticsLog.
					WithField("name", "mongodb-"+dbKey_+"-"+dbType).
					WithField("use", stats.Use).
					WithField("idle", stats.Idle).
					Info()
				time.Sleep(statisticsLogDuration)
			}
		}()
	}
}

func BaseClose(dbType string, dbKey ...string) {
	if dbType != "master" && dbType != "slave" {
		panic("dbType error")
	}
	dbKey_ := getDbKey(dbKey...)
	//防止并发写map异常
	dbMapGuard.Lock()
	if dbSt, ok := dbMap[dbKey_]; ok {
		var dbs []*Pool
		if dbType == "master" {
			dbs = dbSt.masters
		} else if dbType == "slave" {
			dbs = dbSt.slaves
		}
		for _, db := range dbs {
			db.Client.Disconnect(db.ctx)
		}
		if dbType == "master" {
			dbSt.masters = nil
		} else if dbType == "slave" {
			dbSt.slaves = nil
		}
	}
	dbMapGuard.Unlock()
}

func Stats(dbKey ...string) (Statistics, error) {
	var statistics Statistics
	masterStats, masterErr := MasterStats(dbKey...)
	if masterErr != nil {
		return statistics, masterErr
	}
	slaveStats, slaveErr := SlaveStats(dbKey...)
	if slaveErr != nil {
		return statistics, slaveErr
	}
	statistics.Use += masterStats.Use
	statistics.Idle += masterStats.Idle
	statistics.Use += slaveStats.Use
	statistics.Idle += slaveStats.Idle
	return statistics, nil
}

func MasterStats(dbKey ...string) (Statistics, error) {
	return BaseStats("master", dbKey...)
}

func SlaveStats(dbKey ...string) (Statistics, error) {
	return BaseStats("slave", dbKey...)
}

func BaseStats(dbType string, dbKey ...string) (Statistics, error) {
	if dbType != "master" && dbType != "slave" {
		panic("dbType error")
	}
	dbKey_ := getDbKey(dbKey...)
	var statistics Statistics
	var dbs []*Pool
	//防止并发写map异常
	dbMapGuard.RLock()
	if dbSt, ok := dbMap[dbKey_]; ok {
		if dbType == "master" {
			dbs = dbSt.masters
		} else if dbType == "slave" {
			dbs = dbSt.slaves
		}
	}
	dbMapGuard.RUnlock()
	if len(dbs) == 0 {
		return statistics, errors.New("dbs len is 0")
	}
	for _, db := range dbs {
		stats := db.Stats
		statistics.Use += stats.Use
		statistics.Idle += stats.Idle
	}
	return statistics, nil
}

func DB(dbKey ...string) *mongo.Database {
	return Master(dbKey...)
}

func Master(dbKey ...string) *mongo.Database {
	dbKey_ := getDbKey(dbKey...)
	var dbs []*Pool
	//防止并发写map异常
	dbMapGuard.RLock()
	if dbSt, ok := dbMap[dbKey_]; ok {
		dbs = dbSt.masters
	}
	dbMapGuard.RUnlock()
	db, _ := pick(dbKey_, "master", dbs)
	return db.Client.Database(db.DatabaseName)
}

func Slave(dbKey ...string) *mongo.Database {
	dbKey_ := getDbKey(dbKey...)
	var dbs []*Pool
	//防止并发写map异常
	dbMapGuard.RLock()
	if dbSt, ok := dbMap[dbKey_]; ok {
		dbs = dbSt.slaves
	}
	dbMapGuard.RUnlock()
	db, _ := pick(dbKey_, "slave", dbs)
	return db.Client.Database(db.DatabaseName)
}
