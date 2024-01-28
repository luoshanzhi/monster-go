package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/luoshanzhi/monster-go"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	dbMap      = map[string]*dbStore{}
	dbMapGuard sync.RWMutex
)

var pick = func(dbKey string, dbType string, dbs []*sql.DB) (*sql.DB, error) {
	if dbType != "master" && dbType != "slave" {
		return nil, errors.New("dbType error")
	}
	if len(dbs) == 0 {
		return nil, errors.New("dbs len is 0")
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
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		return dbs[r.Intn(len(dbs))], nil
	} else {
		return maxCanDB, nil
	}
}

func SetPick(pk func(dbKey string, dbType string, dbs []*sql.DB) (*sql.DB, error)) {
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

func Open(options Options, dbKey ...string) {
	OpenMaster(options, dbKey...)
	OpenSlave(options, dbKey...)
}

func OpenMaster(options Options, dbKey ...string) {
	BaseOpen("master", options, dbKey...)
}

func OpenSlave(options Options, dbKey ...string) {
	BaseOpen("slave", options, dbKey...)
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

func BaseOpen(dbType string, options Options, dbKey ...string) {
	if dbType != "master" && dbType != "slave" {
		panic("dbType error")
	}
	dbKey_ := getDbKey(dbKey...)
	sqlConfig, ok := monster.CurEnvConfig.Sql[dbKey_]
	if !ok {
		panic("dbKey error")
	}
	var dbArr []monster.SqlSettingItem
	if dbType == "master" {
		dbArr = sqlConfig.Master
	} else if dbType == "slave" {
		dbArr = sqlConfig.Slave
	}
	if len(dbArr) == 0 {
		panic("config error")
	}
	var dbs []*sql.DB
	for _, item := range dbArr {
		driverName := item.DriverName
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
		db, err := sql.Open(driverName, user+":"+password+"@tcp("+host+":"+strconv.Itoa(port)+")/"+dBase+"?charset="+charset+interpolateParams)
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
	//防止并发写map异常
	dbMapGuard.Lock()
	if _, ok := dbMap[dbKey_]; !ok {
		dbMap[dbKey_] = &dbStore{}
	}
	dbSt := dbMap[dbKey_]
	if dbType == "master" {
		dbSt.masters = dbs
		monster.CommonLog.Info("数据库(" + dbKey_ + "): 主库启动成功")
	} else if dbType == "slave" {
		dbSt.slaves = dbs
		monster.CommonLog.Info("数据库(" + dbKey_ + "): 从库启动成功")
	}
	dbMapGuard.Unlock()
	if options.StatisticsLog {
		statisticsLogDuration := options.StatisticsLogDuration
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
					WithField("name", "sql-"+dbKey_+"-"+dbType).
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
		var dbs []*sql.DB
		if dbType == "master" {
			dbs = dbSt.masters
		} else if dbType == "slave" {
			dbs = dbSt.slaves
		}
		for _, db := range dbs {
			db.Close()
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
	var dbs []*sql.DB
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
		stats := db.Stats()
		statistics.Use += stats.InUse
		statistics.Idle += stats.Idle
	}
	return statistics, nil
}

func DB(dbKey ...string) *sql.DB {
	return Master(dbKey...)
}

func Master(dbKey ...string) *sql.DB {
	dbKey_ := getDbKey(dbKey...)
	var dbs []*sql.DB
	//防止并发写map异常
	dbMapGuard.RLock()
	if dbSt, ok := dbMap[dbKey_]; ok {
		dbs = dbSt.masters
	}
	dbMapGuard.RUnlock()
	db, _ := pick(dbKey_, "master", dbs)
	return db
}

func Slave(dbKey ...string) *sql.DB {
	dbKey_ := getDbKey(dbKey...)
	var dbs []*sql.DB
	//防止并发写map异常
	dbMapGuard.RLock()
	if dbSt, ok := dbMap[dbKey_]; ok {
		dbs = dbSt.slaves
	}
	dbMapGuard.RUnlock()
	db, _ := pick(dbKey_, "slave", dbs)
	return db
}

func Query(handler Handler, col interface{}, query string, args ...interface{}) error {
	return QueryContext(context.Background(), handler, col, query, args...)
}

func QueryRow(handler Handler, col interface{}, query string, args ...interface{}) error {
	return QueryRowContext(context.Background(), handler, col, query, args...)
}

func Exec(handler Handler, query string, args ...interface{}) (sql.Result, error) {
	return ExecContext(context.Background(), handler, query, args...)
}

func Prepare(handler Handler, query string) (*sql.Stmt, error) {
	return PrepareContext(context.Background(), handler, query)
}

func QueryContext(ctx context.Context, handler Handler, col interface{}, query string, args ...interface{}) error {
	if handler == nil {
		return errors.New("handler is nil")
	}
	colValueElem, colItemType, colItemTagMap, reflectErr := colReflect(col)
	if reflectErr != nil {
		return reflectErr
	}
	monster.CommonLog.Trace("sql("+fmt.Sprintf("%p", handler)+"):", query)
	rows, err := handler.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	scan_, scanErr := rowsScan(rows, colItemType, colItemTagMap)
	if scanErr != nil {
		return scanErr
	}
	for rows.Next() {
		appendErr := colAppend(rows, &scan_, colValueElem, colItemType)
		if appendErr != nil {
			return appendErr
		}
	}
	return nil
}

func QueryRowContext(ctx context.Context, handler Handler, col interface{}, query string, args ...interface{}) error {
	if handler == nil {
		return errors.New("handler is nil")
	}
	colValueElem, colItemType, colItemTagMap, reflectErr := colReflect(col)
	if reflectErr != nil {
		return reflectErr
	}
	monster.CommonLog.Trace("sql("+fmt.Sprintf("%p", handler)+"):", query)
	rows, err := handler.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	scan_, scanErr := rowsScan(rows, colItemType, colItemTagMap)
	if scanErr != nil {
		return scanErr
	}
	if rows.Next() {
		appendErr := colAppend(rows, &scan_, colValueElem, colItemType)
		if appendErr != nil {
			return appendErr
		}
	} else {
		return errors.New("not exists")
	}
	return nil
}

func ExecContext(ctx context.Context, handler Handler, query string, args ...interface{}) (sql.Result, error) {
	if handler == nil {
		return nil, errors.New("handler is nil")
	}
	monster.CommonLog.Trace("sql("+fmt.Sprintf("%p", handler)+"):", query)
	return handler.ExecContext(ctx, query, args...)
}

func PrepareContext(ctx context.Context, handler Handler, query string) (*sql.Stmt, error) {
	monster.CommonLog.Trace("sql("+fmt.Sprintf("%p", handler)+"):", query)
	return handler.PrepareContext(ctx, query)
}

func StmtQueryContext(ctx context.Context, stmt *sql.Stmt, col interface{}, args ...interface{}) error {
	if stmt == nil {
		return errors.New("stmt is nil")
	}
	colValueElem, colItemType, colItemTagMap, reflectErr := colReflect(col)
	if reflectErr != nil {
		return reflectErr
	}
	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	scan_, scanErr := rowsScan(rows, colItemType, colItemTagMap)
	if scanErr != nil {
		return scanErr
	}
	for rows.Next() {
		appendErr := colAppend(rows, &scan_, colValueElem, colItemType)
		if appendErr != nil {
			return appendErr
		}
	}
	return nil
}

func StmtQueryRowContext(ctx context.Context, stmt *sql.Stmt, col interface{}, args ...interface{}) error {
	if stmt == nil {
		return errors.New("stmt is nil")
	}
	colValueElem, colItemType, colItemTagMap, reflectErr := colReflect(col)
	if reflectErr != nil {
		return reflectErr
	}
	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	scan_, scanErr := rowsScan(rows, colItemType, colItemTagMap)
	if scanErr != nil {
		return scanErr
	}
	if rows.Next() {
		appendErr := colAppend(rows, &scan_, colValueElem, colItemType)
		if appendErr != nil {
			return appendErr
		}
	} else {
		return errors.New("not exists")
	}
	return nil
}

func StmtExecContext(ctx context.Context, stmt *sql.Stmt, args ...interface{}) (sql.Result, error) {
	return stmt.ExecContext(ctx, args...)
}

func colReflect(col interface{}) (colValueElem reflect.Value, colItemType reflect.Type, colItemTagMap map[string]string, err error) {
	colValue := reflect.ValueOf(col)
	if colValue.Kind() != reflect.Ptr {
		err = errors.New("col is not ptr")
		return
	}
	if colValue.IsNil() {
		err = errors.New("col is nil")
		return
	}
	colValueElem = colValue.Elem()
	colItemType = colValueElem.Type()
	if colItemType.Kind() == reflect.Slice {
		//获取切片里面item类型
		colItemType = colItemType.Elem()
	}
	if colItemType.Kind() != reflect.Struct {
		err = errors.New("colItem is not struct")
		return
	}
	colItemTagMap = make(map[string]string)
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
	return
}

func rowsScan(rows *sql.Rows, colItemType reflect.Type, colItemTagMap map[string]string) (scan, error) {
	var scan_ scan
	columns, columnErr := rows.Columns()
	if columnErr != nil {
		return scan_, columnErr
	}
	length := len(columns)
	dest := make([]interface{}, length)
	column := make([]string, length)
	for i, item := range columns {
		if name, ok := colItemTagMap[item]; ok {
			item = name
		} else {
			//字段没设置tag,就按首字母大写找字段
			item = monster.FirstUpper(item)
		}
		field, ok := colItemType.FieldByName(item)
		if !ok {
			return scan_, errors.New(item + " is not in col")
		}
		var addr interface{}
		//防止数据库字段null出错
		switch field.Type.Kind() {
		case reflect.String:
			addr = &sql.NullString{}
		case reflect.Int, reflect.Int64:
			addr = &sql.NullInt64{}
		case reflect.Int32:
			addr = &sql.NullInt32{}
		case reflect.Int16:
			addr = &sql.NullInt16{}
		case reflect.Float32, reflect.Float64:
			addr = &sql.NullFloat64{}
		case reflect.Bool:
			addr = &sql.NullBool{}
		case reflect.Struct:
			if field.Type == reflect.TypeOf((*time.Time)(nil)).Elem() {
				addr = &sql.NullTime{}
			}
		case reflect.Uint8:
			addr = &sql.NullByte{}
		default:
			return scan_, errors.New("colField type error")
		}
		dest[i] = addr
		column[i] = item
	}
	scan_.dest = dest
	scan_.column = column
	return scan_, nil
}

func colAppend(rows *sql.Rows, scan_ *scan, colValueElem reflect.Value, colItemType reflect.Type) error {
	err := rows.Scan(scan_.dest...)
	if err != nil {
		return err
	}
	newValue := reflect.New(colItemType).Elem()
	for i, item := range scan_.column {
		colField := newValue.FieldByName(item)
		switch obj := scan_.dest[i].(type) {
		case *sql.NullString:
			colField.SetString(obj.String)
		case *sql.NullInt64:
			colField.SetInt(obj.Int64)
		case *sql.NullInt32:
			colField.SetInt(int64(obj.Int32))
		case *sql.NullInt16:
			colField.SetInt(int64(obj.Int16))
		case *sql.NullFloat64:
			colField.SetFloat(obj.Float64)
		case *sql.NullBool:
			colField.SetBool(obj.Bool)
		case *sql.NullTime:
			colField.Set(reflect.ValueOf(obj.Time))
		case *sql.NullByte:
			colField.Set(reflect.ValueOf(obj.Byte))
		}
	}
	if colValueElem.Kind() == reflect.Slice {
		colValueElem.Set(reflect.Append(colValueElem, newValue))
	} else {
		colValueElem.Set(newValue)
	}
	return nil
}
