package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/luoshanzhi/monster-go"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var masters []*sql.DB
var slaves []*sql.DB
var pick = func(dbType string, dbs []*sql.DB) (*sql.DB, error) {
	if dbType != "master" && dbType != "slave" {
		return nil, errors.New("dbType参数错误")
	}
	if len(dbs) == 0 {
		return nil, errors.New("dbs为空")
	}
	var maxCan = -1
	var maxCanDB *sql.DB
	uniMap := make(map[int]bool)
	for _, item := range dbs {
		stats := item.Stats()
		maxOpenConnections := stats.MaxOpenConnections
		inUse := stats.InUse
		can := maxOpenConnections - inUse
		uniMap[can] = true
		if maxCan == -1 || can > maxCan {
			maxCan = can
			maxCanDB = item
		}
	}
	if len(uniMap) == 1 {
		//所有都相等就随机选一个
		rand.Seed(time.Now().UnixNano())
		return dbs[rand.Intn(len(dbs))], nil
	} else {
		return maxCanDB, nil
	}
}

func SetPick(pk func(dbType string, dbs []*sql.DB) (*sql.DB, error)) {
	if pk != nil {
		pick = pk
	}
}

func Open(options Options) {
	OpenMaster(options)
	OpenSlave(options)
}

func OpenMaster(options Options) {
	BaseOpen("master", options)
}

func OpenSlave(options Options) {
	BaseOpen("slave", options)
}

func Close() {
	CloseMaster()
	CloseSlave()
}

func CloseMaster() {
	BaseClose("master")
}

func CloseSlave() {
	BaseClose("slave")
}

func BaseOpen(dbType string, options Options) {
	if dbType != "master" && dbType != "slave" {
		panic("dbType参数错误")
	}
	mysqlConfig := monster.CurEnvConfig.Mysql
	var dbArr []monster.MysqlSettingItem
	if dbType == "master" {
		dbArr = mysqlConfig.Master
	} else if dbType == "slave" {
		dbArr = mysqlConfig.Slave
	}
	if len(dbArr) == 0 {
		panic("数据库配置异常")
	}
	var dbs []*sql.DB
	for _, item := range dbArr {
		host := item.Host
		user := item.User
		password := item.Password
		dBase := item.Database
		port := item.Port
		charset := strings.TrimSpace(options.Charset)
		interpolateParams := ""
		if charset == "" {
			charset = "utf8"
		}
		if options.InterpolateParams {
			interpolateParams = "&interpolateParams=true"
		}
		db, err := sql.Open("mysql", user+":"+password+"@tcp("+host+":"+strconv.Itoa(port)+")/"+dBase+"?charset="+charset+interpolateParams)
		if err != nil {
			panic(err)
		}
		//设置<=0数，将不限制时间
		db.SetConnMaxLifetime(options.ConnMaxLifetime)
		db.SetMaxOpenConns(options.MaxOpenConns)
		db.SetMaxIdleConns(options.MaxIdleConns)
		err = db.Ping()
		if err != nil {
			panic(err)
		}
		dbs = append(dbs, db)
	}
	if dbType == "master" {
		masters = dbs
		monster.CommonLog.Info("数据库: 主库启动成功")
	} else if dbType == "slave" {
		slaves = dbs
		monster.CommonLog.Info("数据库: 从库启动成功")
	}
	if options.StatisticsLog {
		statisticsLogDuration := options.StatisticsLogDuration
		if statisticsLogDuration <= 0 {
			statisticsLogDuration = time.Second * 5
		}
		go func() {
			for {
				stats, err := BaseStats(dbType)
				if err != nil {
					return
				}
				monster.StatisticsLog.
					WithField("name", "database-"+dbType).
					WithField("use", stats.Use).
					WithField("idle", stats.Idle).
					Info()
				time.Sleep(statisticsLogDuration)
			}
		}()
	}
}

func BaseClose(dbType string) {
	if dbType != "master" && dbType != "slave" {
		panic("dbType参数错误")
	}
	var dbs []*sql.DB
	if dbType == "master" {
		dbs = masters
	} else if dbType == "slave" {
		dbs = slaves
	}
	for _, db := range dbs {
		db.Close()
	}
	if dbType == "master" {
		masters = nil
	} else if dbType == "slave" {
		slaves = nil
	}
}

