package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentLifePaymentInstructionDeclaration(t *testing.T) {
	binding := BindPermanentSet("unpaid")
	subjects := MatchingPermanents(IsLand)
	body := DestroyEach(FromPermanentSet(binding))
	instruction := PermanentLifePaymentInstruction{Subjects: subjects, Amount: 1, Unpaid: binding, Body: body}
	effect := instruction.Effect()
	got, ok := InspectPermanentLifePaymentInstruction(effect)
	if !ok || got.Amount != 1 || got.Unpaid != binding || got.Body != body {
		t.Fatal("lost payment declaration")
	}
	if err := ValidateCardDefinition("Offer", NewInstant("Offer", "", Spell(effect))); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []PermanentLifePaymentInstruction{
		{}, {Subjects: subjects, Amount: -1, Unpaid: binding, Body: body},
		{Subjects: subjects, Unpaid: &PermanentSetBinding{}, Body: body},
		{Subjects: subjects, Unpaid: binding},
		{Subjects: subjects, Unpaid: binding, Body: (*sequenceEffect)(nil)},
	} {
		if bad.Effect() != nil {
			t.Fatal("invalid payment accepted")
		}
	}
	for _, bad := range []Effect{nil, (*sequenceEffect)(nil), Sequence(), Sequence(body, effect),
		Sequence(OfferCostForEachPermanent(subjects, EachPlayer, Costs(LifePayCostInto(1, BindNumber("paid"))), binding), body),
		Sequence(OfferCostForEachPermanent(subjects, EachPlayer, Costs(LifePayCost(1), LifePayCost(1)), binding), body),
		Sequence(OfferCostForEachPermanent(subjects, OnePlayer(You), Costs(LifePayCost(1)), binding), body),
	} {
		if _, ok := InspectPermanentLifePaymentInstruction(bad); ok {
			t.Fatal("invalid shape accepted")
		}
	}
	foreign := (PermanentLifePaymentInstruction{Subjects: subjects, Unpaid: BindPermanentSet("unpaid"), Body: body}).Effect()
	if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", Spell(foreign))); err == nil {
		t.Fatal("foreign result accepted")
	}
}

func TestPermanentLifePaymentInstructionQuery(t *testing.T) {
	for _, state := range []string{"alone", "prefix", "suffix", "foreign", "opaque subject", "opaque body", "unknown suffix", "illegal", "unknown illegal", "escaped", "forward", "duplicate", "copy", "clone", "zero", "empty", "no regeneration"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			binding := BindPermanentSet("unpaid")
			subjects := MatchingPermanents(IsLand)
			result := binding
			known, certain := true, false
			amount := 1
			switch state {
			case "foreign":
				result = BindPermanentSet("unpaid")
				known = false
			case "opaque subject", "unknown illegal":
				subjects = MatchingPermanents(NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter called") }))
				known = false
			case "zero":
				amount = 0
			case "empty":
				subjects = MatchingPermanents(And(IsLand, IsCreature))
			}
			body := DestroyEach(FromPermanentSet(result))
			if state == "no regeneration" {
				body = DestroyEachWithoutRegeneration(FromPermanentSet(result))
			}
			if state == "opaque body" {
				body = Sequence(body, destructionInspectionTrap{})
				known = false
			}
			effect := (PermanentLifePaymentInstruction{Subjects: subjects, Amount: amount, Unpaid: binding, Body: body}).Effect()
			obj.Effects = []Effect{effect}
			switch state {
			case "prefix":
				obj.Effects = []Effect{effect, Destroy(target)}
				known = false
			case "suffix", "copy", "clone":
				obj.Effects = []Effect{Destroy(target), effect}
				certain = true
			case "unknown suffix":
				obj.Effects = []Effect{Destroy(target), effect, destructionInspectionTrap{}}
				known = false
			case "illegal", "unknown illegal":
				p.GrantBaseAttr(core.Shroud)
			case "escaped":
				obj.Effects = []Effect{effect, body}
				known = false
			case "forward":
				obj.Effects = []Effect{body, effect}
				known = false
			case "duplicate":
				obj.Effects = []Effect{effect, effect}
				known = false
			}
			if state == "copy" {
				obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
			}
			if state == "clone" {
				g = g.Clone()
				obj = g.StackPeek()
			}
			g.SetRandomResults([]int{7})
			life := g.GetPlayer(obj.Controller).Life()
			for range 3 {
				got, k := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}})
				if got != certain || k != known {
					t.Fatalf("got %v/%v want %v/%v", got, k, certain, known)
				}
			}
			if g.GetPlayer(obj.Controller).Life() != life || len(g.resolution.createdTokenBindings) != 0 || g.FindPermanent(p.ID()) == nil || g.RandIntn(10) != 7 {
				t.Fatal("inspection changed state")
			}
		})
	}
}
