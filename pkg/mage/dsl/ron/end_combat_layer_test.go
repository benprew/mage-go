package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONEndCombatLayerGlyph(t *testing.T) {
	for _, scenario := range []string{"live", "phase-cycle", "phase-cycle-after", "phased-at-expiry", "returned-before", "phased-before", "returned-after", "clone", "copy", "removed-from-combat", "regenerated"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			g.SetStep(core.DeclareBlockers)
			wallCard := mage.NewCreature("Wall", "", 0, 6, mage.WithSubTypes("Wall"))
			wallCard.SetOwner(a.PlayerID())
			wall := g.PutOnBattlefield(wallCard, a.PlayerID())
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 20), b.PlayerID())
			g.GetCombat().AddAttacker(attacker.ID(), a.PlayerID())
			g.GetCombat().AddBlocker(wall.ID(), attacker.ID())
			spell := repositoryCard(t, "LEG", "Glyph of Destruction")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{wall.ID()}, 0); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("failed payment changed state")
			}
			a.ManaPool().Add(core.Red, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{attacker.ID()}, 0); err == nil || a.ManaPool().Count(core.Red) != 1 {
				t.Fatal("illegal target spent mana")
			}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{wall.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if wall.CurrentPower(g) != 0 {
				t.Fatal("bonus started before resolution")
			}
			pending := g.Clone()
			switch scenario {
			case "returned-before":
				g.BouncePermanentToHand(g.FindPermanent(wall.ID()))
				a.RemoveFromHand(wallCard.ID())
				g.PutOnBattlefield(wallCard, a.PlayerID())
			case "phased-before":
				g.PhaseOut(wall.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(wall.ID()))
				g.GetCombat().AddBlocker(wall.ID(), attacker.ID())
			case "copy":
				if g.CopySpellOnStack(spell.ID(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			g.ApplyContinuousEffects()
			valid := scenario != "returned-before" && scenario != "phased-before"
			want := 0
			if valid {
				want = 10
			}
			if scenario == "copy" {
				want = 20
			}
			if g.FindPermanentIncludingPhased(wall.ID()).CurrentPower(g) != want {
				t.Fatal("wrong bonus or incarnation")
			}
			if scenario == "phase-cycle-after" {
				g.PhaseIn(g.PhaseOut(wall.ID()))
				if g.FindPermanent(wall.ID()).CurrentPower(g) != 10 {
					t.Fatal("phase cycle ended the resolved bonus")
				}
			}
			if scenario == "live" {
				g.ExecuteCombatDamage()
				if g.FindPermanent(wall.ID()).Damage != 0 || g.FindPermanent(attacker.ID()).Damage != 10 {
					t.Fatal("Glyph combat damage did not use bonus and prevention")
				}
			}
			if scenario == "returned-after" {
				g.BouncePermanentToHand(g.FindPermanent(wall.ID()))
				a.RemoveFromHand(wallCard.ID())
				g.PutOnBattlefield(wallCard, a.PlayerID())
				valid = false
			}
			if scenario == "removed-from-combat" {
				g.RemoveFromCombat(wall.ID())
			}
			if scenario == "clone" {
				pending.ResolveStack()
				pending.RunStepWithPriority(core.EndCombat)
				if pending.FindPermanent(wall.ID()).CurrentPower(pending) != 0 || g.FindPermanent(wall.ID()).CurrentPower(g) != 10 {
					t.Fatal("clone shared duration")
				}
			}
			if valid {
				g.DealDamageToPermanent(g.FindPermanent(wall.ID()), 3, attacker.ID())
				if g.FindPermanent(wall.ID()).Damage != 0 {
					t.Fatal("Glyph did not prevent damage")
				}
			}
			if scenario == "phased-at-expiry" {
				g.PhaseOut(wall.ID())
			} else {
				g.PhaseIn([]uuid.UUID{wall.ID()})
			}
			g.RunStepWithPriority(core.EndCombat)
			g.PhaseIn([]uuid.UUID{wall.ID()})
			g.ApplyContinuousEffects()
			if g.FindPermanent(wall.ID()).CurrentPower(g) != 0 {
				t.Fatal("bonus lasted beyond combat")
			}
			if valid {
				g.DealDamageToPermanent(g.FindPermanent(wall.ID()), 3, attacker.ID())
				if g.FindPermanent(wall.ID()).Damage != 0 {
					t.Fatal("combat expiry ended turn prevention")
				}
			}
			if scenario == "regenerated" {
				g.AddRegenerationShield(wall.ID())
			}
			g.RunStepWithPriority(core.EndStep)
			survives := !valid || scenario == "regenerated"
			if (g.FindPermanent(wall.ID()) != nil) != survives {
				t.Fatal("delayed destruction lost target identity or regeneration")
			}
			if scenario == "regenerated" {
				if !g.FindPermanent(wall.ID()).Tapped {
					t.Fatal("destruction did not use regeneration")
				}
				g.RunStepWithPriority(core.Cleanup)
				g.DealDamageToPermanent(g.FindPermanent(wall.ID()), 1, attacker.ID())
				if g.FindPermanent(wall.ID()).Damage != 1 {
					t.Fatal("prevention survived cleanup")
				}
			}
		})
	}
}

func TestRONEndCombatLayerDelayedResolution(t *testing.T) {
	g, a, _ := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheNextEndOfCombat,ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AdjustPT(4,0),AddAbility([Flying])],UntilEndOfCombat)]))]))`)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.FireEvent(core.GameEvent{Type: core.EvtEndOfCombat, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	g.ApplyContinuousEffects()
	if source.CurrentPower(g) != 4 || !source.HasKeyword(core.Flying) {
		t.Fatal("delayed body did not install combat layers")
	}
	g.GetEffects().RemoveEndOfCombat()
	g.ApplyContinuousEffects()
	if source.CurrentPower(g) != 0 || source.HasKeyword(core.Flying) {
		t.Fatal("delayed layers did not expire")
	}
}

func TestRONEndCombatLayerInvalid(t *testing.T) {
	for _, action := range []string{
		`CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AdjustPT(1,0)],UntilEndOfCombat)`,
		`CreatePermanentLayerEffectUntil(ThisPermanent,[],UntilEndOfCombat)`,
		`CreatePermanentLayerEffectUntil(ThisPermanent,[AdjustPT(1,0)],UntilEndOfCombat())`,
		`CreatePermanentLayerEffectUntil(ThisPermanent,[AdjustPT(1,0)],UntilEndOfCombat(You))`,
		`CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([Flying()])],UntilEndOfCombat)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+action+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONEndCombatLayerUpkeep(t *testing.T) {
	g, a, _ := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AdjustPT(3,0),AddAbility([Banding])],UntilEndOfCombat)]))`)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	g.RunStepWithPriority(core.Upkeep)
	if g.FindPermanent(source.ID()).CurrentPower(g) != 3 || !g.FindPermanent(source.ID()).HasKeyword(core.Banding) {
		t.Fatal("upkeep body lost combat duration")
	}
	g.RunStepWithPriority(core.BeginCombat)
	if g.FindPermanent(source.ID()).CurrentPower(g) != 3 {
		t.Fatal("begin combat ended duration")
	}
	g.RunStepWithPriority(core.EndCombat)
	if g.FindPermanent(source.ID()).CurrentPower(g) != 0 || g.FindPermanent(source.ID()).HasKeyword(core.Banding) {
		t.Fatal("upkeep layers did not expire")
	}
}
