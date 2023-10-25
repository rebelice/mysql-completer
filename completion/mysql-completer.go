package completion

import (
	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
)

type TableReference struct {
	Schema string
	Table  string
	Alias  string
}

type AutoCompletionContext struct {
	Candidates *CandidatesCollection
	// A hierarchical view of all table references in the code, updated by visiting all relevant FROM clauses after
	// the candidate collection.
	// Organized as stack to be able to easily remove sets of references when changing nesting level.
	ReferencesStack [][]*TableReference
	// A flat list of possible references for easier lookup.
	References []*TableReference
}

func (c *AutoCompletionContext) CollectCandidates(parser *mysql.MySQLParser, scanner *Scanner, caretOffset int, caretLine int) {
	c3 := NewCodeCompletionCore(parser)
	c3.IgnoredTokens = map[int]bool{
		mysql.MySQLParserEOF:                      true,
		mysql.MySQLLexerEQUAL_OPERATOR:            true,
		mysql.MySQLLexerASSIGN_OPERATOR:           true,
		mysql.MySQLLexerNULL_SAFE_EQUAL_OPERATOR:  true,
		mysql.MySQLLexerGREATER_OR_EQUAL_OPERATOR: true,
		mysql.MySQLLexerGREATER_THAN_OPERATOR:     true,
		mysql.MySQLLexerLESS_OR_EQUAL_OPERATOR:    true,
		mysql.MySQLLexerLESS_THAN_OPERATOR:        true,
		mysql.MySQLLexerNOT_EQUAL_OPERATOR:        true,
		mysql.MySQLLexerNOT_EQUAL2_OPERATOR:       true,
		mysql.MySQLLexerPLUS_OPERATOR:             true,
		mysql.MySQLLexerMINUS_OPERATOR:            true,
		mysql.MySQLLexerMULT_OPERATOR:             true,
		mysql.MySQLLexerDIV_OPERATOR:              true,
		mysql.MySQLLexerMOD_OPERATOR:              true,
		mysql.MySQLLexerLOGICAL_NOT_OPERATOR:      true,
		mysql.MySQLLexerBITWISE_NOT_OPERATOR:      true,
		mysql.MySQLLexerSHIFT_LEFT_OPERATOR:       true,
		mysql.MySQLLexerSHIFT_RIGHT_OPERATOR:      true,
		mysql.MySQLLexerLOGICAL_AND_OPERATOR:      true,
		mysql.MySQLLexerBITWISE_AND_OPERATOR:      true,
		mysql.MySQLLexerBITWISE_XOR_OPERATOR:      true,
		mysql.MySQLLexerLOGICAL_OR_OPERATOR:       true,
		mysql.MySQLLexerBITWISE_OR_OPERATOR:       true,
		mysql.MySQLLexerDOT_SYMBOL:                true,
		mysql.MySQLLexerCOMMA_SYMBOL:              true,
		mysql.MySQLLexerSEMICOLON_SYMBOL:          true,
		mysql.MySQLLexerCOLON_SYMBOL:              true,
		mysql.MySQLLexerOPEN_PAR_SYMBOL:           true,
		mysql.MySQLLexerCLOSE_PAR_SYMBOL:          true,
		mysql.MySQLLexerOPEN_CURLY_SYMBOL:         true,
		mysql.MySQLLexerCLOSE_CURLY_SYMBOL:        true,
		mysql.MySQLLexerUNDERLINE_SYMBOL:          true,
		mysql.MySQLLexerAT_SIGN_SYMBOL:            true,
		mysql.MySQLLexerAT_AT_SIGN_SYMBOL:         true,
		mysql.MySQLLexerNULL2_SYMBOL:              true,
		mysql.MySQLLexerPARAM_MARKER:              true,
		mysql.MySQLLexerCONCAT_PIPES_SYMBOL:       true,
		mysql.MySQLLexerAT_TEXT_SUFFIX:            true,
		mysql.MySQLLexerBACK_TICK_QUOTED_ID:       true,
		mysql.MySQLLexerSINGLE_QUOTED_TEXT:        true,
		mysql.MySQLLexerDOUBLE_QUOTED_TEXT:        true,
		mysql.MySQLLexerNCHAR_TEXT:                true,
		mysql.MySQLLexerUNDERSCORE_CHARSET:        true,
		mysql.MySQLLexerIDENTIFIER:                true,
		mysql.MySQLLexerINT_NUMBER:                true,
		mysql.MySQLLexerLONG_NUMBER:               true,
		mysql.MySQLLexerULONGLONG_NUMBER:          true,
		mysql.MySQLLexerDECIMAL_NUMBER:            true,
		mysql.MySQLLexerBIN_NUMBER:                true,
		mysql.MySQLLexerHEX_NUMBER:                true,
	}

	c3.PreferredRules = map[int]bool{
		mysql.MySQLParserRULE_schemaRef:            true,
		mysql.MySQLParserRULE_tableRef:             true,
		mysql.MySQLParserRULE_tableRefWithWildcard: true,
		mysql.MySQLParserRULE_filterTableRef:       true,
		mysql.MySQLParserRULE_columnRef:            true,
		mysql.MySQLParserRULE_columnInternalRef:    true,
		mysql.MySQLParserRULE_tableWild:            true,
		mysql.MySQLParserRULE_functionRef:          true,
		mysql.MySQLParserRULE_functionCall:         true,
		mysql.MySQLParserRULE_runtimeFunctionCall:  true,
		mysql.MySQLParserRULE_triggerRef:           true,
		mysql.MySQLParserRULE_viewRef:              true,
		mysql.MySQLParserRULE_procedureRef:         true,
		mysql.MySQLParserRULE_logfileGroupRef:      true,
		mysql.MySQLParserRULE_tablespaceRef:        true,
		mysql.MySQLParserRULE_engineRef:            true,
		mysql.MySQLParserRULE_collationName:        true,
		mysql.MySQLParserRULE_charsetName:          true,
		mysql.MySQLParserRULE_eventRef:             true,
		mysql.MySQLParserRULE_serverRef:            true,
		mysql.MySQLParserRULE_user:                 true,
		mysql.MySQLParserRULE_userVariable:         true,
		mysql.MySQLParserRULE_systemVariable:       true,
		mysql.MySQLParserRULE_labelRef:             true,
		mysql.MySQLParserRULE_setSystemVariable:    true,
		mysql.MySQLParserRULE_parameterName:        true,
		mysql.MySQLParserRULE_procedureName:        true,
		mysql.MySQLParserRULE_identifier:           true,
		mysql.MySQLParserRULE_labelIdentifier:      true,
	}

	noSeparatorRequiredFor := map[int]bool{
		mysql.MySQLLexerEQUAL_OPERATOR:            true,
		mysql.MySQLLexerASSIGN_OPERATOR:           true,
		mysql.MySQLLexerNULL_SAFE_EQUAL_OPERATOR:  true,
		mysql.MySQLLexerGREATER_OR_EQUAL_OPERATOR: true,
		mysql.MySQLLexerGREATER_THAN_OPERATOR:     true,
		mysql.MySQLLexerLESS_OR_EQUAL_OPERATOR:    true,
		mysql.MySQLLexerLESS_THAN_OPERATOR:        true,
		mysql.MySQLLexerNOT_EQUAL_OPERATOR:        true,
		mysql.MySQLLexerNOT_EQUAL2_OPERATOR:       true,
		mysql.MySQLLexerPLUS_OPERATOR:             true,
		mysql.MySQLLexerMINUS_OPERATOR:            true,
		mysql.MySQLLexerMULT_OPERATOR:             true,
		mysql.MySQLLexerDIV_OPERATOR:              true,
		mysql.MySQLLexerMOD_OPERATOR:              true,
		mysql.MySQLLexerLOGICAL_NOT_OPERATOR:      true,
		mysql.MySQLLexerBITWISE_NOT_OPERATOR:      true,
		mysql.MySQLLexerSHIFT_LEFT_OPERATOR:       true,
		mysql.MySQLLexerSHIFT_RIGHT_OPERATOR:      true,
		mysql.MySQLLexerLOGICAL_AND_OPERATOR:      true,
		mysql.MySQLLexerBITWISE_AND_OPERATOR:      true,
		mysql.MySQLLexerBITWISE_XOR_OPERATOR:      true,
		mysql.MySQLLexerLOGICAL_OR_OPERATOR:       true,
		mysql.MySQLLexerBITWISE_OR_OPERATOR:       true,
		mysql.MySQLLexerDOT_SYMBOL:                true,
		mysql.MySQLLexerCOMMA_SYMBOL:              true,
		mysql.MySQLLexerSEMICOLON_SYMBOL:          true,
		mysql.MySQLLexerCOLON_SYMBOL:              true,
		mysql.MySQLLexerOPEN_PAR_SYMBOL:           true,
		mysql.MySQLLexerCLOSE_PAR_SYMBOL:          true,
		mysql.MySQLLexerOPEN_CURLY_SYMBOL:         true,
		mysql.MySQLLexerCLOSE_CURLY_SYMBOL:        true,
		mysql.MySQLLexerPARAM_MARKER:              true,
	}

	// Certain tokens (like identifiers) must be treated as if the char directly following them still belongs to that
	// token (e.g. a whitespace after a name), because visually the caret is placed between that token and the
	// whitespace creating the impression we are still at the identifier (and we should show candidates for this
	// identifier position).
	// Other tokens (like operators) don't need a separator and hence we can take the caret index as is for them
	caretIndex := scanner.TokenIndex()
	if caretIndex > 0 && !noSeparatorRequiredFor[scanner.LookBack(false /* skipHidden */)] {
		caretIndex--
	}
	c.ReferencesStack = append([][]*TableReference{{}}, c.ReferencesStack...)
	parser.Reset()
	context := parser.Query()

	c.Candidates = c3.CollectCandidates(caretIndex, context)

	// Post processing some entries.
	if len(c.Candidates.Tokens[mysql.MySQLLexerNOT2_SYMBOL]) > 0 {
		// NOT2 is a NOT with special meaning in the operator precedence chain.
		// For code completion it's the same as NOT
		c.Candidates.Tokens[mysql.MySQLLexerNOT_SYMBOL] = c.Candidates.Tokens[mysql.MySQLLexerNOT2_SYMBOL]
		delete(c.Candidates.Tokens, mysql.MySQLLexerNOT2_SYMBOL)
	}

	// If a column reference is required then we have to continue scanning the query for table references.
	for ruleName := range c.Candidates.Rules {
		if ruleName == mysql.MySQLParserRULE_columnRef {
			c.CollectLeadingTableReferences(parser, scanner, caretIndex, false /* forTableAlter */)
			c.TakeReferencesSnapshot()
			c.CollectRemainingTableReferences(parser, scanner)
			c.TakeReferencesSnapshot()
			break
		} else if ruleName == mysql.MySQLParserRULE_columnInternalRef {
			// Note:: rule columnInternalRef is not only used for ALTER TABLE, but atm. we only support that here.
			c.CollectLeadingTableReferences(parser, scanner, caretIndex, true /* forTableAlter */)
			c.TakeReferencesSnapshot()
		}
	}
}

