package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCombatCollectionQuery(t *testing.T) {
	for _, syntax := range []string{
		`IsAttacking`, `IsntAttacking`, `IsBlocking`,
		`And([IsCardtype(Creature),Or([IsAttacking,IsBlocking])])`,
		`And([IsAttacking,ControlledByAPlayer(Opponent)])`,
	} {
		build, err := compileDestructionCollection(nameScopeNode(t, syntax), scope{resolving: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := mage.InspectContextualPermanentCollection(build(instanceBindings{})); !ok {
			t.Fatalf("lost current-combat declaration: %s", syntax)
		}
	}
	for _, syntax := range []string{
		`IsAttacking()`, `IsBlocking(You)`, `IsntAttacking(IsAttacking)`,
		`And([IsAttacking,Or([])])`, `Or([IsBlocking,Unknown])`,
	} {
		if _, err := compileDestructionCollection(nameScopeNode(t, syntax), scope{resolving: true}); err == nil {
			t.Fatalf("accepted malformed combat collection %s", syntax)
		}
	}
}

func TestRONCombatCollectionQueryGameplay(t *testing.T) {
	for _, name := range []string{"Army of Allah", "Piety"} {
		for _, state := range []string{"normal", "removed", "phased", "cycle", "returned"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				record := repositoryRecord(t, "ARN", name, "ARN.ron")
				collection := contextualQueryNode(t, record, "CreateEachPermanentLayerEffectUntil").Args[0]
				build, err := compileDestructionCollection(collection, scope{resolving: true})
				if err != nil {
					t.Fatal(err)
				}
				plan, ok := mage.InspectContextualPermanentCollection(build(instanceBindings{}))
				if !ok {
					t.Fatal("real record lost combat collection")
				}
				g, a, b := ronGame()
				g.SetStep(core.DeclareBlockers)
				attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 4), a.PlayerID())
				blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 2, 4), b.PlayerID())
				idle := g.PutOnBattlefield(mage.NewCreature("Idle", "", 2, 4), b.PlayerID())
				g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
				g.GetCombat().AddBlocker(blocker.ID(), attacker.ID())
				subject := attacker
				if name == "Piety" {
					subject = blocker
				}
				castRONSpell(t, g, a, repositoryCard(t, "ARN", name))
				switch state {
				case "removed":
					g.RemoveFromCombat(subject.ID())
				case "phased":
					g.PhaseOut(subject.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(subject.ID()))
				case "returned":
					g.RemoveFromBattlefield(subject)
					subject = g.PutOnBattlefield(subject.Card, a.PlayerID())
				}
				for _, game := range []*mage.Game{g, g.Clone()} {
					game.ApplyContinuousEffects()
					visits := plan.Select(&mage.EffectContext{Game: game, Controller: a.PlayerID()})
					selected := state == "normal"
					if selected && (len(visits) != 1 || visits[0].ID != subject.ID()) || !selected && len(visits) != 0 {
						t.Fatalf("wrong current combat visits: %v", visits)
					}
					game.ResolveStack()
					p := game.FindPermanentIncludingPhased(subject.ID())
					power, toughness := 2, 4
					if selected && name == "Army of Allah" {
						power += 2
					}
					if selected && name == "Piety" {
						toughness += 3
					}
					if p == nil || p.CurrentPower(game) != power || p.CurrentToughness(game) != toughness {
						t.Fatal("native layer membership disagrees with pure combat query")
					}
					game.RemoveFromCombat(subject.ID())
					game.GetCombat().AddAttacker(idle.ID(), a.PlayerID())
					game.GetCombat().AddBlocker(idle.ID(), attacker.ID())
					game.ApplyContinuousEffects()
					p = game.FindPermanentIncludingPhased(subject.ID())
					if selected && (p.CurrentPower(game) != power || p.CurrentToughness(game) != toughness) {
						t.Fatal("resolution layer lost its captured membership")
					}
					late := game.FindPermanent(idle.ID())
					if late.CurrentPower(game) != 2 || late.CurrentToughness(game) != 4 {
						t.Fatal("later combat member entered the captured layer set")
					}
				}
			})
		}
	}
}
