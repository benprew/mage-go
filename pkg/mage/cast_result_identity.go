package mage

import "github.com/google/uuid"

// CastHandCardFaceDownWithoutPayingInto saves the exact spell from a legal cast.
// A failed cast clears the result and does not stop later instructions.
func CastHandCardFaceDownWithoutPayingInto(player PlayerRef, card CardRef, result *DamageSourceBinding) ResolvingEffect {
	return &faceDownCastEffect{player: player, card: card, result: result, captureResult: true}
}

// MayCastHandCardFaceDownWithoutPayingInto saves an accepted, legal cast.
// A declined or failed cast clears the result.
func MayCastHandCardFaceDownWithoutPayingInto(player PlayerRef, card CardRef, result *DamageSourceBinding) ResolvingEffect {
	return &optionalFaceDownCast{faceDownCastEffect{player: player, card: card, result: result, captureResult: true}}
}

type resolvedSpellPermanentRef struct{ spell DamageSourceRef }

// PermanentFromResolvedSpell reads only the permanent that one exact spell became.
// A countered spell, a departed permanent, or a phased permanent has no subject.
func PermanentFromResolvedSpell(spell DamageSourceRef) PermanentRef {
	return resolvedSpellPermanentRef{spell: spell}
}
func (resolvedSpellPermanentRef) permanentRef() {}
func (r resolvedSpellPermanentRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-ref", Text: "permanent from resolved spell", children: []definitionEdge{{"spell", damageSourceDefinition{r.spell}}}}
}
func (r resolvedSpellPermanentRef) resolvePermanent(ctx *EffectContext) (*Permanent, bool) {
	spell := r.spell.resolveDamageSource(ctx)
	if !spell.hasExactIdentity() {
		return nil, false
	}
	p := ctx.Game.FindPermanent(spell.ID())
	if p == nil || p.PhasedOut || p.resolvedSpellID != spell.StackID() || spell.StackID() == uuid.Nil {
		return nil, false
	}
	return p, true
}

func (e *faceDownCastEffect) clearResult(ctx *EffectContext) {
	if e.captureResult {
		delete(ctx.Game.resolution.damageSourceBindings, e.result)
	}
}

func (resolvedSpellPermanentRef) damageRecipient() {}
func (r resolvedSpellPermanentRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return permanentRecipient(r, ctx)
}

func (r resolvedSpellPermanentRef) informationIdentity(ctx *EffectContext) (capturedPermanent, bool) {
	spell := r.spell.resolveDamageSource(ctx)
	if !spell.hasExactIdentity() || spell.StackID() == uuid.Nil {
		return capturedPermanent{}, false
	}
	if p := ctx.Game.FindPermanentIncludingPhased(spell.ID()); p != nil && p.resolvedSpellID == spell.StackID() {
		return capturedPermanent{id: p.ID(), incarnation: p.incarnationID}, true
	}
	for incarnation, lki := range ctx.Game.zones.damageSourceLKI {
		if lki.ID == spell.ID() && lki.Snapshot != nil && lki.Snapshot.resolvedSpellID == spell.StackID() {
			return capturedPermanent{id: lki.ID, incarnation: incarnation}, true
		}
	}
	return capturedPermanent{}, false
}
