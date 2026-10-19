package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDestructionStateChangeRead(t *testing.T) {
	g, p, _, _ := targetQueryFixture(t)
	visit := PermanentVisit{p.ID(), p.incarnationID}
	for _, protected := range []bool{false, true} {
		if protected {
			p.GrantBaseAttr(core.Indestructible)
		}
		g.AddRegenerationShield(p.ID())
		g.effects.AddReplacement(&proposalTrap{})
		before := g.Clone()
		for range 2 {
			if change, known := g.InspectDestructionStateChange([]PermanentVisit{visit, visit}); change != !protected || !known {
				t.Fatalf("protected=%v change=%v/%v", protected, change, known)
			}
		}
		if !reflect.DeepEqual(g.effects.replacements, before.effects.replacements) || !reflect.DeepEqual(g.random, before.random) || !reflect.DeepEqual(g.resolution, before.resolution) || p.Tapped {
			t.Fatal("inspection changed state")
		}
		clone := g.Clone()
		clone.MutablePermanent(p.ID()).RevokeBaseAttr(core.Indestructible)
		if change, known := clone.InspectDestructionStateChange([]PermanentVisit{visit}); !change || !known {
			t.Fatal("query did not use current clone properties")
		}
	}
}

func TestDestructionStateChangeInvalid(t *testing.T) {
	g, p, _, _ := targetQueryFixture(t)
	visit := PermanentVisit{p.ID(), p.incarnationID}
	if _, known := (*Game)(nil).InspectDestructionStateChange(nil); known {
		t.Fatal("nil game accepted")
	}
	if change, known := g.InspectDestructionStateChange(nil); change || !known {
		t.Fatal("empty group can change state")
	}
	for _, bad := range []PermanentVisit{{}, {p.ID(), uuid.New()}, {uuid.New(), uuid.New()}} {
		if _, known := g.InspectDestructionStateChange([]PermanentVisit{visit, bad}); known {
			t.Fatal("possible change hid invalid suffix")
		}
	}
	g.PhaseOut(p.ID())
	if _, known := g.InspectDestructionStateChange([]PermanentVisit{visit}); known {
		t.Fatal("phased visit accepted")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	if change, known := g.InspectDestructionStateChange([]PermanentVisit{visit}); !change || !known {
		t.Fatal("phase cycle lost visit")
	}
	g.BouncePermanentToHand(p)
	g.PutOnBattlefield(p.Card, p.ControllerID())
	if _, known := g.InspectDestructionStateChange([]PermanentVisit{visit}); known {
		t.Fatal("query followed a returned object")
	}
}

func TestDestructionStateChangeSequence(t *testing.T) {
	for _, state := range []string{"indestructible", "mixed", "regenerate", "cancel", "exile", "hand", "ordinary"} {
		for _, noRegen := range []bool{false, true} {
			t.Run(state+map[bool]string{false: "/regen", true: "/no-regen"}[noRegen], func(t *testing.T) {
				g, land, obj, binding := targetQueryFixture(t)
				p := g.PutOnBattlefield(NewCreature("Prefix", "", 2, 2), obj.Controller)
				switch state {
				case "indestructible", "mixed":
					p.GrantBaseAttr(core.Indestructible)
					if state == "mixed" {
						g.PutOnBattlefield(NewCreature("Other", "", 2, 2), obj.Controller)
					}
				case "regenerate":
					g.AddRegenerationShield(p.ID())
				case "cancel":
					g.effects.AddReplacement(&nextDestructionReplacement{subject: p.ID(), incarnation: p.incarnationID})
				case "exile":
					g.effects.AddReplacement(&temporaryDeathExileReplacement{subject: p.ID(), incarnation: p.incarnationID})
				case "hand":
					p.RuntimeAbilities = append(p.RuntimeAbilities, ReturnToHandInsteadOfDyingWithPlayLock())
				}
				prefix := (DirectDestructionInstruction{Subjects: MatchingPermanents(IsCreature), CannotRegenerate: noRegen}).Effect()
				obj.Effects = []Effect{Sequence(prefix, Destroy(binding))}
				watched := []PermanentVisit{{land.ID(), land.incarnationID}}
				plan, ok := InspectOrderedDestruction(Sequence(prefix, DestroyEach(MatchingPermanents(IsLand))))
				if !ok {
					t.Fatal("native sequence rejected")
				}
				want := state == "indestructible"
				for _, game := range []*Game{g, g.Clone()} {
					if certain, known := game.InspectSpellDestruction(game.StackPeek(), watched); certain != want || known != want {
						t.Fatalf("spell proof=%v/%v want=%v", certain, known, want)
					}
					if certain, known := plan.Certain(&EffectContext{Game: game, Controller: obj.Controller}, watched); certain != want || known != want {
						t.Fatalf("collection proof=%v/%v want=%v", certain, known, want)
					}
				}
				g.ResolveStack()
				if g.FindPermanent(land.ID()) != nil {
					t.Fatal("later destruction failed")
				}
				if want && g.FindPermanent(p.ID()) == nil {
					t.Fatal("indestructible prefix moved")
				}
			})
		}
	}
}
