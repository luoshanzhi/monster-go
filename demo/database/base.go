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
