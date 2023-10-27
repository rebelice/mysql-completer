package completion

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type test struct {
	input string
}

func TestCompleter(t *testing.T) {
	tests := []test{
		{
			input: "SELECT * FROM |",
		},
		{
			input: "SELECT | FROM t",
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
		fmt.Printf("CANDIDATES FOR: %s\n", test.input)
		fmt.Println("TOKENS -----------------")
		for k, candidate := range context.Candidates.Tokens {
			fmt.Printf("%s: ", lexer.SymbolicNames[k])
			for _, token := range candidate {
				fmt.Printf("%s ", lexer.SymbolicNames[token])
			}
			fmt.Println("")
		}
		fmt.Println("RULES -----------------")
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

type candidatesTest struct {
	Input string
	Want  []string
}

func TestCandidates(t *testing.T) {
	tests := []candidatesTest{}

	const (
		record = true
	)
	var (
		filepath = "testdata/data.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		text, caretOffset := catchCaret(t.Input)
		input := antlr.NewInputStream(text)
		lexer := mysql.NewMySQLLexer(input)
		tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
		parser := mysql.NewMySQLParser(tokens)
		parser.RemoveErrorListeners()
		result := GetCodeCompletionList(1, caretOffset, "db", true, parser)
		if record {
			tests[i].Want = result
		} else {
			a.Equal(t.Want, result)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}
