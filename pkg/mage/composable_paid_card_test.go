package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPaidSacrificeCardSpellAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCard("paid")
	spell := NewInstant("Offering", "", WithAbility(Spell(IfCardHasType(FromCard(paid), core.TypeCreature, GainLifeFor(You, Fixed(2))))),
		WithAdditionalCost(SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	creature := NewCreature("Paid", "", 1, 1)
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatal("paid card escaped a stack instance")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("paid card was not available to both stack copies")
	}
}

func TestPaidSacrificeCardVisitAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCard("paid")
	spell := NewInstant("Visit", "", WithAbility(Spell(Sequence(
		ReturnCardToHand(FromCard(paid)),
		IfCardHasType(FromCard(paid), core.TypeCreature, GainLifeFor(You, Fixed(2))),
	))), WithAdditionalCost(SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	creature := NewCreature("Paid", "", 1, 1)
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 20 || len(a.Hand()) != 1 || a.Hand()[0].ID() != creature.ID() {
		t.Fatal("saved card followed a later zone visit")
	}

	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", paid), ""},
		{SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", nil), "non-nil"},
		{SacrificeMatchingCostWithCard(IsCreature, "", paid), "text"},
	} {
		card := NewInstant("Offering", "", WithAbility(Spell(ReturnCardToHand(FromCard(paid)))), WithAdditionalCost(tc.cost))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	missing := NewInstant("Missing", "", WithAbility(Spell(ReturnCardToHand(FromCard(paid)))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing payment producer: %v", err)
	}
	duplicate := NewInstant("Duplicate", "", WithAbility(Spell(ReturnCardToHand(FromCard(paid)))),
		WithAdditionalCost(SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", paid)),
		WithAdditionalCost(SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", paid)))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate payment producer: %v", err)
	}
}

func TestPaidSacrificeCardFailedPayment(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCard("paid")
	spell := NewInstant("Offering", "", WithAbility(Spell(ReturnCardToHand(FromCard(paid)))),
		WithAdditionalCost(SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err == nil {
		t.Fatal("cast without a creature succeeded")
	}
	if len(a.Hand()) != 1 || !g.stack.IsEmpty() || len(g.resolution.cardBindings) != 0 {
		t.Fatal("failed payment changed the game or saved a card")
	}
}

func TestPaidSacrificeTokenHasNoGraveyardCard(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCard("paid")
	source := NewArtifact("Offering", "", WithAbility(Activate(
		Costs(SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", paid)),
		IfCardHasType(FromCard(paid), core.TypeCreature, GainLifeFor(You, Fixed(2))),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	token := NewToken("Token", 1, 1, []core.CardType{core.TypeCreature}, nil)
	g.PutOnBattlefield(token, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("token produced a graveyard card")
	}
}

func TestPaidSacrificeCardActivationAndStaleChoice(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCard("paid")
	source := NewArtifact("Offering", "", WithAbility(Activate(
		Costs(SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", paid)),
		IfCardHasType(FromCard(paid), core.TypeCreature, GainLifeFor(You, Fixed(2))),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	creature := NewCreature("Paid", "", 1, 1)
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("activation lost its paid card")
	}

	for _, change := range []func(*Game, Card){
		func(g *Game, p Card) { g.PhaseOut(p.ID()) },
		func(g *Game, p Card) {
			g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
			g.PutOnBattlefield(p, a.PlayerID())
		},
	} {
		other := NewCreature("Other", "", 1, 1)
		g.PutOnBattlefield(other, a.PlayerID())
		tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
			Controller: a.PlayerID(), SourceID: source.ID(),
			Costs: []Cost{SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", paid)},
		})
		if err != nil {
			t.Fatal(err)
		}
		change(g, other)
		if err := tx.Commit(); err == nil || len(tx.paidCards) != 0 {
			t.Fatal("stale chosen incarnation produced a paid card")
		}
		g.RemoveFromBattlefield(g.FindPermanent(other.ID()))
	}
}
