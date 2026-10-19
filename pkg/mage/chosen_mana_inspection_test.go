package mage

import (
	"reflect"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type manaRestrictionInspectionTrap struct{}

func (manaRestrictionInspectionTrap) Description() string { panic("restriction callback") }
func (manaRestrictionInspectionTrap) IsSatisfiedBy(core.SpellPaymentContext) bool {
	panic("restriction callback")
}

func TestChosenManaInspectionDeclaration(t *testing.T) {
	amount := FromNumber(BindNumber("paid"))
	for _, restriction := range []core.ManaRestriction{nil, core.CreatureSpellsOnly{}, core.ArtifactSpellsOnly{}} {
		d := ManaProductionInstruction{Subject: You, Color: core.AnyColor, Amount: amount, Restriction: restriction}
		effect := d.Effect()
		got, ok := InspectManaProductionInstructions(effect)
		if effect == nil || !ok || len(got) != 1 || got[0] != d {
			t.Fatal("choice or restriction operands lost")
		}
	}
	fixedDeclaration := ManaProductionInstruction{Subject: You, Color: core.Red, Amount: Fixed(2), Restriction: core.CreatureSpellsOnly{}}
	if got, ok := InspectManaProductionInstructions(fixedDeclaration.Effect()); !ok || len(got) != 1 || got[0] != fixedDeclaration {
		t.Fatal("fixed restriction declaration lost")
	}
	batch := AddRestrictedMana([]ManaProduction{{Color: core.Red, Amount: 2}, {Color: core.Colorless, Amount: 3}}, core.ArtifactSpellsOnly{})
	got, ok := InspectManaProductionInstructions(batch)
	if !ok || len(got) != 2 || got[0].Color != core.Red || got[1].Color != core.Colorless || got[0].Restriction != (core.ArtifactSpellsOnly{}) {
		t.Fatal("fixed restricted batch lost")
	}
	got[0].Color = core.Blue
	again, _ := InspectManaProductionInstructions(batch)
	if again[0].Color != core.Red {
		t.Fatal("query shares batch storage")
	}
	var missing *manaAnyOneColorEffect
	var fixed *restrictedFixedManaEffect
	var restriction *core.ArtifactSpellsOnly
	for _, effect := range []Effect{missing, fixed,
		AddRestrictedManaOfAnyOneColor(Fixed(2), nil),
		AddRestrictedManaOfAnyOneColor(Fixed(2), restriction),
		AddRestrictedManaOfAnyOneColor(Fixed(2), manaRestrictionInspectionTrap{}),
		AddRestrictedManaOfAnyOneColor(nil, core.ArtifactSpellsOnly{}),
		AddRestrictedMana(nil, core.ArtifactSpellsOnly{}),
		AddRestrictedMana([]ManaProduction{{Color: core.AnyColor, Amount: 2}}, core.ArtifactSpellsOnly{}),
		AddRestrictedMana([]ManaProduction{{Color: core.Red, Amount: 0}}, core.ArtifactSpellsOnly{}),
		Sequence(batch, destructionInspectionTrap{}),
	} {
		if _, ok := InspectManaProductionInstructions(effect); ok {
			t.Fatal("invalid production accepted")
		}
	}
	for _, d := range []ManaProductionInstruction{
		{Subject: ActivePlayer, Color: core.AnyColor, Amount: Fixed(2)},
		{Subject: You, Color: core.AnyColor, Amount: Fixed(2), Restriction: restriction},
		{Subject: You, Color: core.Red, Amount: amount, Restriction: core.ArtifactSpellsOnly{}},
	} {
		if d.Effect() != nil {
			t.Fatal("unsupported declaration built")
		}
	}
}

func TestChosenManaInspectionSpell(t *testing.T) {
	for _, state := range []string{"only", "suffix", "prefix", "zero prefix", "opaque restriction", "opaque amount", "unknown suffix", "all illegal unknown", "phase", "return"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, land := targetQueryFixture(t)
			g.GetPlayer(obj.Controller).ManaPool().AddRestricted(core.Green, 3, core.ArtifactSpellsOnly{})
			amount := ResolutionValue(Fixed(3))
			restriction := core.ManaRestriction(core.CreatureSpellsOnly{})
			want, known := false, true
			if state == "zero prefix" {
				amount = Fixed(0)
			}
			if state == "opaque amount" {
				amount = damageInspectionValueTrap{}
				known = false
			}
			if state == "opaque restriction" || state == "all illegal unknown" {
				restriction = manaRestrictionInspectionTrap{}
				known = false
			}
			mana := AddRestrictedManaOfAnyOneColor(amount, restriction)
			obj.Effects = []Effect{mana}
			switch state {
			case "suffix":
				obj.Effects = []Effect{Sequence(Destroy(land), mana)}
				want = true
			case "prefix", "zero prefix":
				obj.Effects = []Effect{Sequence(mana, Destroy(land))}
				known = false
			case "unknown suffix":
				obj.Effects = []Effect{Sequence(Destroy(land), mana, destructionInspectionTrap{})}
				known = false
			case "all illegal unknown":
				p.GrantBaseAttr(core.Shroud)
				obj.Effects = []Effect{Sequence(Destroy(land), mana)}
			case "phase":
				g.PhaseOut(p.ID())
				obj.Effects = []Effect{Sequence(Destroy(land), mana)}
			case "return":
				g.BouncePermanentToHand(p)
				g.GetPlayer(p.ControllerID()).RemoveFromHand(p.Card.ID())
				g.PutOnBattlefield(p.Card, p.ControllerID())
				obj.Effects = []Effect{Sequence(Destroy(land), mana)}
			}
			before := g.Clone()
			for _, game := range []*Game{g, g.Clone()} {
				for range 2 {
					certain, k := game.InspectSpellDestruction(game.StackPeek(), []PermanentVisit{{p.ID(), p.incarnationID}})
					if certain != want || k != known {
						t.Fatalf("got %v/%v want %v/%v", certain, k, want, known)
					}
				}
			}
			if !reflect.DeepEqual(g.GetPlayer(obj.Controller).ManaPool(), before.GetPlayer(obj.Controller).ManaPool()) || !reflect.DeepEqual(g.random, before.random) || g.StackSize() != before.StackSize() {
				t.Fatal("inspection changed game")
			}
		})
	}
}

