module github.com/etclabscore/go-openrpc-document

go 1.13

require (
	github.com/alecthomas/jsonschema v0.0.2
	github.com/davecgh/go-spew v1.1.1
	github.com/etclabscore/go-jsonschema-walk v0.0.4
	github.com/go-openapi/jsonreference v0.19.3
	github.com/go-openapi/spec v0.19.7
	github.com/gregdhill/go-openrpc v0.0.1
)

replace github.com/alecthomas/jsonschema => github.com/etclabscore/go-jsonschema-reflect v0.0.2

replace github.com/gregdhill/go-openrpc => github.com/meowsbits/go-openrpc v0.0.1
