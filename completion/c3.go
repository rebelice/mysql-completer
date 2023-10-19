package completion

import (
	"github.com/antlr4-go/antlr/v4"
)

type CandidatesCollection struct {
	Tokens map[int][]int
	Rules  map[int][]int
}

type RuleEndStatus map[int]bool

type FollowSetWithPath struct {
	intervals antlr.IntervalSet
	path      []int
	following []int
}

type FollowSetsList []FollowSetWithPath

type FollowSetsHolder struct {
	sets     FollowSetsList
	combined antlr.IntervalSet
}

type FollowSetsPerState map[int]FollowSetsHolder

type CodeCompletionCore struct {
	setsPerState FollowSetsPerState

	parser antlr.Parser
	atn    *antlr.ATN

	candidates      *CandidatesCollection
	shortcutMap     map[int]map[int]RuleEndStatus
	statesProcessed int
	tokenStartIndex int
	tokens          []int
}

func NewCodeCompletionCore(parser antlr.Parser) *CodeCompletionCore {
	return &CodeCompletionCore{
		parser: parser,
		atn:    parser.GetATN(),
	}
}

func (c *CodeCompletionCore) CollectCandidates(caretTokenIndex int, context antlr.ParserRuleContext) {
	// init
	c.candidates = &CandidatesCollection{
		Tokens: make(map[int][]int),
		Rules:  make(map[int][]int),
	}
	c.shortcutMap = make(map[int]map[int]RuleEndStatus)
	c.statesProcessed = 0

	if context == nil {
		c.tokenStartIndex = 0
	} else {
		c.tokenStartIndex = context.GetStart().GetTokenIndex()
	}
	c.tokens = []int{}

	// Get all token types, starting with the rule start token, up to the caret token.
	// These should all have been read already and hence do not place any time penalty on us here.
	tokenStream := c.parser.GetTokenStream()
	currentOffset := tokenStream.Index()
	tokenStream.Seek(c.tokenStartIndex)
	offset := 1
	for {
		token := tokenStream.LT(offset)
		offset++
		c.tokens = append(c.tokens, token.GetTokenType())

		if token.GetTokenIndex() >= caretTokenIndex || c.tokens[len(c.tokens)-1] == antlr.TokenEOF {
			break
		}
	}
	tokenStream.Seek(currentOffset)

	var callStack []int
	var startRule int
	if context == nil {
		startRule = 0
	} else {
		startRule = context.GetRuleIndex()
	}
	c.ProcessRule(c.atn.GetRuleToStartState(startRule), 0, callStack, "")
}

func (c *CodeCompletionCore) DetermineFollowSets(start, stop antlr.ATNState) FollowSetsList {
	seen := make(map[antlr.ATNState]bool)
	ruleStack := []int{}
	result := FollowSetsList{}
	c.CollectFollowSets(start, stop, &result, seen, &ruleStack)
	return result
}

func (c *CodeCompletionCore) CollectFollowSets(s antlr.ATNState, stopState antlr.ATNState, followSets *FollowSetsList, seen map[antlr.ATNState]bool, ruleStack *[]int) {
	if _, exists := seen[s]; exists {
		return
	}

	seen[s] = true

	if s == stopState || s.GetStateType() == antlr.ATNStateRuleStop {
		interval := antlr.NewIntervalSet()
		interval.AddInterval(antlr.NewInterval(antlr.TokenEpsilon, antlr.TokenEpsilon))
		*followSets = append(*followSets, FollowSetWithPath{
			intervals: *interval,
			path:      *ruleStack,
			following: []int{},
		})
		return
	}

	for _, transition := range s.GetTransitions() {
		if transition.GetSerializationType() == antlr.TransitionRULE {
			ruleTransition, ok := transition.(*antlr.RuleTransition)
			if !ok {
				panic("should RuleTransition")
			}
			if existsInSlice(*ruleStack, ruleTransition.GetTarget().GetRuleIndex()) {
				continue
			}

			*ruleStack = append(*ruleStack, ruleTransition.GetTarget().GetRuleIndex())
			c.CollectFollowSets(transition.GetTarget(), stopState, followSets, seen, ruleStack)
			*ruleStack = (*ruleStack)[:len(*ruleStack)-1]
		} else if transition.GetSerializationType() == antlr.TransitionPRECEDENCE {
			predicateTransition, ok := transition.(*antlr.PredicateTransition)
			if !ok {
				panic("should PredicateTransition")
			}
			if c.checkPredicate(predicateTransition) {
				c.CollectFollowSets(transition.GetTarget(), stopState, followSets, seen, ruleStack)
			}
		} else if transition.GetIsEpsilon() {
			c.CollectFollowSets(transition.GetTarget(), stopState, followSets, seen, ruleStack)
		} else if transition.GetSerializationType() == antlr.TransitionWILDCARD {
			interval := antlr.NewIntervalSet()
			interval.AddInterval(antlr.NewInterval(antlr.TokenMinUserTokenType, c.atn.GetMaxTokenType()))
			*followSets = append(*followSets, FollowSetWithPath{
				intervals: *interval,
				path:      *ruleStack,
				following: []int{},
			})
		} else {
			set := transition.GetLabel()
			if set != nil && len(set.GetIntervals()) > 0 {
				if transition.GetSerializationType() == antlr.TransitionNOTSET {
					set = set.Complement(antlr.TokenMinUserTokenType, c.atn.GetMaxTokenType())
				}
				*followSets = append(*followSets, FollowSetWithPath{
					intervals: *set,
					path:      *ruleStack,
					following: getFollowingTokens(transition),
				})
			}
		}
	}
}

func getFollowingTokens(transition antlr.Transition) []int {
	result := []int{}
	seen := []antlr.ATNState{}
	pipeline := []antlr.ATNState{transition.GetTarget()}

	for len(pipeline) > 0 {
		state := pipeline[len(pipeline)-1]
		pipeline = pipeline[:len(pipeline)-1]

		for _, transition := range state.GetTransitions() {
			if transition.GetSerializationType() == antlr.TransitionATOM {
				if !transition.GetIsEpsilon() {
					list := transition.GetLabel().ToList()
				}
			}
		}
	}
}

func (c *CodeCompletionCore) checkPredicate(transition *antlr.PredicateTransition) bool {
	return transition.GetPredicate().Evaluate(c.parser, antlr.ParserRuleContextEmpty)
}

func existsInSlice(list []int, want int) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

func (c *CodeCompletionCore) ProcessRule(startState antlr.ATNState, tokenIndex int, callStack []int, indentation string) RuleEndStatus {
	// Start with rule specific handling before going into the ATN traversal.

	// Check first if we've taken this path with the same input before.
	positionMap := c.shortcutMap[startState.GetRuleIndex()]
	if positionMap != nil && positionMap[tokenIndex] != nil {
		return positionMap[tokenIndex]
	}

	var result RuleEndStatus

	// For rule start states we determine and cache the follow set, which gives us 3 advantages:
	// 1. We can quickly check if a symbol would be matched when we follow that rule. We can so check in advance
	//    and can save us all the intermediate steps if there is no match.
	// 2. We'll have all symbols that are collectable already together when we are at the caret when entering a rule.
	// 3. We get this lookup for free with any 2nd or further visit of the same rule, which often happens
	//    in non trivial grammars, especially with (recursive) expressions and of course when invoking code completion
	//    multiple times.
	if c.setsPerState != nil && len(c.setsPerState[startState.GetStateNumber()].sets) > 0 {
		stop := c.atn.GetRuleToStopState(startState.GetRuleIndex())
		followSets := determineFollowSets(startState, stop)
	}
}
