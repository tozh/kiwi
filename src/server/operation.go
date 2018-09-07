package server

type Op struct {
	Argc   int64    // count of arguments
	Argv   []string // arguments of current command
	DbId   int64
	Target int64
	Cmd    *Command
}

type OpArray struct {
	Ops   []*Op
	OpNum int64
}

// Op functions and methods
func OpCreate(argc int64, argv []string, dbid int64, target int64, cmd *Command) *Op {
	return &Op {
		argc,
		argv,
		dbid,
		target,
		cmd,
	}
}

func (op *Op) Init() {
	op.Argc = 0
	op.Argv = nil
	op.DbId = 0
	op.Target = 0
	op.Cmd = nil
}


// OpArray functions and methods
func OpArrayCreate() *OpArray {
	return &OpArray{nil, 0}
}

func (oa *OpArray) Init() {
	oa.Ops = nil
	oa.OpNum = 0
}

func (oa *OpArray) Append(argc int64, argv []string, dbid int64, target int64, cmd *Command) {
	op := OpCreate(argc, argv, dbid, target, cmd)
	oa.Ops = append(oa.Ops, op)
	oa.OpNum++
}


