package openrpc_go_document

import (
	"context"
	"encoding/json"
	"reflect"
	"unicode"
)

var (
	contextTypeEthereum = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorTypeEthereum           = reflect.TypeOf((*error)(nil)).Elem()
	subscriptionTypeEthereum    = reflect.TypeOf(Subscription{})
	stringType          = reflect.TypeOf("")
)

type ID string
type Subscription struct {
	ID        ID
	namespace string
	err       chan error // closed on unsubscribe
}

// Err returns a channel that is closed when the client send an unsubscribe request.
func (s *Subscription) Err() <-chan error {
	return s.err
}

// MarshalJSON marshals a subscription as its ID.
func (s *Subscription) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.ID)
}

// ethereumCallback is a method ethereumCallback which was registered in the server
type ethereumCallback struct {
	fn          reflect.Value  // the function
	rcvr        reflect.Value  // receiver object of method, set if fn is method
	argTypes    []reflect.Type // input argument types
	retTypes    []reflect.Type // return types
	hasCtx      bool           // method's first argument is a context (not included in argTypes)
	errPos      int            // err return idx, of -1 when method cannot return error
	isSubscribe bool           // true if this is a subscription ethereumCallback
}

func (e *ethereumCallback) Callback() *Callback {
	return &Callback{
		Receiver: e.rcvr,
		Fn:       e.fn,
	}
}

// DefaultServiceCallbacksEthereum provides a set of ethereum/go-ethereum/rpc default callbacks.
// This function and associated function duplicate that library's business logic in order
// to pass openrpc-ready names and methods to the openrpc document library.
//
//
func DefaultServiceCallbacksEthereum(service interface{}) func() map[string]Callback {
	return func() map[string]Callback{
		v := reflect.ValueOf(service)
		callbacks := suitableCallbacks(v)
		out := make(map[string]Callback)
		for k, v := range callbacks {
			out[k] = *v.Callback()
		}
		return out
	}
}

// suitableCallbacks iterates over the methods of the given type. It determines if a method
// satisfies the criteria for a RPC ethereumCallback or a subscription ethereumCallback and adds it to the
// collection of callbacks. See server documentation for a summary of these criteria.
func suitableCallbacks(receiver reflect.Value) map[string]*ethereumCallback {
	typ := receiver.Type()
	callbacks := make(map[string]*ethereumCallback)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		if method.PkgPath != "" {
			continue // method not exported
		}
		cb := newEthereumCallback(receiver, method.Func)
		if cb == nil {
			continue // function invalid
		}
		name := formatEthereumName(method.Name)
		callbacks[name] = cb
	}
	return callbacks
}

// newCallback turns fn (a function) into a ethereumCallback object. It returns nil if the function
// is unsuitable as an RPC ethereumCallback.
func newEthereumCallback(receiver, fn reflect.Value) *ethereumCallback {
	fntype := fn.Type()
	c := &ethereumCallback{fn: fn, rcvr: receiver, errPos: -1, isSubscribe: isPubSub(fntype)}
	// Determine parameter types. They must all be exported or builtin types.
	c.makeArgTypes()

	// Verify return types. The function must return at most one error
	// and/or one other non-error value.
	outs := make([]reflect.Type, fntype.NumOut())
	for i := 0; i < fntype.NumOut(); i++ {
		outs[i] = fntype.Out(i)
	}
	if len(outs) > 2 {
		return nil
	}
	// If an error is returned, it must be the last returned value.
	switch {
	case len(outs) == 1 && isErrorType(outs[0]):
		c.errPos = 0
	case len(outs) == 2:
		if isErrorType(outs[0]) || !isErrorType(outs[1]) {
			return nil
		}
		c.errPos = 1
	}
	return c
}

// makeArgTypes composes the argTypes list.
func (c *ethereumCallback) makeArgTypes() {
	fntype := c.fn.Type()
	// Skip receiver and context.Context parameter (if present).
	firstArg := 0
	if c.rcvr.IsValid() {
		firstArg++
	}
	if fntype.NumIn() > firstArg && fntype.In(firstArg) == contextTypeEthereum {
		c.hasCtx = true
		firstArg++
	}
	// Add all remaining parameters.
	c.argTypes = make([]reflect.Type, fntype.NumIn()-firstArg)
	for i := firstArg; i < fntype.NumIn(); i++ {
		c.argTypes[i-firstArg] = fntype.In(i)
	}
}

// makeRetTypes composes the argTypes list.
func (c *ethereumCallback) makeRetTypes() {
	fntype := c.fn.Type()
	// Add all remaining parameters.
	c.retTypes = make([]reflect.Type, fntype.NumOut())
	for i := 0; i < fntype.NumOut(); i++ {
		c.retTypes[i] = fntype.Out(i)
	}
}

// Is t context.Context or *context.Context?
func isContextType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t == contextTypeEthereum
}

// Does t satisfy the error interface?
func isErrorType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Implements(errorTypeEthereum)
}

// Is t Subscription or *Subscription?
func isSubscriptionType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t == subscriptionTypeEthereum
}

// isPubSub tests whether the given method has as as first argument a context.Context and
// returns the pair (Subscription, error).
func isPubSub(methodType reflect.Type) bool {
	// numIn(0) is the receiver type
	if methodType.NumIn() < 2 || methodType.NumOut() != 2 {
		return false
	}
	return isContextType(methodType.In(1)) &&
		isSubscriptionType(methodType.Out(0)) &&
		isErrorType(methodType.Out(1))
}

// formatName converts to first character of name to lowercase.
func formatEthereumName(name string) string {
	ret := []rune(name)
	if len(ret) > 0 {
		ret[0] = unicode.ToLower(ret[0])
	}
	return string(ret)
}
