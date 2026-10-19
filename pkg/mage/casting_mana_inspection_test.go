package mage

import (
	"maps"
	"reflect"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCastingManaInspection(t *testing.T) {
	for _, state := range []string{"saved", "sum", "product", "zero", "suffix", "prefix", "missing", "foreign", "negative", "no cost", "optional", "multiple", "wrong capture", "invalid cost", "opaque", "invalid arithmetic", "overwrite before", "overwrite after", "opaque suffix", "all illegal", "unknown all illegal", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, permanent, obj, target := targetQueryFixture(t)
			paid := BindNumber("paid value")
			cost := SacrificeMatchingIncludingSourceCostWithManaValue(IsCreature, "Sacrifice a creature", paid)
			base := obj.Card.(*BaseCard)
			base.additionalCosts = []Cost{cost}
			obj.capturedNumbers = map[*NumberBinding]int{paid: 4}
			amount := FromNumber(paid)
			want, known := false, true
			switch state {
			case "sum":
				amount = AddValues(Fixed(1), amount)
			case "product":
				amount = MultiplyValues(Fixed(2), AddValues(amount, ValueAtResolution(XValue())))
			case "zero":
				obj.capturedNumbers[paid] = 0
			case "missing":
				delete(obj.capturedNumbers, paid)
				known = false
			case "foreign":
				amount = FromNumber(BindNumber("paid value"))
				known = false
			case "negative":
				obj.capturedNumbers[paid] = -1
				known = false
			case "no cost":
				base.additionalCosts = nil
				known = false
			case "optional":
				base.additionalCosts = []Cost{OptionalCostInto(cost, "Pay?", BindNumber("optional"))}
				known = false
			case "multiple":
				base.additionalCosts = append(base.additionalCosts, cost)
				known = false
			case "wrong capture":
				cost.(*sacrificeMatchingCost).captureManaValue = false
				known = false
			case "invalid cost":
				cost.(*sacrificeMatchingCost).count = 2
				known = false
			case "opaque":
				amount = MultiplyValues(amount, damageInspectionValueTrap{})
				known = false
			case "invalid arithmetic":
				amount = arithmeticValue{amount, Fixed(1), arithmeticOperation(99)}
				known = false
			case "suffix":
				want = true
			case "prefix":
				known = false
			case "all illegal":
				permanent.GrantBaseAttr(core.Shroud)
			case "unknown all illegal":
				permanent.GrantBaseAttr(core.Shroud)
				delete(obj.capturedNumbers, paid)
				known = false
			}
			mana := (ManaProductionInstruction{Subject: You, Color: core.Black, Amount: amount}).Effect()
			obj.Effects = []ResolvingEffect{mana}
			switch state {
			case "suffix", "opaque suffix":
				obj.Effects = []ResolvingEffect{Destroy(target), mana}
				if state == "opaque suffix" {
					obj.Effects = append(obj.Effects, destructionInspectionTrap{})
					known = false
				}
			case "prefix":
				obj.Effects = append(obj.Effects, Destroy(target))
			case "overwrite before", "overwrite after":
				writer := (DestructionResultInstruction{Subjects: MatchingPermanents(IsCreature), Result: paid}).Effect()
				obj.Effects = []ResolvingEffect{writer, mana}
				if state == "overwrite after" {
					obj.Effects = []ResolvingEffect{mana, writer}
				}
				known = false
			}
			pool := g.GetPlayer(obj.Controller).ManaPool()
			pool.AddRestricted(core.Green, 2, core.ArtifactSpellsOnly{})
			beforeGame := g.Clone()
			before := beforeGame.GetPlayer(obj.Controller).ManaPool()
			numbers := maps.Clone(obj.capturedNumbers)
			if state == "clone" {
				g = g.Clone()
				obj = g.StackPeek()
			}
			for range 2 {
				certain, gotKnown := g.InspectSpellDestruction(obj, []PermanentVisit{{ID: permanent.ID(), Incarnation: permanent.incarnationID}})
				if certain != want || gotKnown != known {
					t.Fatalf("query = %v/%v, want %v/%v", certain, gotKnown, want, known)
				}
			}
			if !reflect.DeepEqual(before, pool) || !reflect.DeepEqual(beforeGame.random, g.random) || !maps.Equal(numbers, obj.capturedNumbers) || g.StackSize() != 1 {
				t.Fatal("inspection changed state")
			}
		})
	}
}

func TestCastingManaInspectionPayment(t *testing.T) {
	for _, state := range []string{"normal", "exile", "cancel", "rollback"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			paid := BindNumber("paid value")
			effect := (ManaProductionInstruction{Subject: You, Color: core.Black, Amount: FromNumber(paid)}).Effect()
			card := NewInstant("Payment spell", "{B}", WithAbility(Spell(effect)), WithAdditionalCost(SacrificeMatchingIncludingSourceCostWithManaValue(IsCreature, "Sacrifice a creature", paid)))
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			victim := NewCreature("Payment", "{4}", 1, 1)
			victim.SetOwner(b.PlayerID())
			permanent := g.PutOnBattlefield(victim, a.PlayerID())
			if state != "rollback" {
				a.ManaPool().Add(core.Black, 1)
			}
			if state == "exile" {
				g.AddExileIfWouldGoToGraveyardThisTurn(permanent.ID(), permanent.ID())
			}
			if state == "cancel" {
				g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(a Action) bool { _, ok := a.(*SacrificePermanentAction); return ok }, replaceFunc: func(Action, *Game) Action { return nil }})
			}
			err := g.CastSpellByName(a.PlayerID(), card.Name(), nil)
			if state == "rollback" {
				if err == nil || g.StackSize() != 0 || g.FindPermanent(permanent.ID()) == nil || len(a.Hand()) != 1 || len(g.resolution.numberBindings) != 0 {
					t.Fatal("failed payment published a result")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if state == "cancel" && g.FindPermanent(permanent.ID()) == nil {
				t.Fatal("replacement did not suppress the move")
			}
			obj := g.StackPeek()
			if obj.capturedNumbers[paid] != 4 {
				t.Fatal("wrong capture")
			}
			for range 2 {
				if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
					t.Fatal("paid native production unknown")
				}
			}
			if a.ManaPool().TotalMana() != 0 {
				t.Fatal("inspection produced mana")
			}
			g.ResolveStack()
			if a.ManaPool().Count(core.Black) != 4 {
				t.Fatal("payment value lost")
			}
		})
	}
}
