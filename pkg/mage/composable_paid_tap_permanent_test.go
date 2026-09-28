package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPaidTapPermanentKeepsPaymentSourceThroughCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindPermanent("paid")
	source := NewArtifact("Source", "", WithAbility(Activate(
		Costs(TapNOtherMatchingCostWithPermanent(IsCreature, "Tap another creature", paid)),
		DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(2)),
	)))
	creature := NewCreature("Paid", "", 1, 1, WithKeyword(core.Lifelink))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(creature.ID()).Tapped {
		t.Fatal("payment did not tap the chosen creature")
	}
	g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
	g.PutOnBattlefield(creature, b.PlayerID())
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatalf("clone payment source: clone A=%d, live A=%d", clone.GetPlayer(a.PlayerID()).Life(), a.Life())
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 18 {
		t.Fatalf("paid source identity: A=%d B=%d", a.Life(), b.Life())
	}
}

func TestPaidTapPermanentRejectsStalePayment(t *testing.T) {
	a := NewBasePlayer("A")
	for _, change := range []func(*Game, Card){
		func(g *Game, c Card) { g.PhaseOut(c.ID()) },
		func(g *Game, c Card) {
			g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
			g.PutOnBattlefield(c, a.PlayerID())
		},
	} {
		g := NewGame(a, NewBasePlayer("B"))
		paid := BindPermanent("paid")
		source := NewArtifact("Source", "")
		creature := NewCreature("Paid", "", 1, 1)
		g.PutOnBattlefield(source, a.PlayerID())
		g.PutOnBattlefield(creature, a.PlayerID())
		tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{
			TapNOtherMatchingCostWithPermanent(IsCreature, "Tap another creature", paid), LifePayCost(2),
		}})
		if err != nil {
			t.Fatal(err)
		}
		change(g, creature)
		if err := tx.Commit(); err == nil || len(tx.paidPermanents) != 0 || a.Life() != 20 {
			t.Fatalf("stale payment changed live state: err=%v, paid=%v, life=%d", err, tx.paidPermanents, a.Life())
		}
	}
}

func TestPaidTapPermanentSpellAdditionalCost(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindPermanent("paid")
	spell := NewInstant("Spell", "", WithAbility(Spell(
		DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(2)))),
		WithAdditionalCost(TapNOtherMatchingCostWithPermanent(IsCreature, "Tap a creature", paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	creature := NewCreature("Paid", "", 1, 1, WithKeyword(core.Lifelink))
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(creature.ID()).Tapped {
		t.Fatal("additional cost did not tap the chosen creature")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatalf("paid source characteristics: A=%d B=%d", a.Life(), b.Life())
	}
}

func TestPaidTapPermanentDeclaration(t *testing.T) {
	paid := BindPermanent("paid")
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{TapNOtherMatchingCostWithPermanent(IsCreature, "Tap another creature", paid), ""},
		{TapNOtherMatchingCostWithPermanent(IsCreature, "", paid), "text"},
		{TapNOtherMatchingCostWithPermanent(IsCreature, "Tap another creature", nil), "non-nil"},
		{TapNOtherMatchingCostWithPermanent(IsCreature, "Tap another creature", BindPermanent("")), "name"},
	} {
		card := NewArtifact("Source", "", WithAbility(Activate(Costs(tc.cost), TapSubject(From(paid)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("definition = %v, want %q", err, tc.want)
		}
	}
	missing := NewArtifact("Missing", "", WithAbility(Activate(Costs(), TapSubject(From(paid)))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing producer = %v", err)
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(
		TapNOtherMatchingCostWithPermanent(IsCreature, "Tap another creature", paid),
		TapNOtherMatchingCostWithPermanent(IsCreature, "Tap another creature", paid),
	), TapSubject(From(paid)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate producer = %v", err)
	}
	resolutionCost := NewArtifact("Resolution", "", WithAbility(Activate(Costs(),
		IfPaid(You, Costs(TapNOtherMatchingCostWithPermanent(IsCreature, "Tap another creature", paid)),
			TapSubject(From(paid))))))
	if err := ValidateCardDefinition(resolutionCost.Name(), resolutionCost); err == nil || !strings.Contains(err.Error(), "activated ability cost") {
		t.Fatalf("resolution payment binding = %v", err)
	}
}
