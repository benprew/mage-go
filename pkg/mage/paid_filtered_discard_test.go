package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPaidFilteredDiscardLockedVisit(t *testing.T) {
	for _, mode := range []string{"normal", "returned", "missing", "changed", "clone"} {
		t.Run(mode, func(t *testing.T) {
			a := &filteredDiscardPlayer{BasePlayer: NewBasePlayer("A")}
			g := NewGame(a, NewBasePlayer("B"))
			paid := BindCard("paid")
			first, second, land := NewArtifact("First", ""), NewArtifact("Second", ""), NewLand("Land")
			for _, c := range []Card{land, first, second} {
				a.AddToHand(c)
			}
			a.choice = second
			allowed := true
			filter := NewCardFilter("artifact card", func(c Card) bool { return allowed && c.HasType(TypeArtifact) })
			a.ManaPool().Add(Red, 1)
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{ManaPayment("{R}"), LifePayCost(2), DiscardMatchingCardCostInto(filter, paid)}})
			if err != nil {
				t.Fatal(err)
			}
			if len(a.candidates) != 2 || a.candidates[0].ID() != first.ID() || a.candidates[1].ID() != second.ID() {
				t.Fatal("choice included a nonmatching card")
			}
			switch mode {
			case "returned", "missing":
				c, _ := a.RemoveFromHand(second.ID())
				if mode == "returned" {
					a.AddToHand(c)
				}
			case "changed":
				allowed = false
			case "clone":
				clone := g.Clone()
				copied := *tx
				copied.game = clone
				if err := copied.Commit(); err != nil {
					t.Fatal(err)
				}
				if len(a.Hand()) != 3 || len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 1 {
					t.Fatal("clone payment changed original")
				}
			}
			err = tx.Commit()
			if mode == "returned" || mode == "missing" || mode == "changed" {
				if err == nil {
					t.Fatal("stale or ineligible choice paid")
				}
				if a.Life() != 20 || a.ManaPool().Count(Red) != 1 || len(a.Graveyard()) != 0 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 {
					t.Fatal("failed payment spent resources")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != second.ID() || a.Life() != 18 {
					t.Fatal("payment used another card")
				}
			}
			if err == nil && tx.paidCards[paid].id != second.ID() {
				t.Fatal("capture did not use the chosen card")
			}
			if err != nil && len(tx.paidCards) != 0 {
				t.Fatal("failed payment published capture")
			}
			if a.calls != 1 {
				t.Fatalf("choice calls = %d", a.calls)
			}
		})
	}
}

func TestPaidFilteredDiscardInvalidAndReplacement(t *testing.T) {
	for _, mode := range []string{"nonmatching", "outside", "nil", "empty", "prevented", "hidden", "recheck"} {
		t.Run(mode, func(t *testing.T) {
			a := &filteredDiscardPlayer{BasePlayer: NewBasePlayer("A")}
			g := NewGame(a, NewBasePlayer("B"))
			paid := BindCard("paid")
			artifact, land := NewArtifact("Artifact", ""), NewLand("Land")
			a.AddToHand(land)
			if mode != "empty" {
				a.AddToHand(artifact)
			}
			a.choice = artifact
			switch mode {
			case "nonmatching":
				a.choice = land
			case "outside":
				a.choice = NewArtifact("Outside", "")
			case "nil":
				a.choice = nil
			case "prevented":
				g.effects.AddReplacement(&preventOneCostDiscardReplacement{card: artifact.ID()})
			case "hidden":
				g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
			case "recheck":
				g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
				g.effects.AddReplacement(&discardToHandReplacement{})
			}
			cost := DiscardMatchingCardCostInto(IsArtifactCard, paid)
			before := len(a.Hand())
			if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), cost}}); err == nil {
				t.Fatal("invalid payment prepared")
			}
			if err := cost.Pay(uuid.Nil, a.PlayerID(), g); err == nil {
				t.Fatal("invalid direct payment succeeded")
			}
			if a.Life() != 20 || len(a.Hand()) != before || len(a.Graveyard()) != 0 || len(a.Library()) != 0 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 {
				t.Fatal("invalid payment changed state")
			}
		})
	}
}

