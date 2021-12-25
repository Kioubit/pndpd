package modules

var ModuleList []*Module

type Module struct {
	Name                string
	Option              string
	OptionDescription   string
	CommandLineCallback func([]string)
	ConfigCallback      func([]string)
}

func RegisterModule(name string, option string, description string, commandLineCallback func([]string), configCallback func([]string)) {
	ModuleList = append(ModuleList, &Module{
		Name:                name,
		Option:              option,
		OptionDescription:   description,
		CommandLineCallback: commandLineCallback,
		ConfigCallback:      configCallback,
	})
}
