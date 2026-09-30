package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetLegalityPoppedSpellColors(t *testing.T) {
	for _, protection := range []Color{Red, Blue} {
		for _, override := range []string{"printed", "blue", "colorless"} {
			for _, mixed := range []bool{false, true} {
				t.Run(protection.String()+"/"+override+"/"+map[bool]string{false: "singular", true: "mixed"}[mixed], func(t *testing.T) {
					g, a, b := newSpellCopyGame()
					targets := DeclareAnyNumberDamageTargets("recipients")
					spell := NewInstant("Color source", "{R}", Spell(Sequence(DealDamageEach(This, targets, Fixed(2)), GainLifeFor(You, Fixed(1))), WithTargets(targets)))
					spell.SetOwner(a.PlayerID())
					a.AddToHand(spell)
					a.ManaPool().Add(Red, 1)
					victim := g.PutOnBattlefield(NewCreature("Recipient", "", 2, 9), b.PlayerID())
					ids := []uuid.UUID{victim.ID()}
					if mixed {
						ids = append(ids, b.PlayerID())
					}
					if err := g.CastSpellByName(a.PlayerID(), spell.Name(), ids, 0); err != nil {
						t.Fatal(err)
					}
					obj := g.StackPeek()
					if override == "blue" {
						if !g.ChangeObjectColor(obj.ID, Blue) {
							t.Fatal("color change failed")
						}
					}
					if override == "colorless" {
						colors := []Color{}
						obj.ColorOverride = &colors
					}
					victim.RuntimeAbilities = append(victim.RuntimeAbilities, ProtectionFromColor(protection))
					illegal := override == "printed" && protection == Red || override == "blue" && protection == Blue
					clone := g.Clone()
					clone.ResolveTopOfStack()
					if a.Life() != 20 || b.Life() != 20 || g.FindPermanent(victim.ID()).Damage != 0 || g.StackSize() != 1 {
						t.Fatal("clone changed the original game")
					}
					g.ResolveStack()
					for _, game := range []*Game{g, clone} {
						wantDamage, wantGain, wantPlayer := 2, 1, 0
						if illegal {
							wantDamage = 0
							if !mixed {
								wantGain = 0
							}
						}
						if mixed {
							wantPlayer = 2
						}
						if game.FindPermanent(victim.ID()).Damage != wantDamage || game.GetPlayer(a.PlayerID()).Life() != 20+wantGain || game.GetPlayer(b.PlayerID()).Life() != 20-wantPlayer {
							t.Fatalf("illegal=%v: damage=%d life=%d/%d", illegal, game.FindPermanent(victim.ID()).Damage, game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
						}
					}
				})
			}
		}
	}
}

func TestTargetLegalityNewSpellCopyColors(t *testing.T) {
	for _, modal := range []bool{false, true} {
		for _, protected := range []Color{Red, Blue} {
			t.Run(map[bool]string{false: "spell", true: "mode"}[modal]+protected.String(), func(t *testing.T) {
				g, a, b := newSpellCopyGame()
				target := DeclareDamageTarget("recipient")
				body := Sequence(DealDamageFrom(This, target, Fixed(2)), GainLifeFor(You, Fixed(1)))
				var ability Ability = Spell(body, WithTargets(target))
				if modal {
					ability = SpellModes(ModeOf("Damage", body, WithTargets(target)), ModeOf("Life", GainLifeFor(You, Fixed(3))))
				}
				spell := NewInstant("Copy source", "{R}", WithAbility(ability))
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				a.ManaPool().Add(Red, 1)
				victim := g.PutOnBattlefield(NewCreature("Protected", "", 2, 9, WithAbility(ProtectionFromColor(protected))), b.PlayerID())
				if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
				original := g.StackPeek()
				if !g.ChangeObjectColor(original.ID, Blue) {
					t.Fatal("color change failed")
				}
				a.chooseQueue = [][]uuid.UUID{{victim.ID()}}
				copied := g.CopyStackObjectDirect(original, a.PlayerID(), true)
				wantTarget := victim.ID()
				if protected == Red {
					wantTarget = b.PlayerID()
				}
				if copied.Targets[0] != wantTarget || original.Targets[0] != b.PlayerID() {
					t.Fatal("copy accepted illegal target or changed original")
				}
				if copied.ColorOverride != nil {
					t.Fatal("copy retained a noncopiable color change")
				}
				g.ResolveTopOfStack()
				wantDamage, wantLife := 2, 20
				if protected == Red {
					wantDamage, wantLife = 0, 18
				}
				if g.FindPermanent(victim.ID()).Damage != wantDamage || b.Life() != wantLife || a.Life() != 21 {
					t.Fatal("copy resolution lost its printed colors")
				}
			})
		}
	}
}

func TestTargetLegalitySpellColorViewIsolation(t *testing.T) {
	g, _, _ := newSpellCopyGame()
	spell := NewInstant("Spell", "{R}")
	colors := []Color{Blue}
	obj := &StackObject{SourceID: spell.ID(), Card: spell, ColorOverride: &colors}
	view := g.resolutionTargetSource(obj)
	colors[0] = Green
	if !ProtectionFromColor(Blue).BlocksInGame(view, g) || ProtectionFromColor(Green).BlocksInGame(view, g) {
		t.Fatal("color view shares mutable colors")
	}
	unrelated := NewInstant("Unrelated", "{G}")
	obj.TargetSource = unrelated
	if g.resolutionTargetSource(obj) != unrelated {
		t.Fatal("unrelated source inherited spell colors")
	}
	obj.Card, obj.TargetSource = nil, nil
	if g.resolutionTargetSource(obj) != nil {
		t.Fatal("missing source acquired colors")
	}
}
