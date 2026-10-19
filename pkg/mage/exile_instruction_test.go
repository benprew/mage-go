package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestExileInstructionDeclaration(t *testing.T) {
	binding := DeclarePermanentTarget("subject", AnyPermanent)
	set := MatchingPermanents(IsLand)
	for _, declaration := range []PermanentExileInstruction{{Subject: binding}, {Subjects: set}} {
		effect := declaration.Effect()
		got, ok := InspectPermanentExileInstructions(Sequence(effect, CompositeEffects("nested", effect)))
		if !ok || len(got) != 2 || got[0].Subject != declaration.Subject || InspectDefinition(got[0].Subjects).Text != InspectDefinition(declaration.Subjects).Text {
			t.Fatal("lost exile declaration")
		}
		got[0] = PermanentExileInstruction{}
		again, _ := InspectPermanentExileInstructions(effect)
		if again[0].Subject != declaration.Subject || InspectDefinition(again[0].Subjects).Text != InspectDefinition(declaration.Subjects).Text {
			t.Fatal("inspection shares result storage")
		}
	}
	var absent *PermanentTargetBinding
	for _, declaration := range []PermanentExileInstruction{{}, {Subject: absent}, {Subject: binding, Subjects: set}} {
		if declaration.Effect() != nil {
			t.Fatal("malformed declaration built an effect")
		}
	}
	var absentEffect *exileSubjectEffect
	var absentGroup *exileEachEffect
	for _, effect := range []Effect{nil, absentEffect, absentGroup, Exile(nil), ExileEach(nil), ExileWithResult(binding), ExileWithResult(binding, ExiledThisWayInto(BindNumber("count"))), ExileWithResult(binding, nil), Sequence(Exile(binding), destructionInspectionTrap{}), Destroy(binding)} {
		if _, ok := InspectPermanentExileInstructions(effect); ok {
			t.Fatal("incomplete or result-bearing tree accepted")
		}
	}
}

func TestExileInstructionSpell(t *testing.T) {
	for _, state := range []string{"only", "group", "prefix", "suffix", "empty prefix", "missing", "foreign", "duplicate", "opaque subject", "opaque collection", "opaque child", "opaque suffix", "result suffix", "shroud", "phase", "phase cycle", "returned", "copy", "replacement"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, binding := targetQueryFixture(t)
			exile := Exile(binding)
			obj.Effects = []Effect{exile}
			want, known := false, true
			switch state {
			case "group":
				obj.Effects = []Effect{ExileEach(MatchingPermanents(IsLand))}
			case "prefix":
				obj.Effects = []Effect{Sequence(exile, Destroy(binding))}
				known = false
			case "suffix":
				obj.Effects = []Effect{Sequence(Destroy(binding), exile)}
				want = true
			case "empty prefix":
				obj.Effects = []Effect{Sequence(ExileEach(MatchingPermanents(IsArtifact)), Destroy(binding))}
				want = true
			case "missing":
				obj.TargetSpecs = nil
				obj.Targets = nil
				known = false
			case "foreign":
				obj.Effects = []Effect{Exile(DeclarePermanentTarget("foreign", IsLand))}
				known = false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, binding.targetSpec())
				obj.Targets = append(obj.Targets, p.ID())
				known = false
			case "opaque subject":
				obj.Effects = []Effect{Exile(This)}
				known = false
			case "opaque collection":
				obj.Effects = []Effect{ExileEach(MatchingPermanents(NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("filter called") })))}
				known = false
			case "opaque child":
				obj.Effects = []Effect{Sequence(exile, destructionInspectionTrap{})}
				known = false
			case "opaque suffix":
				obj.Effects = []Effect{Sequence(Destroy(binding), ExileEach(MatchingPermanents(NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("filter called") }))))}
				known = false
			case "result suffix":
				obj.Effects = []Effect{Sequence(Destroy(binding), ExileWithResult(binding, ExiledThisWayInto(BindNumber("count"))))}
				known = false
			case "shroud":
				p.GrantBaseAttr(core.Shroud)
			case "phase":
				g.PhaseOut(p.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "returned":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, obj.Controller)
			case "copy":
				obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
			case "replacement":
				g.effects.AddReplacement(&proposalTrap{})
			}
			before := g.Clone()
			for _, game := range []*Game{g, g.Clone()} {
				for range 2 {
					if got, k := game.InspectSpellDestruction(game.stack.FindByID(obj.ID), []PermanentVisit{{p.ID(), p.incarnationID}}); got != want || k != known {
						t.Fatalf("proof=%v/%v want=%v/%v", got, k, want, known)
					}
				}
			}
			if !reflect.DeepEqual(g.random, before.random) || !reflect.DeepEqual(g.effects.replacements, before.effects.replacements) || len(g.GetExile()) != 0 || g.StackSize() != before.StackSize() {
				t.Fatal("inspection changed state")
			}
		})
	}
}

