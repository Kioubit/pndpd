package modules

var ModuleList []*Module

type Module struct {
	Name             string
	Commands         []Command
	InitCallback     func(CallbackInfo)
	CompleteCallback func()
	ShutdownCallback func()
}

type Command struct {
	CommandText        string
	Description        string
	BlockTerminate     bool
	CommandLineEnabled bool
	ConfigEnabled      bool
}

type CallbackType int

const (
	CommandLine CallbackType = 0
	Config      CallbackType = 1
)

type CallbackInfo struct {
	CallbackType CallbackType
	Command      Command
	Arguments    []string
}

func RegisterModule(name string, commands []Command, initCallback func(CallbackInfo), CompleteCallback func(), shutdownCallback func()) {
	ModuleList = append(ModuleList, &Module{
		Name:             name,
		Commands:         commands,
		InitCallback:     initCallback,
		CompleteCallback: CompleteCallback,
		ShutdownCallback: shutdownCallback,
	})
}

func GetCommand(target string, scope CallbackType) (*Module, Command) {
	for i := range ModuleList {
		for _, command := range ModuleList[i].Commands {
			if command.CommandText == target {
				if scope == CommandLine && command.CommandLineEnabled {
					return ModuleList[i], command
				}
				if scope == Config && command.ConfigEnabled {
					return ModuleList[i], command
				}
				return nil, Command{}
			}
		}
	}
	return nil, Command{}
}

var runningModules []*Module

func ExecuteInit(module *Module, info CallbackInfo) {
	if info.Command.BlockTerminate {
		found := false
		for _, n := range runningModules {
			if n == module {
				found = true
				break
			}
		}
		if !found {
			runningModules = append(runningModules, module)
		}
	}
	module.InitCallback(info)
}

func ExecuteComplete() {
	for i := range runningModules {
		(*runningModules[i]).CompleteCallback()
	}
}

func ShutdownAll() {
	for i := range runningModules {
		(*runningModules[i]).ShutdownCallback()
	}
}

func ExistsBlockingModule() bool {
	return len(runningModules) != 0
}
