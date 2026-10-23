package compiler

// The assembly AST is separate from the C AST, so code generation and text
// emission remain distinct compiler passes.
type AssemblyProgram struct {
	Function AssemblyFunction
}

type AssemblyFunction struct {
	Name         string
	Instructions []Instruction
}

type Instruction interface{ instruction() }

type Mov struct {
	Source      Immediate
	Destination Register
}

type Ret struct{}

func (Mov) instruction() {}
func (Ret) instruction() {}

type Immediate struct{ Value int32 }
type Register string

const AX Register = "eax"

func Generate(program Program) AssemblyProgram {
	return AssemblyProgram{Function: AssemblyFunction{
		Name: program.Function.Name,
		Instructions: []Instruction{
			Mov{Source: Immediate{Value: program.Function.Body.Value.Value}, Destination: AX},
			Ret{},
		},
	}}
}
