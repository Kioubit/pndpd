package modules

var ModuleList []*Module

type Module struct {
	Name     string
	Option   []Option
	Callback func(Callback)
}

type Option struct {
	Option      string
	Description string
}

type CallbackType int

const (
	CommandLine CallbackType = 0
	Config      CallbackType = 1
)

type Callback struct {
	CallbackType CallbackType
	Option       string
	Arguments    []string
}

func RegisterModule(name string, option []Option, Callback func(Callback)) {
	ModuleList = append(ModuleList, &Module{
		Name:     name,
		Option:   option,
		Callback: Callback,
	})
}
