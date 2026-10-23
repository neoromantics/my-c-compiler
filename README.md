# My C compiler

A Go implementation of Chapter 1, “A Minimal Compiler,” from Nora Sandler's
[*Writing a C Compiler*](https://norasandler.com/book/).

The supported program is a single function with no parameters and one return
statement whose expression is a nonnegative decimal integer constant:

```c
int main(void) {
    return 42;
}
```

Function names may be any identifier. Constants must fit in a signed 32-bit
integer. Building an executable requires a function named `main`. Operators,
variables, multiple statements, and additional functions belong to later chapters.

## Build and use

Requirements: Go 1.24 or newer and `gcc` on the PATH (Apple's Clang alias works).
The compiler emits x86-64 assembly for macOS or Linux. Running the resulting
executables requires x86-64 Linux/macOS, or Rosetta on an Apple Silicon Mac.
On macOS, the driver explicitly passes `-arch x86_64` to the system toolchain;
on Linux it passes `-m64`.

```sh
go build -o mycc .
./mycc examples/return_42.c
./examples/return_42
echo $? # 42
```

The executable is written beside the source, with the `.c` extension removed.
Preprocessing and assembly intermediates are cleaned up automatically.

```sh
./mycc --lex examples/return_42.c
./mycc --parse examples/return_42.c
./mycc --codegen examples/return_42.c
./mycc -S examples/return_42.c
```

The three long flags stop after their respective passes, report errors with a
nonzero exit status, and produce no output files. `-S` writes readable assembly
to `examples/return_42.s` without assembling or linking. Flags may appear before
or after the input filename. `--help` prints usage.

## Compiler passes

| File | Responsibility |
| --- | --- |
| `main.go` | Parse command-line arguments; preprocess using `gcc -E -P`; run compiler passes; assemble and link using `gcc`. |
| `internal/compiler/lexer.go` | Recognize keywords, identifiers, integer constants, and punctuation. |
| `internal/compiler/ast.go` | Define the C abstract syntax tree. |
| `internal/compiler/parser.go` | Parse the token stream with recursive descent and reject trailing tokens. |
| `internal/compiler/codegen.go` | Convert the C AST into an assembly AST containing `Mov` and `Ret`. |
| `internal/compiler/emit.go` | Render AT&T assembly, including macOS symbol prefixes and Linux's non-executable-stack directive. |

Comments and preprocessor directives are handled by the system preprocessor.
Lexer/parser diagnostic locations refer to its output.

## Tests

```sh
go test ./...
go vet ./...

# Run the existing book test suite from this directory.
python3 writing-a-c-compiler-tests/test_compiler ./mycc --chapter 1 --stage lex
python3 writing-a-c-compiler-tests/test_compiler ./mycc --chapter 1 --stage parse
python3 writing-a-c-compiler-tests/test_compiler ./mycc --chapter 1 --stage codegen
python3 writing-a-c-compiler-tests/test_compiler ./mycc --chapter 1
```

The Go tests cover invalid tokens and syntax, truncated input, integer bounds,
both assembly formats, preprocessing, stage flags, paths containing spaces,
failed-build cleanup, and executable generation. On Apple Silicon without
Rosetta, the executable test builds and inspects the x86-64 binary, then skips
execution. The book's full test run requires the ability to execute x86-64
binaries; its intermediate-stage tests do not.

For an environment that restricts writes to this workspace, prefix Go commands
with `GOCACHE="$PWD/.cache/go-build"`.
