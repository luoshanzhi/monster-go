package database

import "time"

type Options struct {
	ConnMaxLifetime time.Duration //连接最大存活时间
	MaxOpenConns    int           //最大打开连接
	MaxIdleConns    int           //最大空闲
}
