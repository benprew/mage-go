package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestNoncolorTargetQuery(t *testing.T) {
	for _, tc := range []struct {
		name       string
		protection func() *ProtectionAbility
		blocked    bool
	}{
		{"matching type", func() *ProtectionAbility { return ProtectionFromCardType(TypeSorcery) }, true},
		{"other type", func() *ProtectionAbility { return ProtectionFromCardType(TypeArtifact) }, false},
		{"matching subtype", func() *ProtectionAbility { return ProtectionFromSubType("Arcane") }, true},
		{"other subtype", func() *ProtectionAbility { return ProtectionFromSubType("Goblin") }, false},
		{"everything", ProtectionFromAll, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, p, obj, binding := targetQueryFixture(t)
			obj.Card.(*BaseCard).subTypes = []string{"Arcane"}
			protection := tc.protection()
			p.RuntimeAbilities = append(p.RuntimeAbilities, protection)
			plan, _ := InspectPermanentTarget(binding)
			got, known := plan.Select(g, obj)
			if !known || (len(got) == 0) != tc.blocked {
				t.Fatalf("got %v known %v blocked %v", got, known, tc.blocked)
			}
			if p.canBeTargetedByCharacteristics(g.resolutionTargetSource(obj), obj.Controller, g) == tc.blocked {
				t.Fatal("native legality differs")
			}
			protection.ExemptCardID = obj.SourceID
			if got, known := plan.Select(g, obj); !known || len(got) != 1 {
				t.Fatal("exact exemption lost")
			}
			protection.ExemptCardID = p.ID()
			if got, known := plan.Select(g, obj); !known || (len(got) == 0) != tc.blocked {
				t.Fatal("unrelated exemption applied")
			}
			clone := g.Clone()
			if got, known := plan.Select(clone, clone.StackPeek()); !known || (len(got) == 0) != tc.blocked {
				t.Fatal("clone differs")
			}
		})
	}
}

func TestNoncolorTargetQueryNoCallbacks(t *testing.T) {
	for _, protection := range []*ProtectionAbility{ProtectionFromCardType(TypeArtifact), ProtectionFromSubType("Goblin"), ProtectionFromAll()} {
		g, p, obj, binding := targetQueryFixture(t)
		protection.Filter.fn = func(Card) bool { panic("query called protection callback") }
		p.RuntimeAbilities = append(p.RuntimeAbilities, protection)
		plan, _ := InspectPermanentTarget(binding)
		random := g.random.Clone()
		if _, known := plan.Select(g, obj); !known {
			t.Fatal("typed protection unknown")
		}
		if !reflect.DeepEqual(random, g.random) || g.StackSize() != 1 {
			t.Fatal("query changed state")
		}
	}
	for _, filter := range []CardFilter{
		NewCardFilter("everything", func(Card) bool { panic("opaque protection called") }),
		OrCardFilter(AnyCardFilter(), NewCardFilter("opaque", func(Card) bool { panic("opaque child called") })),
	} {
		g, p, obj, binding := targetQueryFixture(t)
		p.RuntimeAbilities = append(p.RuntimeAbilities, &ProtectionAbility{Filter: filter})
		plan, _ := InspectPermanentTarget(binding)
		if _, known := plan.Select(g, obj); known {
			t.Fatal("opaque protection accepted")
		}
	}
}

func TestNoncolorTargetQuerySourceInformation(t *testing.T) {
	for _, quality := range []string{"type", "subtype"} {
		for _, state := range []string{"live", "layer", "departed", "returned", "phased", "phase-cycle", "copy", "face-down", "target-face-down", "wrapped"} {
			t.Run(quality+"/"+state, func(t *testing.T) {
				g, a, b := newSpellCopyGame()
				filter := HasCardTypeFilter(TypeLand)
				if state == "target-face-down" {
					filter = AnyPermanent
				}
				target := DeclarePermanentTarget("subject", filter)
				card := NewCreature("Source", "", 2, 2, WithSubTypes("Goblin"), WithAbility(Activate(Costs(ManaPayment("{0}")), Destroy(target), WithTargets(target))))
				source := g.PutOnBattlefield(card, a.PlayerID())
				victim := g.PutOnBattlefield(NewLand("Subject"), b.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
					t.Fatal(err)
				}
				obj := g.StackPeek()
				if state == "layer" {
					var modification = AddCardType(This, TypeArtifact)
					if quality == "subtype" {
						modification = AddSubtypeFamily(This, SubtypeCreature, "Elf")
					}
					if err := ApplyEffect(g, Until(EndOfTurn, modification), source.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				switch state {
				case "departed", "returned":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					if state == "returned" {
						g.PutOnBattlefield(card, b.PlayerID()).FaceDown = true
					}
				case "phased", "phase-cycle":
					g.PhaseOut(source.ID())
					if state == "phase-cycle" {
						g.PhaseIn([]uuid.UUID{source.ID()})
					}
				case "copy":
					if g.CopyStackObjectDirect(obj, b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				case "face-down":
					g.FindPermanent(source.ID()).FaceDown = true
				}
				protection := ProtectionFromCardType(TypeCreature)
				if quality == "subtype" {
					protection = ProtectionFromSubType("Goblin")
				}
				if state == "layer" {
					protection = ProtectionFromCardType(TypeArtifact)
					if quality == "subtype" {
						protection = ProtectionFromSubType("Elf")
					}
				}
				var ability Ability = protection
				if state == "wrapped" {
					ability = WrapGrantedAbility(ability)
				}
				victim = g.FindPermanent(victim.ID())
				victim.RuntimeAbilities = append(victim.RuntimeAbilities, ability)
				if state == "target-face-down" {
					victim.FaceDown = true
				}
				plan, ok := InspectPermanentTarget(target)
				if !ok {
					t.Fatal("target lost")
				}
				want := (state == "face-down" && quality == "subtype") || state == "target-face-down"
				for _, game := range []*Game{g, g.Clone()} {
					got, known := plan.Select(game, game.StackPeek())
					if !known || (len(got) == 1) != want {
						t.Fatalf("got %v known %v want %v", got, known, want)
					}
					game.ResolveStack()
					if (game.FindPermanent(victim.ID()) == nil) != want {
						t.Fatal("native resolution differs")
					}
				}
			})
		}
	}
}
