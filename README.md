# monster

```text
一个简单的go语言web开发框架; 
简单的 工厂模式；
简单的 MVC模式,支持https,请求拦截器,支持热更新,优雅关闭服务器,
简单的 数据库(mysql主从)；
简单的 缓存(redis)
```

> ### 1. 工厂使用 [查看demo, base.go是入口文件][demoFactory]

```go
package main

import (
	"fmt"
	"github.com/luoshanzhi/monster-go"
	"time"
)

type Bird struct {
	//框架自动注入 Common 工厂实例, 带不带*都可以，私有属性也可以注入
	common *Common
}

func (the *Bird) Fly() {
	fmt.Println("flying: " + the.common.Now())
}

type Dog struct {
	common *Common
}

func (the *Dog) Init() {
	//工厂模块初始化时，Init方法被调用
	fmt.Println("Init Dog")
}

func (the *Dog) Use() {
	//工厂模块每次使用时, Use方法被调用
	fmt.Println("Use Dog")
}

func (the *Dog) Run() {
	fmt.Println("running: " + the.common.Now())
}

type Common struct {
	//下面是各种自动注入值
	str    string   `val:"怪兽"`
	num    int      `val:"123"`
	bl     bool     `val:"true"`
	numArr []int    `val:"[12,34,56,78,90]"`
	strArr []string `val:"[\"aa\",\"bb\",\"cc\",\"dd\",\"ee\"]"`
	ob     struct {
		Age  int
		Name string
		Man  bool
	} `val:"{\"age\":18,\"man\":true,\"name\":\"怪兽\"}"`
}

func (the *Common) Now() string {
	return time.Now().String()
}

func (the *Common) Print() {
	fmt.Println(the.str, the.num, the.bl, the.numArr, the.strArr, the.ob)
}

//object 实现空 Multiton 方法就代表多例，每次工厂取出都是全新的实例

var factoryMap = map[string]interface{}{
	"Bird":   (*Bird)(nil),
	"Dog":    (*Dog)(nil),
	"Common": (*Common)(nil),
}

func main() {
	monster.Init(factoryMap) //初始化工厂
	bird := monster.Factory("Bird").(*Bird)
	dog := monster.Factory("Dog").(*Dog)
	common := monster.Factory("Common").(*Common)
	bird.Fly()
	dog.Run()
	common.Print()
}


输出:
Init Dog
Use Dog
flying: 2022-03-26 16:37:07.403121 +0800 CST m = +0.001406368
running: 2022-03-26 16:37:07.403492 +0800 CST m = +0.001777576
怪兽 123 true [12 34 56 78 90] [aa bb cc dd ee] {18 怪兽 true}
```

> ### 2. MVC使用 [查看demo, base.go是入口文件][demoMvc]

```text
支持https,优雅关闭服务器,即使热更新时存在端口不同也会优雅关闭服务
优雅关闭服务器: kill pid
重启(支持热更新): kill -USR2 pid
```

