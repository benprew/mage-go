package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPaidPermanentSpellIdentityAndCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindPermanent("paid")
	spell := NewInstant("Offering", "", WithAbility(Spell(
		DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(2)))),
		WithAdditionalCost(SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	creature := NewCreature("Paid", "", 2, 2, WithKeyword(core.Lifelink))
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(creature.ID()) != nil {
		t.Fatal("cost did not sacrifice chosen creature")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 18 || a.Life() != 20 {
		t.Fatalf("copy or clone lost paid source controller and lifelink: clone A=%d B=%d, live A=%d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life(), a.Life())
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 18 {
		t.Fatal("paid source did not survive stack copy")
	}
}

func TestPaidPermanentFailedCommitAndDefinition(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindPermanent("paid")
	for _, change := range []func(*Game, Card){
		func(g *Game, p Card) { g.PhaseOut(p.ID()) },
		func(g *Game, p Card) {
			g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
			g.PutOnBattlefield(p, a.PlayerID())
		},
	} {
		creature := NewCreature("Paid", "", 2, 2)
		g.PutOnBattlefield(creature, a.PlayerID())
		tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{
			SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", paid),
		}})
		if err != nil {
			t.Fatal(err)
		}
		change(g, creature)
		if err := tx.Commit(); err == nil || len(tx.paidPermanents) != 0 {
			t.Fatal("stale choice produced a paid permanent")
		}
		if p := g.FindPermanent(creature.ID()); p != nil {
			g.RemoveFromBattlefield(p)
		}
	}
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", paid), ""},
		{SacrificeMatchingCostWithPermanent(IsCreature, "", paid), "text"},
		{SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", nil), "non-nil"},
	} {
		card := NewInstant("Test", "", WithAbility(Spell(DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(1)))), WithAdditionalCost(tc.cost))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	missing := NewInstant("Missing", "", WithAbility(Spell(DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(1)))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing binding validation = %v", err)
	}
	duplicate := NewInstant("Duplicate", "", WithAbility(Spell(DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(1)))),
		WithAdditionalCost(SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", paid)),
		WithAdditionalCost(SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", paid)))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate binding validation = %v", err)
	}
	resolutionCost := NewArtifact("Invalid", "", WithAbility(Activate(Costs(),
		IfPaid(You, Costs(SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", paid)),
			GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(resolutionCost.Name(), resolutionCost); err == nil || !strings.Contains(err.Error(), "activated ability cost") {
		t.Fatalf("resolution payment validation = %v", err)
	}
}

func TestPaidPermanentActivationRetainsFormerIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindPermanent("paid")
	source := NewArtifact("Offering", "", WithAbility(Activate(
		Costs(SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", paid)),
		DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(3)),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	creature := NewCreature("Paid", "", 2, 2, WithKeyword(core.Lifelink))
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(creature, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatalf("life A=%d B=%d, want former controller's lifelink", a.Life(), b.Life())
	}
}

func TestPaidPermanentAlternateCostCast(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindPermanent("paid")
	spell := NewInstant("Offering", "", WithAbility(Spell(
		DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(2)))),
		WithAdditionalCost(SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	g.PutOnBattlefield(NewCreature("Paid", "", 1, 1, WithKeyword(core.Lifelink)), a.PlayerID())
	if err := g.CastCardFromZoneWithAlternateCost(a.PlayerID(), spell.ID(), core.ZoneHand, core.ManaCost{}, nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatalf("life = %d, want damage and lifelink from paid creature", a.Life())
	}
}