func Stats() (Statistics, error) {
	var statistics Statistics
	masterStats, masterErr := MasterStats()
	if masterErr != nil {
		return statistics, masterErr
	}
	slaveStats, slaveErr := SlaveStats()
	if slaveErr != nil {
		return statistics, slaveErr
	}
	statistics.Use += masterStats.Use
	statistics.Idle += masterStats.Idle
	statistics.Use += slaveStats.Use
	statistics.Idle += slaveStats.Idle
	return statistics, nil
}

func MasterStats() (Statistics, error) {
	return BaseStats("master")
}

func SlaveStats() (Statistics, error) {
	return BaseStats("slave")
}

func BaseStats(dbType string) (Statistics, error) {
	var statistics Statistics
	var dbs []*sql.DB
	if dbType == "master" {
		dbs = masters
	} else if dbType == "slave" {
		dbs = slaves
	}
	if len(dbs) == 0 {
		return statistics, errors.New("数据库不存在")
	}
	for _, db := range dbs {
		stats := db.Stats()
		statistics.Use += stats.InUse
		statistics.Idle += stats.Idle
	}
	return statistics, nil
}

func DB() *sql.DB {
	return Master()
}

func Master() *sql.DB {
	db, _ := pick("master", masters)
	return db
}

func Slave() *sql.DB {
	db, _ := pick("slave", slaves)
	return db
}

func Query(handler Handler, col interface{}, query string, args ...interface{}) error {
	return BaseQuery(handler, false, col, query, args...)
}

func QueryRow(handler Handler, col interface{}, query string, args ...interface{}) error {
	return BaseQueryRow(handler, false, col, query, args...)
}

func Exec(handler Handler, query string, args ...interface{}) (sql.Result, error) {
	return BaseExec(handler, false, query, args...)
}

func PrepareQuery(handler Handler, col interface{}, query string, args ...interface{}) error {
	return BaseQuery(handler, true, col, query, args...)
}
func PrepareQueryRow(handler Handler, col interface{}, query string, args ...interface{}) error {
	return BaseQueryRow(handler, true, col, query, args...)
}

func PrepareExec(handler Handler, query string, args ...interface{}) (sql.Result, error) {
	return BaseExec(handler, true, query, args...)
}

func QueryContext(ctx context.Context, handler Handler, col interface{}, query string, args ...interface{}) error {
	return BaseQueryContext(ctx, handler, false, col, query, args...)
}

func QueryRowContext(ctx context.Context, handler Handler, col interface{}, query string, args ...interface{}) error {
	return BaseQueryRowContext(ctx, handler, false, col, query, args...)
}

func ExecContext(ctx context.Context, handler Handler, query string, args ...interface{}) (sql.Result, error) {
	return BaseExecContext(ctx, handler, false, query, args...)
}

func PrepareQueryContext(ctx context.Context, handler Handler, col interface{}, query string, args ...interface{}) error {
	return BaseQueryContext(ctx, handler, true, col, query, args...)
}
func PrepareQueryRowContext(ctx context.Context, handler Handler, col interface{}, query string, args ...interface{}) error {
	return BaseQueryRowContext(ctx, handler, true, col, query, args...)
}

func PrepareExecContext(ctx context.Context, handler Handler, query string, args ...interface{}) (sql.Result, error) {
	return BaseExecContext(ctx, handler, true, query, args...)
}

func BaseQuery(handler Handler, prepare bool, col interface{}, query string, args ...interface{}) error {
	return BaseQueryContext(context.Background(), handler, prepare, col, query, args...)
}

func BaseQueryRow(handler Handler, prepare bool, col interface{}, query string, args ...interface{}) error {
	return BaseQueryRowContext(context.Background(), handler, prepare, col, query, args...)
}

func BaseExec(handler Handler, prepare bool, query string, args ...interface{}) (sql.Result, error) {
	return BaseExecContext(context.Background(), handler, prepare, query, args...)
}

