package mage

import (
	"reflect"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRecipientDiscardInstructionDeclaration(t *testing.T) {
	result := BindPlayerNumbers("counts")
	member := BindPlayer("recipient")
	producer := DiscardUpToInstruction{Subjects: EachPlayer, Maximum: Fixed(3), Result: result}
	got, ok := InspectDiscardUpToInstruction(producer.Effect())
	if !ok || got.Result != result || got.Subjects != producer.Subjects {
		t.Fatal("lost producer")
	}
	amount := SubtractValues(Fixed(3), PlayerNumber(result, FromPlayer(member)))
	read, ok := InspectRecipientDiscardAmount(amount, member)
	if !ok || read.Result != result || read.Base != 3 {
		t.Fatal("lost local result identity")
	}
	for _, value := range []ResolutionValue{nil, damageInspectionValueTrap{}, PlayerNumber(result, FromPlayer(member)), SubtractValues(Fixed(3), PlayerNumber(result, You)), SubtractValues(Fixed(3), PlayerNumber(result, FromPlayer(BindPlayer("recipient")))), SubtractValues(damageInspectionValueTrap{}, PlayerNumber(result, FromPlayer(member))), AddValues(Fixed(3), PlayerNumber(result, FromPlayer(member))), SubtractValues(Fixed(3), PlayerNumber(nil, FromPlayer(member))), SubtractValues(Fixed(3), PlayerNumber(BindPlayerNumbers(""), FromPlayer(member)))} {
		if _, ok := InspectRecipientDiscardAmount(value, member); ok {
			t.Fatalf("accepted %T", value)
		}
	}
	if _, ok := InspectRecipientDiscardAmount(amount, nil); ok {
		t.Fatal("nil member")
	}
	for _, instruction := range []DiscardUpToInstruction{{}, {EachPlayer, nil, result}, {nil, Fixed(3), result}, {EachPlayer, Fixed(3), nil}, {EachPlayer, Fixed(3), BindPlayerNumbers("")}} {
		if instruction.Effect() != nil {
			t.Fatal("invalid descriptor")
		}
	}
	for _, effect := range []ResolvingEffect{nil, Sequence(producer.Effect()), DiscardUpToEach(EachPlayer, Fixed(3), nil)} {
		if _, ok := InspectDiscardUpToInstruction(effect); ok {
			t.Fatal("invalid producer")
		}
	}
}

func TestRecipientDiscardInstructionSpell(t *testing.T) {
	for _, state := range []string{"only", "prefix", "suffix", "zero prefix", "unbound", "forward", "duplicate", "foreign", "unknown suffix", "all illegal", "unbound all illegal", "copy", "clone", "opaque maximum", "opaque subjects", "opaque source"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, land := targetQueryFixture(t)
			result := BindPlayerNumbers("counts")
			member := BindPlayer("recipient")
			producer := DiscardUpToInstruction{EachPlayer, Fixed(3), result}
			read := result
			want, known := false, true
			switch state {
			case "foreign":
				read = BindPlayerNumbers("counts")
				known = false
			case "opaque maximum":
				producer.Maximum = damageInspectionValueTrap{}
				known = false
			case "opaque subjects":
				producer.Subjects = collectionDamagePlayerTrap{}
				known = false
			case "zero prefix":
				producer.Maximum = Fixed(0)
			}
			source := DamageSourceRef(This)
			if state == "opaque source" {
				source = damageSourceQueryTrap{}
				known = false
			}
			damage := PlayerCollectionDamageInstruction{source, EachPlayer, member, SubtractValues(Fixed(3), PlayerNumber(read, FromPlayer(member)))}.Effect()
			obj.Effects = []ResolvingEffect{Sequence(producer.Effect(), damage)}
			switch state {
			case "prefix", "zero prefix":
				obj.Effects = append(obj.Effects, Destroy(land))
				known = false
			case "suffix":
				obj.Effects = append([]ResolvingEffect{Destroy(land)}, obj.Effects...)
				want = true
			case "unbound", "unbound all illegal":
				obj.Effects = []ResolvingEffect{damage}
				known = false
			case "forward":
				obj.Effects = []ResolvingEffect{damage, producer.Effect()}
				known = false
			case "duplicate":
				obj.Effects = []ResolvingEffect{producer.Effect(), producer.Effect(), damage}
				known = false
			case "unknown suffix":
				obj.Effects = append([]ResolvingEffect{Destroy(land)}, append(obj.Effects, destructionInspectionTrap{})...)
				known = false
			case "copy":
				obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
			case "clone":
				g = g.Clone()
				obj = g.StackPeek()
			}
			if state == "all illegal" || state == "unbound all illegal" {
				p.GrantBaseAttr(core.Shroud)
			}
			rng := g.random.Clone()
			g.resolution.numberBindings = map[*NumberBinding]int{result.members[0]: -999, result.members[1]: 1000}
			before := g.Clone()
			if state != "suffix" {
				g.effects.AddReplacement(&proposalTrap{})
			}
			for range 2 {
				got, k := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}})
				if got != want || k != known {
					t.Fatalf("got %v/%v want %v/%v", got, k, want, known)
				}
			}
			if !reflect.DeepEqual(rng, g.random) || !reflect.DeepEqual(g.resolution.numberBindings, before.resolution.numberBindings) || g.GetPlayer(obj.Controller).Life() != 20 {
				t.Fatal("query changed state")
			}
		})
	}
}

