package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDeathHistoryDestinationPlan(t *testing.T) {
	action := `CreateEachPermanentRuleEffectUntil(WasDealtDamageByPermanentThisTurn(ThisPermanent,IsCardtype(Creature)),[CantBeRegenerated],UntilEndOfTurn)`
	nodes, err := Parse(strings.NewReader(action), "plan.ron")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := compileCollectionDamageHistoryRulePlan(nodes[0], scope{permanent: true})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		g, a, b := ronGame()
		source := g.PutOnBattlefield(mage.NewCreature("Source", "", 1, 8), a.PlayerID())
		p := g.PutOnBattlefield(mage.NewCreature("Subject", "", 1, 8), b.PlayerID())
		if err := mage.ApplyEffect(g, plan.build(instanceBindings{}), source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.DealDamageToPermanent(p, 1, source.ID())
		g.AddRegenerationShield(p.ID())
		query, known := mage.InspectPermanentCollection(mage.MatchingPermanents(mage.IsCreature))
		if !known {
			t.Fatal("missing collection declaration")
		}
		found := false
		for _, visit := range query.Select(g) {
			if visit.ID == p.ID() {
				found = true
				if got, known := g.InspectDestructionDestinations(visit, false); !known || got != mage.DestructionToGraveyard {
					t.Fatal("plan lost history")
				}
			}
		}
		if !found {
			t.Fatal("missing recipient visit")
		}
	}
	for _, bad := range []string{
		strings.Replace(action, "ThisPermanent", "Ref_TargetPermanent", 1),
		strings.Replace(action, "IsCardtype(Creature)", "IsCardtype(Artifact)", 1),
		strings.Replace(action, "CantBeRegenerated]", "CantBeRegenerated()]", 1),
		strings.Replace(action, "UntilEndOfTurn", "UntilEndOfCombat", 1),
	} {
		nodes, err := Parse(strings.NewReader(bad), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileCollectionDamageHistoryRulePlan(nodes[0], scope{permanent: true}); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}

func TestRONDeathHistoryDestination(t *testing.T) {
	for _, name := range []string{"Stone Rain", "Wrath of God"} {
		for _, change := range []string{"damage", "no damage", "source left", "clone damage", "cleanup", "phase cycle"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				sword := g.PutOnBattlefield(repositoryCard(t, "DRK", "Runesword"), a.PlayerID())
				attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 8), a.PlayerID())
				attacker.RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
				a.ManaPool().Add(core.Colorless, 3)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), sword.ID(), 0, []uuid.UUID{attacker.ID()}); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				p := g.PutOnBattlefield(mage.NewCreature("Subject", "", 1, 8), b.PlayerID())
				p.Card.SetOwner(b.PlayerID())
				if change != "no damage" && change != "clone damage" {
					g.DealDamageToPermanent(p, 1, attacker.ID())
				}
				if change == "source left" {
					g.BouncePermanentToHand(attacker)
					g.PutTriggersOnStack()
					g.ResolveStack()
				}
				if change == "cleanup" {
					g.RunStepWithPriority(core.Cleanup)
					g.SetStep(core.PrecombatMain)
				}
				if change == "phase cycle" {
					g.PhaseIn(g.PhaseOut(p.ID()))
				}
				if name == "Stone Rain" {
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.AddCardType(mage.This, core.TypeLand)), p.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				g.AddRegenerationShield(p.ID())
				var spell *mage.StackObject
				if name == "Stone Rain" {
					spell = castRONSpell(t, g, a, repositoryCard(t, "LEA", name), p.ID())
				} else {
					spell = castRONSpell(t, g, a, repositoryCard(t, "LEA", name))
				}
				instructions, ok := mage.InspectDirectDestruction(spell.Effects[0])
				if !ok || len(instructions) != 1 {
					t.Fatal("destruction declaration")
				}
				query, ok := mage.InspectPermanentCollection(mage.MatchingPermanents(mage.IsCreature))
				if !ok {
					t.Fatal("collection")
				}
				var visit mage.PermanentVisit
				for _, v := range query.Select(g) {
					if v.ID == p.ID() {
						visit = v
					}
				}
				for _, game := range []*mage.Game{g.Clone(), g} {
					damaged := change != "no damage" && change != "cleanup" && change != "clone damage"
					if change == "clone damage" && game != g {
						game.DealDamageToPermanent(game.FindPermanent(p.ID()), 1, attacker.ID())
						damaged = true
					}
					want := mage.DestructionToExile
					if !damaged {
						want = mage.DestructionToGraveyard
						if name == "Stone Rain" {
							want = mage.DestructionNoMove
						}
					}
					for range 2 {
						if got, known := game.InspectDestructionDestinations(visit, instructions[0].CannotRegenerate); !known || got != want {
							t.Fatalf("got %v known %v want %v", got, known, want)
						}
					}
					game.ResolveStack()
					if (game.FindExiledCard(p.Card.ID()) != nil) != (want == mage.DestructionToExile) || (game.FindPermanent(p.ID()) != nil) != (want == mage.DestructionNoMove) || (len(game.GetPlayer(b.PlayerID()).Graveyard()) == 1) != (want == mage.DestructionToGraveyard) {
						t.Fatal("native resolution disagrees")
					}
				}
			})
		}
	}
}
