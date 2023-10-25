package completion

import "github.com/antlr4-go/antlr/v4"

type Scanner struct {
	input      *antlr.CommonTokenStream
	index      int
	tokens     []antlr.Token
	tokenStack []int
}

func NewScanner(input *antlr.CommonTokenStream) *Scanner {
	input.Fill()
	return &Scanner{
		input:  input,
		index:  0,
		tokens: input.GetAllTokens(),
	}
}

func (s *Scanner) TokenIndex() int {
	return s.index
}

func (s *Scanner) LookBack(skipHidden bool) int {
	index := s.index
	for index > 0 {
		index--
		if s.tokens[index].GetChannel() == antlr.TokenDefaultChannel || !skipHidden {
			return s.tokens[index].GetTokenType()
		}
	}

	return antlr.TokenInvalidType
}

func (s *Scanner) Push() {
	s.tokenStack = append(s.tokenStack, s.index)
}

func (s *Scanner) Pop() bool {
	if len(s.tokenStack) == 0 {
		return false
	}
	s.index = s.tokenStack[len(s.tokenStack)-1]
	s.tokenStack = s.tokenStack[:len(s.tokenStack)-1]
	return true
}

func (s *Scanner) Previous(skipHidden bool) bool {
	for s.index > 0 {
		s.index--
		if s.tokens[s.index].GetChannel() == 0 || !skipHidden {
			return true
		}
	}
	return false
}

func (s *Scanner) TokenType() int {
	return s.tokens[s.index].GetTokenType()
}

func (s *Scanner) SkipTokenSequence(list []int) bool {
	if s.index >= len(s.tokens) {
		return false
	}

	for _, token := range list {
		if s.tokens[s.index].GetTokenType() != token {
			return false
		}

		s.index++
		for s.index < len(s.tokens) && s.tokens[s.index].GetChannel() != antlr.TokenDefaultChannel {
			s.index++
		}

		if s.index >= len(s.tokens) {
			return false
		}
	}
	return true
}

func (s *Scanner) TokenText() string {
	return s.tokens[s.index].GetText()
}

func (s *Scanner) Next(skipHidden bool) bool {
	for s.index < len(s.tokens)-1 {
		s.index++
		if s.tokens[s.index].GetChannel() == antlr.TokenDefaultChannel || !skipHidden {
			return true
		}
	}
	return false
}

func (s *Scanner) Is(tokenType int) bool {
	return s.tokens[s.index].GetTokenType() == tokenType
}

func (s *Scanner) Seek(index int) {
	if index < len(s.tokens) {
		s.index = index
	}
}

func (s *Scanner) TokenSubText() string {
	cs := s.tokens[s.index].GetTokenSource().GetInputStream()
	return cs.GetText(s.tokens[s.index].GetStart(), cs.Size()-1)
}

func (s *Scanner) AdvanceToPosition(line, offset int) bool {
	if len(s.tokens) == 0 {
		return false
	}

	i := 0
	for ; i < len(s.tokens); i++ {
		run := s.tokens[i]
		tokenLine := run.GetLine()
		if tokenLine >= line {
			tokenOffset := run.GetColumn()
			tokenLength := run.GetStop() - run.GetStart() + 1
			if tokenLine == line && tokenOffset <= offset && offset < tokenOffset+tokenLength {
				s.index = i
				break
			}

			if tokenLine > line || tokenOffset > offset {
				if i == 0 {
					return false
				}

				s.index = i - 1
				break
			}
		}
	}

	if i == len(s.tokens) {
		s.index = i - 1
	}

	return true
}
