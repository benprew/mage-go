package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// RegenerateInsteadOfDestroy replaces each eligible destruction of the source.
// Current abilities must contain this declaration when destruction is proposed.
func RegenerateInsteadOfDestroy() Ability {
	return &staticRegenerationAbility{BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic}}
}

type staticRegenerationAbility struct{ BaseAbility }

func (*staticRegenerationAbility) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-destruction-replacement", Text: "regenerate instead of destruction", battlefieldSource: true}
}

type staticRegenerationKey struct{ incarnation, ability uuid.UUID }
type staticRegenerationReplacement struct {
	replacementBase
	key staticRegenerationKey
}

func (r *staticRegenerationReplacement) IsActive(g GameReader) bool {
	p := g.FindPermanent(r.sourceID)
	if p == nil || p.PhasedOut || p.FaceDown || p.incarnationID != r.key.incarnation {
		return false
	}
	for _, a := range p.RuntimeAbilities {
		if a.AbilityID() == r.key.ability {
			if _, ok := UnwrapAbility(a).(*staticRegenerationAbility); ok {
				return true
			}
		}
	}
	return false
}
func (r *staticRegenerationReplacement) Matches(action Action, g GameReader) bool {
	a, ok := action.(*DestroyPermanentAction)
	return ok && !a.cannotRegenerate && a.PermanentID() == r.sourceID && r.IsActive(g) && g.CanRegeneratePermanent(g.FindPermanent(r.sourceID))
}
func (r *staticRegenerationReplacement) Replace(action Action, g *Game) Action {
	if !r.Matches(action, g) {
		return action
	}
	g.regeneratePermanent(g.MutablePermanent(r.sourceID))
	return nil
}
func (r *staticRegenerationReplacement) Clone() ReplacementEffect { c := *r; return &c }
func (*staticRegenerationReplacement) Text() string               { return "regenerate instead of destruction" }
func (a *DestroyPermanentAction) staticRegenerationCandidates(g *Game) []ReplacementEffect {
	p := g.FindPermanent(a.permanentID)
	if p == nil || p.PhasedOut || p.FaceDown || a.cannotRegenerate {
		return nil
	}
	if a.regeneration == nil {
		a.regeneration = make(map[staticRegenerationKey]*staticRegenerationReplacement)
	}
	var candidates []ReplacementEffect
	for _, ability := range p.RuntimeAbilities {
		if _, ok := UnwrapAbility(ability).(*staticRegenerationAbility); !ok {
			continue
		}
		key := staticRegenerationKey{p.incarnationID, ability.AbilityID()}
		r := a.regeneration[key]
		if r == nil {
			r = &staticRegenerationReplacement{replacementBase: replacementBase{sourceID: p.ID()}, key: key}
			a.regeneration[key] = r
		}
		if r.Matches(a, g) {
			candidates = append(candidates, r)
		}
	}
	return candidates
}

// ReplaceNextDestructionThisTurn removes damage instead of one destruction.
// The saved visit survives source departure and cannot protect a returned object.
func ReplaceNextDestructionThisTurn(subject PermanentRef) ResolvingEffect {
	return &nextDestructionEffect{subject: subject}
}

type nextDestructionEffect struct{ subject PermanentRef }

func (*nextDestructionEffect) Text() string {
	return "remove damage instead of next destruction this turn"
}
func (*nextDestructionEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *nextDestructionEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "future-destruction-replacement", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
}
func (e *nextDestructionEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject)
}
func (e *nextDestructionEffect) Apply(ctx *EffectContext) error {
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok || p == nil || p.PhasedOut {
		return nil
	}
	ctx.Game.effects.AddReplacement(&nextDestructionReplacement{replacementBase: replacementBase{sourceID: ctx.SourceID, duration: core.EndOfTurn}, subject: p.ID(), incarnation: p.incarnationID})
	return nil
}

type nextDestructionReplacement struct {
	replacementBase
	subject, incarnation uuid.UUID
	consumed             bool
}

func (*nextDestructionReplacement) survivesSourceExit() bool   { return true }
func (r *nextDestructionReplacement) IsActive(GameReader) bool { return !r.consumed }
func (r *nextDestructionReplacement) Matches(action Action, g GameReader) bool {
	a, ok := action.(*DestroyPermanentAction)
	if !ok || a.PermanentID() != r.subject || r.consumed {
		return false
	}
	p := g.FindPermanent(r.subject)
	return p != nil && !p.PhasedOut && p.incarnationID == r.incarnation
}
func (r *nextDestructionReplacement) Replace(action Action, g *Game) Action {
	if !r.Matches(action, g) {
		return action
	}
	r.consumed = true
	removeMarkedDamage(g.MutablePermanent(r.subject))
	return nil
}
func (r *nextDestructionReplacement) Clone() ReplacementEffect { c := *r; return &c }
func (*nextDestructionReplacement) Text() string               { return "remove damage instead of next destruction" }
func removeMarkedDamage(p *Permanent) {
	p.Damage = 0
	p.DeathtouchDamage = false
}
func (g *Game) regeneratePermanent(p *Permanent) {
	g.TapPermanent(p)
	removeMarkedDamage(p)
	g.RemoveFromCombat(p.ID())
}
