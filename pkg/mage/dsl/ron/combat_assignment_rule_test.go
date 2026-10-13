package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRONCombatAssignmentCube(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		t.Run(map[bool]string{false: "unblocked", true: "blocked"}[blocked], func(t *testing.T) {
			a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, "FEM", "Delif's Cube")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid Cube cast")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			cube := g.FindPermanent(card.ID())
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 4, 8), a.PlayerID())
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			g.PutOnBattlefield(mage.NewCreature("Blocker", "", 0, 8), b.PlayerID())
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), cube.ID(), 0, []uuid.UUID{attacker.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			a.SetAttackers(1, []string{"Attacker"})
			if blocked {
				b.SetBlockers(1, map[string]string{"Blocker": "Attacker"})
			}
			g.RunStepWithPriority(core.DeclareAttackers)
			g.RunStepWithPriority(core.DeclareBlockers)
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.RunStepWithPriority(core.CombatDamage)
				if game.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("Cube did not suppress combat assignment")
				}
				want := int(1)
				if blocked {
					want = 0
				}
				if game.FindPermanent(cube.ID()).Counters[core.Cube] != want {
					t.Fatal("counter or block timing")
				}
			}
			if blocked {
				return
			}
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), cube.ID(), 1, []uuid.UUID{attacker.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(cube.ID()).Counters[core.Cube] != 0 {
				t.Fatal("regeneration did not pay counter")
			}
			g.DestroyPermanent(g.FindPermanent(attacker.ID()))
			if g.FindPermanent(attacker.ID()) == nil {
				t.Fatal("second Cube ability did not regenerate")
			}
			g.RunStepWithPriority(core.Cleanup)
			g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
			g.ExecuteCombatDamage()
			if b.Life() != 16 {
				t.Fatal("assignment suppression survived cleanup")
			}
		})
	}
}

func TestRONCombatAssignmentInvalid(t *testing.T) {
	valid := `CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[AssignsNoCombatDamage],UntilEndOfTurn)`
	for _, action := range []string{
		strings.Replace(valid, "AssignsNoCombatDamage", "AssignsNoCombatDamage()", 1),
		strings.Replace(valid, "UntilEndOfTurn", "UntilEndOfCombat", 1),
		strings.Replace(valid, "Ref_TargetPermanent", "Trigger_ThatCreature", 1),
		strings.Replace(valid, "Ref_TargetPermanent", "Ref_TargetPlayer", 1),
		strings.Replace(valid, "[AssignsNoCombatDamage]", "[]", 1),
		strings.Replace(valid, "[AssignsNoCombatDamage]", "[AssignsNoCombatDamage,Unknown]", 1),
	} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + action + `])))])`
		nodes, err := Parse(strings.NewReader(src), "bad-assignment.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", action)
		}
	}
}

func TestRONCombatAssignmentCubeRollbackAndIdentity(t *testing.T) {
	for _, change := range []string{"unpaid", "wrong-controller", "shroud", "returned", "phased", "phase-cycle", "source-left", "source-return", "control", "abilities", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			cube := g.PutOnBattlefield(repositoryCard(t, "FEM", "Delif's Cube"), a.PlayerID())
			card := mage.NewCreature("Watched", "", 4, 8)
			card.SetOwner(a.PlayerID())
			controller := a.PlayerID()
			if change == "wrong-controller" {
				controller = b.PlayerID()
			}
			p := g.PutOnBattlefield(card, controller)
			if change != "unpaid" {
				a.ManaPool().Add(core.Colorless, 2)
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), cube.ID(), 0, []uuid.UUID{p.ID()})
			if change == "unpaid" || change == "wrong-controller" {
				want := 0
				if change == "wrong-controller" {
					want = 2
				}
				if err == nil || g.StackSize() != 0 || g.FindPermanent(cube.ID()).Tapped || a.ManaPool().TotalMana() != want {
					t.Fatal("illegal payment changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "shroud":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
			case "returned":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				p = g.PutOnBattlefield(card, a.PlayerID())
			case "phased":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			if change == "phased" {
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			if change == "source-left" || change == "source-return" {
				g.BouncePermanentToHand(g.FindPermanent(cube.ID()))
				if change == "source-return" {
					a.RemoveFromHand(cube.ID())
					g.PutOnBattlefield(cube.Card, a.PlayerID())
				}
			}
			if change == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: uuid.New(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if change == "abilities" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			}
			g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
			g.ExecuteBlockers(nil)
			g.ExecuteCombatDamage()
			active := change != "shroud" && change != "returned" && change != "phased"
			want := 16
			if active {
				want = 20
			}
			if b.Life() != want {
				t.Fatalf("life=%d want=%d", b.Life(), want)
			}
			if change != "source-left" {
				count := int(0)
				if active {
					count = 1
				}
				if change == "copy" {
					count = 2
				}
				if change == "source-return" {
					count = 0
				}
				if g.FindPermanent(cube.ID()).Counters[core.Cube] != count {
					t.Fatal("delayed binding or copy lost")
				}
			}
		})
	}
	g, a, b := ronGame()
	cube := g.PutOnBattlefield(repositoryCard(t, "FEM", "Delif's Cube"), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 8), b.PlayerID())
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), cube.ID(), 1, []uuid.UUID{other.ID()}); err == nil || a.ManaPool().TotalMana() != 2 || g.StackSize() != 0 {
		t.Fatal("missing counter payment spent mana")
	}
}

