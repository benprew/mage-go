package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPaidDiscardFactsLifecycle(t *testing.T) {
	for _, mode := range []string{"normal", "nonland", "moved", "clone", "hidden", "exile", "prevented"} {
		t.Run(mode, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			result := BindNumber("land discarded")
			cost := DiscardCardCostWithFacts(CardPaymentFact{Filter: IsLandCard, Result: result})
			source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(cost), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2)))))), a.PlayerID())
			var card Card = NewLand("Paid")
			if mode == "nonland" {
				card = NewArtifact("Paid", "")
			}
			a.AddToHand(card)
			switch mode {
			case "hidden":
				g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
			case "exile":
				g.effects.AddReplacement(&factsDiscardToExile{})
			case "prevented":
				g.effects.AddReplacement(&preventCostDiscardReplacement{})
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			if mode == "prevented" {
				if err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 || a.Life() != 20 {
					t.Fatal("cancelled payment changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if mode == "moved" {
				if _, ok := g.MoveFromGraveyard(a.PlayerID(), card.ID(), core.ZoneHand); !ok {
					t.Fatal("move failed")
				}
				a.RemoveFromHand(card.ID())
				a.AddToGraveyard(card)
			}
			g.RemoveFromBattlefield(source)
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			clone.ResolveStack()
			want := 22
			if mode == "hidden" || mode == "nonland" {
				want = 20
			}
			if a.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != want || clone.GetPlayer(b.PlayerID()).Life() != want {
				t.Fatal("clone lost saved payment fact")
			}
			g.ResolveStack()
			if a.Life() != want || b.Life() != want {
				t.Fatal("resolution lost saved payment fact")
			}
		})
	}
}

type factsDiscardToExile struct{ replacementBase }

func (*factsDiscardToExile) Matches(a Action, _ GameReader) bool {
	_, ok := a.(*DiscardAction)
	return ok
}
func (*factsDiscardToExile) Replace(a Action, _ *Game) Action {
	return a.(*DiscardAction).withDestination(core.ZoneExile)
}
func (*factsDiscardToExile) IsActive(GameReader) bool { return true }
func (*factsDiscardToExile) Clone() ReplacementEffect { return &factsDiscardToExile{} }

func TestPaidDiscardFactsTransactions(t *testing.T) {
	a := &factsDiscardPlayer{filteredDiscardPlayer: &filteredDiscardPlayer{BasePlayer: NewBasePlayer("A")}}
	g := NewGame(a, NewBasePlayer("B"))
	result := BindNumber("land")
	cost := DiscardCardCostWithFacts(CardPaymentFact{IsLandCard, result})
	land, artifact := NewLand("Land"), NewArtifact("Artifact", "")
	a.AddToHand(land)
	a.AddToHand(artifact)
	a.choice = land
	one, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	a.choice = artifact
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
	if one.paidNumbers[result] != 1 || two.paidNumbers[result] != 0 {
		t.Fatal("interleaved transactions shared facts")
	}
	a.AddToHand(land)
	a.choice = land
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), cost}})
	if err != nil {
		t.Fatal(err)
	}
	a.RemoveFromHand(land.ID())
	a.AddToHand(land)
	if err = tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || a.Life() != 20 || len(a.Hand()) != 1 {
		t.Fatal("stale hand visit did not roll back")
	}
	if !cost.CanPay(uuid.Nil, a.PlayerID(), g) || len(a.Hand()) != 1 {
		t.Fatal("query changed payment state")
	}
}

