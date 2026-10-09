package mage

import (
	_ "embed"
	"slices"
)

// CardNameCatalog keeps legal English name choices separate from card factories.
type CardNameCatalog struct{ names []string }

//go:embed oracle_card_names.txt
var oracleCardNamesText string

var oracleCardNames = loadOracleCardNameCatalog()

// OracleCardNameCatalog returns the retained complete nontoken name snapshot.
// Separate faces are choices. Combined names are not choices under CR 201.4.
func OracleCardNameCatalog() *CardNameCatalog { return oracleCardNames }

// Names returns separate storage for the catalog's sorted names.
func (c *CardNameCatalog) Names() []string { return slices.Clone(c.names) }

// Contains tests an exact English name without a registry lookup.
func (c *CardNameCatalog) Contains(name string) bool {
	_, found := slices.BinarySearch(c.names, name)
	return found
}
func (c *CardNameCatalog) inspectDefinition() definitionNode {
	n := definitionNode{Role: "card-name-catalog"}
	if len(c.names) == 0 {
		n.problem = "supply a complete card name catalog"
	}
	return n
}

// CardNameBinding declares a name result local to one resolution frame.
type CardNameBinding struct{ name string }

// BindCardName declares a separate result with a diagnostic name.
func BindCardName(name string) *CardNameBinding { return &CardNameBinding{name: name} }

// Resolve reads the saved English name in the current frame.
func (b *CardNameBinding) Resolve(ctx *EffectContext) (string, bool) {
	name, ok := ctx.Game.resolution.cardNameBindings[b]
	return name, ok
}
func (b *CardNameBinding) bindingName() string { return b.name }
func (b *CardNameBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "card-name-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}

type cardNameChoice struct {
	player  PlayerRef
	catalog *CardNameCatalog
	binding *CardNameBinding
}

// ChooseCardName saves a required legal name at resolution without targeting.
// An absent chooser clears the result. An invalid answer uses the first name.
func ChooseCardName(player PlayerRef, catalog *CardNameCatalog, binding *CardNameBinding) ResolvingEffect {
	return &cardNameChoice{player: player, catalog: catalog, binding: binding}
}
func (*cardNameChoice) Text() string                 { return "choose a card name" }
func (*cardNameChoice) Properties() EffectProperties { return EffectProperties{} }
func (e *cardNameChoice) Apply(ctx *EffectContext) error {
	state := &ctx.Game.resolution
	if state.cardNameBindings == nil {
		state.cardNameBindings = make(map[*CardNameBinding]string)
	}
	delete(state.cardNameBindings, e.binding)
	chooser, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	answer := ctx.Game.GetPlayer(chooser).ChooseMode(e.catalog.Names(), e.Text())
	if answer < 0 || answer >= len(e.catalog.names) {
		answer = 0
	}
	state.cardNameBindings[e.binding] = e.catalog.names[answer]
	return nil
}
func (e *cardNameChoice) inspectDefinition() definitionNode {
	return definitionNode{Role: "card-name-choice", bindingWrite: e.binding, children: []definitionEdge{{"player", e.player}, {"catalog", e.catalog}, {"binding", e.binding}}}
}
func (e *cardNameChoice) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}

// CardNameRef reads a name result without a choice or a card lookup.
type CardNameRef interface {
	Resolve(*EffectContext) (string, bool)
	inspectDefinition() definitionNode
}

type savedCardNameRef struct{ binding *CardNameBinding }

// FromCardName reads a saved name in the current resolution frame.
func FromCardName(binding *CardNameBinding) CardNameRef { return savedCardNameRef{binding} }

func (r savedCardNameRef) Resolve(ctx *EffectContext) (string, bool) {
	return r.binding.Resolve(ctx)
}
func (r savedCardNameRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "card-name-ref", bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}
