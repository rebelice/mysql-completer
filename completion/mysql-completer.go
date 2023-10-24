package completion

import (
	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
)

type AutoCompletionContext struct {
}

func (c *AutoCompletionContext) CollectCandidates(parser *mysql.MySQLParser, tokenString antlr.TokenStream, caretOffset int, caretLine int) {
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
}
