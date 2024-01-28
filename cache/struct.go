package cache

import "time"

type Options struct {
	Wait                  bool          //超出最大使用是否阻塞等待
	ConnMaxLifetime       time.Duration //保存连接的最大存活时间
	MaxOpenConns          int           //最大打开连接
	MaxIdleConns          int           //最大保存多少空闲连接
	StatisticsLog         bool          //是否记录统计日志
	StatisticsLogDuration time.Duration //多长时间记录一次统计日志, 默认5秒钟记录一次
}

type Statistics struct {
	Use  int //正在使用
	Idle int //正在空闲
}
