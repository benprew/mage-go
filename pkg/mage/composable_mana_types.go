package mage

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ManaTypesBinding names the mana types saved for one trigger instance.
type ManaTypesBinding struct{ name string }

func cloneManaTypesBindings(src map[*ManaTypesBinding][]core.Color) map[*ManaTypesBinding][]core.Color {
	if src == nil {
		return nil
	}
	dst := make(map[*ManaTypesBinding][]core.Color, len(src))
	for binding, types := range src {
		dst[binding] = slices.Clone(types)
	}
	return dst
}

// BindManaTypes declares an event-time set of mana types.
func BindManaTypes(name string) *ManaTypesBinding { return &ManaTypesBinding{name: name} }

func (b *ManaTypesBinding) bindingName() string { return b.name }
func (b *ManaTypesBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "mana-types-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}

type eventManaTypesCapture struct {
	binding *ManaTypesBinding
	types   []core.Color
}

func (*eventManaTypesCapture) Text() string                 { return "remember produced mana types" }
func (*eventManaTypesCapture) Properties() EffectProperties { return EffectProperties{} }
func (e *eventManaTypesCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-mana-types-capture", bindingWrite: e.binding, children: []definitionEdge{{"types", e.binding}}}
}
func (e *eventManaTypesCapture) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.manaTypesBindings == nil {
		ctx.Game.resolution.manaTypesBindings = make(map[*ManaTypesBinding][]core.Color)
	}
	ctx.Game.resolution.manaTypesBindings[e.binding] = slices.Clone(e.types)
	return nil
}

type manaFromProducedTypesEffect struct {
	player PlayerRef
	types  *ManaTypesBinding
	amount ResolutionValue
}

// AddManaOfProducedTypeToPlayer adds mana of one event-time produced type.
func AddManaOfProducedTypeToPlayer(player PlayerRef, types *ManaTypesBinding, amount ResolutionValue) ResolvingEffect {
	return &manaFromProducedTypesEffect{player: player, types: types, amount: amount}
}

func (*manaFromProducedTypesEffect) Text() string                 { return "add mana of a produced type to player" }
func (*manaFromProducedTypesEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *manaFromProducedTypesEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), bindingRead: e.types,
		children: []definitionEdge{{"player", e.player}, {"types", e.types}, {"amount", e.amount}}}
}
func (e *manaFromProducedTypesEffect) Apply(ctx *EffectContext) error {
	playerID, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	types := ctx.Game.resolution.manaTypesBindings[e.types]
	if len(types) == 0 {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	player := ctx.Game.GetPlayer(playerID)
	selected := types[0]
	if len(types) > 1 {
		if choice := player.ChooseManaColor("add mana of a produced type"); slices.Contains(types, choice) {
			selected = choice
		}
	}
	child := *ctx
	child.Controller = playerID
	return AddMana(selected, amount).Apply(&child)
}
