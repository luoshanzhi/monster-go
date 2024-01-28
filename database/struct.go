package database

import (
	"database/sql"
	"time"
)

type Options struct {
	Charset               string        //字符集,默认utf8
	InterpolateParams     bool          //默认false, 所有带参数(?)的sql都被转成Stmt(预处理); true，参数(?)将被简单的escape防止注入
	ConnMaxLifetime       time.Duration //保存连接的最大存活时间
	MaxOpenConns          int           //最大打开连接
	MaxIdleConns          int           //最大保存多少空闲连接
	StatisticsLog         bool          //是否记录统计日志
	StatisticsLogDuration time.Duration //多长时间记录一次统计日志, 默认5秒钟记录一次
}

type dbStore struct {
	masters []*sql.DB
	slaves  []*sql.DB
}

type Statistics struct {
	Use  int //正在使用
	Idle int //正在空闲
}

type scan struct {
	dest   []interface{}
	column []string
}
