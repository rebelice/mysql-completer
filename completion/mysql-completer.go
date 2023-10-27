package completion

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

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

type AutoCompletionImageType int

const (
	AutoCompletionImageTypeNone AutoCompletionImageType = iota
	AutoCompletionImageTypeKeyword
	AutoCompletionImageTypeSchema
	AutoCompletionImageTypeTable
	AutoCompletionImageTypeRoutine
	AutoCompletionImageTypeFunction
	AutoCompletionImageTypeView
	AutoCompletionImageTypeColumn
	AutoCompletionImageTypeOperator
	AutoCompletionImageTypeEngine
	AutoCompletionImageTypeTrigger
	AutoCompletionImageTypeLogFileGroup
	AutoCompletionImageTypeUserVar
	AutoCompletionImageTypeSystemVar
	AutoCompletionImageTypeTableSpace
	AutoCompletionImageTypeEvent
	AutoCompletionImageTypeIndex
	AutoCompletionImageTypeUser
	AutoCompletionImageTypeCharset
	AutoCompletionImageTypeCollation
)

type AutoCompletionEntry struct {
	ImageType AutoCompletionImageType
	Text      string
}

func (e *AutoCompletionEntry) String() string {
	return fmt.Sprintf("%d(%s)", e.ImageType, e.Text)
}

type CompletionMap map[string]bool

