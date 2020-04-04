module github.com/etclabscore/openrpc-go-document

go 1.13

require (
	github.com/alecthomas/jsonschema v0.0.4
	github.com/davecgh/go-spew v1.1.1
	github.com/etclabscore/go-jsonschema-traverse v0.0.4
	github.com/go-openapi/spec v0.19.7
	github.com/gregdhill/go-openrpc v0.0.0-00010101000000-000000000000
)

replace github.com/alecthomas/jsonschema => github.com/meowsbits/jsonschema v0.0.4

replace github.com/etclabscore/go-jsonschema-traverse => github.com/meowsbits/go-jsonschema-traverse v0.0.4

replace github.com/gregdhill/go-openrpc => github.com/meowsbits/go-openrpc v0.0.1
