package go_openrpc_reflect

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"unicode"

	"github.com/davecgh/go-spew/spew"
	goopenrpcT "github.com/gregdhill/go-openrpc/types"
)

/*
EthereumRPCDescriptor provides an instance of a reflection service provider
that satisfies the defaults needed to reflect a github.com/ethereum/go-ethereum/rpc package-based
RPC service into an OpenRPC Document.

Background:

go-ethereum/rpc is a popular RPC library with offers an alternative
design compared to the standard library's offering.
The conventions it uses to determine valid API methods from go declarations is
different; go requires a strict signature structure, while go-ethereum/rpc
allows a wider and different variety and rules.
To understand the difference, compare the values of EthereumCallbackToMethod
and StandardCallbackToMethod, their API method naming algorithms (ie 'a_b' vs 'A.B'),
and their eventual assembly into openrpc method types.
*/
var EthereumRPCDescriptor = &ReceiverServiceDescriptorT{
	ProviderParseOptions:               EthereumParseOptions(),
	ServiceCallbacksFullyQualifiedName: EthereumMethodName,
	ServiceCallbacksFromReceiverFn:     EthereumCallbacks,
	ServiceCallbackToMethodFn:          EthereumCallbackToMethod,
}

// EthereumCallbacks provides a set of ethereum/go-ethereum/rpc default callbacks.
// This function and associated function duplicate that library's business logic in order
// to pass openrpc-ready names and methods to the openrpc document library.
//
//
var EthereumCallbacks = func(service interface{}) map[string]Callback {
	v := reflect.ValueOf(service)
	callbacks := suitableCallbacks(v)
	out := make(map[string]Callback)
	for k, v := range callbacks {
		out[k] = *v.Callback()
	}
	return out
}

/*
EthereumMethodName replicates the default method naming logic of the
go-ethereum/rpc library. The actual logic also handles '.` joining module and method names,
but this is optional. Supporting the documentation of this feature
would require essentially duplicating the method list, and I'm favoring human readability
over technical conceptual completeness.
*/
var EthereumMethodName = func(receiver interface{}, receiverName, methodName string) string {
	if receiverName != "" {
		return receiverName + "_" + methodName
	}
	ty := reflect.TypeOf(receiver)
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	return formatEthereumName(ty.Name()) + "_" + methodName
}

// EthereumCallbackToMethod will parse a method to an openrpc method.
// Note that this will collect only the broad strokes:
// - all args, result[0] values => params, result
//
// ContentDescriptors and/or Schema filters must be applied separately.
var EthereumCallbackToMethod = func(opts *DocumentProviderParseOpts, name string, cb Callback) (*goopenrpcT.Method, error) {
	pcb, err := newParsedCallback(cb)
	if err != nil {
		if strings.Contains(err.Error(), "autogenerated") {
			return nil, errParseCallbackAutoGenerate
		}
		log.Println("parse ethereumCallback", err)
		return nil, err
	}
	method, err := makeEthereumMethod(opts, name, pcb)
	if err != nil {
		return nil, fmt.Errorf("make method error method=%s cb=%s error=%v", name, spew.Sdump(cb), err)
	}
	return method, nil
}

// EthereumParseOptions modifies the default parse options to
// skip context argment type (if in position 1), and to skip the return value
// if it's an error.
var EthereumParseOptions = func() *DocumentProviderParseOpts {
	opts := DefaultParseOptions()
	opts.ContentDescriptorTypeSkipFn = func(isArgs bool, index int, ty reflect.Type, cd *goopenrpcT.ContentDescriptor) bool {
		if isArgs && index == 0 && isContextType(ty) {
			return true
		}
		if !isArgs && isErrorType(ty) {
			return true
		}
		return false
	}
	return opts
}

// ethereumCallback is a method ethereumCallback which was registered in the server
type ethereumCallback struct {
	fn       reflect.Value  // the function
	rcvr     reflect.Value  // receiver object of method, set if fn is method
	argTypes []reflect.Type // input argument types
	retTypes []reflect.Type // return types
	hasCtx   bool           // method's first argument is a context (not included in argTypes)
	errPos   int            // err return idx, of -1 when method cannot return error
}

func (e *ethereumCallback) Callback() *Callback {
	return &Callback{
		Receiver: e.rcvr,
		Fn:       e.fn,
	}
}

func makeEthereumMethod(opts *DocumentProviderParseOpts, name string, pcb *parsedCallback) (*goopenrpcT.Method, error) {

	argTypes := pcb.cb.getArgTypes()
	retTyptes := pcb.cb.getRetTypes()

	argASTFields := []*NamedField{}
	if pcb.fdecl.Type != nil &&
		pcb.fdecl.Type.Params != nil &&
		pcb.fdecl.Type.Params.List != nil {
		for _, f := range pcb.fdecl.Type.Params.List {
			argASTFields = append(argASTFields, expandASTField(f)...)
		}
	}

	retASTFields := []*NamedField{}
	if pcb.fdecl.Type != nil &&
		pcb.fdecl.Type.Results != nil &&
		pcb.fdecl.Type.Results.List != nil {
		for _, f := range pcb.fdecl.Type.Results.List {
			retASTFields = append(retASTFields, expandASTField(f)...)
		}
	}

	params := func(skipFn func(isArgs bool, index int, ty reflect.Type, descriptor *goopenrpcT.ContentDescriptor) bool) ([]*goopenrpcT.ContentDescriptor, error) {
		out := []*goopenrpcT.ContentDescriptor{}
		for i, a := range argTypes {
			cd, err := opts.contentDescriptor(a, argASTFields[i])
			if err != nil {
				return nil, err
			}
			if skipFn != nil && skipFn(true, i, a, cd) {
				continue
			}
			for _, fn := range opts.ContentDescriptorMutationFns {
				fn(true, i, cd)
			}
			out = append(out, cd)
		}
		return out, nil
	}

	rets := func(skipFn func(isArgs bool, index int, ty reflect.Type, descriptor *goopenrpcT.ContentDescriptor) bool) ([]*goopenrpcT.ContentDescriptor, error) {
		out := []*goopenrpcT.ContentDescriptor{}
		for i, r := range retTyptes {
			cd, err := opts.contentDescriptor(r, retASTFields[i])
			if err != nil {
				return nil, err
			}
			if skipFn != nil && skipFn(false, i, r, cd) {
				continue
			}
			for _, fn := range opts.ContentDescriptorMutationFns {
				fn(false, i, cd)
			}
			out = append(out, cd)
		}
		if len(out) == 0 {
			out = append(out, nullContentDescriptor)
		}
		return out, nil
	}

	collectedParams, err := params(opts.ContentDescriptorTypeSkipFn)
	if err != nil {
		return nil, err
	}
	collectedResults, err := rets(opts.ContentDescriptorTypeSkipFn)
	if err != nil {
		return nil, err
	}
	res := collectedResults[0] // OpenRPC Document specific (can has only one result).

	return makeMethod(name, pcb, collectedParams, res), nil
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
	c := &ethereumCallback{fn: fn, rcvr: receiver, errPos: -1}
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
	if fntype.NumIn() > firstArg && fntype.In(firstArg) == contextType {
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

// formatName converts to first character of name to lowercase.
func formatEthereumName(name string) string {
	ret := []rune(name)
	if len(ret) > 0 {
		ret[0] = unicode.ToLower(ret[0])
	}
	return string(ret)
}