func TestPaidDiscardFactsDefinition(t *testing.T) {
	result := BindNumber("land")
	valid := func(c Cost) Card {
		return NewArtifact("Test", "", WithAbility(Activate(Costs(c), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(1))))))
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "valid", Card: valid(DiscardCardCostWithFacts(CardPaymentFact{IsLandCard, result}))},
		{Name: "nil result", Card: valid(DiscardCardCostWithFacts(CardPaymentFact{IsLandCard, nil})), WantErrSubstr: "non-nil"},
		{Name: "zero filter", Card: valid(DiscardCardCostWithFacts(CardPaymentFact{CardFilter{}, result})), WantErrSubstr: "filter"},
		{Name: "duplicate", Card: valid(DiscardCardCostWithFacts(CardPaymentFact{IsLandCard, result}, CardPaymentFact{IsArtifactCard, result})), WantErrSubstr: "distinct"},
		{Name: "missing", Card: valid(DiscardCost(1)), WantErrSubstr: "producer"},
		{Name: "multiple producers", Card: NewArtifact("Test", "", WithAbility(Activate(Costs(DiscardCardCostWithFacts(CardPaymentFact{IsLandCard, result}), DiscardCardCostWithFacts(CardPaymentFact{IsLandCard, result})), GainLifeFor(You, Fixed(1))))), WantErrSubstr: "duplicate"},
	})
}

type factsDiscardPlayer struct{ *filteredDiscardPlayer }

func (p *factsDiscardPlayer) ChooseCardsFromHand(_ int, _ string, _ GameReader) []Card {
	return []Card{p.choice}
}

func TestPaidDiscardFactsBoundaries(t *testing.T) {
	result := BindNumber("fact")
	cost := DiscardCardCostWithFacts(CardPaymentFact{IsLandCard, result})
	valid := func(c Cost) Card {
		return NewArtifact("Test", "", WithAbility(Activate(Costs(c), GainLifeFor(You, Fixed(1)))))
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "optional", Card: valid(OptionalCost(cost, "Pay?")), WantErrSubstr: "direct cost"},
		{Name: "alternative", Card: valid(EitherCost(cost, LifePayCost(1))), WantErrSubstr: "direct cost"},
		{Name: "resolution", Card: NewInstant("Test", "", WithAbility(Spell(IfPaid(You, Costs(cost), GainLifeFor(You, Fixed(1)))))), WantErrSubstr: "activated ability cost or spell additional cost"},
	})
}

func TestPaidDiscardFactsMultipleAndHidden(t *testing.T) {
	for _, hidden := range []bool{false, true} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		yes, no := BindNumber("land"), BindNumber("nonartifact")
		cost := DiscardCardCostWithFacts(CardPaymentFact{IsLandCard, yes}, CardPaymentFact{NewCardFilter("nonartifact", func(c Card) bool { return !c.HasType(core.TypeArtifact) }), no})
		a.AddToHand(NewLand("Paid"))
		if hidden {
			g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
		}
		tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost}})
		if err != nil {
			t.Fatal(err)
		}
		if err = tx.Commit(); err != nil {
			t.Fatal(err)
		}
		want := 1
		if hidden {
			want = 0
		}
		if tx.paidNumbers[yes] != want || tx.paidNumbers[no] != want {
			t.Fatal("unknown characteristics matched predicate")
		}
	}
}

func TestPaidDiscardFactsChangedReplacementRollback(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	result := BindNumber("land")
	a.AddToHand(NewLand("Paid"))
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), DiscardCardCostWithFacts(CardPaymentFact{IsLandCard, result})}})
	if err != nil {
		t.Fatal(err)
	}
	g.effects.AddReplacement(&preventCostDiscardReplacement{})
	if err = tx.Commit(); err == nil || a.Life() != 20 || len(a.Hand()) != 1 || len(tx.paidNumbers) != 0 {
		t.Fatal("changed replacement spent payment resources", err, a.Life())
	}
}

func TestPaidDiscardFactsSpellAdditionalCost(t *testing.T) {
	a, b := &paidLandChoicePlayer{NewBasePlayer("A")}, NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindNumber("land")
	spell := NewInstant("Test", "", WithAbility(Spell(IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))))), WithAdditionalCost(DiscardCardCostWithFacts(CardPaymentFact{IsLandCard, result})))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.AddToHand(NewLand("Paid"))
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("additional-cost fact lost in spell copy")
	}
}
