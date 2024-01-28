package main

import (
	"fmt"
	"github.com/luoshanzhi/monster-go"
	"github.com/luoshanzhi/monster-go/cache"
	"time"
)

type Common struct {
}

var factoryMap = map[string]interface{}{
	"Common": (*Common)(nil),
}

func main() {
	monster.SetSettingFile("setting.json")
	monster.Init(factoryMap) //初始化工厂
	defer cache.Close()
	cache.Open(cache.Options{
		ConnMaxLifetime: time.Minute * 3, //保存的连接最大存活3分钟就释放掉
		MaxOpenConns:    10,              //连接池,最大打开10连接,超过阻塞等待
		MaxIdleConns:    10,              //连接池,最大保存5个空闲连接
		//StatisticsLog:   true,            //开启统计日志, 记录连接池的基本状态, 文件: log/statistics.log
	})
	type User struct {
		UserId   int
		NickName string
	}
	conn := cache.Conn()
	defer conn.Close()
	user1 := User{
		UserId:   1,
		NickName: "怪兽",
	}
	key := "user"

	//设置缓存, timeout(单位:秒) <=0 不限制时间
	setErr := cache.Set(conn, key, user1, 5)
	if setErr != nil {
		panic(setErr)
	}
	fmt.Println("缓存设置成功")

	//判断缓存是否存在
	exists, existsErr := cache.Exists(conn, key)
	if existsErr != nil {
		panic(existsErr)
	}
	fmt.Println("存在: ", exists)

	//查询目前缓存key: * 代表查询所有
	keys, keysErr := cache.Keys(conn, "us*")
	if keysErr != nil {
		panic(keysErr)
	}
	fmt.Println("keys: ", keys)

	//查询key的ttl(Time To Live, 生存时间值)
	seconds, ttlErr := cache.Ttl(conn, "user")
	if ttlErr != nil {
		panic(ttlErr)
	}
	fmt.Println("seconds: ", seconds)

	//读取缓存
	var user2 User
	getErr := cache.Get(conn, key, &user2)
	if getErr != nil {
		panic(getErr)
	}
	fmt.Println(user2)

	//睡眠5秒，等待缓存user超时被删除
	//time.Sleep(time.Second * 5)

	//删除缓存
	delErr := cache.Del(conn, key)
	if delErr != nil {
		panic(delErr)
	}
	fmt.Println("缓存删除成功")

	//判断缓存是否存在
	exists, existsErr = cache.Exists(conn, key)
	if existsErr != nil {
		panic(existsErr)
	}
	fmt.Println("存在: ", exists)
}
