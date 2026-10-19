package mage

import (
	"reflect"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestNativeCreatureDeathDeclarations(t *testing.T) {
	result, member := BindPermanentSet("deaths"), BindPermanent("member")
	producer := CreatureDeathInstruction{Subjects: MatchingPermanents(IsCreature), Result: result, CannotRegenerate: true}
	got, ok := InspectCreatureDeathInstruction(producer.Effect())
	if !ok || got.Result != result || !got.CannotRegenerate {
		t.Fatal("lost producer")
	}
	consumer := CreatureDeathReturnInstruction{Result: result, Member: member, Graveyard: You}
	read, ok := InspectCreatureDeathReturnInstruction(consumer.Effect())
	if !ok || read.Result != result || read.Member != member {
		t.Fatal("lost continuation")
	}
	for _, bad := range []ResolvingEffect{
		nil, (*destroyEachEffect)(nil), DestroyEach(producer.Subjects),
		DestroyEach(producer.Subjects, CreatureDeathsInto(nil)),
		DestroyEach(producer.Subjects, CreatureDeathsInto(BindPermanentSet(""))),
		DestroyEach(producer.Subjects, CreatureDeathsInto(result), CreatureDeathsInto(result)),
		DestroyEach(producer.Subjects, CreaturesDiedThisWayInto(BindNumber("count"))),
		DestroyEach(producer.Subjects, PermanentsPutIntoGraveyardThisWayInto(BindNumber("moves"))),
	} {
		if _, ok := InspectCreatureDeathInstruction(bad); ok {
			t.Fatal("invalid producer accepted")
		}
	}
	for _, bad := range []ResolvingEffect{
		nil, (*deathResultReturn)(nil),
		ReturnChosenGraveyardCardsForEachPermanent(result, member, You, IsCreatureCard),
		ReturnChosenGraveyardCardsForEachPermanent(result, member, You, HasCardTypeCardFilter(core.TypeArtifact)),
		(CreatureDeathReturnInstruction{Result: result, Member: member}).Effect(),
	} {
		if _, ok := InspectCreatureDeathReturnInstruction(bad); ok {
			t.Fatal("invalid continuation accepted")
		}
	}
	if (CreatureDeathInstruction{Result: result}).Effect() != nil || (CreatureDeathReturnInstruction{Result: result, Member: BindPermanent(""), Graveyard: You}).Effect() != nil {
		t.Fatal("missing operand accepted")
	}
}

func TestNativeCreatureDeathInspection(t *testing.T) {
	for _, state := range []string{"normal", "regen", "no regen", "indestructible", "exile", "phase", "cycle", "return", "copy", "clone", "historical", "foreign blocker", "foreign member", "foreign blocker illegal", "foreign member illegal", "foreign result", "forward", "duplicate", "duplicate consumer", "unknown suffix", "unknown illegal", "unbound illegal", "return prefix", "return suffix", "opaque player"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			result, member := BindPermanentSet("deaths"), BindPermanent("member")
			producer := (CreatureDeathInstruction{Subjects: MatchingPermanents(IsLand), Result: result, CannotRegenerate: state == "no regen"}).Effect()
			var graveyard PlayerRef = You
			if state == "historical" || state == "foreign blocker" || state == "foreign member" || state == "foreign member illegal" || state == "foreign blocker illegal" {
				attacker := From(member)
				blocker := PermanentRef(target)
				if state == "foreign member" || state == "foreign member illegal" {
					attacker = From(BindPermanent("member"))
				}
				if state == "foreign blocker" || state == "foreign blocker illegal" {
					blocker = DeclarePermanentTarget("target", AnyPermanent)
				}
				graveyard = ControllerWhenLastBlockedThisTurn(attacker, blocker)
			}
			if state == "opaque player" {
				graveyard = FromPlayer(BindPlayer("player"))
			}
			consumer := (CreatureDeathReturnInstruction{Result: result, Member: member, Graveyard: graveyard}).Effect()
			obj.Effects = []ResolvingEffect{Sequence(producer, consumer)}
			want, known := true, true
			switch state {
			case "regen":
				g.AddRegenerationShield(p.ID())
				want = false
			case "no regen":
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
			case "foreign result":
				obj.Effects = []ResolvingEffect{producer, (CreatureDeathReturnInstruction{Result: BindPermanentSet("deaths"), Member: member, Graveyard: You}).Effect()}
				want, known = false, false
			case "forward":
				obj.Effects = []ResolvingEffect{consumer, producer}
				want, known = false, false
			case "duplicate":
				obj.Effects = []ResolvingEffect{producer, producer, consumer}
				want, known = false, false
			case "duplicate consumer":
				obj.Effects = []ResolvingEffect{producer, consumer, consumer}
				want, known = false, false
			case "foreign blocker", "foreign member", "foreign blocker illegal", "foreign member illegal", "opaque player":
				want, known = false, false
			case "unknown suffix", "unknown illegal":
				obj.Effects = append(obj.Effects, DealDamageEach(This, PermanentDamageRecipients(collectionQueryTrap{}), Fixed(1)))
				want, known = false, false
			case "unbound illegal":
				obj.Effects = []ResolvingEffect{consumer}
				want, known = false, false
			case "return prefix":
				obj.Effects = []ResolvingEffect{Sequence((CreatureDeathInstruction{Subjects: MatchingPermanents(IsArtifact), Result: result}).Effect(), consumer), Destroy(target)}
				want, known = false, false
			case "return suffix":
				obj.Effects = []ResolvingEffect{Destroy(target), Sequence((CreatureDeathInstruction{Subjects: MatchingPermanents(IsArtifact), Result: result}).Effect(), consumer)}
			}
			if state == "unknown illegal" || state == "unbound illegal" || state == "foreign blocker illegal" || state == "foreign member illegal" {
				p.GrantBaseAttr(core.Shroud)
			}
			before, rng, stack, shields := g.resolution.Clone(), g.random.Clone(), g.StackSize(), g.HasRegenerationShield(p.ID())
			for range 2 {
				if got, k := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}}); got != want || k != known {
					t.Fatalf("got %v/%v want %v/%v", got, k, want, known)
				}
			}
			if !reflect.DeepEqual(before, g.resolution.Clone()) || !reflect.DeepEqual(rng, g.random) || stack != g.StackSize() || shields != g.HasRegenerationShield(p.ID()) {
				t.Fatal("inspection changed state")
			}
		})
	}
}