func TestRecipientDiscardInstructionBindings(t *testing.T) {
	first, second := BindPlayerNumbers("same"), BindPlayerNumbers("same")
	member := BindPlayer("recipient")
	producer := func(binding *PlayerNumbersBinding) ResolvingEffect {
		return DiscardUpToInstruction{EachPlayer, Fixed(3), binding}.Effect()
	}
	consumer := func(binding *PlayerNumbersBinding) ResolvingEffect {
		return PlayerCollectionDamageInstruction{This, EachPlayer, member, SubtractValues(Fixed(3), PlayerNumber(binding, FromPlayer(member)))}.Effect()
	}
	alias := &PlayerNumbersBinding{members: first.members}
	for _, tc := range []struct {
		name    string
		effects []ResolvingEffect
		valid   bool
	}{
		{"nested", []ResolvingEffect{Sequence(producer(first), Sequence(consumer(first)))}, true},
		{"independent", []ResolvingEffect{producer(first), producer(second), consumer(first), consumer(second)}, true},
		{"foreign same name", []ResolvingEffect{producer(first), consumer(second)}, false},
		{"aliased producer", []ResolvingEffect{producer(first), producer(alias), consumer(first)}, false},
		{"foreign alias read", []ResolvingEffect{producer(first), consumer(alias)}, false},
		{"number producer before", []ResolvingEffect{DestroyEach(MatchingPermanents(IsLand), PermanentsPutIntoGraveyardThisWayInto(first.members[0])), producer(first)}, false},
		{"number producer after", []ResolvingEffect{producer(first), DestroyEach(MatchingPermanents(IsLand), PermanentsPutIntoGraveyardThisWayInto(first.members[1]))}, false},
		{"negative maximum", []ResolvingEffect{DiscardUpToInstruction{EachPlayer, Fixed(-1), first}.Effect()}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := inspectSpellPermanentInstructions(tc.effects); ok != tc.valid {
				t.Fatalf("valid %v want %v", ok, tc.valid)
			}
		})
	}
	for _, bad := range []*PlayerNumbersBinding{{}, {members: [2]*NumberBinding{first.members[0], first.members[0]}}} {
		if effect := (DiscardUpToInstruction{EachPlayer, Fixed(3), bad}).Effect(); effect != nil {
			t.Fatal("malformed player numbers")
		}
	}
	for _, base := range []ResolutionValue{Fixed(-1), Fixed(0), ValueAtResolution(Fixed(3))} {
		if _, ok := InspectRecipientDiscardAmount(SubtractValues(base, PlayerNumber(first, FromPlayer(member))), member); !ok {
			t.Fatal("native fixed operand rejected")
		}
	}
}
