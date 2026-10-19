package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestOrderedDestruction(t *testing.T) {
	for _, state := range []string{"normal", "shield", "no regen", "indestructible", "phase", "return", "empty prefix", "changed prefix", "shield prefix", "certain prefix", "empty"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			land := g.PutOnBattlefield(NewLand("Land"), a.PlayerID())
			watched := []PermanentVisit{{ID: land.ID(), Incarnation: land.incarnationID}}
			effect := DestroyEach(MatchingPermanents(IsLand))
			want, known := true, true
			switch state {
			case "shield":
				g.AddRegenerationShield(land.ID())
				want = false
			case "no regen":
				g.AddRegenerationShield(land.ID())
				effect = DestroyEachWithoutRegeneration(MatchingPermanents(IsLand))
			case "indestructible":
				land.GrantBaseAttr(core.Indestructible)
				want = false
			case "phase":
				g.PhaseOut(land.ID())
				want = false
			case "return":
				g.RemoveFromBattlefield(land)
				g.PutOnBattlefield(land.Card, a.PlayerID())
				want = false
			case "empty prefix":
				effect = Sequence(DestroyEach(MatchingPermanents(IsCreature)), effect)
			case "changed prefix", "shield prefix":
				creature := g.PutOnBattlefield(NewCreature("Earlier", "", 1, 1), b.PlayerID())
				if state == "shield prefix" {
					g.AddRegenerationShield(creature.ID())
				}
				effect = Sequence(DestroyEach(MatchingPermanents(IsCreature)), effect)
				want, known = false, false
			case "certain prefix":
				effect = Sequence(effect, DestroyEach(MatchingPermanents(IsCreature)))
			case "empty":
				effect = Sequence()
				want = false
			}
			plan, ok := InspectOrderedDestruction(effect)
			if !ok {
				t.Fatal("complete collection declaration rejected")
			}
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			for range 2 {
				got, gotKnown := plan.Certain(ctx, watched)
				if got != want || gotKnown != known {
					t.Fatalf("certainty=%v/%v want=%v/%v", got, gotKnown, want, known)
				}
			}
			clone := g.Clone()
			got, gotKnown := plan.Certain(&EffectContext{Game: clone, Controller: a.PlayerID()}, watched)
			if got != want || gotKnown != known || g.FindPermanentIncludingPhased(land.ID()) == nil {
				t.Fatal("query changed the game or clone result")
			}
		})
	}
}

func TestOrderedDestructionSimultaneous(t *testing.T) {
	a := &proposalChoiceTrap{NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	first := g.PutOnBattlefield(NewLand("First"), a.PlayerID())
	second := g.PutOnBattlefield(NewLand("Second"), a.PlayerID())
	g.AddRegenerationShield(first.ID())
	watch := []PermanentVisit{{ID: first.ID(), Incarnation: first.incarnationID}, {ID: second.ID(), Incarnation: second.incarnationID}}
	operands := []PermanentSet{MatchingPermanents(IsLand), MatchingPermanents(IsLand)}
	plan, ok := InspectOrderedDestruction(DestroyEach(UnionPermanents(operands...)))
	if !ok {
		t.Fatal("complete union rejected")
	}
	operands[0] = &destructionInspectionSetTrap{}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	before := g.Clone()
	if certain, known := plan.Certain(ctx, watch); !known || !certain {
		t.Fatal("one regenerated member obscured another certain member")
	}
	if first.Tapped || !reflect.DeepEqual(g.random, before.random) || !reflect.DeepEqual(g.effects.replacements, before.effects.replacements) {
		t.Fatal("inspection used a shield, a decision or RNG")
	}
	g.PhaseOut(second.ID())
	if certain, known := plan.Certain(ctx, watch); !known || certain {
		t.Fatal("phased member supplied a proof")
	}
	g.PhaseIn([]uuid.UUID{second.ID()})
	if certain, known := plan.Certain(ctx, watch); !known || !certain {
		t.Fatal("phase cycle lost current member")
	}
	replacements := append([]ReplacementEffect(nil), g.effects.replacements...)
	for _, cycle := range []bool{false, true} {
		g.effects.replacements = append([]ReplacementEffect(nil), replacements...)
		g.effects.cycleReplacements = nil
		if cycle {
			g.effects.cycleReplacements = []ReplacementEffect{&proposalTrap{}}
		} else {
			g.effects.replacements = append(g.effects.replacements, &proposalTrap{})
		}
		if certain, known := plan.Certain(ctx, watch); known || certain {
			t.Fatal("unknown replacement supplied certainty")
		}
	}
}

func TestOrderedDestructionDestinations(t *testing.T) {
	for _, state := range []string{"hand", "exile", "cancel", "mixed"} {
		t.Run(state, func(t *testing.T) {
			a := &proposalChoiceTrap{NewBasePlayer("A")}
			g := NewGame(a, NewBasePlayer("B"))
			p := g.PutOnBattlefield(NewCreature("Victim", "", 1, 1), a.PlayerID())
			switch state {
			case "hand":
				p.RuntimeAbilities = append(p.RuntimeAbilities, ReturnToHandInsteadOfDyingWithPlayLock())
			case "exile", "mixed":
				g.effects.AddReplacement(&temporaryDeathExileReplacement{subject: p.ID(), incarnation: p.incarnationID})
				if state == "mixed" {
					g.AddRegenerationShield(p.ID())
				}
			case "cancel":
				g.effects.AddReplacement(&nextDestructionReplacement{subject: p.ID(), incarnation: p.incarnationID})
			}
			plan, ok := InspectOrderedDestruction(DestroyEach(MatchingPermanents(IsCreature)))
			if !ok {
				t.Fatal("complete declaration rejected")
			}
			if certain, known := plan.Certain(&EffectContext{Game: g, Controller: a.PlayerID()}, []PermanentVisit{{ID: p.ID(), Incarnation: p.incarnationID}}); !known || certain {
				t.Fatal("replacement supplied a certain graveyard move")
			}
		})
	}
}

func TestOrderedDestructionUnknown(t *testing.T) {
	trap := NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("query called filter") })
	for _, effect := range []ResolvingEffect{
		nil, Destroy(This), DestroyEach(MatchingPermanents(trap)),
		Sequence(DestroyEach(MatchingPermanents(IsLand)), destructionInspectionTrap{}),
		Sequence(DestroyEach(MatchingPermanents(IsLand)), DestroyEach(MatchingPermanents(Or(AnyPermanent, trap)))),
		DestroyEach(&destructionInspectionSetTrap{}),
	} {
		if _, ok := InspectOrderedDestruction(effect); ok {
			t.Fatal("unsupported descendant supplied a partial proof")
		}
	}
	plan, ok := InspectOrderedDestruction(DestroyEach(MatchingPermanents(IsLand)))
	if !ok {
		t.Fatal("valid plan rejected")
	}
	for _, ctx := range []*EffectContext{nil, {}, {Game: NewGame(NewBasePlayer("A"), NewBasePlayer("B"))}} {
		if certain, known := plan.Certain(ctx, nil); certain || known {
			t.Fatal("invalid context supplied certainty")
		}
	}
	if certain, known := (OrderedDestructionPredicate{}).Certain(nil, nil); certain || known {
		t.Fatal("zero query supplied certainty")
	}
}
