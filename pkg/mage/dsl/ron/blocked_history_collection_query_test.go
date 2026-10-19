package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNativeBlockedHistoryCollectionLowering(t *testing.T) {
	for _, name := range []string{"Glyph of Doom", "Glyph of Reincarnation"} {
		record := hostUntapRecord(t, "LEG", name)
		if _, err := Compile(record); err != nil {
			t.Fatal(err)
		}
		actionName := "DestroyEachPermanent"
		if name == "Glyph of Reincarnation" {
			actionName = "DestroyEachPermanentNoRegen"
		}
		action := contextualQueryNode(t, record, actionName)
		target := &permanentTargetPlan{name: "Ref_TargetPermanent"}
		build, err := compileDestructionCollection(action.Args[0], scope{resolving: true, permanentTarget: target})
		if err != nil {
			t.Fatal(err)
		}
		for range 2 {
			bindings := instanceBindings{permanents: map[*permanentTargetPlan]*mage.PermanentTargetBinding{target: mage.DeclarePermanentTarget("wall", mage.IsCreature)}}
			subjects := build(bindings)
			if _, known := mage.InspectContextualPermanentCollection(subjects); !known {
				t.Fatalf("%s lost native history declaration", name)
			}
			effect := (mage.DirectDestructionInstruction{Subjects: subjects, CannotRegenerate: name == "Glyph of Reincarnation"}).Effect()
			if _, known := mage.InspectOrderedDestruction(effect); !known {
				t.Fatal("native destruction clause is opaque")
			}
		}
	}
	for _, syntax := range []string{
		`WasBlockedByDefenderThisTurn`, `WasBlockedByDefenderThisTurn()`,
		`WasBlockedByDefenderThisTurn(You)`, `WasBlockedByDefenderThisTurn(Ref_TargetPermanent)`,
		`WasBlockedByDefenderThisTurn(Ref_TargetPermanent,Ref_TargetPermanent)`,
		`And([IsCardtype(Creature),WasBlockedByDefenderThisTurn(Ref_TargetPermanent)])`,
	} {
		if _, err := compileDestructionCollection(nameScopeNode(t, syntax), scope{resolving: true}); err == nil {
			t.Fatalf("unbound or invalid history accepted: %s", syntax)
		}
	}
}

func TestRONNativeBlockedHistoryCollectionGameplay(t *testing.T) {
	for _, name := range []string{"Glyph of Doom", "Glyph of Reincarnation"} {
		for _, state := range []string{"normal", "phase attacker", "cycle", "return attacker", "illegal wall", "control", "copy", "clone", "regeneration"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 0, 5, mage.WithSubTypes("Wall")), b.PlayerID())
				card := mage.NewCreature("Attacker", "", 2, 5)
				card.SetOwner(a.PlayerID())
				attacker := g.PutOnBattlefield(card, a.PlayerID())
				visits, _ := mage.InspectPermanentCollection(mage.MatchingPermanents(mage.IsCreature))
				originalVisit := visits.Select(g)[1]
				idle := g.PutOnBattlefield(mage.NewCreature("Idle", "", 2, 5), a.PlayerID())
				g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: wall.ID(), AttackerID: attacker.ID()}})
				g.GetCombat().Reset()
				if name == "Glyph of Reincarnation" {
					g.SetStep(core.EndCombat)
					g.SetStep(core.PostcombatMain)
				}
				spell := repositoryCard(t, "LEG", name)
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				a.ManaPool().Add(core.Black, 1)
				a.ManaPool().Add(core.Green, 1)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{wall.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				switch state {
				case "phase attacker":
					g.PhaseOut(attacker.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(attacker.ID()))
				case "return attacker":
					g.RemoveFromBattlefield(attacker)
					g.PutOnBattlefield(card, a.PlayerID())
				case "illegal wall":
					wall.GrantBaseAttr(core.Shroud)
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{TargetID: attacker.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "copy":
					if g.CopySpellOnStack(spell.ID(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				case "clone":
					g = g.Clone()
				case "regeneration":
					g.AddRegenerationShield(attacker.ID())
				}
				if certain, known := g.InspectSpellDestruction(g.StackPeek(), nil); certain || !known {
					t.Fatal("wrong bounded delay or death-return inspection status")
				}
				g.ResolveStack()
				if name == "Glyph of Doom" {
					g.RunStepWithPriority(core.EndCombat)
					g.ResolveStack()
				}
				destroyed := state != "phase attacker" && state != "return attacker" && state != "illegal wall"
				if name == "Glyph of Doom" && state == "regeneration" {
					destroyed = false
				}
				if name == "Glyph of Reincarnation" {
					if destroyed {
						found := false
						for _, visit := range visits.Select(g) {
							if visit.ID == card.ID() && visit.Incarnation != originalVisit.Incarnation {
								found = true
							}
						}
						if !found {
							t.Fatal("death continuation lost actual return")
						}
					}
					if destroyed && len(g.GetPlayer(a.PlayerID()).Graveyard()) != 1 {
						t.Fatal("wrong post-return graveyard")
					}
				} else if (g.FindPermanentIncludingPhased(attacker.ID()) == nil) != destroyed {
					t.Fatal("wrong delayed destruction membership")
				}
				if g.FindPermanent(idle.ID()) == nil {
					t.Fatal("history included idle creature")
				}
			})
		}
	}
}