func (c *AutoCompletionContext) TakeReferencesSnapshot() {
	for _, references := range c.ReferencesStack {
		c.References = append(c.References, references...)
	}
}

// Called if one of the candidates is a column reference, for table references *after* the caret.
// The function attempts to get table references together with aliases where possible. This is the only place
// where we actually look beyond the caret and hence different rules apply: the query doesn't need to be valid
// beyond that point. We simply scan forward until we find a FROM keyword and work from there. This makes it much
// easier to work on incomplete queries, which nonetheless need e.g. columns from table references.
// Because inner queries can use table references from outer queries we can simply scan for all outer FROM clauses
// (skip over subqueries).
func (c *AutoCompletionContext) CollectRemainingTableReferences(parser *mysql.MySQLParser, scanner *Scanner) {
	scanner.Push()

	level := 0
	for {
		found := scanner.TokenType() == mysql.MySQLLexerFROM_SYMBOL
		for !found {
			if !scanner.Next(false /* skipHidden */) {
				break
			}

			switch scanner.TokenType() {
			case mysql.MySQLLexerOPEN_PAR_SYMBOL:
				level++
			case mysql.MySQLLexerCLOSE_PAR_SYMBOL:
				if level > 0 {
					level--
				}
			case mysql.MySQLLexerFROM_SYMBOL:
				// Open and close parentheses don't need to match, if we come from within a subquery.
				if level == 0 {
					found = true
				}
			}
		}

		if !found {
			scanner.Pop()
			return // No more FROM clause found.
		}

		c.ParseTableReferences(scanner.TokenSubText(), parser)
		if scanner.TokenType() == mysql.MySQLLexerFROM_SYMBOL {
			scanner.Next(false /* skipHidden */)
		}
	}
}