```go
package main

import (
	"errors"
	"fmt"
	"github.com/luoshanzhi/monster-go"
	"github.com/luoshanzhi/monster-go/demo/mvc/object"
	"github.com/luoshanzhi/monster-go/mvc"
	"net/http"
	"regexp"
)

var factoryMap = map[string]interface{}{
	"Bird":            (*object.Bird)(nil),
	"Dog":             (*object.Dog)(nil),
	"Common":          (*object.Common)(nil),
	"Json":            (*object.Json)(nil),
	"InterceptorName": (*object.InterceptorName)(nil),
}

func handler(req *http.Request) (mvc.Route, error) {
	var route mvc.Route
	path := req.URL.Path
	pathReg := regexp.MustCompile(`^/(\w+)/(\w+)`)
	pathRes := pathReg.FindStringSubmatch(path)
	if len(pathRes) != 3 {
		return route, errors.New("错误的路由")
	}
	// 通过url解析，到相关的控制器模块，控制器模块要先在工厂注册
	route.ControllerName = monster.FirstUpper(pathRes[1])
	//控制器模块对应的方法
	route.MethodName = monster.FirstUpper(pathRes[2])
	return route, nil
}

func prepare(server *mvc.Server, httpServer *http.Server) {
	//这里可以更改 http.Server 实例信息
	fmt.Println("prepare(" + server.Addr + ")")
}

//测试连接:
//http://127.0.0.1:9022/bird/fly
//http://127.0.0.1:9022/dog/run?str=a&num=1&bl=true&strArr1=bbb&strArr1=ccc&numArr1=333&numArr1=444&StrArr2=eee&StrArr2=fff&StrArr2=hhh&numArr2=123&&numArr2=456
//http://127.0.0.1:9022/bird/upload 单个上传图片, 参数file
//http://127.0.0.1:9022/bird/uploads 多个上传图片, 参数files
//http://127.0.0.1:9021/bigBird/fly 只有9021端口可以调用
func main() {
	monster.Init(factoryMap) //初始化工厂
	//9021端口添加一个拦截器
	var interceptors9021 = []mvc.Interceptor{
		monster.Factory("InterceptorName").(mvc.Interceptor),
	}
	mvc.Serve(
		&mvc.Server{Addr: ":9020", Handler: handler, Prepare: prepare},
		&mvc.Server{Addr: ":9021", Handler: handler, Prepare: prepare, Interceptors: interceptors9021},
		&mvc.Server{Addr: ":9022", Handler: handler, Prepare: prepare},
		//CertFile 和 KeyFile 同时不为空就是 https
		&mvc.Server{Addr: ":9023", Handler: handler, Prepare: prepare, CertFile: "server.crt", KeyFile: "server.key"},
	)
}
```

> ### 3. 数据库使用 [查看demo, base.go是入口文件][demoDatabase]

```text
支持数据库查询绑定到对象，数据库事务
```

```go
package main

import (
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/luoshanzhi/monster-go"
	"github.com/luoshanzhi/monster-go/database"
	"strconv"
	"time"
)

type Common struct {
}

var factoryMap = map[string]interface{}{
	"Common": (*Common)(nil),
}

func doSql(handler database.Handler) error {
	//database.Handler 满足 sql.DB 和 sql.Conn 和 sql.Tx, 所以下面的 handler都可以
	
	//执行sql：用在 insert,update,delete 等等...
	result, execErr := database.Exec(handler, "update t_user set nickName='怪兽 "+strconv.Itoa(int(time.Now().Unix()))+"' where userId=? limit 1", 1)
	rowsAffected, _ := result.RowsAffected()
	fmt.Println(rowsAffected, execErr)

	//查询多条
	var userList []struct {
		UserId   int    //如果没设置tag,就按首字母大写映射到字段
		UserName string `db:"nickName"` //指定字段映射到该字段
	}
	listErr := database.Query(handler, &userList, "select userId,nickName from t_user order by userId asc limit 1")
	fmt.Println(userList, listErr)

	//查询单条
	var userInfo struct {
		UserId   int
		NickName string
	}
	infoErr := database.QueryRow(handler, &userInfo, "select userId,nickName from t_user where userId=?", 1)
	fmt.Println(userInfo, infoErr)

	if execErr != nil || listErr != nil || infoErr != nil {
		return errors.New("出现错误")
	}
	return nil
}

func main() {
	monster.SetSettingFile("setting.json")
	monster.Init(factoryMap) //初始化工厂
	defer database.Close()
	database.Open(database.Options{
		ConnMaxLifetime: time.Minute * 3, //保存的连接最大存活3分钟就释放掉
		MaxOpenConns:    10,              //连接池,最大打开10连接,超过阻塞等待
		MaxIdleConns:    5,               //连接池,最大保存5个空闲连接
		//StatisticsLog:   true,            //开启统计日志, 记录连接池的基本状态, 文件: log/statistics.log
	})

	//database.Handler 满足 sql.DB 和 sql.Conn 和 sql.Tx, 所以下面的 handler都可以
	
	//sql.DB版本
	db := database.DB()
	doSql(db)

	//sql.Conn版本
	//sql.Conn 和 sql.DB区别在于 sql.Conn自己释放回连接池, 用在http服务里和 http.Request 上下文绑定就比较合理
	/*ctx := context.Background()
	conn, err := database.DB().Conn(ctx)
	if err != nil {
		panic(err)
	}
	doSql(conn)
	conn.Close()*/

	//sql.Tx版本(事务)
	/*db := database.DB()
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	doErr := doSql(tx)
	if doErr != nil {
		tx.Rollback()
	}
	commitErr := tx.Commit()
	fmt.Println(commitErr)*/
}
```

