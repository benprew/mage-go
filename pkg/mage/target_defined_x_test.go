package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetDefinedXPayment(t *testing.T) {
	g, a, b := newSpellCopyGame()
	target := DeclareSpellTarget("spell").WithFilter(SpellWithSingleTarget())
	action := NewAction(ActionActivated, WithCost(ManaCostOf("{X}")), WithCost(Tap()), WithTarget(target.targetSpec()), WithEffect(ChangeSingleSpellTargetToPlayer(target)), WithActivationManaXFromSpell(target, 2))
	c := NewArtifact("Mirror", "", WithAbility(action))
	if err := ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(c, a.PlayerID())
	spell := NewInstant("X spell", "{X}{R}", Spell(GainLifeFor(You, Fixed(1))))
	obj := &StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: b.PlayerID(), Targets: []uuid.UUID{a.PlayerID()}, XValue: 3}
	g.pushStack(obj)
	g.SetXValue(91)
	action = UnwrapAbility(g.FindPermanent(p.ID()).RuntimeAbilities[0]).(*ActionDefinition)
	a.ManaPool().Add(core.Colorless, 7)
	if action.CanActivate(a.PlayerID(), g) {
		t.Fatal("query ignored target cost")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{obj.ID}); err == nil {
		t.Fatal("underpaid target value")
	}
	if a.ManaPool().TotalMana() != 7 || g.FindPermanent(p.ID()).Tapped || g.XValue() != 91 {
		t.Fatal("rollback")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if !action.CanActivate(a.PlayerID(), g) || g.XValue() != 91 {
		t.Fatal("query changed state or used ambient X")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{obj.ID}); err != nil {
		t.Fatal(err)
	}
	if g.StackPeek().XValue != 8 || a.ManaPool().TotalMana() != 0 || g.XValue() != 91 {
		t.Fatal("X spell mana value")
	}
}

func TestTargetDefinedXInvalid(t *testing.T) {
	for _, variant := range []string{"unbound", "multiple", "no-X", "zero"} {
		target := DeclareSpellTarget("spell")
		opts := []ActionOption{WithCost(ManaCostOf("{X}")), WithEffect(ChangeSingleSpellTargetToPlayer(target)), WithTarget(target.targetSpec()), WithActivationManaXFromSpell(target, 2)}
		switch variant {
		case "unbound":
			opts[len(opts)-1] = WithActivationManaXFromSpell(DeclareSpellTarget("other"), 2)
		case "multiple":
			opts = append(opts, WithTarget(DeclareSpellTarget("other").targetSpec()))
		case "no-X":
			opts = []ActionOption{WithCost(ManaCostOf("{1}")), WithTarget(target.targetSpec()), WithEffect(ChangeSingleSpellTargetToPlayer(target)), WithActivationManaXFromSpell(target, 2)}
		case "zero":
			opts = []ActionOption{WithCost(ManaCostOf("{X}")), WithTarget(target.targetSpec()), WithEffect(ChangeSingleSpellTargetToPlayer(target)), WithActivationManaXFromSpell(target, 0)}
		}
		if err := ValidateCardDefinition("Bad", NewArtifact("Bad", "", WithAbility(NewAction(ActionActivated, opts...)))); err == nil {
			t.Fatal("invalid", variant)
		}
	}
}

func TestTargetDefinedXIdentityAndZero(t *testing.T) {
	g, a, b := newSpellCopyGame()
	target := DeclareSpellTarget("spell").WithFilter(SpellWithSingleTarget())
	action := NewAction(ActionActivated, WithCost(ManaCostOf("{X}")), WithCost(Tap()), WithTarget(target.targetSpec()), WithEffect(ChangeSingleSpellTargetToPlayer(target)), WithActivationManaXFromSpell(target, 2))
	p := g.PutOnBattlefield(NewArtifact("Mirror", "", WithAbility(action)), a.PlayerID())
	action = UnwrapAbility(g.FindPermanent(p.ID()).RuntimeAbilities[0]).(*ActionDefinition)
	g.SetXValue(23)
	if action.CanActivate(a.PlayerID(), g) {
		t.Fatal("query without target")
	}
	spell := NewInstant("Zero", "", NewTargetedSpell(TargetPlayer(), DealDamage(Fixed(3))))
	obj := &StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: b.PlayerID(), Targets: []uuid.UUID{a.PlayerID()}, TargetSpecs: spell.CastTargets(), Effects: []Effect{DealDamage(Fixed(3))}}
	g.pushStack(obj)
	for _, ids := range [][]uuid.UUID{nil, {spell.ID()}, {a.PlayerID()}, {uuid.New()}, {obj.ID, obj.ID}} {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, ids); err == nil {
			t.Fatal("invalid identity or count")
		}
		if g.FindPermanent(p.ID()).Tapped || g.XValue() != 23 || g.StackSize() != 1 {
			t.Fatal("invalid announcement changed state")
		}
	}
	g.PhaseOut(p.ID())
	if action.CanActivate(a.PlayerID(), g) {
		t.Fatal("phased source query")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{obj.ID}); err == nil {
		t.Fatal("phased source activation")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	if !action.CanActivate(a.PlayerID(), g) {
		t.Fatal("zero payment unavailable")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{obj.ID}); err != nil {
		t.Fatal(err)
	}
	if g.StackPeek().XValue != 0 || g.XValue() != 23 {
		t.Fatal("zero mana value")
	}
	a.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 17 {
		t.Fatal("zero value retarget")
	}
}
