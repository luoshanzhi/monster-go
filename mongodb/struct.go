package mongodb

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type Options struct {
	ConnMaxLifetime       time.Duration //保存连接的最大存活时间
	MaxOpenConns          int           //最大打开连接
	MaxIdleConns          int           //最大保存多少空闲连接
	StatisticsLog         bool          //是否记录统计日志
	StatisticsLogDuration time.Duration //多长时间记录一次统计日志, 默认5秒钟记录一次
}

type dbStore struct {
	masters []*Pool
	slaves  []*Pool
}

type Statistics struct {
	Use  int //正在使用
	Idle int //正在空闲
}

type Pool struct {
	MaxOpenConns int
	DatabaseName string
	Stats        *Statistics
	Client       *mongo.Client
	ctx          context.Context
}
