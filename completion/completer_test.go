package completion

import (
	"fmt"
	"testing"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
)

type test struct {
	input string
}

func TestCompleter(t *testing.T) {
	tests := []test{
		{
			input: "SELECT * FROM |",
		},
	}

	for _, test := range tests {
		text, caretOffset := catchCaret(test.input)
		input := antlr.NewInputStream(text)
		lexer := mysql.NewMySQLLexer(input)
		tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
		parser := mysql.NewMySQLParser(tokens)

		scanner := NewScanner(tokens)
		// Move to caret position and store that on the scanner stack.
		scanner.AdvanceToPosition(1, caretOffset)
		scanner.Push()
		context := AutoCompletionContext{}
		context.CollectCandidates(parser, scanner, caretOffset, 1)
		for k, candidate := range context.Candidates.Tokens {
			fmt.Printf("%s: ", lexer.SymbolicNames[k])
			for _, token := range candidate {
				fmt.Printf("%s ", lexer.SymbolicNames[token])
			}
			fmt.Println("")
		}
		for k, candidate := range context.Candidates.Rules {
			fmt.Printf("%s: ", parser.RuleNames[k])
			for _, r := range candidate {
				fmt.Printf("%s ", parser.RuleNames[r])
			}
			fmt.Println("")
		}
	}
}

func catchCaret(s string) (string, int) {
	for i, c := range s {
		if c == '|' {
			return s[:i] + s[i+1:], i
		}
	}
	return s, -1
}
