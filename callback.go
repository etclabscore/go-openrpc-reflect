package go_openrpc_reflect

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"reflect"
	"runtime"
)

// Callback defines a basic type of <|receiver>:<method>
// which will be parsed by reflect, runtime, and ast.
// A Callback will eventually turn into an openrpc Method type.
type Callback struct {
	Receiver, Fn reflect.Value
}

type parsedCallback struct {
	cb       *Callback
	runtimeF *runtime.Func
	fdecl    *ast.FuncDecl
	printed  []byte
}

func (cb *Callback) Rcvr() reflect.Value {
	return cb.Receiver
}

func (cb *Callback) Func() reflect.Value {
	if cb.Receiver.IsValid() {
		return cb.Fn
	}

	return cb.Receiver
}

func (cb *Callback) String() string {
	return cb.Func().Type().String()
}

func (cb *Callback) HasReceiver() bool {
	if cb.Receiver == NoReceiverValue || reflect.TypeOf(cb.Receiver) == NoReceiver {
		return false
	}

	// Optimistic receiver
	return true
}

func (cb *Callback) HasContext() bool {
	fntype := cb.Func().Type()
	// Skip receiver and context.Context parameter (if present).
	firstArg := 0
	if cb.HasReceiver() {
		firstArg++
	}
	if fntype.NumIn() > firstArg && fntype.In(firstArg) == contextType {
		return true
	}
	return false
}

func (cb *Callback) getArgTypes() (argTypes []reflect.Type) {
	fntype := cb.Func().Type()

	// Skip receiver if present.
	firstArg := 0
	if cb.HasReceiver() {
		firstArg++
	}
	//
	argTypes = make([]reflect.Type, fntype.NumIn()-firstArg)
	for i := firstArg; i < fntype.NumIn(); i++ {
		argTypes[i-firstArg] = fntype.In(i)
	}
	return
}

func (cb *Callback) getRetTypes() (retTypes []reflect.Type) {
	fntype := cb.Func().Type()
	// Add all remaining parameters.
	retTypes = make([]reflect.Type, fntype.NumOut())
	for i := 0; i < fntype.NumOut(); i++ {
		retTypes[i] = fntype.Out(i)
	}
	return
}

func newParsedCallback(cb Callback) (*parsedCallback, error) {
	rcvrVal, fnVal := cb.Rcvr(), cb.Func()

	runtimeFunc := runtime.FuncForPC(cb.Func().Pointer())
	runtimeFile, _ := runtimeFunc.FileLine(runtimeFunc.Entry())

	tokenFileSet := token.NewFileSet()
	astFile, err := parser.ParseFile(tokenFileSet, runtimeFile, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	astFuncDecl := documentGetAstFunc(Callback{rcvrVal, fnVal}, astFile, runtimeFunc)
	if astFuncDecl == nil {
		return nil, fmt.Errorf("nil ast func cb=%v", cb)
	}

	pcb := &parsedCallback{
		cb:       &cb,
		runtimeF: runtimeFunc,
		fdecl:    astFuncDecl,
		printed: []byte{},
	}

	out := []byte{}
	buf := bytes.NewBuffer(out)
	err = printer.Fprint(buf, tokenFileSet, astFuncDecl)
	if err != nil {
		return nil, err
	}
	pcb.printed = buf.Bytes()

	return pcb, nil
}
