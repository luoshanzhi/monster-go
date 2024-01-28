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
