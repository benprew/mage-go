package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCombatTargetRealDeclarations(t *testing.T) {
	for _, name := range []string{"Lesser Werewolf", "Sentinel"} {
		t.Run(name, func(t *testing.T) {
			record := hostUntapRecord(t, "LEG", name)
			rule := record.Fields["Rules"].Args[0]
			targetNode := rule.Args[1].Args[0].Args[0]
			build, _, err := compileTarget(targetNode, scope{permanent: true, activation: true})
			if err != nil {
				t.Fatal(err)
			}
			first, second := build(instanceBindings{}), build(instanceBindings{})
			if first == second {
				t.Fatal("shared target declaration")
			}
			g, a, b := ronGame()
			source := g.PutOnBattlefield(mage.NewCreature("Source", "", 2, 8, mage.WithAbility(mage.Activate(mage.Costs(mage.ManaPayment("{B}")), mage.TapSubject(first.(*mage.PermanentTargetBinding)), mage.WithTargets(first.(*mage.PermanentTargetBinding))))), a.PlayerID())
			victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 8), b.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{source.ID()})
			a.ManaPool().Add(core.Black, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err == nil {
				t.Fatal("target accepted before block")
			}
			if a.ManaPool().Count(core.Black) != 1 {
				t.Fatal("target rejection spent mana")
			}
			g.ExecuteBlockers([]mage.BlockAssignment{{AttackerID: source.ID(), BlockerID: victim.ID()}})
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if !g.FindPermanent(victim.ID()).Tapped {
				t.Fatal("real target declaration failed in native combat")
			}
			if _, err := Compile(record); err == nil {
				t.Fatal("remaining full-record blockers disappeared")
			}
		})
	}
}

func TestRONCombatTargetDeclarations(t *testing.T) {
	for _, filter := range []string{
		`IsBlockingAttacker`, `IsBlockingAttacker()`, `IsBlockingAttacker(ThisPermanent())`, `IsBlockingAttacker(You)`, `IsBlockingAttacker(Ref_TargetPermanent)`, `IsBlockedByDefender(HostPermanent)`,
		`Or([])`, `Or([IsBlockingAttacker(ThisPermanent)])`, `Or([IsBlockingAttacker(ThisPermanent),IsBlockingAttacker(ThisPermanent)])`, `Or([IsBlockingAttacker(ThisPermanent),IsTapped])`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:8),Rules:[Activated(PayMana([]),Targeted([TargetPermanent(` + filter + `)],ActionList([TapPermanent(Ref_TargetPermanent)])))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-combat-target.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid target", filter)
		}
	}
	for _, filter := range []string{`IsBlockingAttacker(ThisPermanent)`, `IsBlockedByDefender(ThisPermanent)`, `Or([IsBlockedByDefender(ThisPermanent),IsBlockingAttacker(ThisPermanent)])`} {
		factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(`+filter+`)],ActionList([TapPermanent(Ref_TargetPermanent)])))`)
		if err := mage.ValidateCardDefinition("Scope", factory()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRONCombatTargetNativeFactories(t *testing.T) {
	for _, direction := range []string{"IsBlockingAttacker", "IsBlockedByDefender"} {
		for _, change := range []string{"live", "exit", "copy", "clone"} {
			t.Run(direction+"/"+change, func(t *testing.T) {
				nodes, err := Parse(strings.NewReader(`Card(Name:"Combat source",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:8),Rules:[Activated(PayMana([ManaCostB]),Targeted([TargetPermanent(And([IsCardtype(Creature),`+direction+`(ThisPermanent)]))],ActionList([TapPermanent(Ref_TargetPermanent)])))])`), "combat-source.ron")
				if err != nil {
					t.Fatal(err)
				}
				factory, err := Compile(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				card := factory()
				source := g.PutOnBattlefield(card, a.PlayerID())
				victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 8), b.PlayerID())
				attacker, blocker := source, victim
				active := a
				if direction == "IsBlockedByDefender" {
					attacker, blocker = victim, source
					active = b
				}
				attacker.RevokeBaseAttr(core.AttrSummonSick)
				g.SetActivePlayerIndex(0)
				if active == b {
					g.SetActivePlayerIndex(1)
				}
				g.ExecuteAttackers(active.PlayerID(), []uuid.UUID{attacker.ID()})
				g.ExecuteBlockers([]mage.BlockAssignment{{AttackerID: attacker.ID(), BlockerID: blocker.ID()}})
				g.UntapPermanent(victim)
				a.ManaPool().Add(core.Black, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
					t.Fatal(err)
				}
				clone := g.Clone()
				switch change {
				case "exit":
					g.RemoveFromCombat(victim.ID())
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				case "clone":
					clone.PhaseOut(victim.ID())
				}
				g.ResolveStack()
				if g.FindPermanent(victim.ID()).Tapped != (change != "exit") {
					t.Fatal("RON target recheck")
				}
				clone.ResolveStack()
				if change != "clone" && !clone.FindPermanent(victim.ID()).Tapped {
					t.Fatal("clone target resolution")
				}
			})
		}
	}
}