func (c *AutoCompletionContext) CollectLeadingTableReferences(parser *mysql.MySQLParser, scanner *Scanner, caretIndex int, forTableAlter bool) {
	scanner.Push()

	if forTableAlter {
		// lexer := parser.GetTokenStream().GetTokenSource().(*mysql.MySQLLexer)

		for scanner.Previous(false /* skipHidden */) && scanner.TokenType() != mysql.MySQLLexerALTER_SYMBOL {
			// Skip all tokens until FROM
		}

		if scanner.TokenType() == mysql.MySQLLexerALTER_SYMBOL {
			scanner.SkipTokenSequence([]int{mysql.MySQLLexerALTER_SYMBOL, mysql.MySQLLexerTABLE_SYMBOL})

			var reference TableReference
			reference.Table = unquote(scanner.TokenText())
			if scanner.Next(false /* skipHidden */) && scanner.Is(mysql.MySQLLexerDOT_SYMBOL) {
				reference.Schema = reference.Table
				scanner.Next(false /* skipHidden */)
				scanner.Next(false /* skipHidden */) // Why twice?
				reference.Table = unquote(scanner.TokenText())
			}
			c.ReferencesStack[0] = append(c.ReferencesStack[0], &reference)
		}
	} else {
		scanner.Seek(0)

		level := 0
		for {
			found := scanner.TokenType() == mysql.MySQLLexerFROM_SYMBOL
			for !found {
				if !scanner.Next(false /* skipHidden */) || scanner.TokenIndex() >= caretIndex {
					break
				}

				switch scanner.TokenType() {
				case mysql.MySQLLexerOPEN_PAR_SYMBOL:
					level++
					c.ReferencesStack = append([][]*TableReference{{}}, c.ReferencesStack...)
				case mysql.MySQLLexerCLOSE_PAR_SYMBOL:
					if level == 0 {
						scanner.Pop()
						return // We cannot go above the initial nesting level.
					}

					level--
					c.ReferencesStack = c.ReferencesStack[1:]
				case mysql.MySQLLexerFROM_SYMBOL:
					found = true
				}
			}

			if !found {
				scanner.Pop()
				return // No more FROM clause found.
			}

			c.ParseTableReferences(scanner.TokenSubText(), parser)
			if scanner.TokenType() == mysql.MySQLLexerFROM_SYMBOL {
				scanner.Next(false /* skipHidden */)
			}
		}
	}
}

