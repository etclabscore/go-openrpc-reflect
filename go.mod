module github.com/etclabscore/go-openrpc-reflect

go 1.13

// require github.com/open-rpc/meta-schema v0.0.43
require github.com/open-rpc/meta-schema v0.0.0-20201023174056-aa7982132ac2

require (
	github.com/alecthomas/jsonschema v0.0.0-20200530073317-71f438968921
	github.com/etclabscore/go-jsonschema-walk v0.0.6
	github.com/ethereum/go-ethereum v1.9.12
	github.com/go-openapi/spec v0.19.11
	github.com/go-openapi/swag v0.19.11 // indirect
	github.com/iancoleman/orderedmap v0.1.0 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/tidwall/gjson v1.6.0
	golang.org/x/net v0.0.0-20201022231255-08b38378de70 // indirect
)

//replace github.com/open-rpc/meta-schema => github.com/meowsbits/meta-schema v0.0.43

//HACKING:replace github.com/open-rpc/meta-schema => /home/ia/dev/open-rpc/meta-schema
//replace github.com/open-rpc/meta-schema => /home/ia/dev/open-rpc/meta-schema
