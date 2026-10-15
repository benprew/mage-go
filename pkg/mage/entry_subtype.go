package mage

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type enterWithoutSubtypeOption struct {
	family        SubtypeFamily
	subtype       string
	asEnchantment bool
}

// EnterWithoutSubtype removes one subtype from the exact entrant in layer 4.
// The printed card and later battlefield visits keep their normal subtypes.
func EnterWithoutSubtype(family SubtypeFamily, subtype string) BattlefieldEntryOption {
	return enterWithoutSubtypeOption{family: family, subtype: subtype}
}

// EnterAsNonAuraEnchantment sets the entrant's card type and removes Aura.
// A copied Aura ability can return a card with a different printed type.
func EnterAsNonAuraEnchantment() BattlefieldEntryOption {
	return enterWithoutSubtypeOption{family: SubtypeEnchantment, subtype: "Aura", asEnchantment: true}
}
func (enterWithoutSubtypeOption) battlefieldEntryOption() {}
func (o enterWithoutSubtypeOption) applyBattlefieldEntry(e *battlefieldEntryOptions) {
	e.withoutSubtype = &o
	e.withoutSubtypeCount++
}
func (o enterWithoutSubtypeOption) apply(g *Game, p *Permanent) {
	if o.asEnchantment {
		setCardTypesAtCurrentLayer(g, p, core.TypeEnchantment)
	}
	if !p.supportsSubtypeFamily(o.family) {
		return
	}
	var retained []string
	for _, subtype := range p.computedSubtypes() {
		family, ok := subtypeFamilyForCard(p.Card, subtype)
		if ok && family == o.family && subtype != o.subtype {
			retained = append(retained, subtype)
		}
	}
	p.setSubtypeFamily(o.family, retained)
}

type entrySubtypeEffect struct {
	effectSource
	subject capturedPermanent
	removal enterWithoutSubtypeOption
}

func (*entrySubtypeEffect) Text() string                 { return "entry without subtype" }
func (*entrySubtypeEffect) Properties() EffectProperties { return EffectProperties{} }
func (*entrySubtypeEffect) GetLayer() core.Layer         { return core.LayerType }
func (*entrySubtypeEffect) GetDuration() core.Duration   { return core.Indefinite }
func (*entrySubtypeEffect) survivesSourceExit() bool     { return true }
func (e *entrySubtypeEffect) IsActive(g *Game) bool {
	p := g.FindPermanent(e.subject.id)
	return p != nil && !p.PhasedOut && p.incarnationID == e.subject.incarnation
}
func (e *entrySubtypeEffect) ApplyContinuous(ctx *EffectContext) error {
	if e.IsActive(ctx.Game) {
		e.removal.apply(ctx.Game, ctx.Game.MutablePermanent(e.subject.id))
	}
	return nil
}
func (e *entrySubtypeEffect) cloneEffect() ContinuousEffect { clone := *e; return &clone }

func (e battlefieldEntryOptions) prepareSubtype(g *Game, p *Permanent) func() {
	if e.withoutSubtype == nil {
		return func() {}
	}
	effect := &entrySubtypeEffect{subject: capturedPermanent{p.ID(), p.incarnationID}, removal: *e.withoutSubtype}
	g.effects.Add(effect)
	e.withoutSubtype.apply(g, p)
	return func() {
		g.effects.effects = slices.DeleteFunc(g.effects.effects, func(candidate ContinuousEffect) bool { return candidate == effect })
	}
}

func (g *Game) discardUnregisteredEntrySubtype(p *Permanent) {
	if p == nil {
		return
	}
	if live := g.FindPermanentIncludingPhased(p.ID()); live != nil && live.incarnationID == p.incarnationID {
		return
	}
	g.effects.effects = slices.DeleteFunc(g.effects.effects, func(candidate ContinuousEffect) bool {
		effect, ok := candidate.(*entrySubtypeEffect)
		return ok && effect.subject == (capturedPermanent{p.ID(), p.incarnationID})
	})
}

func setCardTypesAtCurrentLayer(g *Game, p *Permanent, kinds ...core.CardType) {
	p.cardTypeOverride = slices.Clone(kinds)
	for _, item := range []struct {
		kind core.CardType
		attr core.Attr
	}{
		{core.TypeCreature, core.AttrIsCreature},
		{core.TypeArtifact, core.AttrIsArtifact},
		{core.TypeEnchantment, core.AttrIsEnchantment},
		{core.TypeLand, core.AttrIsLand},
	} {
		if slices.Contains(kinds, item.kind) {
			g.effects.GrantAttr(p.ID(), item.attr)
			p.grantedAttrs[item.attr] = 1
		} else {
			g.effects.RevokeAttr(p.ID(), item.attr)
			p.grantedAttrs[item.attr] = -(p.baseAttrs[item.attr] + 1)
		}
	}
	if !slices.Contains(kinds, core.TypeCreature) {
		g.effects.RevokeAttr(p.ID(), core.AttrHasPowerToughness)
		p.grantedAttrs[core.AttrHasPowerToughness] = -(p.baseAttrs[core.AttrHasPowerToughness] + 1)
	}
	refreshIntrinsicBasicLandManaAbilities(p)
}