> ### 4. 缓存使用 [查看demo, base.go是入口文件][demoCache]

```text
支持简单缓存,只简单封装了: GET,SET,EXISTS,DEL,KEYS,TTL
有需要的，自己可以获取conn, 自定义封装
```

```go
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
		MaxIdleConns:    5,               //连接池,最大保存5个空闲连接
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

输出:
缓存设置成功
存在:  true
keys:  [user]
seconds:  5
{1 怪兽}
缓存删除成功
存在:  false
```

> ### 5. MongoDB使用 [查看demo, base.go是入口文件][demoMongoDB]

```text
简单封装了MongoDB
```

```go
package main

import (
	"context"
	"fmt"
	"github.com/luoshanzhi/monster-go"
	"github.com/luoshanzhi/monster-go/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type Common struct {
}

var factoryMap = map[string]interface{}{
	"Common": (*Common)(nil),
}

func doNoSql(db *mongo.Database) error {
	ctx := context.Background()
	coll := db.Collection("user")
	//添加
	insertResult, err := coll.InsertOne(
		ctx,
		bson.M{
			"age":  20,
			"name": "ABC",
		})
	fmt.Println("添加：", insertResult, err)
	//查询1
	cursor1, err := coll.Find(
		ctx,
		bson.D{},
	)
	var results1 []bson.M
	err = cursor1.All(ctx, &results1)
	fmt.Println("查询1：", results1, err)
	//更新
	updateResult, err := coll.UpdateOne(
		ctx,
		bson.D{
			{"name", "ABC"},
		},
		bson.D{
			{"$set", bson.D{
				{"name", "EFG"},
			}},
		},
	)
	fmt.Println("更新：", updateResult, err)
	//查询2
	cursor2, err := coll.Find(
		ctx,
		bson.D{{"name", "EFG"}},
	)
	var results2 []bson.M
	err = cursor2.All(ctx, &results2)
	fmt.Println("查询2：", results2, err)
	//删除
	deleteResult, err := coll.DeleteOne(
		ctx,
		bson.D{{"name", "EFG"}},
	)
	fmt.Println("删除：", deleteResult, err)
	//查询3
	cursor3, err := coll.Find(
		ctx,
		bson.D{},
	)
	var results3 []bson.M
	err = cursor3.All(ctx, &results3)
	fmt.Println("查询3：", results3, err)
	return nil
}

func main() {
	monster.SetSettingFile("setting.json")
	monster.Init(factoryMap) //初始化工厂
	defer mongodb.Close()
	mongodb.Open(mongodb.Options{
		ConnMaxLifetime: time.Minute * 3, //保存的连接最大存活3分钟就释放掉
		MaxOpenConns:    10,              //连接池,最大打开10连接,超过阻塞等待
		MaxIdleConns:    5,               //连接池,最大保存5个空闲连接
		//StatisticsLog:   true,            //开启统计日志, 记录连接池的基本状态, 文件: log/statistics.log
	})

	db := mongodb.DB()
	doNoSql(db)
}

输出:
添加： &{ObjectID("649a9d77865d4a9084c935db")} <nil>
查询1： [map[_id:ObjectID("649a9d77865d4a9084c935db") age:20 name:ABC]] <nil>
更新： &{1 1 0 <nil>} <nil>
查询2： [map[_id:ObjectID("649a9d77865d4a9084c935db") age:20 name:EFG]] <nil>
删除： &{1} <nil>
查询3： [] <nil>
```

[demoFactory]: https://github.com/luoshanzhi/monster-go/tree/main/demo/factory

[demoMvc]: https://github.com/luoshanzhi/monster-go/tree/main/demo/mvc

[demoDatabase]: https://github.com/luoshanzhi/monster-go/tree/main/demo/database

[demoCache]: https://github.com/luoshanzhi/monster-go/tree/main/demo/cache

[demoMongoDB]: https://github.com/luoshanzhi/monster-go/tree/main/demo/mongodb