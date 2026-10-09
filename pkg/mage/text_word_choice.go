package mage

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type wordPairResult struct{ first, second int }
type wordPairBinding struct{ name string }

func (b *wordPairBinding) bindingName() string { return b.name }
func (b *wordPairBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "text-word-pair-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}

type wordPairChoice struct {
	player  PlayerRef
	binding *wordPairBinding
	labels  []string
}

func (*wordPairChoice) Text() string                 { return "choose a word and a different word" }
func (*wordPairChoice) Properties() EffectProperties { return EffectProperties{} }
func (e *wordPairChoice) Apply(ctx *EffectContext) error {
	state := &ctx.Game.resolution
	delete(state.wordPairBindings, e.binding)
	player, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	chooser := ctx.Game.GetPlayer(player)
	first := chooser.ChooseMode(slices.Clone(e.labels), "choose the word to replace")
	if first < 0 || first >= len(e.labels) {
		first = 0
	}
	options := slices.Clone(e.labels)
	options = slices.Delete(options, first, first+1)
	second := chooser.ChooseMode(slices.Clone(options), "choose the replacement word")
	if second < 0 || second >= len(options) {
		second = 0
	}
	if second >= first {
		second++
	}
	if state.wordPairBindings == nil {
		state.wordPairBindings = make(map[*wordPairBinding]wordPairResult)
	}
	state.wordPairBindings[e.binding] = wordPairResult{first + 1, second + 1}
	return nil
}
func (e *wordPairChoice) inspectDefinition() definitionNode {
	return definitionNode{Role: "text-word-pair-choice", bindingWrite: e.binding, children: []definitionEdge{{"player", e.player}, {"binding", e.binding}}}
}
func (e *wordPairChoice) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}

// ColorWordPairBinding declares a typed pair of words in one resolution frame.
type ColorWordPairBinding struct{ binding *wordPairBinding }

// BindColorWordPair declares a separate word-pair result.
func BindColorWordPair(name string) *ColorWordPairBinding {
	return &ColorWordPairBinding{&wordPairBinding{name}}
}

// ChooseColorWordPair saves two different legal words at resolution.
func ChooseColorWordPair(player PlayerRef, result *ColorWordPairBinding) ResolvingEffect {
	var binding *wordPairBinding
	if result != nil {
		binding = result.binding
	}
	return &wordPairChoice{player: player, binding: binding, labels: []string{"White", "Blue", "Black", "Red", "Green"}}
}

// ColorWordRef reads one typed word without making a choice.
type ColorWordRef interface {
	Resolve(*EffectContext) (core.Color, bool)
	inspectDefinition() definitionNode
}
type savedColorWord struct {
	binding *wordPairBinding
	second  bool
}

// FirstColorWord reads the saved word to replace.
func FirstColorWord(result *ColorWordPairBinding) ColorWordRef {
	var binding *wordPairBinding
	if result != nil {
		binding = result.binding
	}
	return savedColorWord{binding: binding}
}

// SecondColorWord reads the saved replacement word.
func SecondColorWord(result *ColorWordPairBinding) ColorWordRef {
	var binding *wordPairBinding
	if result != nil {
		binding = result.binding
	}
	return savedColorWord{binding: binding, second: true}
}
func (r savedColorWord) Resolve(ctx *EffectContext) (core.Color, bool) {
	pair, ok := ctx.Game.resolution.wordPairBindings[r.binding]
	if r.second {
		return core.Color(pair.second), ok
	}
	return core.Color(pair.first), ok
}
func (r savedColorWord) inspectDefinition() definitionNode {
	return definitionNode{Role: "saved-color-word", bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}

// BasicLandWordPairBinding declares a typed pair of words in one resolution frame.
type BasicLandWordPairBinding struct{ binding *wordPairBinding }

// BindBasicLandWordPair declares a separate word-pair result.
func BindBasicLandWordPair(name string) *BasicLandWordPairBinding {
	return &BasicLandWordPairBinding{&wordPairBinding{name}}
}

// ChooseBasicLandWordPair saves two different legal words at resolution.
func ChooseBasicLandWordPair(player PlayerRef, result *BasicLandWordPairBinding) ResolvingEffect {
	var binding *wordPairBinding
	if result != nil {
		binding = result.binding
	}
	return &wordPairChoice{player: player, binding: binding, labels: []string{"Plains", "Island", "Swamp", "Mountain", "Forest"}}
}

// BasicLandWordRef reads one typed word without making a choice.
type BasicLandWordRef interface {
	Resolve(*EffectContext) (core.BasicLandType, bool)
	inspectDefinition() definitionNode
}
type savedBasicLandWord struct {
	binding *wordPairBinding
	second  bool
}

// FirstBasicLandWord reads the saved word to replace.
func FirstBasicLandWord(result *BasicLandWordPairBinding) BasicLandWordRef {
	var binding *wordPairBinding
	if result != nil {
		binding = result.binding
	}
	return savedBasicLandWord{binding: binding}
}

// SecondBasicLandWord reads the saved replacement word.
func SecondBasicLandWord(result *BasicLandWordPairBinding) BasicLandWordRef {
	var binding *wordPairBinding
	if result != nil {
		binding = result.binding
	}
	return savedBasicLandWord{binding: binding, second: true}
}
func (r savedBasicLandWord) Resolve(ctx *EffectContext) (core.BasicLandType, bool) {
	pair, ok := ctx.Game.resolution.wordPairBindings[r.binding]
	if r.second {
		return core.BasicLandType(pair.second), ok
	}
	return core.BasicLandType(pair.first), ok
}
func (r savedBasicLandWord) inspectDefinition() definitionNode {
	return definitionNode{Role: "saved-basicland-word", bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}
