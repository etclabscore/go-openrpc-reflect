module github.com/etclabscore/openrpc-go-document

go 1.13

require (
	github.com/alecthomas/jsonschema v0.0.2
	github.com/davecgh/go-spew v1.1.1
	github.com/etclabscore/go-jsonschema-traverse v0.0.4
	github.com/go-openapi/spec v0.19.7
	github.com/gregdhill/go-openrpc v0.0.1
	golang.org/x/tools v0.0.0-20190628153133-6cdbf07be9d0
)

replace github.com/alecthomas/jsonschema => github.com/meowsbits/jsonschema v0.0.2

replace github.com/gregdhill/go-openrpc => github.com/meowsbits/go-openrpc v0.0.1

replace github.com/etclabscore/go-jsonschema-traverse => github.com/meowsbits/go-jsonschema-traverse v0.0.4