func TestPaidFilteredDiscardCardActivationAndClone(t *testing.T) {
	a, b := &paidLandChoicePlayer{NewBasePlayer("A")}, NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCard("discarded")
	source := NewArtifact("Edge", "", WithAbility(Activate(Costs(DiscardMatchingCardCostInto(IsLandCard, paid)),
		IfCardHasType(FromCard(paid), TypeLand, GainLifeFor(You, Fixed(2))))))
	g.PutOnBattlefield(source, a.PlayerID())
	a.AddToHand(NewLand("Paid land"))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatal("paid discard capture escaped cloned resolution")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("paid land type was not available to the activation")
	}
}

func TestPaidFilteredDiscardCardDefinition(t *testing.T) {
	paid := BindCard("discarded")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid discard cost",
			Card: NewArtifact("Edge", "", WithAbility(Activate(Costs(DiscardMatchingCardCostInto(IsLandCard, paid)), IfCardHasType(FromCard(paid), TypeLand, GainLifeFor(You, Fixed(2)))))),
		},
		{
			Name:          "nil binding",
			Card:          NewArtifact("Edge", "", WithAbility(Activate(Costs(DiscardMatchingCardCostInto(IsLandCard, nil)), IfCardHasType(FromCard(paid), TypeLand, GainLifeFor(You, Fixed(2)))))),
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "missing producer",
			Card:          NewArtifact("Missing", "", WithAbility(Activate(Costs(), IfCardHasType(FromCard(paid), TypeLand, GainLifeFor(You, Fixed(2)))))),
			WantErrSubstr: "producer",
		},
		{
			Name:          "duplicate producer",
			Card:          NewArtifact("Duplicate", "", WithAbility(Activate(Costs(DiscardMatchingCardCostInto(IsLandCard, paid), DiscardMatchingCardCostInto(IsLandCard, paid)), GainLifeFor(You, Fixed(1))))),
			WantErrSubstr: "duplicate",
		},
	})
}

func TestPaidFilteredDiscardCardSpellCopyAndVisit(t *testing.T) {
	a, b := &paidLandChoicePlayer{NewBasePlayer("A")}, NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCard("discarded")
	spell := NewInstant("Discard spell", "", WithAbility(Spell(
		IfCardHasType(FromCard(paid), TypeLand, GainLifeFor(You, Fixed(2))))),
		WithAdditionalCost(DiscardMatchingCardCostInto(IsLandCard, paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	land := NewLand("Paid land")
	a.AddToHand(land)
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("stack copy did not keep its paid card")
	}

	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	g.resolution.cardBindings = make(map[*CardBinding]capturedCard)
	g.resolution.cardBindings[paid] = capturedCard{id: land.ID(), visit: g.graveyardVisit(land.ID()), zone: ZoneGraveyard}
	if err := ReturnCardToHand(FromCard(paid)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	a.RemoveFromHand(land.ID())
	a.AddToGraveyard(land)
	if _, ok := FromCard(paid).resolveCard(ctx); ok {
		t.Fatal("paid card followed a later zone visit")
	}
}

func TestPaidFilteredDiscardUnsupportedContexts(t *testing.T) {
	paid := BindCard("paid")
	cost := DiscardMatchingCardCostInto(IsArtifactCard, paid)
	valid := func(c Cost) Card {
		return NewArtifact("Synthetic", "", WithAbility(Activate(Costs(c), GainLifeFor(You, Fixed(1)))))
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "zero filter", Card: valid(DiscardMatchingCardCostInto(CardFilter{}, paid))},
		{Name: "unnamed", Card: valid(DiscardMatchingCardCostInto(IsArtifactCard, BindCard(""))), WantErrSubstr: "name"},
		{Name: "optional", Card: valid(OptionalCost(cost, "Pay?")), WantErrSubstr: "direct cost"},
		{Name: "alternative", Card: valid(EitherCost(cost, LifePayCost(2))), WantErrSubstr: "direct cost"},
		{Name: "resolution", Card: NewInstant("Synthetic", "", Spell(IfPaid(You, Costs(cost), GainLifeFor(You, Fixed(1))))), WantErrSubstr: "activated ability cost or spell additional cost"},
	})
}
func TestPaidFilteredDiscardInterleaved(t *testing.T) {
	a := &filteredDiscardPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindCard("paid")
	cost := DiscardMatchingCardCostInto(IsArtifactCard, paid)
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	a.AddToHand(first)
	a.AddToHand(second)
	a.choice = first
	one, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	a.choice = second
	two, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	if err = two.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = one.Commit(); err != nil {
		t.Fatal(err)
	}
	if one.paidCards[paid].id != first.ID() || two.paidCards[paid].id != second.ID() {
		t.Fatal("payments shared captures")
	}
}
