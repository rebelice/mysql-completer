package completion

import (
	"github.com/antlr4-go/antlr/v4"
)

type PipelineEntry struct {
	State      antlr.ATNState
	TokenIndex int
}

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

	parser         antlr.Parser
	atn            *antlr.ATN
	IgnoredTokens  map[int]bool
	PreferredRules map[int]bool

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
					following: c.getFollowingTokens(transition),
				})
			}
		}
	}
}

func (c *CodeCompletionCore) getFollowingTokens(transition antlr.Transition) []int {
	result := []int{}
	// seen := []antlr.ATNState{}
	pipeline := []antlr.ATNState{transition.GetTarget()}

	for len(pipeline) > 0 {
		state := pipeline[len(pipeline)-1]
		pipeline = pipeline[:len(pipeline)-1]

		for _, transition := range state.GetTransitions() {
			if transition.GetSerializationType() == antlr.TransitionATOM {
				if !transition.GetIsEpsilon() {
					list := transition.GetLabel().ToList()
					if len(list) == 1 {
						if _, exists := c.IgnoredTokens[list[0]]; !exists {
							result = append(result, list[0])
							pipeline = append(pipeline, transition.GetTarget())
						}
					}
				} else {
					pipeline = append(pipeline, transition.GetTarget())
				}
			}
		}
	}

	return result
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

	result := make(RuleEndStatus)

	// For rule start states we determine and cache the follow set, which gives us 3 advantages:
	// 1. We can quickly check if a symbol would be matched when we follow that rule. We can so check in advance
	//    and can save us all the intermediate steps if there is no match.
	// 2. We'll have all symbols that are collectable already together when we are at the caret when entering a rule.
	// 3. We get this lookup for free with any 2nd or further visit of the same rule, which often happens
	//    in non trivial grammars, especially with (recursive) expressions and of course when invoking code completion
	//    multiple times.
	if c.setsPerState != nil && (len(c.setsPerState[startState.GetStateNumber()].sets) == 0) {
		stop := c.atn.GetRuleToStopState(startState.GetRuleIndex())
		followSets := c.DetermineFollowSets(startState, stop)
		sets, exists := c.setsPerState[startState.GetStateNumber()]
		if !exists {
			sets = FollowSetsHolder{
				sets:     followSets,
				combined: *antlr.NewIntervalSet(),
			}
			c.setsPerState[startState.GetStateNumber()] = sets
		} else {
			sets.sets = followSets
			c.setsPerState[startState.GetStateNumber()] = sets
		}

		combined := antlr.NewIntervalSet()
		for _, set := range sets.sets {
			combined.AddAll(&set.intervals)
		}
		sets.combined = *combined
		c.setsPerState[startState.GetStateNumber()] = sets
	}

	followSets := c.setsPerState[startState.GetStateNumber()]
	callStack = append(callStack, startState.GetRuleIndex())

	if tokenIndex >= len(c.tokens)-1 { // At caret?
		if _, exists := c.PreferredRules[startState.GetRuleIndex()]; exists {
			// No need to go deeper when collecting entries and we reach a rule that we want to collect anyway.
			c.translateToRuleIndex(callStack)
		} else {
			// Convert all follow sets to either single symbols or their associated preferred rule and add
			// the result to our candidates list.
			for _, set := range followSets.sets {
				var fullPath []int
				fullPath = append(fullPath, callStack...)
				fullPath = append(fullPath, set.path...)
				if !c.translateToRuleIndex(fullPath) {
					for _, symbol := range set.intervals.ToList() {
						if _, exists := c.IgnoredTokens[symbol]; !exists {
							if _, exists := c.candidates.Tokens[symbol]; !exists {
								c.candidates.Tokens[symbol] = set.following
							} else {
								equal := len(c.candidates.Tokens[symbol]) == len(set.following)
								if equal {
									for i, v := range c.candidates.Tokens[symbol] {
										if v != set.following[i] {
											equal = false
											break
										}
									}
								}
								if !equal {
									c.candidates.Tokens[symbol] = []int{}
								}
							}
						}
					}
				}
			}
		}

		callStack = callStack[:len(callStack)-1]
		return RuleEndStatus{}
	} else {
		// Process the rule if we either could pass it without consuming anything (epsilon transition)
		// or if the current input symbol will be matched somewhere after this entry point.
		currentSymbol := c.tokens[tokenIndex]
		if !followSets.combined.Contains(antlr.TokenEpsilon) && !followSets.combined.Contains(currentSymbol) {
			callStack = callStack[:len(callStack)-1]
			return RuleEndStatus{}
		}
	}

	// The current state execution pipeline contains all yet-to-be-processed ATN states in this rule.
	// For each such state we store the token index + a list of rules that lead to it
	var statePipeline []PipelineEntry
	var currentEntry PipelineEntry

	// Bootstrap the pipeline.
	statePipeline = append(statePipeline, PipelineEntry{
		State:      startState,
		TokenIndex: tokenIndex,
	})

	for len(statePipeline) != 0 {
		currentEntry = statePipeline[len(statePipeline)-1]
		statePipeline = statePipeline[:len(statePipeline)-1]
		c.statesProcessed++

		atCaret := currentEntry.TokenIndex >= len(c.tokens)-1

		switch currentEntry.State.GetStateType() {
		case antlr.ATNStateRuleStart:
			indentation += "  "
		case antlr.ATNStateRuleStop:
			result[currentEntry.TokenIndex] = true
			continue
		}

		for _, transition := range currentEntry.State.GetTransitions() {
			switch transition.GetSerializationType() {
			case antlr.TransitionRULE:
				endStatus := c.ProcessRule(transition.GetTarget(), currentEntry.TokenIndex, callStack, indentation)
				ruleTransition := transition.(*antlr.RuleTransition)
				for status := range endStatus {
					statePipeline = append(statePipeline, PipelineEntry{
						State:      ruleTransition.GetFollowState(),
						TokenIndex: status,
					})
				}
			case antlr.TransitionPREDICATE:
				if c.checkPredicate(transition.(*antlr.PredicateTransition)) {
					statePipeline = append(statePipeline, PipelineEntry{
						State:      transition.GetTarget(),
						TokenIndex: currentEntry.TokenIndex,
					})
				}
			case antlr.TransitionWILDCARD:
				if atCaret {
					if !c.translateToRuleIndex(callStack) {
						interval := antlr.NewIntervalSet()
						interval.AddInterval(antlr.NewInterval(antlr.TokenMinUserTokenType, c.atn.GetMaxTokenType()))
						for _, token := range interval.ToList() {
							if _, exists := c.IgnoredTokens[token]; !exists {
								if _, exists := c.candidates.Tokens[token]; !exists {
									c.candidates.Tokens[token] = []int{}
								}
							}
						}
					}
				} else {
					statePipeline = append(statePipeline, PipelineEntry{
						State:      transition.GetTarget(),
						TokenIndex: currentEntry.TokenIndex + 1,
					})
				}
			default:
				if transition.GetIsEpsilon() {
					if atCaret {
						c.translateToRuleIndex(callStack)
					}

					// Jump over simple states with a single outgoing epsilon transition.
					statePipeline = append(statePipeline, PipelineEntry{
						State:      transition.GetTarget(),
						TokenIndex: currentEntry.TokenIndex,
					})
				}

				set := transition.GetLabel()
				if set != nil && len(set.GetIntervals()) > 0 {
					if transition.GetSerializationType() == antlr.TransitionNOTSET {
						set = set.Complement(antlr.TokenMinUserTokenType, c.atn.GetMaxTokenType())
					}
					if atCaret {
						if !c.translateToRuleIndex(callStack) {
							list := set.ToList()
							addFollowing := len(list) == 1
							for _, symbol := range list {
								if _, exists := c.IgnoredTokens[symbol]; !exists {
									if addFollowing {
										c.candidates.Tokens[symbol] = c.getFollowingTokens(transition)
									} else {
										c.candidates.Tokens[symbol] = []int{}
									}
								}
							}
						}
					} else {
						currentSymbol := c.tokens[currentEntry.TokenIndex]
						if set.Contains(currentSymbol) {
							statePipeline = append(statePipeline, PipelineEntry{
								State:      transition.GetTarget(),
								TokenIndex: currentEntry.TokenIndex + 1,
							})
						}
					}
				}
			}
		}
	}

	callStack = callStack[:len(callStack)-1]
	return result
}

func (c *CodeCompletionCore) translateToRuleIndex(ruleStack []int) bool {
	if len(c.PreferredRules) == 0 {
		return false
	}

	// Loop over the rule stack from highest to lowest rule level. This way we properly handle the higher rule
	// if it contains a lower one that is also a preferred rule
	for i, entry := range ruleStack {
		if _, exists := c.PreferredRules[entry]; exists {
			// Add the rule to our candidates list along with the current rule path,
			// but only if there isn't already an entry like that.
			var path []int
			path = append(path, ruleStack[:i]...)
			addNew := true
			for k, v := range c.candidates.Rules {
				if k != entry || len(v) != len(path) {
					continue
				}

				// Found an entry for this rule. Same path? If so don't add a new (duplicate) entry.
				equal := true
				for j, p := range path {
					if p != v[j] {
						equal = false
						break
					}
				}

				if !equal {
					addNew = false
					break
				}
			}

			if addNew {
				c.candidates.Rules[ruleStack[i]] = path
			}
			return true
		}
	}

	return false
}
