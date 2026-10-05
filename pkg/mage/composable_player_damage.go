package mage

import (
	"maps"

	"github.com/google/uuid"
)

type playerDamageEachEffect struct {
	source   DamageSourceRef
	subjects PlayerSet
	member   *PlayerBinding
	amount   ResolutionValue
}

// DealDamageEachPlayer reads each player's amount before one damage batch.
// The member binding is available only to the amount expression.
func DealDamageEachPlayer(source DamageSourceRef, subjects PlayerSet, member *PlayerBinding, amount ResolutionValue) ResolvingEffect {
	return &playerDamageEachEffect{source, subjects, member, amount}
}

func (*playerDamageEachEffect) Text() string { return "deal damage to each player" }
func (*playerDamageEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *playerDamageEachEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", children: []definitionEdge{
		{"source", damageSourceDefinition{e.source}}, {"subjects", e.subjects},
		{"amount", playerDamageAmount{e.member, e.amount}},
	}}
}

type playerDamageAmount struct {
	member *PlayerBinding
	amount ResolutionValue
}

func (v playerDamageAmount) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-damage-amount", isolatedBindings: true, children: []definitionEdge{
		{"member", damagePlayerMember{v.member}}, {"value", v.amount},
	}}
}

type damagePlayerMember struct{ binding *PlayerBinding }

func (m damagePlayerMember) inspectDefinition() definitionNode {
	return definitionNode{Role: "damage-player-member", bindingWrite: m.binding, children: []definitionEdge{{"binding", m.binding}}}
}

func (e *playerDamageEachEffect) Apply(ctx *EffectContext) error {
	source := e.source.resolveDamageSource(ctx)
	if !source.Valid() {
		return nil
	}
	ids := e.subjects.resolvePlayers(ctx)
	assignments := make([]DamageAssignment, 0, len(ids))
	seen := make(map[uuid.UUID]bool, len(ids))
	outer := ctx.Game.resolution.playerBindings
	for _, id := range ids {
		player := ctx.Game.GetPlayer(id)
		if player == nil || seen[id] {
			continue
		}
		seen[id] = true
		amount := func() int {
			frame := maps.Clone(outer)
			if frame == nil {
				frame = make(map[*PlayerBinding]uuid.UUID)
			}
			frame[e.member] = id
			ctx.Game.resolution.playerBindings = frame
			defer func() { ctx.Game.resolution.playerBindings = outer }()
			return e.amount.resolveValue(ctx)
		}()
		if amount > 0 {
			assignments = append(assignments, PlayerDamageAssignment(player, amount, source))
		}
	}
	ctx.Game.DealSimultaneousDamage(assignments...)
	return nil
}
