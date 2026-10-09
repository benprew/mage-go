package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestActivationDefinedXPayment(t *testing.T) {
	for _, count := range []int{0, 2, 4} {
		g, a, _ := newSpellCopyGame()
		action := NewAction(ActionActivated, WithCost(ManaCostOf("{X}{X}")), WithCost(Tap()), WithEffect(GainLifeFor(You, CountersOnWithLKI(This, core.Pin))), WithActivationManaXFromSourceCounters(core.Pin))
		card := NewArtifact("Defined X", "", WithAbility(action))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		p := g.PutOnBattlefield(card, a.PlayerID())
		g.MutablePermanent(p.ID()).Counters[core.Pin] = uint8(count)
		g.SetXValue(99)
		if count > 0 {
			a.ManaPool().Add(core.Colorless, 2*count-1)
			if action := UnwrapAbility(g.FindPermanent(p.ID()).RuntimeAbilities[0]).(ActivatedAbility); action.CanActivate(a.PlayerID(), g) {
				t.Fatal("unaffordable query")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
				t.Fatal("underpayment accepted")
			}
			if g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 2*count-1 || g.StackSize() != 0 || g.XValue() != 99 {
				t.Fatal("failed payment changed state")
			}
			a.ManaPool().Add(core.Colorless, 1)
		}
		if action := UnwrapAbility(g.FindPermanent(p.ID()).RuntimeAbilities[0]).(ActivatedAbility); !action.CanActivate(a.PlayerID(), g) {
			t.Fatal("query used ambient X")
		}
		if g.XValue() != 99 || g.FindPermanent(p.ID()).Tapped {
			t.Fatal("query changed state")
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if g.StackPeek().XValue != count || a.ManaPool().TotalMana() != 0 || !g.FindPermanent(p.ID()).Tapped || g.XValue() != 99 {
			t.Fatal("wrong announcement")
		}
		g.MutablePermanent(p.ID()).Counters[core.Pin] = 5
		g.ResolveStack()
		if a.Life() != 25 {
			t.Fatal("resolution used payment count")
		}
	}
}

func TestActivationDefinedXDeclarations(t *testing.T) {
	for _, tc := range []struct {
		name string
		card Card
	}{
		{"spell", NewInstant("Bad", "{X}", WithAbility(NewAction(ActionSpell, WithEffect(GainLifeFor(You, Fixed(1))), WithActivationManaXFromSourceCounters(core.Pin))))},
		{"invalid", NewArtifact("Bad", "", WithAbility(NewAction(ActionActivated, WithCost(ManaCostOf("{X}")), WithEffect(GainLifeFor(You, Fixed(1))), WithActivationManaXFromSourceCounters(core.NumCounters))))},
		{"duplicate", NewArtifact("Bad", "", WithAbility(NewAction(ActionActivated, WithCost(ManaCostOf("{X}")), WithEffect(GainLifeFor(You, Fixed(1))), WithActivationManaXFromSourceCounters(core.Pin), WithActivationManaXFromSourceCounters(core.Pin))))},
		{"no X", NewArtifact("Bad", "", WithAbility(NewAction(ActionActivated, WithCost(ManaCostOf("{1}")), WithEffect(GainLifeFor(You, Fixed(1))), WithActivationManaXFromSourceCounters(core.Pin))))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateCardDefinition("Bad", tc.card); err == nil {
				t.Fatal("invalid declaration accepted")
			}
		})
	}
}

func TestActivationDefinedXLockedCostAndClones(t *testing.T) {
	g, a, b := newSpellCopyGame()
	newCard := func() Card {
		return NewArtifact("Defined", "", WithAbility(NewAction(ActionActivated,
			WithCost(ManaCostOf("{X}{X}")), WithCost(RemoveCountersCost(core.Pin, 1)),
			WithEffect(GainLifeFor(You, CountersOnWithLKI(This, core.Pin))),
			WithActivationManaXFromSourceCounters(core.Pin))))
	}
	first := g.PutOnBattlefield(newCard(), a.PlayerID())
	second := g.PutOnBattlefield(newCard(), a.PlayerID())
	g.MutablePermanent(first.ID()).Counters[core.Pin] = 2
	g.MutablePermanent(second.ID()).Counters[core.Pin] = 3
	clone := g.Clone()
	clone.GetPlayer(a.PlayerID()).ManaPool().Add(core.Colorless, 4)
	clone.MutablePermanent(first.ID()).Counters[core.Pin] = 1
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if clone.StackPeek().XValue != 1 || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 2 {
		t.Fatal("clone payment")
	}
	if g.StackSize() != 0 || g.FindPermanent(first.ID()).Counters[core.Pin] != 2 {
		t.Fatal("clone changed original")
	}
	a.ManaPool().Add(core.Colorless, 10)
	g.SetXValue(90)
	for _, p := range []*Permanent{first, second} {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	if a.ManaPool().TotalMana() != 0 || g.StackPeek().XValue != 3 || g.XValue() != 90 {
		t.Fatal("cost was not locked before counter removal")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy")
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 22 {
		t.Fatal("independent pending instances")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("clone counter removal")
	}
}

func TestActivationDefinedXReductionAndSacrifice(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	card := NewArtifact("Defined", "", WithAbility(NewAction(ActionActivated,
		WithCost(ManaCostOf("{X}{X}")), WithCost(SacrificeSourceCost()),
		WithEffect(GainLifeFor(You, CountersOnWithLKI(This, core.Pin))), WithActivationManaXFromSourceCounters(core.Pin))))
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.MutablePermanent(p.ID()).Counters[core.Pin] = 3
	aura := g.PutOnBattlefield(NewAura("Reduction", "", WithCastTarget(TargetArtifact()), WithStaticAbility(ReduceActivationCostStatic(AttachedPermanent, 2, 1))), a.PlayerID())
	g.Attach(aura.ID(), p.ID())
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.StackPeek().XValue != 3 || a.ManaPool().TotalMana() != 0 || g.FindPermanent(p.ID()) != nil {
		t.Fatal("reduction before X or failed sacrifice")
	}
	g.ResolveStack()
	if a.Life() != 23 {
		t.Fatal("sacrifice lost source count")
	}
}
