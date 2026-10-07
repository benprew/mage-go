package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCantBeTargetedBySpellsIf(t *testing.T) {
	g, a, b := newSpellCopyGame()
	card := NewCreature("Conditional", "", 2, 3, WithAbility(Static(CantBeTargetedBySpellsIf(This, NewCardFilter("any card", func(Card) bool { return true }), IsYourTurn()))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	spell := NewInstant("Spell", "")
	check := func(game *Game, allowed bool) {
		t.Helper()
		if got := game.FindPermanent(p.ID()).CanBeTargetedBy(SpellTargetSource(spell), b.PlayerID(), game); got != allowed {
			t.Fatalf("target allowed=%v, want %v", got, allowed)
		}
	}
	check(g, false)
	clone := g.Clone()
	g.SetActivePlayerIndex(1)
	check(g, true)
	check(clone, false)
	g.AddControlEffect(ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	check(g, false)
	g.PhaseIn(g.PhaseOut(p.ID()))
	check(g, false)
	if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), p.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	check(g, true)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	check(g, false)
	if !g.FindPermanent(p.ID()).CanBeTargetedBy(spell, a.PlayerID(), g) {
		t.Fatal("spell condition restricted an ability")
	}
}

func TestCantBeTargetedBySpellsIfInvalid(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "nil condition", Card: NewCreature("Bad", "", 2, 3, WithAbility(Static(CantBeTargetedBySpellsIf(This, NewCardFilter("any card", func(Card) bool { return true }), nil)))), WantErr: true},
		{Name: "invalid condition", Card: NewCreature("Bad", "", 2, 3, WithAbility(Static(CantBeTargetedBySpellsIf(This, NewCardFilter("any card", func(Card) bool { return true }), AllConditions())))), WantErr: true},
		{Name: "nil subject", Card: NewCreature("Bad", "", 2, 3, WithAbility(Static(CantBeTargetedBySpellsIf(nil, NewCardFilter("any card", func(Card) bool { return true }), IsYourTurn())))), WantErr: true},
		{Name: "empty filter", Card: NewCreature("Bad", "", 2, 3, WithAbility(Static(CantBeTargetedBySpellsIf(This, CardFilter{}, IsYourTurn())))), WantErr: true},
	})
}

func TestCantBeTargetedBySpellsIfAttachment(t *testing.T) {
	g, a, b := newSpellCopyGame()
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 3), b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 2, 3), a.PlayerID())
	auraCard := NewAura("Condition source", "", WithAbility(Static(CantBeTargetedBySpellsIf(AttachedPermanent,
		NewCardFilter("instant", func(c Card) bool { return c.HasType(core.TypeInstant) }), IsYourTurn()))))
	aura := g.PutOnBattlefield(auraCard, a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	spell := NewInstant("Spell", "")
	allowed := func(game *Game, id uuid.UUID) bool {
		return game.FindPermanent(id).CanBeTargetedBy(SpellTargetSource(spell), b.PlayerID(), game)
	}
	if allowed(g, host.ID()) || !allowed(g, other.ID()) {
		t.Fatal("wrong conditional host")
	}
	clone := g.Clone()
	g.PhaseOut(aura.ID())
	if !allowed(g, host.ID()) || allowed(clone, host.ID()) {
		t.Fatal("phased source or clone condition")
	}
	g.PhaseIn([]uuid.UUID{aura.ID()})
	g.Attach(aura.ID(), other.ID())
	if !allowed(g, host.ID()) || allowed(g, other.ID()) {
		t.Fatal("restriction did not follow attachment")
	}
	g.RemoveFromBattlefield(g.FindPermanent(other.ID()))
	returned := g.PutOnBattlefield(other.Card, a.PlayerID())
	if !allowed(g, returned.ID()) {
		t.Fatal("old attachment followed returned host")
	}
	g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
	if !allowed(g, host.ID()) {
		t.Fatal("departed source retained rule")
	}
}