func BaseQueryContext(ctx context.Context, handler Handler, prepare bool, col interface{}, query string, args ...interface{}) error {
	if handler == nil {
		return errors.New("handler为nil")
	}
	colValue := reflect.ValueOf(col)
	if colValue.Kind() != reflect.Ptr {
		return errors.New("col不是指针")
	}
	if colValue.IsNil() {
		return errors.New("col等于nil")
	}
	colValueElem := colValue.Elem()
	//获取切片里面item类型
	colItemType := colValueElem.Type().Elem()
	if colItemType.Kind() != reflect.Struct {
		return errors.New("col里的item不是结构体")
	}
	colItemTagMap := make(map[string]string)
	colItemNumField := colItemType.NumField()
	for i := 0; i < colItemNumField; i++ {
		field := colItemType.Field(i)
		name := field.Name
		val := field.Tag.Get("db")
		if val == "" {
			continue
		}
		colItemTagMap[val] = name
	}
	monster.CommonLog.Trace("sql("+fmt.Sprintf("%p", handler)+"):", query)
	var rows *sql.Rows
	var err error
	if prepare {
		stmt, pErr := handler.PrepareContext(ctx, query)
		if pErr != nil {
			return pErr
		}
		defer stmt.Close()
		rows, err = stmt.Query(args...)
	} else {
		rows, err = handler.QueryContext(ctx, query, args...)
	}
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		columns, columnErr := rows.Columns()
		if columnErr != nil {
			return columnErr
		}
		newValue := reflect.New(colItemType).Elem()
		var dest []interface{}
		var destColumn []string
		for _, column := range columns {
			if name, ok := colItemTagMap[column]; ok {
				column = name
			} else {
				//字段没设置tag,就按首字母大写找字段
				column = monster.FirstUpper(column)
			}
			if _, ok := colItemType.FieldByName(column); ok {
				colField := newValue.FieldByName(column)
				addr := colField.Addr().Interface()
				//防止数据库字段null出错
				switch colField.Interface().(type) {
				case string:
					addr = &sql.NullString{}
				case int, int64:
					addr = &sql.NullInt64{}
				case int32:
					addr = &sql.NullInt32{}
				case int16:
					addr = &sql.NullInt16{}
				case float32, float64:
					addr = &sql.NullFloat64{}
				case bool:
					addr = &sql.NullBool{}
				case time.Time:
					addr = &sql.NullTime{}
				case byte:
					addr = &sql.NullByte{}
				}
				dest = append(dest, addr)
				destColumn = append(destColumn, column)
			}
		}
		err := rows.Scan(dest...)
		if err != nil {
			return err
		}
		for i, column := range destColumn {
			if _, ok := colItemType.FieldByName(column); ok {
				colField := newValue.FieldByName(column)
				destValue := reflect.ValueOf(dest[i])
				destObj := destValue.Elem().Interface()
				switch obj := destObj.(type) {
				case sql.NullString:
					colField.SetString(obj.String)
				case sql.NullInt64:
					colField.SetInt(obj.Int64)
				case sql.NullInt32:
					colField.SetInt(int64(obj.Int32))
				case sql.NullInt16:
					colField.SetInt(int64(obj.Int16))
				case sql.NullFloat64:
					colField.SetFloat(obj.Float64)
				case sql.NullBool:
					colField.SetBool(obj.Bool)
				case sql.NullTime:
					colField.Set(reflect.ValueOf(obj.Time))
				case sql.NullByte:
					colField.Set(reflect.ValueOf(obj.Byte))
				}
			}
		}
		colValueElem.Set(reflect.Append(colValueElem, newValue))
	}
	return nil
}

func BaseQueryRowContext(ctx context.Context, handler Handler, prepare bool, col interface{}, query string, args ...interface{}) error {
	colValue := reflect.ValueOf(col)
	if colValue.Kind() != reflect.Ptr {
		return errors.New("col不是指针")
	}
	if colValue.IsNil() {
		return errors.New("col等于nil")
	}
	colValue = colValue.Elem()
	if colValue.Kind() != reflect.Struct {
		return errors.New("col不是结构体")
	}
	colType := colValue.Type()
	sliceType := reflect.SliceOf(colType)
	sliceValue := reflect.New(sliceType).Elem()
	err := BaseQueryContext(ctx, handler, prepare, sliceValue.Addr().Interface(), query, args...)
	if err != nil {
		return err
	}
	if sliceValue.Len() > 0 {
		colValue.Set(sliceValue.Index(0))
	} else {
		return errors.New("数据不存在")
	}
	return nil
}

func BaseExecContext(ctx context.Context, handler Handler, prepare bool, query string, args ...interface{}) (sql.Result, error) {
	if handler == nil {
		return nil, errors.New("handler为nil")
	}
	monster.CommonLog.Trace("sql("+fmt.Sprintf("%p", handler)+"):", query)
	if prepare {
		stmt, err := handler.PrepareContext(ctx, query)
		if err != nil {
			return nil, err
		}
		defer stmt.Close()
		return stmt.Exec(args...)
	} else {
		return handler.ExecContext(ctx, query, args...)
	}
}
