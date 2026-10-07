package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

// A clause fixture cannot establish support for the complete card.
func unblockedDelayClause(t *testing.T, name string) mage.CardFactory {
	t.Helper()
	n := hostUntapRecord(t, "FEM", name)
	if _, err := Compile(n); err == nil {
		t.Fatal("complete card unexpectedly supported")
	}
	rule := n.Fields["Rules"].Args[0]
	delay := rule.Args[1].Args[1].Args[0].Args[0]
	body := delay.Args[1].Args[0].Args
	if name == "Delif's Cone" {
		delay.Args[1].Args[0].Args = []Node{body[0].Args[0]}
	} else {
		delay.Args[1].Args[0].Args = []Node{body[1]}
	}
	n.Fields["Name"] = Node{Kind: String, String: "Unblocked delay clause"}
	n.Fields["Rules"] = Node{Kind: List, Args: []Node{rule}}
	factory, err := Compile(n)
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition(factory().Name(), factory()); err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONUnblockedDelayRealClauses(t *testing.T) {
	for _, name := range []string{"Delif's Cone", "Delif's Cube"} {
		for _, blocked := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/blocked=%t", name, blocked), func(t *testing.T) {
				a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				factory := unblockedDelayClause(t, name)
				card := factory()
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				creature := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 4, 8), a.PlayerID())
				creature.RevokeBaseAttr(core.AttrSummonSick)
				g.PutOnBattlefield(mage.NewCreature("Blocker", "", 0, 8), b.PlayerID())
				if name == "Delif's Cube" {
					a.ManaPool().Add(core.Colorless, 2)
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{creature.ID()}); err != nil {
					t.Fatal(err)
				}
				if len(g.DelayedTriggers()) != 0 {
					t.Fatal("subscribed before resolution")
				}
				g.ResolveStack()
				if len(g.DelayedTriggers()) != 1 {
					t.Fatal("missing subscription")
				}
				a.SetAttackers(1, []string{"Attacker"})
				if blocked {
					b.SetBlockers(1, map[string]string{"Blocker": "Attacker"})
				}
				g.RunStepWithPriority(core.DeclareAttackers)
				if a.Life() != 20 {
					t.Fatal("fired before blocker declaration")
				}
				g.RunStepWithPriority(core.DeclareBlockers)
				if name == "Delif's Cone" {
					want := 20
					if !blocked {
						want = 24
					}
					if a.Life() != want {
						t.Fatalf("life=%d want=%d", a.Life(), want)
					}
				} else {
					want := 0
					if !blocked {
						want = 1
					}
					if g.FindPermanent(source.ID()).Counters[core.Cube] != uint8(want) {
						t.Fatal("counter timing or source identity")
					}
				}
			})
		}
	}
}

func TestRONUnblockedDelayBindings(t *testing.T) {
	rule := `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateTriggerUntil(WhenACreatureAttacksAndIsntBlocked(SinglePermanent(Ref_TargetPermanent)),ActionList([GainLife(PowerOfPermanent(Trigger_ThatCreature))]),UntilEndOfTurn)])))`
	factory := permanentScopeFactory(t, "Artifact", rule)
	if factory().Abilities()[0].(*mage.ActionDefinition).Targets()[0] == factory().Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("shared targets")
	}
	g, a, b := ronGame()
	first := g.PutOnBattlefield(factory(), a.PlayerID())
	second := g.PutOnBattlefield(factory(), b.PlayerID())
	sources := []*mage.Permanent{first, second, first}
	creatures := []*mage.Permanent{
		g.PutOnBattlefield(mage.NewCreature("First", "", 3, 8), a.PlayerID()),
		g.PutOnBattlefield(mage.NewCreature("Second", "", 7, 8), b.PlayerID()),
		g.PutOnBattlefield(mage.NewCreature("Third", "", 5, 8), a.PlayerID()),
	}
	for i, source := range sources {
		player := a
		if i == 1 {
			player = b
		}
		if err := g.ActivateAbilityByIndex(player.PlayerID(), source.ID(), 0, []uuid.UUID{creatures[i].ID()}); err != nil {
			t.Fatal(err)
		}
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	g.BouncePermanentToHand(g.FindPermanent(sources[0].ID()))
	for _, game := range []*mage.Game{g, clone} {
		for _, creature := range creatures {
			game.GetCombat().AddAttacker(creature.ID(), b.PlayerID())
		}
		game.ExecuteBlockers(nil)
		if game.GetPlayer(a.PlayerID()).Life() != 28 || game.GetPlayer(b.PlayerID()).Life() != 27 {
			t.Fatal("pending bindings or controller isolation")
		}
	}
}