func TestRONCombatAssignmentEventAndInstances(t *testing.T) {
	rule := `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateTriggerUntil(WhenACreatureAttacksAndIsntBlocked(SinglePermanent(Ref_TargetPermanent)),ActionList([CreatePermanentRuleEffectUntil(Trigger_ThatCreature,[AssignsNoCombatDamage],UntilEndOfTurn)]),UntilEndOfTurn)])))`
	factory := permanentScopeFactory(t, "Artifact", rule)
	g, a, b := ronGame()
	first := g.PutOnBattlefield(factory(), a.PlayerID())
	second := g.PutOnBattlefield(factory(), b.PlayerID())
	p := g.PutOnBattlefield(mage.NewCreature("First", "", 4, 8), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 8), b.PlayerID())
	untouched := g.PutOnBattlefield(mage.NewCreature("Untouched", "", 1, 8), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), second.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.GetCombat().AddAttacker(p.ID(), b.PlayerID())
		game.GetCombat().AddAttacker(q.ID(), a.PlayerID())
		game.GetCombat().AddAttacker(untouched.ID(), b.PlayerID())
		game.ExecuteBlockers(nil)
		game.ExecuteCombatDamage()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 19 {
			t.Fatal("event or instance binding isolation")
		}
	}
}

func TestRONCombatAssignmentSourceAndHost(t *testing.T) {
	for _, ref := range []string{"ThisPermanent", "HostPermanent"} {
		t.Run(ref, func(t *testing.T) {
			source := `Card(Name:"Rule source",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:4,Toughness:8),Rules:[Activated(PayMana([]),ActionList([CreatePermanentRuleEffectUntil(ThisPermanent,[AssignsNoCombatDamage],UntilEndOfTurn)]))])`
			if ref == "HostPermanent" {
				source = `Card(Name:"Rule source",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),Activated(PayMana([]),ActionList([CreatePermanentRuleEffectUntil(HostPermanent,[AssignsNoCombatDamage],UntilEndOfTurn)]))])`
			}
			nodes, err := Parse(strings.NewReader(source), "source-assignment.ron")
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
			permanent := g.PutOnBattlefield(factory(), a.PlayerID())
			subject := permanent
			if ref == "HostPermanent" {
				subject = g.PutOnBattlefield(mage.NewCreature("Host", "", 4, 8), a.PlayerID())
				g.Attach(permanent.ID(), subject.ID())
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), permanent.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if ref == "HostPermanent" {
				g.BouncePermanentToHand(g.FindPermanent(permanent.ID()))
			}
			g.GetCombat().AddAttacker(subject.ID(), b.PlayerID())
			g.ExecuteCombatDamage()
			if b.Life() != 20 {
				t.Fatal("source or host rule failed")
			}
		})
	}
}

func TestRONCombatAssignmentCubeRepeatedCombat(t *testing.T) {
	g, a, b := ronGame()
	cube := g.PutOnBattlefield(repositoryCard(t, "FEM", "Delif's Cube"), a.PlayerID())
	creature := g.PutOnBattlefield(mage.NewCreature("Watched", "", 4, 8), a.PlayerID())
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), cube.ID(), 0, []uuid.UUID{creature.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	for i := 1; i <= 2; i++ {
		g.GetCombat().Reset()
		g.GetCombat().AddAttacker(creature.ID(), b.PlayerID())
		g.ExecuteBlockers(nil)
		g.ExecuteCombatDamage()
		if b.Life() != 20 || g.FindPermanent(cube.ID()).Counters[core.Cube] != i {
			t.Fatal("subscription did not persist through separate combats")
		}
	}
	g.RunStepWithPriority(core.Cleanup)
	g.GetCombat().Reset()
	g.GetCombat().AddAttacker(creature.ID(), b.PlayerID())
	g.ExecuteBlockers(nil)
	g.ExecuteCombatDamage()
	if b.Life() != 16 || g.FindPermanent(cube.ID()).Counters[core.Cube] != 2 {
		t.Fatal("cleanup left subscription or assignment rule active")
	}
}
