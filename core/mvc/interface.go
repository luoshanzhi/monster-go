package mvc

import (
	"net/http"
	"reflect"
)

type OPTIONS interface {
	Method
}
type GET interface {
	Method
}
type HEAD interface {
	Method
}
type POST interface {
	Method
}
type PUT interface {
	Method
}
type DELETE interface {
	Method
}
type TRACE interface {
	Method
}
type CONNECT interface {
	Method
}
type Method interface {
	Method()
}
type Interceptor interface {
	Before(w http.ResponseWriter, req *http.Request, throughout []reflect.Value, throughoutIndex int) interface{}
	Invoke(w http.ResponseWriter, req *http.Request, method reflect.Value, in []reflect.Value, throughout []reflect.Value, throughoutIndex int) interface{}
	After(w http.ResponseWriter, req *http.Request, ret reflect.Value, throughout []reflect.Value, throughoutIndex int) interface{}
}
type View interface {
	Out(http.ResponseWriter, *http.Request)
}
