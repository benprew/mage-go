package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// AbilityRef keeps an ability separate from its source and from spells.
type AbilityRef interface {
	StackObjectRef
	abilityRef()
}

// AbilityTargetBinding stores no choices; each stack object keeps its targets.
type AbilityTargetBinding struct{ *targetDeclaration }

// DeclareActivatedAbilityTarget requires the current or last source type.
func DeclareActivatedAbilityTarget(name string, sourceType core.CardType) *AbilityTargetBinding {
	return &AbilityTargetBinding{&targetDeclaration{name: name, spec: &activatedAbilityTarget{BaseTarget: BaseTarget{min: 1, max: 1}, sourceType: sourceType}}}
}

func (*AbilityTargetBinding) abilityRef()     {}
func (*AbilityTargetBinding) stackObjectRef() {}
func (b *AbilityTargetBinding) resolveStackObject(ctx *EffectContext) (*StackObject, bool) {
	value, ok := firstTargetValue(ctx.targetBindings[b.targetDeclaration])
	if !ok {
		return nil, false
	}
	obj := ctx.Game.stack.FindByID(value.id)
	return obj, obj != nil && obj.IsAbility && obj.activatedAbility
}
func (b *AbilityTargetBinding) inspectDefinition() definitionNode {
	return targetReadNode(b.targetDeclaration, "ability-ref")
}

type activatedAbilityTarget struct {
	BaseTarget
	sourceType core.CardType
}

func (t *activatedAbilityTarget) Possible(_ uuid.UUID, _ Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, obj := range g.stack.Objects() {
		if obj.IsAbility && obj.activatedAbility && abilitySourceHasType(obj, g, t.sourceType) {
			result = append(result, obj.ID)
		}
	}
	return result
}
func (*activatedAbilityTarget) Choose(uuid.UUID, Card, *Game, []uuid.UUID) error { return nil }
func (t *activatedAbilityTarget) inspectDefinition() definitionNode {
	n := definitionNode{Role: "ability-target", Text: "activated ability with source type"}
	if !slices.Contains(core.CardTypeValues(), t.sourceType) {
		n.problem = "supply a valid source card type"
	}
	return n
}

func abilitySourceHasType(obj *StackObject, g *Game, kind core.CardType) bool {
	if obj.sourceIncarnation != uuid.Nil {
		ctx := &EffectContext{Game: g, SourceID: obj.SourceID, sourceCaptured: true, sourceIncarnation: obj.sourceIncarnation}
		p, lki := readPermanentInformation(This, ctx)
		if p != nil {
			return p.HasType(kind)
		}
		return lki != nil && lki.HasType(kind)
	}
	if obj.abilitySourceVisit != uuid.Nil && g.graveyardVisit(obj.SourceID) == obj.abilitySourceVisit {
		for _, player := range g.AllPlayers() {
			for _, card := range player.Graveyard() {
				if card.ID() == obj.SourceID {
					return card.HasType(kind)
				}
			}
		}
	}
	return obj.TargetSource != nil && obj.TargetSource.HasType(kind)
}

// ControllerOfAbility reads the ability controller while the ability exists.
func ControllerOfAbility(ability AbilityRef) PlayerRef { return abilityControllerRef{ability} }

type abilityControllerRef struct{ ability AbilityRef }

func (abilityControllerRef) playerRef()       {}
func (abilityControllerRef) damageRecipient() {}
func (r abilityControllerRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (r abilityControllerRef) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	obj, ok := r.ability.resolveStackObject(ctx)
	if !ok {
		return uuid.Nil, false
	}
	return obj.Controller, ctx.Game.GetPlayer(obj.Controller) != nil
}
func (r abilityControllerRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-ref", Text: "ability controller", children: []definitionEdge{{"ability", r.ability}}}
}

// CounterAbility uses the same counter operation as spells.
func CounterAbility(ability AbilityRef) ResolvingEffect {
	return &counterReferencedEffect{spell: ability, ability: true}
}