func TestRONUnblockedDelayInvalid(t *testing.T) {
	valid := `CreateTriggerUntil(WhenACreatureAttacksAndIsntBlocked(SinglePermanent(Ref_TargetPermanent)),ActionList([GainLife(PowerOfPermanent(Trigger_ThatCreature))]),UntilEndOfTurn)`
	for _, action := range []string{
		strings.Replace(valid, "UntilEndOfTurn", "UntilEndOfCombat", 1),
		strings.Replace(valid, "UntilEndOfTurn", "UntilEndOfTurn()", 1),
		strings.Replace(valid, "SinglePermanent(Ref_TargetPermanent)", "Ref_TargetPermanent", 1),
		strings.Replace(valid, "Ref_TargetPermanent", "Ref_TargetPlayer", 1),
		strings.Replace(valid, "Ref_TargetPermanent", "Trigger_ThatCreature", 1),
		strings.Replace(valid, "SinglePermanent(Ref_TargetPermanent)", "SinglePermanent(Ref_TargetPermanent,ThisPermanent)", 1),
		strings.Replace(valid, "WhenACreatureAttacksAndIsntBlocked", "WhenACreatureAttacks", 1),
		strings.Replace(valid, "Trigger_ThatCreature", "Trigger_ThatPermanent", 1),
		strings.Replace(valid, "ActionList([GainLife(PowerOfPermanent(Trigger_ThatCreature))])", "ActionList([])", 1),
		valid + `,GainLife(PowerOfPermanent(Trigger_ThatCreature))`,
	} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + action + `])))])`
		nodes, err := Parse(strings.NewReader(src), "bad-delay.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", action)
		}
	}
}

func TestRONUnblockedDelayLegalityAndRollback(t *testing.T) {
	for _, change := range []string{"unpaid", "wrong-controller", "shroud", "returned", "phased", "phase-cycle"} {
		t.Run(change, func(t *testing.T) {
			factory := unblockedDelayClause(t, "Delif's Cube")
			g, a, b := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			card := mage.NewCreature("Attacker", "", 2, 8)
			card.SetOwner(a.PlayerID())
			controller := a.PlayerID()
			if change == "wrong-controller" {
				controller = b.PlayerID()
			}
			target := g.PutOnBattlefield(card, controller)
			if change != "unpaid" {
				a.ManaPool().Add(core.Colorless, 2)
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()})
			if change == "unpaid" || change == "wrong-controller" {
				want := 0
				if change == "wrong-controller" {
					want = 2
				}
				if err == nil || g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != want || len(g.DelayedTriggers()) != 0 {
					t.Fatal("illegal activation changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "shroud":
				g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
			case "returned":
				g.BouncePermanentToHand(target)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "phased":
				g.PhaseOut(target.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			}
			g.ResolveStack()
			want := 0
			if change == "phase-cycle" {
				want = 1
			}
			if len(g.DelayedTriggers()) != want {
				t.Fatal("resolution ignored target legality")
			}
		})
	}
}

func TestRONUnblockedDelayInventory(t *testing.T) {
	report, err := InspectFS(os.DirFS("../../../../cards/mtgish"), "FEM.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct{ name, role string }{{"WhenACreatureAttacksAndIsntBlocked", "delayed_combat_event"}, {"SinglePermanent", "delayed_combat_subject"}, {"Ref_TargetPermanent", "permanent_reference"}} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == want.name && signature.Role == want.role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s/%s", want.name, want.role)
		}
	}
}

func TestRONUnblockedDelayConePaidBranch(t *testing.T) {
	n := hostUntapRecord(t, "FEM", "Delif's Cone")
	branch := n.Fields["Rules"].Args[0].Args[1].Args[1].Args[0].Args[0].Args[1].Args[0].Args[1]
	if branch.Name != "If" || branch.Args[0].Name != "CostWasPaid" {
		t.Fatal("no-damage rule must follow accepted life gain")
	}
}

func TestRONUnblockedDelayParentEventBinding(t *testing.T) {
	src := `Card(Name:"Parent capture",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:3,Toughness:8),Rules:[TriggerA(WhenAPermanentEntersTheBattlefield(IsCardtype(Land)),ActionList([CreateTriggerUntil(WhenACreatureAttacksAndIsntBlocked(SinglePermanent(ThisPermanent)),ActionList([TapPermanent(Trigger_ThatPermanent),GainLife(PowerOfPermanent(Trigger_ThatCreature))]),UntilEndOfTurn)]))])`
	nodes, err := Parse(strings.NewReader(src), "parent-delay.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition(factory().Name(), factory()); err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	creature := g.PutOnBattlefield(factory(), a.PlayerID())
	land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if land.Tapped || len(g.DelayedTriggers()) != 1 {
		t.Fatal("parent capture changed during subscription")
	}
	g.GetCombat().AddAttacker(creature.ID(), b.PlayerID())
	g.ExecuteBlockers(nil)
	if !g.FindPermanent(land.ID()).Tapped || g.FindPermanent(creature.ID()).Tapped || a.Life() != 23 {
		t.Fatal("parent and delayed event references were combined")
	}
}
