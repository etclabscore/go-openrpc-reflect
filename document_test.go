package openrpc_go_document

type MyService struct {
}

type MyVal struct {
	Name     string
	Age      string `json:"age"`
	Exists   bool
	thoughts []string
}

func (m *MyService) Fetch() (result *MyVal, err error) {
	return &MyVal{}, nil
}

func Test