func (m CompletionMap) toSLice() []string {
	var result []string
	for key := range m {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func (m CompletionMap) Insert(entry AutoCompletionEntry) {
	m[entry.String()] = true
}

func (m CompletionMap) insertSchemas() {
	// TODO: load schemas
	m.Insert(AutoCompletionEntry{
		ImageType: AutoCompletionImageTypeSchema,
		Text:      "db",
	})
}

func (m CompletionMap) insertTables() {
	// TODO: load tables
	for i := 0; i < 5; i++ {
		m.Insert(AutoCompletionEntry{
			ImageType: AutoCompletionImageTypeTable,
			Text:      fmt.Sprintf("table%d", i),
		})
	}
}

func (m CompletionMap) insertViews() {
	// TODO: load views
	for i := 0; i < 5; i++ {
		m.Insert(AutoCompletionEntry{
			ImageType: AutoCompletionImageTypeView,
			Text:      fmt.Sprintf("view%d", i),
		})
	}
}

func (m CompletionMap) insertColumns(tables map[string]bool) {
	// TODO: load columns
	for table := range tables {
		id, err := strconv.Atoi(table[len(table)-1:])
		if err != nil {
			panic(err)
		}
		for i := 0; i < id; i++ {
			m.Insert(AutoCompletionEntry{
				ImageType: AutoCompletionImageTypeColumn,
				Text:      fmt.Sprintf("c%d", i),
			})
		}
	}
}

func GetCodeCompletionList(caretLine int, caretOffset int, defaultSchema string, uppercaseKeywords bool, parser *mysql.MySQLParser) []string {
	context := AutoCompletionContext{}

	// A set for each object type. This will sort the groups alphabetically and avoids duplicates,
	// but allows to add them as groups to the final list.
	schemaEntries := make(CompletionMap)
	tableEntries := make(CompletionMap)
	columnEntries := make(CompletionMap)
	viewEntries := make(CompletionMap)
	functionEntries := make(CompletionMap)
	// udfEntries := make(CompletionMap)
	runtimeFunctionEntries := make(CompletionMap)
	procedureEntries := make(CompletionMap)
	triggerEntries := make(CompletionMap)
	engineEntries := make(CompletionMap)
	logFileGroupEntries := make(CompletionMap)
	tableSpaceEntries := make(CompletionMap)
	systemVarEntries := make(CompletionMap)
	keywordEntries := make(CompletionMap)
	collationEntries := make(CompletionMap)
	charsetEntries := make(CompletionMap)
	eventEntries := make(CompletionMap)

	// Handled but needs meat yet.
	// userVarEntries := make(CompletionMap)

	// To be done yet.
	userEntries := make(CompletionMap)
	indexEntries := make(CompletionMap)
	pluginEntries := make(CompletionMap)
	// fkEntries := make(CompletionMap)
	labelEntries := make(CompletionMap)

	synonyms := map[int][]string{
		mysql.MySQLLexerCHAR_SYMBOL:         {"CHARACTER"},
		mysql.MySQLLexerNOW_SYMBOL:          {"CURRENT_TIMESTAMP", "LOCALTIME", "LOCALTIMESTAMP"},
		mysql.MySQLLexerDAY_SYMBOL:          {"DAYOFMONTH", "SQL_TSI_DAY"},
		mysql.MySQLLexerDECIMAL_SYMBOL:      {"DEC"},
		mysql.MySQLLexerDISTINCT_SYMBOL:     {"DISTINCTROW"},
		mysql.MySQLLexerCOLUMNS_SYMBOL:      {"FIELDS"},
		mysql.MySQLLexerFLOAT_SYMBOL:        {"FLOAT4"},
		mysql.MySQLLexerDOUBLE_SYMBOL:       {"FLOAT8"},
		mysql.MySQLLexerINT_SYMBOL:          {"INTEGER", "INT4"},
		mysql.MySQLLexerRELAY_THREAD_SYMBOL: {"IO_THREAD"},
		mysql.MySQLLexerSUBSTRING_SYMBOL:    {"MID", "SUBSTR"},
		mysql.MySQLLexerMID_SYMBOL:          {"MEDIUMINT"},
		mysql.MySQLLexerMEDIUMINT_SYMBOL:    {"MIDDLEINT", "INT3"},
		mysql.MySQLLexerNDBCLUSTER_SYMBOL:   {"NDB"},
		mysql.MySQLLexerREGEXP_SYMBOL:       {"RLIKE"},
		mysql.MySQLLexerDATABASE_SYMBOL:     {"SCHEMA"},
		mysql.MySQLLexerDATABASES_SYMBOL:    {"SCHEMAS"},
		mysql.MySQLLexerUSER_SYMBOL:         {"SESSION_USER"},
		mysql.MySQLLexerSTD_SYMBOL:          {"STDDEV", "STDDEV"},
		mysql.MySQLLexerVARCHAR_SYMBOL:      {"VARCHARACTER"},
		mysql.MySQLLexerVARIANCE_SYMBOL:     {"VAR_POP"},
		mysql.MySQLLexerTINYINT_SYMBOL:      {"INT1"},
		mysql.MySQLLexerSMALLINT_SYMBOL:     {"INT2"},
		mysql.MySQLLexerBIGINT_SYMBOL:       {"INT8"},
		mysql.MySQLLexerSECOND_SYMBOL:       {"SQL_TSI_SECOND"},
		mysql.MySQLLexerMINUTE_SYMBOL:       {"SQL_TSI_MINUTE"},
		mysql.MySQLLexerHOUR_SYMBOL:         {"SQL_TSI_HOUR"},
		mysql.MySQLLexerWEEK_SYMBOL:         {"SQL_TSI_WEEK"},
		mysql.MySQLLexerMONTH_SYMBOL:        {"SQL_TSI_MONTH"},
		mysql.MySQLLexerQUARTER_SYMBOL:      {"SQL_TSI_QUARTER"},
		mysql.MySQLLexerYEAR_SYMBOL:         {"SQL_TSI_YEAR"},
	}

	scanner := NewScanner(parser.GetTokenStream().(*antlr.CommonTokenStream))
	lexer := parser.GetTokenStream().GetTokenSource().(*mysql.MySQLLexer)

	// Move to caret position and store that on the scanner stack.
	scanner.AdvanceToPosition(caretLine, caretOffset)
	scanner.Push()

	context.CollectCandidates(parser, scanner, caretOffset, caretLine)

	for token, value := range context.Candidates.Tokens {
		entry := parser.SymbolicNames[token]
		if strings.HasSuffix(entry, "_SYMBOL") {
			entry = entry[:len(entry)-7]
		} else {
			entry = unquote(entry)
		}

		list := 0
		if len(value) > 0 {
			// A function call?
			if value[0] == mysql.MySQLLexerOPEN_PAR_SYMBOL {
				list = 1
			} else {
				for _, item := range value {
					subEntry := parser.SymbolicNames[item]
					if strings.HasSuffix(subEntry, "_SYMBOL") {
						subEntry = subEntry[:len(subEntry)-7]
					} else {
						subEntry = unquote(subEntry)
					}
					entry += " " + subEntry
				}
			}
		}

		switch list {
		case 1:
			runtimeFunctionEntries.Insert(AutoCompletionEntry{
				ImageType: AutoCompletionImageTypeFunction,
				Text:      strings.ToLower(entry) + "()",
			})
		default:
			if !uppercaseKeywords {
				entry = strings.ToLower(entry)
			}

			keywordEntries.Insert(AutoCompletionEntry{
				ImageType: AutoCompletionImageTypeKeyword,
				Text:      entry,
			})

			// Add also synonyms, if there are any.
			if synonyms[token] != nil {
				for _, synonym := range synonyms[token] {
					if !uppercaseKeywords {
						synonym = strings.ToLower(synonym)
					}

					keywordEntries.Insert(AutoCompletionEntry{
						ImageType: AutoCompletionImageTypeKeyword,
						Text:      synonym,
					})
				}
			}
		}
	}

	for candidate := range context.Candidates.Rules {
		// Restore the scanner position to the caret position and store that value again for the next round.
		scanner.Pop()
		scanner.Push()

		switch candidate {
		case mysql.MySQLParserRULE_runtimeFunctionCall:
			// TODO: load runtime functions
			runtimeFunctionEntries.Insert(AutoCompletionEntry{
				ImageType: AutoCompletionImageTypeFunction,
				Text:      "runtimeFunction()",
			})
		case mysql.MySQLParserRULE_schemaRef:
			schemaEntries.insertSchemas()
		case mysql.MySQLParserRULE_tableRefWithWildcard:
			// A special form of table references (id.id.*) used only in multi-table delete.
			// Handling is similar as for column references (just that we have table/view objects instead of column refs).
			schema, _, flags := determineSchemaTableQualifier(scanner, lexer)
			if flags&ObjectFlagsShowSchemas != 0 {
				schemaEntries.insertSchemas()
			}

			schemas := make(map[string]bool)
			if len(schema) == 0 {
				schemas[defaultSchema] = true
			} else {
				schemas[schema] = true
			}
			if flags&ObjectFlagsShowTables != 0 {
				tableEntries.insertTables()
				viewEntries.insertViews()
			}
		case mysql.MySQLParserRULE_tableRef, mysql.MySQLParserRULE_filterTableRef:
			qualifier, flags := determineQualifier(scanner, lexer, caretOffset)

			if flags&ObjectFlagsShowFirst != 0 {
				schemaEntries.insertSchemas()
			}

			if flags&ObjectFlagsShowSecond != 0 {
				schemas := make(map[string]bool)
				if len(qualifier) == 0 {
					schemas[defaultSchema] = true
				} else {
					schemas[qualifier] = true
				}

				tableEntries.insertTables()
				viewEntries.insertViews()
			}
		case mysql.MySQLParserRULE_tableWild, mysql.MySQLParserRULE_columnRef:
			schema, table, flags := determineSchemaTableQualifier(scanner, lexer)
			if flags&ObjectFlagsShowSchemas != 0 {
				schemaEntries.insertSchemas()
			}

			schemas := make(map[string]bool)
			if len(schema) != 0 {
				schemas[schema] = true
			} else if len(context.References) > 0 {
				for _, reference := range context.References {
					if len(reference.Schema) != 0 {
						schemas[reference.Schema] = true
					}
				}
			}

			if len(schemas) == 0 {
				schemas[defaultSchema] = true
			}

			if flags&ObjectFlagsShowTables != 0 {
				tableEntries.insertTables()
				if candidate == mysql.MySQLParserRULE_columnRef {
					viewEntries.insertViews()

					for _, reference := range context.References {
						if (len(schema) == 0 && len(reference.Schema) == 0) || schemas[reference.Schema] {
							if len(reference.Alias) == 0 {
								tableEntries.Insert(AutoCompletionEntry{
									ImageType: AutoCompletionImageTypeTable,
									Text:      reference.Table,
								})
							} else {
								tableEntries.Insert(AutoCompletionEntry{
									ImageType: AutoCompletionImageTypeTable,
									Text:      reference.Alias,
								})
							}
						}
					}
				}
			}

			if flags&ObjectFlagsShowColumns != 0 {
				if schema == table { // Schema and table are equal if it's not clear if we see a schema or table qualifier.
					schemas[defaultSchema] = true
				}

				tables := make(map[string]bool)
				if len(table) != 0 {
					tables[table] = true

					// Could be an alias
					for _, reference := range context.References {
						if strings.EqualFold(reference.Alias, table) {
							tables[reference.Table] = true
							schemas[reference.Schema] = true
						}
					}
				} else if len(context.References) > 0 && candidate == mysql.MySQLParserRULE_columnRef {
					for _, reference := range context.References {
						tables[reference.Table] = true
					}
				}

				if len(tables) > 0 {
					columnEntries.insertColumns(tables)
				}
			}

			// TODO: special handling for triggers.
		}
	}

	scanner.Pop() // Clear the scanner stack.
	var result []string
	result = append(result, keywordEntries.toSLice()...)
	result = append(result, columnEntries.toSLice()...)
	result = append(result, userEntries.toSLice()...)
	result = append(result, labelEntries.toSLice()...)
	result = append(result, tableEntries.toSLice()...)
	result = append(result, viewEntries.toSLice()...)
	result = append(result, schemaEntries.toSLice()...)
	result = append(result, functionEntries.toSLice()...)
	result = append(result, procedureEntries.toSLice()...)
	result = append(result, triggerEntries.toSLice()...)
	result = append(result, indexEntries.toSLice()...)
	result = append(result, eventEntries.toSLice()...)
	result = append(result, userEntries.toSLice()...)
	result = append(result, engineEntries.toSLice()...)
	result = append(result, pluginEntries.toSLice()...)
	result = append(result, logFileGroupEntries.toSLice()...)
	result = append(result, tableSpaceEntries.toSLice()...)
	result = append(result, charsetEntries.toSLice()...)
	result = append(result, collationEntries.toSLice()...)
	result = append(result, runtimeFunctionEntries.toSLice()...)
	result = append(result, systemVarEntries.toSLice()...)

	return result
}

type ObjectFlags int

const (
	ObjectFlagsShowSchemas ObjectFlags = 1 << iota
	ObjectFlagsShowTables
	ObjectFlagsShowColumns
	ObjectFlagsShowFirst
	ObjectFlagsShowSecond
)

func determineSchemaTableQualifier(scanner *Scanner, lexer *mysql.MySQLLexer) (schema, table string, flags ObjectFlags) {
	position := scanner.TokenIndex()
	if scanner.TokenChannel() != 0 {
		scanner.Next(true /* skipHidden */) // First skip to the next non-hidden token.
	}

	tokenType := scanner.TokenType()
	if tokenType != mysql.MySQLLexerDOT_SYMBOL && !lexer.IsIdentifier(scanner.TokenType()) {
		// We are at the end of an incomplete identifier spec. Jump back, so that the other tests succeed.
		scanner.Previous(true /* skipHidden */)
	}

	if position > 0 {
		if lexer.IsIdentifier(scanner.TokenType()) && scanner.LookBack(false /* skipHidden */) == mysql.MySQLLexerDOT_SYMBOL {
			scanner.Previous(true /* skipHidden */)
		}
		if scanner.Is(mysql.MySQLLexerDOT_SYMBOL) && lexer.IsIdentifier(scanner.LookBack(false /* skipHidden */)) {
			scanner.Previous(true /* skipHidden */)

			if scanner.LookBack(false /* skipHidden */) == mysql.MySQLLexerDOT_SYMBOL {
				scanner.Previous(true /* skipHidden */)
				if lexer.IsIdentifier(scanner.LookBack(false /* skipHidden */)) {
					scanner.Previous(true /* skipHidden */)
				}
			}
		}
	}

	schema = ""
	table = ""
	temp := ""
	if lexer.IsIdentifier(scanner.TokenType()) {
		temp = unquote(scanner.TokenText())
		scanner.Next(true /* skipHidden */)
	}

	if !scanner.Is(mysql.MySQLLexerDOT_SYMBOL) || position <= scanner.TokenIndex() {
		return schema, table, ObjectFlagsShowSchemas | ObjectFlagsShowTables | ObjectFlagsShowColumns
	}

	scanner.Next(true /* skipHidden */) // skip dot
	table = temp
	schema = temp
	if lexer.IsIdentifier(scanner.TokenType()) {
		temp = unquote(scanner.TokenText())
		scanner.Next(true /* skipHidden */)

		if !scanner.Is(mysql.MySQLLexerDOT_SYMBOL) || position <= scanner.TokenIndex() {
			return schema, table, ObjectFlagsShowTables | ObjectFlagsShowColumns
		}

		table = temp
		return schema, table, ObjectFlagsShowColumns
	}

	return schema, table, ObjectFlagsShowTables | ObjectFlagsShowColumns
}

func determineQualifier(scanner *Scanner, lexer *mysql.MySQLLexer, offsetInLine int) (string, ObjectFlags) {
	// Five possible positions here:
	//   - In the first id (including the position directly after the last char).
	//   - In the space between first id and a dot.
	//   - On a dot (visually directly before the dot).
	//   - In space after the dot, that includes the position directly after the dot.
	//   - In the second id.
	// All parts are optional (though not at the same time). The on-dot position is considered the same
	// as in first id as it visually belongs to the first id

	position := scanner.TokenIndex()
	if scanner.TokenChannel() != 0 {
		scanner.Next(true /* skipHidden */) // First skip to the next non-hidden token.
	}

	if !scanner.Is(mysql.MySQLLexerDOT_SYMBOL) && !lexer.IsIdentifier(scanner.TokenType()) {
		// We are at the end of an incomplete identifier spec. Jump back, so that the other tests succeed.
		scanner.Previous(true /* skipHidden */)
	}

	// Go left until we find something not related to an id or find at most 1 dot.
	if position > 0 {
		if lexer.IsIdentifier(scanner.TokenType()) && scanner.LookBack(false /* skipHidden */) == mysql.MySQLLexerDOT_SYMBOL {
			scanner.Previous(true /* skipHidden */)
		}
		if scanner.Is(mysql.MySQLLexerDOT_SYMBOL) && lexer.IsIdentifier(scanner.LookBack(false /* skipHidden */)) {
			scanner.Previous(true /* skipHidden */)
		}
	}

	// The scanner is now on the leading identifier or dot (if there's no leading id).
	qualifier := ""
	temp := ""
	if lexer.IsIdentifier(scanner.TokenType()) {
		temp = unquote(scanner.TokenText())
		scanner.Next(true /* skipHidden */)
	}

	if !scanner.Is(mysql.MySQLLexerDOT_SYMBOL) || position <= scanner.TokenIndex() {
		return qualifier, ObjectFlagsShowFirst | ObjectFlagsShowSecond
	}

	qualifier = temp
	return qualifier, ObjectFlagsShowSecond
}
