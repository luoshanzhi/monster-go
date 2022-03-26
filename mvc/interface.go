package mvc

import (
	"net/http"
	"reflect"
)

type OPTIONS interface{}
type GET interface{}
type HEAD interface{}
type POST interface{}
type PUT interface{}
type DELETE interface{}
type TRACE interface{}
type CONNECT interface{}

type Interceptor interface {
	Before(w http.ResponseWriter, req *http.Request, throughout []reflect.Value, throughoutIndex int) interface{}
	Invoke(w http.ResponseWriter, req *http.Request, method reflect.Value, in []reflect.Value, throughout []reflect.Value, throughoutIndex int) interface{}
	After(w http.ResponseWriter, req *http.Request, ret reflect.Value, throughout []reflect.Value, throughoutIndex int) interface{}
}
type View interface {
	Out(http.ResponseWriter, *http.Request) error
}
