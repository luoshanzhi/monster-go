package object

import (
	"github.com/luoshanzhi/monster-go/demo/controller"
	"github.com/luoshanzhi/monster-go/demo/module"
	"github.com/luoshanzhi/monster-go/demo/view"
)

var FactoryMap = map[string]interface{}{
	"controller/Bird": (*controller.Bird)(nil),
	"controller/Dog":  (*controller.Dog)(nil),
	"view/Json":       (*view.Json)(nil),
	"module/Common":   (*module.Common)(nil),
}
