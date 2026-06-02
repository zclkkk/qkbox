package api

const MethodHello = "hello"

var MethodRegistry = map[string]struct{}{
	MethodHello: {},
}
