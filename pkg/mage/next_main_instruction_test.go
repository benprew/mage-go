package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestNextMainInstructionDeclaration(t *testing.T) {
	target := DeclareSpellTarget("spell")
	body := AddManaAmount(core.Colorless, ManaValueOfSpellWithLKI(target))
	effect := (NextMainPhaseInstruction{Body: body}).Effect()
	instruction, ok := InspectNextMainPhaseInstruction(effect)
	if !ok || instruction.Body != body {
		t.Fatal("native body lost")
	}
	var missing *sequenceEffect
	if (NextMainPhaseInstruction{}).Effect() != nil || (NextMainPhaseInstruction{Body: missing}).Effect() != nil {
		t.Fatal("missing body accepted")
	}
	for _, bad := range []ResolvingEffect{nil, (*nextStepEffect)(nil), AtEach(YourMainPhase, body), AtNextThisTurn(YourMainPhase, body), AtNext(YourUpkeep, body), AtNext(YourMainPhase, nil), &nextStepEffect{pattern: YourMainPhase, body: body, condition: ControllerLifeAtLeastStartingLife()}} {
		if _, ok := InspectNextMainPhaseInstruction(bad); ok {
			t.Fatal("unsupported schedule accepted")
		}
	}
}

func TestNextMainInstructionInspection(t *testing.T) {
	for _, state := range []string{"normal", "foreign", "no LKI", "opaque", "opaque suffix", "unknown pattern", "repeated", "restricted", "saved number", "foreign after proof", "illegal foreign", "nested", "fixed", "X"} {
		t.Run(state, func(t *testing.T) {
			g, victim, obj, target := counterInstructionFixture(t)
			g.PutOnBattlefield(NewLand("Watched"), obj.Controller)
			collection, _ := InspectPermanentCollection(MatchingPermanents(AnyPermanent))
			watched := collection.Select(g)
			amount := ManaValueOfSpellWithLKI(target)
			known := state == "normal" || state == "nested" || state == "fixed" || state == "X"
			switch state {
			case "foreign", "illegal foreign", "foreign after proof":
				amount = ManaValueOfSpellWithLKI(DeclareSpellTarget("spell"))
			case "no LKI":
				amount = ManaValueOfSpell(target)
			case "saved number":
				amount = FromNumber(BindNumber("missing"))
			case "nested":
				amount = MultiplyValues(Fixed(2), AddValues(Fixed(1), amount))
			case "fixed":
				amount = Fixed(0)
			case "X":
				amount = ValueAtResolution(XValue())
			}
			body := AddManaAmount(core.Colorless, amount)
			switch state {
			case "opaque":
				body = destructionInspectionTrap{}
			case "restricted":
				body = AddRestrictedManaOfAnyOneColor(amount, core.CreatureSpellsOnly{})
			}
			effect := AtNext(YourMainPhase, body)
			if state == "unknown pattern" {
				effect = AtNext(YourUpkeep, body)
			}
			if state == "repeated" {
				effect = AtEach(YourMainPhase, body)
			}
			obj.Effects = []Effect{effect}
			if state == "opaque suffix" {
				obj.Effects = append(obj.Effects, destructionInspectionTrap{})
			}
			if state == "illegal foreign" {
				g.CounterStackObject(victim.ID)
			}
			if state == "foreign after proof" {
				obj.Effects = append([]Effect{DestroyEach(MatchingPermanents(AnyPermanent))}, obj.Effects...)
			}
			before := g.StackSize()
			for range 2 {
				if certain, got := g.InspectSpellDestruction(obj, watched); certain || got != known {
					t.Fatalf("inspection = %v/%v, want false/%v", certain, got, known)
				}
			}
			if g.StackSize() != before || len(g.DelayedTriggers()) != 0 || len(g.resolution.numberBindings) != 0 {
				t.Fatal("inspection mutated state")
			}
		})
	}
}

func TestNextMainInstructionOrder(t *testing.T) {
	for _, prefix := range []bool{false, true} {
		g, _, obj, target := counterInstructionFixture(t)
		g.PutOnBattlefield(NewLand("Watched"), obj.Controller)
		collection, _ := InspectPermanentCollection(MatchingPermanents(AnyPermanent))
		watched := collection.Select(g)
		delay := (NextMainPhaseInstruction{Body: AddManaAmount(core.Colorless, ManaValueOfSpellWithLKI(target))}).Effect()
		destruction := DestroyEach(MatchingPermanents(AnyPermanent))
		obj.Effects = []Effect{destruction, delay}
		if prefix {
			obj.Effects = []Effect{delay, destruction}
		}
		if certain, known := g.InspectSpellDestruction(obj, watched); certain != !prefix || known != !prefix {
			t.Fatalf("prefix=%v proof=%v/%v", prefix, certain, known)
		}
		if len(g.DelayedTriggers()) != 0 || len(collection.Select(g)) != 1 {
			t.Fatal("query executed body")
		}
	}
}