func TestChosenManaInspectionProvenance(t *testing.T) {
	for _, state := range []string{"paid", "missing", "foreign", "negative", "absent cost", "optional cost", "overwrite", "wrong capture"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			paid := BindNumber("paid")
			cost := SacrificeMatchingIncludingSourceCostWithManaValue(IsCreature, "Sacrifice a creature", paid)
			base := obj.Card.(*BaseCard)
			base.additionalCosts = []Cost{cost}
			obj.capturedNumbers = map[*NumberBinding]int{paid: 3}
			read := paid
			switch state {
			case "missing":
				delete(obj.capturedNumbers, paid)
			case "foreign":
				read = BindNumber("paid")
			case "negative":
				obj.capturedNumbers[paid] = -1
			case "absent cost":
				base.additionalCosts = nil
			case "optional cost":
				base.additionalCosts = []Cost{OptionalCostInto(cost, "Pay?", BindNumber("optional"))}
			case "wrong capture":
				cost.(*sacrificeMatchingCost).captureManaValue = false
			}
			mana := AddRestrictedManaOfAnyOneColor(AddValues(Fixed(1), FromNumber(read)), core.CreatureSpellsOnly{})
			obj.Effects = []Effect{Destroy(target), mana}
			if state == "overwrite" {
				obj.Effects = append(obj.Effects, (DestructionResultInstruction{Subjects: MatchingPermanents(IsCreature), Result: paid}).Effect())
			}
			for range 2 {
				certain, known := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}})
				if certain != (state == "paid") || known != (state == "paid") {
					t.Fatalf("query = %v/%v", certain, known)
				}
			}
		})
	}
}

func TestChosenManaInspectionFixedOrder(t *testing.T) {
	for _, state := range []string{"only", "prefix", "suffix", "unknown restriction"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			restriction := core.ManaRestriction(core.ArtifactSpellsOnly{})
			if state == "unknown restriction" {
				restriction = manaRestrictionInspectionTrap{}
			}
			mana := AddRestrictedMana([]ManaProduction{{Color: core.Red, Amount: 2}, {Color: core.Colorless, Amount: 1}}, restriction)
			obj.Effects = []Effect{mana}
			if state == "prefix" {
				obj.Effects = append(obj.Effects, Destroy(target))
			}
			if state == "suffix" || state == "unknown restriction" {
				obj.Effects = []Effect{Destroy(target), mana}
			}
			certain, known := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}})
			if certain != (state == "suffix") || known != (state == "only" || state == "suffix") {
				t.Fatalf("query = %v/%v", certain, known)
			}
		})
	}
}