func TestExileInstructionPartialLegality(t *testing.T) {
	g, p, obj, first := targetQueryFixture(t)
	second := DeclarePermanentTarget("second", IsLand)
	other := g.PutOnBattlefield(NewLand("Other"), obj.Controller)
	obj.TargetSpecs = append(obj.TargetSpecs, second.targetSpec())
	obj.Targets = append(obj.Targets, other.ID())
	obj.targetIncarnations[other.ID()] = other.incarnationID
	obj.TargetZones[other.ID()] = core.ZoneBattlefield
	obj.Effects = []Effect{Sequence(Exile(first), Destroy(second))}
	p.GrantBaseAttr(core.Shroud)
	watched := []PermanentVisit{{other.ID(), other.incarnationID}}
	if certain, known := g.InspectSpellDestruction(obj, watched); !certain || !known {
		t.Fatal("illegal exile blocked legal destruction")
	}
	other.GrantBaseAttr(core.Shroud)
	if certain, known := g.InspectSpellDestruction(obj, watched); certain || !known {
		t.Fatal("all-illegal tree executed")
	}
	obj.Targets[1] = uuid.Nil
	if certain, known := g.InspectSpellDestruction(obj, watched); certain || known {
		t.Fatal("missing binding accepted")
	}
}

func TestExileInstructionGroupExecution(t *testing.T) {
	for _, replaced := range []bool{false, true} {
		t.Run(map[bool]string{false: "simultaneous", true: "replacement"}[replaced], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewCreature("one", "", 2, 2), b.PlayerID())
			q := g.PutOnBattlefield(NewCreature("two", "", 2, 2), b.PlayerID())
			p.Card.SetOwner(b.PlayerID())
			q.Card.SetOwner(b.PlayerID())
			observations := 0
			watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(evt *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				if evt.FromZone == core.ZoneBattlefield && evt.ToZone == core.ZoneExile {
					observations++
					if g.FindPermanent(p.ID()) != nil || g.FindPermanent(q.ID()) != nil {
						t.Error("partial exile batch visible")
					}
				}
				return false
			})
			g.PutOnBattlefield(NewArtifact("watcher", "", WithAbility(watcher)), a.PlayerID())
			var replacement *exileCountReplacement
			if replaced {
				replacement = &exileCountReplacement{cancel: p.ID(), t: t, all: []uuid.UUID{p.ID(), q.ID()}}
				g.AddReplacementEffect(replacement)
			}
			instruction := PermanentExileInstruction{Subjects: MatchingPermanents(IsCreature)}
			if err := ApplyEffect(g, instruction.Effect(), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if replaced {
				if replacement.calls != 2 || g.FindPermanent(p.ID()) == nil || len(b.Graveyard()) != 1 || len(g.GetExile()) != 0 || observations != 0 {
					t.Fatal("exile bypassed replacements")
				}
			} else if observations != 2 || len(g.GetExile()) != 2 {
				t.Fatal("exile group did not stay simultaneous")
			}
		})
	}
}
