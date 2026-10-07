package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type restrictedManaChooser struct {
	*BasePlayer
	calls int
	color core.Color
}

func (p *restrictedManaChooser) ChooseManaColor(string) core.Color {
	p.calls++
	return p.color
}

func TestRestrictedResolutionMana(t *testing.T) {
	for _, amount := range []int{-1, 0, 3} {
		a := &restrictedManaChooser{BasePlayer: NewBasePlayer("Alice"), color: core.Blue}
		b := NewBasePlayer("Bob")
		g := NewGame(a, b)
		effect := AddRestrictedManaOfAnyOneColor(Fixed(amount), core.CreatureSpellsOnly{})
		card := NewSorcery("Restricted", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
		g.resolution.manaTracking = true
		if err := effect.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		want := max(amount, 0)
		if a.ManaPool().Count(core.Blue) != want || b.ManaPool().TotalMana() != 0 || g.resolution.manaProduced != want {
			t.Fatal("wrong pool or production tracking")
		}
		if a.calls != min(want, 1) {
			t.Fatal("wrong choice count")
		}
		cost := core.ManaCost{Generic: want}
		creature := core.SpellPaymentContext{IsCreature: true}
		if !a.ManaPool().CanPay(cost, &creature) {
			t.Fatal("creature cannot spend mana")
		}
		if want > 0 && a.ManaPool().CanPay(cost, nil) {
			t.Fatal("activation can spend restricted mana")
		}
		clone := g.Clone()
		a.ManaPool().Clear()
		if clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != want {
			t.Fatal("pool clone changed")
		}
	}
}

func TestRestrictedResolutionManaDeclarations(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		AddRestrictedManaOfAnyOneColor(nil, core.CreatureSpellsOnly{}),
		AddRestrictedManaOfAnyOneColor(Fixed(1), nil),
	} {
		card := NewSorcery("Invalid", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestRestrictedResolutionManaPlanning(t *testing.T) {
	a, b := NewBasePlayer("Alice"), NewBasePlayer("Bob")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewLand("Restricted source", WithAbility(Activate(Costs(Tap()), AddRestrictedManaOfAnyOneColor(Fixed(2), core.CreatureSpellsOnly{})))), a.PlayerID())
	ability := source.Card.Abilities()[0]
	productions := ManaProductionsForAbility(ability)
	if len(productions) != 1 || productions[0].Restriction == nil || !IsActivatedManaAbility(ability) {
		t.Fatal("production metadata lost restriction")
	}
	cost := core.ManaCost{Generic: 2}
	creature := core.SpellPaymentContext{IsCreature: true}
	artifact := core.SpellPaymentContext{IsArtifact: true}
	for _, ctx := range []*core.SpellPaymentContext{nil, &artifact} {
		if g.CanAfford(a.PlayerID(), cost, ctx) {
			t.Fatal("planner used restricted source for an illegal payment")
		}
	}
	if !g.CanAfford(a.PlayerID(), cost, &creature) || g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
		t.Fatal("pure planning or matching payment failed")
	}
	clone := g.Clone()
	if err := g.mana.AutoTapForCost(g, a.PlayerID(), cost, AutoTapHint{}, &creature); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 2 || clone.FindPermanent(source.ID()).Tapped {
		t.Fatal("execution or clone isolation")
	}
	if a.ManaPool().CanPay(cost, nil) || a.ManaPool().CanPay(cost, &artifact) {
		t.Fatal("execution added unrestricted mana")
	}
}
