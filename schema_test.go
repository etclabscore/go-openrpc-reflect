package go_openrpc_reflect

import (
	"reflect"
	"testing"
)

func TestFullTypeDescription(t *testing.T) {

	type mystruct struct {}

	cases := map[reflect.Type]string{
		reflect.TypeOf(uint(0)): "uint",
		reflect.TypeOf(new(uint)): "*uint",
		reflect.TypeOf(new(mystruct)): "github.com/etclabscore/go-openrpc-reflect.*mystruct",
		reflect.TypeOf(mystruct{}): "github.com/etclabscore/go-openrpc-reflect.mystruct",
	}

	for k, v := range cases {
		got := fullTypeDescription(k)
		if got != v {
			t.Error("got", got, "want", v)
		}
	}

}
