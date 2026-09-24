package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCleanupHistoryAcrossPriorityRounds(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(a.PlayerID())
	g.PutOnBattlefield(victim, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(victim.ID()))
	attacker := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.trackers.Turn.RecordAttacked(attacker.ID())
	register := func() {
		g.RegisterDelayedTrigger(&DelayedTrigger{EventType: EvtCleanup, Controller: a.PlayerID(), Effects: []Effect{GainLifeFor(You, CreaturesDiedThisTurn())}})
	}
	register()
	added := false
	g.SetOnPriority(func(game *Game, _ int, _ bool) PriorityAction {
		if game.CreatureDeaths() == 0 {
			t.Fatal("cleanup cleared deaths before priority")
		}
		if !added {
			added = true
			next := NewCreature("Cleanup victim", "", 1, 1)
			next.SetOwner(b.PlayerID())
			game.PutOnBattlefield(next, b.PlayerID())
			game.DoSacrifice(game.FindPermanent(next.ID()))
			register()
		}
		return PriorityAction{Type: PriorityPass}
	})
	g.RunStepWithPriority(Cleanup)
	if a.Life() != 24 || g.GetCleanupPriorityRounds() != 2 {
		t.Fatalf("life = %d, priority rounds = %d; want 24 and 2", a.Life(), g.GetCleanupPriorityRounds())
	}
	if g.CreatureDeaths() != 0 || !g.trackers.Turn.AttackedLastTurn(a.PlayerID())[attacker.ID()] {
		t.Fatal("final cleanup did not reset deaths and preserve attacker history")
	}
	if g.trackers.Turn.PlayerLifeGained(a.PlayerID()) != 0 {
		t.Fatal("life history survived final cleanup")
	}
}

func TestCleanupHistoryStateBasedActions(t *testing.T) {
	for _, trigger := range []bool{false, true} {
		t.Run(map[bool]string{false: "without trigger", true: "with trigger"}[trigger], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			victim := NewCreature("Temporary survivor", "", 1, 0)
			victim.SetOwner(a.PlayerID())
			g.PutOnBattlefield(victim, a.PlayerID())
			if err := ApplyEffect(g, Until(EndOfTurn, ModifyPT(This, Fixed(0), Fixed(1))), victim.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if trigger {
				g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(WhenEvent(CreatureDies(), GainLifeFor(You, CreaturesDiedThisTurn())))), a.PlayerID())
			}
			seen := false
			g.SetOnPriority(func(game *Game, _ int, _ bool) PriorityAction {
				seen = true
				if game.CreatureDeaths() != 1 {
					t.Fatalf("cleanup priority deaths = %d, want 1", game.CreatureDeaths())
				}
				return PriorityAction{Type: PriorityPass}
			})
			g.RunStepWithPriority(Cleanup)
			wantLife := 20
			if trigger {
				wantLife++
			}
			if !seen || g.GetCleanupPriorityRounds() != 1 || a.Life() != wantLife || g.CreatureDeaths() != 0 {
				t.Fatalf("priority = %v, rounds = %d, life = %d, deaths = %d", seen, g.GetCleanupPriorityRounds(), a.Life(), g.CreatureDeaths())
			}
		})
	}
}

func TestCleanupHistoryCloneAndActivationLimits(t *testing.T) {
	for _, granted := range []bool{false, true} {
		t.Run(map[bool]string{false: "intrinsic", true: "granted"}[granted], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewArtifact("Source", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), ActivationLimit(1))))
			if granted {
				source = NewArtifact("Source", "")
				g.PutOnBattlefield(NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(MatchingPermanents(IsArtifact), Costs(), GainLifeFor(You, Fixed(1)), ActivationLimit(1))))), a.PlayerID())
			}
			g.PutOnBattlefield(source, a.PlayerID())
			g.ApplyContinuousEffects()
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			g.trackers.Turn.RecordCreatureDeath()
			g.RegisterDelayedTrigger(&DelayedTrigger{EventType: EvtCleanup, Controller: a.PlayerID(), Effects: []Effect{GainLifeFor(You, CreaturesDiedThisTurn())}})
			if !g.doCleanupActions() {
				t.Fatal("cleanup did not request priority")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if clone.doCleanupActions() {
				t.Fatal("final cleanup requested priority")
			}
			if clone.CreatureDeaths() != 0 || g.CreatureDeaths() != 1 {
				t.Fatal("clone reset changed original history")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
				t.Fatal("cleanup priority reset activation limit")
			}
			if err := clone.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal("final cleanup retained activation limit", err)
			}
			if g.GetPlayer(a.PlayerID()).Life() != 21 {
				t.Fatal("clone changed original life")
			}
		})
	}
}

func TestCleanupHistoryPhasedActivationReset(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewArtifact("Source", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), ActivationLimit(1))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.MutablePermanent(source.ID()).PhasedOut = true
	clone := g.Clone()
	clone.RunStepWithPriority(Cleanup)
	clone.MutablePermanentIncludingPhased(source.ID()).PhasedOut = false
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal("phased permanent retained prior turn limit", err)
	}
	g.MutablePermanentIncludingPhased(source.ID()).PhasedOut = false
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("clone reset changed original limit")
	}
}
