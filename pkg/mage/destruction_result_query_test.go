package mage

import (
	"reflect"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDestructionResultQueryDeclaration(t *testing.T) {
	binding := BindNumber("moves")
	instruction := DestructionResultInstruction{Subjects: MatchingPermanents(IsLand), Result: binding, CannotRegenerate: true}
	effect := instruction.Effect()
	got, ok := InspectDestructionResult(effect)
	if !ok || got.Result != binding || !got.CannotRegenerate {
		t.Fatal("lost producer identity")
	}
	if _, ok := InspectDirectDestruction(effect); ok {
		t.Fatal("legacy inspector accepted result writes")
	}
	if _, ok := InspectDestructionResult(DestroyEach(MatchingPermanents(IsLand), CreaturesDiedThisWayInto(binding))); ok {
		t.Fatal("creature result accepted as graveyard result")
	}
	for _, effect := range []ResolvingEffect{
		DestroyEach(MatchingPermanents(IsLand)),
		DestroyEach(MatchingPermanents(IsLand), PermanentsPutIntoGraveyardThisWayInto(nil)),
		DestroyEach(MatchingPermanents(IsLand), PermanentsPutIntoGraveyardThisWayInto(BindNumber(""))),
		DestroyEach(MatchingPermanents(IsLand), PermanentsPutIntoGraveyardThisWayInto(binding), PermanentsPutIntoGraveyardThisWayInto(binding)),
		Sequence(instruction.Effect()),
	} {
		if _, ok := InspectDestructionResult(effect); ok {
			t.Fatal("invalid producer accepted")
		}
	}
	if (DestructionResultInstruction{Result: binding}).Effect() != nil || (DestructionResultInstruction{Subjects: instruction.Subjects}).Effect() != nil {
		t.Fatal("invalid descriptor built")
	}
}

func TestDestructionResultQuerySpell(t *testing.T) {
	for _, state := range []string{"normal", "regeneration", "no regeneration", "indestructible", "exile", "phase", "cycle", "return", "copy", "clone", "unknown suffix", "unbound", "forward", "duplicate", "foreign same name", "empty name", "unbound all illegal", "unknown all illegal"} {
		t.Run(state, func(t *testing.T) {
			g, p, _, obj, targets := pluralQueryFixture(t, 1)
			result := BindNumber("moves")
			if state == "empty name" {
				result = BindNumber("")
			}
			producer := DestroyEach(targets, PermanentsPutIntoGraveyardThisWayInto(result))
			if state == "no regeneration" {
				producer = DestroyEachWithoutRegeneration(targets, PermanentsPutIntoGraveyardThisWayInto(result))
			}
			amount := FromNumber(result)
			if state == "foreign same name" {
				amount = FromNumber(BindNumber("moves"))
			}
			consumer := CollectionDamageInstruction{This, PermanentDamageRecipients(MatchingPermanents(IsCreature)), amount}.Effect()
			obj.Effects = []ResolvingEffect{Sequence(producer, consumer)}
			want, known := true, true
			switch state {
			case "regeneration":
				g.AddRegenerationShield(p.ID())
				want = false
			case "no regeneration":
				g.AddRegenerationShield(p.ID())
			case "indestructible":
				p.GrantBaseAttr(core.Indestructible)
				want = false
			case "exile":
				g.effects.AddReplacement(&temporaryDeathExileReplacement{subject: p.ID(), incarnation: p.incarnationID})
				want = false
			case "phase":
				g.PhaseOut(p.ID())
				want = false
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, p.ControllerID())
				want = false
			case "copy":
				obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
			case "clone":
				g = g.Clone()
				obj = g.StackPeek()
			case "unknown suffix":
				obj.Effects = append(obj.Effects, DealDamageEach(This, PermanentDamageRecipients(collectionQueryTrap{}), Fixed(1)))
				want, known = false, false
			case "unknown all illegal":
				p.GrantBaseAttr(core.Shroud)
				obj.Effects = append(obj.Effects, DealDamageEach(This, PermanentDamageRecipients(collectionQueryTrap{}), Fixed(1)))
				want, known = false, false
			case "unbound all illegal":
				p.GrantBaseAttr(core.Shroud)
				obj.Effects = []ResolvingEffect{consumer}
				want, known = false, false
			case "unbound":
				obj.Effects = []ResolvingEffect{consumer}
				want, known = false, false
			case "forward":
				obj.Effects = []ResolvingEffect{Sequence(consumer, producer)}
				want, known = false, false
			case "duplicate":
				obj.Effects = []ResolvingEffect{Sequence(producer, producer, consumer)}
				want, known = false, false
			case "foreign same name", "empty name":
				want, known = false, false
			}
			rng := g.random.Clone()
			for range 2 {
				if got, k := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}}); got != want || k != known {
					t.Fatalf("got %v/%v want %v/%v", got, k, want, known)
				}
			}
			if len(g.resolution.numberBindings) != 0 || !reflect.DeepEqual(rng, g.random) || p.Damage != 0 {
				t.Fatal("query consumed state")
			}
		})
	}
}

func TestDestructionResultQueryBindings(t *testing.T) {
	first, second := BindNumber("same label"), BindNumber("same label")
	producer := func(binding *NumberBinding) ResolvingEffect {
		return DestructionResultInstruction{Subjects: MatchingPermanents(IsLand), Result: binding}.Effect()
	}
	consumer := func(binding *NumberBinding) ResolvingEffect {
		return DealDamageEach(This, PermanentDamageRecipients(MatchingPermanents(IsCreature)), FromNumber(binding))
	}
	for _, test := range []struct {
		name    string
		effects []ResolvingEffect
		valid   bool
	}{
		{"nested producer", []ResolvingEffect{Sequence(Sequence(producer(first)), consumer(first))}, true},
		{"independent identities", []ResolvingEffect{producer(first), producer(second), consumer(first), consumer(second)}, true},
		{"nested forward", []ResolvingEffect{Sequence(consumer(first)), producer(first)}, false},
		{"foreign identity", []ResolvingEffect{producer(first), consumer(second)}, false},
		{"arithmetic unsupported", []ResolvingEffect{producer(first), DealDamageEach(This, PermanentDamageRecipients(MatchingPermanents(IsCreature)), AddValues(FromNumber(first), Fixed(1)))}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, ok := inspectSpellPermanentInstructions(test.effects); ok != test.valid {
				t.Fatalf("valid=%v want %v", ok, test.valid)
			}
		})
	}
}
