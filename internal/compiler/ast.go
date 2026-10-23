package compiler

// Chapter 1 has one function, one return statement, and a constant expression.
type Program struct {
	Function Function
}

type Function struct {
	Name string
	Body ReturnStatement
}

type ReturnStatement struct {
	Value Constant
}

type Constant struct {
	Value int32
}
