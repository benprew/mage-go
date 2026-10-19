package mage

import "testing"

func TestNativeCreatureDeathCount(t *testing.T) {
	for _, state := range []string{"normal", "foreign", "forward", "duplicate", "opaque amount", "nested foreign", "nil operand", "unknown operation", "suffix", "prefix", "no regeneration", "nested sum"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, _ := targetQueryFixture(t)
			result := BindNumber("deaths")
			producer := (CreatureDeathCountInstruction{Subjects: MatchingPermanents(IsLand), Result: result}).Effect()
			instruction, ok := InspectCreatureDeathCountInstruction(producer)
			if !ok || instruction.Result != result {
				t.Fatal("count identity lost")
			}
			amount := AddValues(FromNumber(result), Fixed(3))
			if state == "foreign" {
				amount = AddValues(FromNumber(BindNumber("deaths")), Fixed(3))
			}
			if state == "opaque amount" {
				amount = &damageInspectionValueTrap{}
			}
			switch state {
			case "nested foreign":
				amount = AddValues(amount, FromNumber(BindNumber("foreign")))
			case "nil operand":
				amount = AddValues(amount, nil)
			case "unknown operation":
				amount = arithmeticValue{left: amount, right: Fixed(1), operation: 255}
			case "nested sum":
				amount = AddValues(Fixed(2), AddValues(FromNumber(result), Fixed(1)))
			case "no regeneration":
				producer = (CreatureDeathCountInstruction{Subjects: MatchingPermanents(IsLand), Result: result, CannotRegenerate: true}).Effect()
				g.AddRegenerationShield(p.ID())
			}
			damage := DealDamageFrom(This, You, amount)
			obj.Effects = []Effect{producer, damage}
			switch state {
			case "forward":
				obj.Effects = []Effect{damage, producer}
			case "duplicate":
				obj.Effects = []Effect{producer, producer, damage}
			case "suffix":
				obj.Effects = append(obj.Effects, DestroyEach(MatchingPermanents(IsLand)))
			case "prefix":
				obj.Effects = []Effect{damage, producer}
			}
			wantKnown := state == "normal" || state == "suffix" || state == "no regeneration" || state == "nested sum"
			for range 2 {
				proof, known := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}})
				if known != wantKnown || proof != wantKnown {
					t.Fatalf("got %v/%v, want %v/%v", proof, known, wantKnown, wantKnown)
				}
			}
			if g.GetPlayer(obj.Controller).Life() != 20 || g.FindPermanent(p.ID()) == nil {
				t.Fatal("query changed state")
			}
			if wantKnown {
				clone := g.Clone()
				for _, game := range []*Game{clone, g} {
					game.ResolveStack()
					if game.GetPlayer(obj.Controller).Life() != 17 || game.FindPermanent(p.ID()) != nil {
						t.Fatal("noncreature death changed the count or clone state")
					}
				}
			}
		})
	}
}

func TestNativeCreatureDeathCountInvalid(t *testing.T) {
	for _, declaration := range []CreatureDeathCountInstruction{{}, {Subjects: MatchingPermanents(IsLand)}, {Result: BindNumber("count")}, {Subjects: MatchingPermanents(IsLand), Result: BindNumber("")}} {
		if declaration.Effect() != nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	result := BindNumber("count")
	for _, effect := range []Effect{DestroyEach(MatchingPermanents(IsLand)), DestroyEach(MatchingPermanents(IsLand), CreaturesDiedThisWayInto(result), CreaturesDiedThisWayInto(result)), DestroyEach(MatchingPermanents(IsLand), PermanentsPutIntoGraveyardThisWayInto(result))} {
		if _, ok := InspectCreatureDeathCountInstruction(effect); ok {
			t.Fatal("invalid producer inspected")
		}
	}
}
