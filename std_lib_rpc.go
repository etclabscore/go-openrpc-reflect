package openrpc_go_document

import (
	"go/token"
	"log"
	"reflect"
	"sync"
)

func DefaultSuitableCallbacksGoRPC(service interface{}) func() map[string]Callback {
	return func() map[string]Callback {
		ty := reflect.TypeOf(service)
		v := reflect.ValueOf(service)
		methods := suitableMethods(ty, true) // debug
		out := make(map[string]Callback)
		for k, vv := range methods {
			out[k] = Callback{v, vv.method.Func} // TODO
		}
		return out
	}
}

// Precompute the reflect type for error. Can't use error directly
// because Typeof takes an empty interface value. This is annoying.
var gorpcTypeOfError = reflect.TypeOf((*error)(nil)).Elem()

type gorpcMethodType struct {
	sync.Mutex // protects counters
	method     reflect.Method
	ArgType    reflect.Type
	ReplyType  reflect.Type
	numCalls   uint
}

//func (m *gorpcMethodType) CallbackWithReceiver(receiverValue reflect.Value) Callback {
//	return Callback{
//		Receiver: receiverValue,
//		Fn:       m.method.Type.,
//	}
//}

type gorpcService struct {
	name   string                      // name of service
	rcvr   reflect.Value               // receiver of methods for the service
	typ    reflect.Type                // type of the receiver
	method map[string]*gorpcMethodType // registered methods
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return token.IsExported(t.Name()) || t.PkgPath() == ""
}

// suitableMethods returns suitable Rpc methods of typ, it will report
// error using log if reportErr is true.
func suitableMethods(typ reflect.Type, reportErr bool) map[string]*gorpcMethodType {
	methods := make(map[string]*gorpcMethodType)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}
		// Method needs three ins: receiver, *args, *reply.
		if mtype.NumIn() != 3 {
			if reportErr {
				log.Printf("rpc.Register: method %q has %d input parameters; needs exactly three\n", mname, mtype.NumIn())
			}
			continue
		}
		// First arg need not be a pointer.
		argType := mtype.In(1)
		if !isExportedOrBuiltinType(argType) {
			if reportErr {
				log.Printf("rpc.Register: argument type of method %q is not exported: %q\n", mname, argType)
			}
			continue
		}
		// Second arg must be a pointer.
		replyType := mtype.In(2)
		if replyType.Kind() != reflect.Ptr {
			if reportErr {
				log.Printf("rpc.Register: reply type of method %q is not a pointer: %q\n", mname, replyType)
			}
			continue
		}
		// Reply type must be exported.
		if !isExportedOrBuiltinType(replyType) {
			if reportErr {
				log.Printf("rpc.Register: reply type of method %q is not exported: %q\n", mname, replyType)
			}
			continue
		}
		// Method needs one out.
		if mtype.NumOut() != 1 {
			if reportErr {
				log.Printf("rpc.Register: method %q has %d output parameters; needs exactly one\n", mname, mtype.NumOut())
			}
			continue
		}
		// The return type of the method must be error.
		if returnType := mtype.Out(0); returnType != gorpcTypeOfError {
			if reportErr {
				log.Printf("rpc.Register: return type of method %q is %q, must be error\n", mname, returnType)
			}
			continue
		}
		methods[mname] = &gorpcMethodType{method: method, ArgType: argType, ReplyType: replyType}
	}
	return methods
}