func (c *AutoCompletionContext) ParseTableReferences(fromClause string, parserTemplate *mysql.MySQLParser) {
	// We use a local parser just for the FROM clause to avoid messing up tokens on the autocompletion
	// parser (which would affect the processing of the found candidates)
	input := antlr.NewInputStream(fromClause)
	lexer := mysql.NewMySQLLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, 0)
	parser := mysql.NewMySQLParser(tokens)

	parser.BuildParseTrees = true
	parser.RemoveErrorListeners()
	tree := parser.FromClause()

	listener := &TableRefListener{
		context:        c,
		fromClauseMode: true,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
}

type TableRefListener struct {
	*mysql.BaseMySQLParserListener

	context        *AutoCompletionContext
	fromClauseMode bool
	done           bool
	level          int
}

func (l *TableRefListener) ExitTableRef(ctx *mysql.TableRefContext) {
	if l.done {
		return
	}

	if !l.fromClauseMode || l.level == 0 {
		reference := &TableReference{}
		if ctx.QualifiedIdentifier() != nil {
			reference.Table = unquote(ctx.QualifiedIdentifier().Identifier().GetText())
			if ctx.QualifiedIdentifier().DotIdentifier() != nil {
				reference.Schema = reference.Table
				reference.Table = unquote(ctx.QualifiedIdentifier().DotIdentifier().Identifier().GetText())
			}
		} else {
			reference.Table = unquote(ctx.DotIdentifier().Identifier().GetText())
		}
		l.context.ReferencesStack[0] = append(l.context.ReferencesStack[0], reference)
	}
}

func (l *TableRefListener) ExitTableAlias(ctx *mysql.TableAliasContext) {
	if l.done {
		return
	}

	if l.level == 0 && len(l.context.ReferencesStack) != 0 && len(l.context.ReferencesStack[0]) != 0 {
		// Appears after a single or derived table.
		// Since derived tables can be very complex it is not possible here to determine possible columns for
		// completion, hence we just walk over them and thus have no field where to store the found alias.
		l.context.ReferencesStack[0][len(l.context.ReferencesStack[0])-1].Alias = unquote(ctx.Identifier().GetText())
	}
}

func (l *TableRefListener) EnterSubquery(ctx *mysql.SubqueryContext) {
	if l.done {
		return
	}

	if l.fromClauseMode {
		l.level++
	} else {
		l.context.ReferencesStack = append([][]*TableReference{{}}, l.context.ReferencesStack...)
	}
}

func (l *TableRefListener) ExitSubquery(ctx *mysql.SubqueryContext) {
	if l.done {
		return
	}

	if l.fromClauseMode {
		l.level--
	} else {
		l.context.ReferencesStack = l.context.ReferencesStack[1:]
	}
}

func unquote(s string) string {
	if len(s) < 2 {
		return s
	}

	if (s[0] == '`' || s[0] == '\'' || s[0] == '"') && s[0] == s[len(s)-1] {
		return s[1 : len(s)-1]
	}
	return s
}

func GetCodeCompletionList(caretLine int, caretOffset int, defaultSchema string, uppercaseKeywords bool, parser *mysql.MySQLParser) {
	scanner := NewScanner(parser.GetTokenStream().(*antlr.CommonTokenStream))

	// Move to caret position and store that on the scanner stack.
	scanner.AdvanceToPosition(caretLine, caretOffset)
	scanner.Push()

	context := AutoCompletionContext{}
	context.CollectCandidates(parser, scanner, caretOffset, caretLine)
}
