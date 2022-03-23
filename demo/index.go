package main

import (
	"errors"
	"github.com/luoshanzhi/monster-go/core"
	"github.com/luoshanzhi/monster-go/core/cache"
	"github.com/luoshanzhi/monster-go/core/database"
	"github.com/luoshanzhi/monster-go/core/mvc"
	"github.com/luoshanzhi/monster-go/demo/object"
	"net/http"
	"regexp"
	"time"
)

func main() {
	core.Init(core.RootPath+"/demo/setting.json", core.RootPath+"/demo/log", object.FactoryMap)
	defer database.Close()
	defer cache.Close()
	database.OpenMaster(time.Minute*3, 10, 10)
	database.OpenSlave(time.Minute*3, 10, 10)
	cache.Open(time.Minute*3, 10, 10)
	mvc.Start(9020, func(req *http.Request) (mvc.Route, error) {
		var route mvc.Route
		path := req.URL.Path
		pathReg := regexp.MustCompile(`^/(\w+)/(\w+)`)
		pathRes := pathReg.FindStringSubmatch(path)
		if len(pathRes) != 3 {
			return route, errors.New("错误的路由")
		}
		route.ControllerName = "controller/" + core.FirstUpper(pathRes[1])
		route.MethodName = core.FirstUpper(pathRes[2])
		return route, nil
	})
}
