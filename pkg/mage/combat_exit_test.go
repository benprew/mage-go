package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombatExitSoleBlockerHistory(t *testing.T) {
	for _, scenario := range []string{"sole", "two", "other-removed", "same-card-returned", "clone", "reset"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 8), a.PlayerID())
			blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 1, 8), b.PlayerID())
			g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
			if scenario == "same-card-returned" {
				g.combat.AddBlockerIncarnation(blocker.ID(), blocker.incarnationID, attacker.ID())
				card := blocker.Card
				g.RemoveFromBattlefield(blocker)
				blocker = g.PutOnBattlefield(card, b.PlayerID())
			}
			g.combat.AddBlockerIncarnation(blocker.ID(), blocker.incarnationID, attacker.ID())
			if scenario == "two" || scenario == "other-removed" || scenario == "reset" {
				g.combat.AddBlockerIncarnation(other.ID(), other.incarnationID, attacker.ID())
				if scenario != "two" {
					g.RemoveFromCombat(other.ID())
				}
			}
			if scenario == "reset" {
				g.combat.Reset()
				g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
				g.combat.AddBlockerIncarnation(blocker.ID(), blocker.incarnationID, attacker.ID())
			}
			original := g
			if scenario == "clone" {
				g = g.Clone()
			}
			if err := ApplyEffect(g, RemovePermanentFromCombatAndUnblock(This), blocker.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			want := scenario == "sole" || scenario == "clone" || scenario == "reset"
			if g.combat.GroupFor(attacker.ID()).Blocked == want || g.combat.IsBlocking(blocker.ID()) {
				t.Fatal("wrong history or combat removal")
			}
			if scenario == "clone" && (!original.combat.IsBlocking(blocker.ID()) || !original.combat.GroupFor(attacker.ID()).Blocked) {
				t.Fatal("clone changed original combat")
			}
			if scenario != "two" {
				g.ExecuteCombatDamage()
				life := 20
				if want {
					life = 18
				}
				if g.GetPlayer(b.PlayerID()).Life() != life {
					t.Fatal("wrong unblocked combat damage")
				}
			}
		})
	}
}

func TestCantBlockDurationAndIdentity(t *testing.T) {
	for _, change := range []string{"none", "ability-removal", "phase-cycle", "return", "clone", "cleanup", "already-blocking"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 1, 8), a.PlayerID())
			blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
			if change == "already-blocking" {
				g.combat.AddAttacker(attacker.ID(), b.PlayerID())
				g.combat.AddBlocker(blocker.ID(), attacker.ID())
			}
			if err := ApplyEffect(g, Until(core.EndOfTurn, CantBlock(This)), blocker.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			original := g
			switch change {
			case "ability-removal":
				g.AddContinuousEffect(RemoveAllAbilities(blocker.ID(), core.Indefinite))
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(blocker.ID()))
			case "return":
				card := blocker.Card
				g.RemoveFromBattlefield(blocker)
				blocker = g.PutOnBattlefield(card, b.PlayerID())
			case "clone":
				g = g.Clone()
				g.effects.RemoveEndOfTurn()
			case "cleanup":
				g.RunStepWithPriority(core.Cleanup)
			}
			g.ApplyContinuousEffects()
			want := change == "return" || change == "clone" || change == "cleanup"
			if CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(attacker.ID()), g) != want {
				t.Fatal("wrong restriction lifetime")
			}
			if change == "clone" && CanBlock(original.FindPermanent(blocker.ID()), original.FindPermanent(attacker.ID()), original) {
				t.Fatal("clone changed original rule")
			}
			if change == "already-blocking" && !g.combat.IsBlocking(blocker.ID()) {
				t.Fatal("restriction removed legal block")
			}
		})
	}
}

func TestCombatExitDeclarations(t *testing.T) {
	for _, ref := range []PermanentRef{nil, From(BindPermanent("missing")), DeclarePermanentTarget("missing", IsCreature)} {
		for _, effect := range []ResolvingEffect{RemovePermanentFromCombatAndUnblock(ref), Until(core.EndOfTurn, CantBlock(ref))} {
			card := NewInstant("Invalid", "", WithAbility(Spell(effect)))
			if ValidateCardDefinition(card.Name(), card) == nil {
				t.Fatal("accepted invalid reference")
			}
		}
	}
	ref := DeclarePermanentTarget("target", IsCreature)
	card := NewInstant("Valid", "", Spell(Sequence(RemovePermanentFromCombatAndUnblock(ref), Until(core.EndOfTurn, CantBlock(ref))), WithTargets(ref)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestCombatExitDepartureKeepsHistory(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 8), a.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 8), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 8), b.PlayerID())
	g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
	g.combat.AddBlockerIncarnation(first.ID(), first.incarnationID, attacker.ID())
	g.combat.AddBlockerIncarnation(second.ID(), second.incarnationID, attacker.ID())
	g.BouncePermanentToHand(first)
	if g.combat.IsBlocking(first.ID()) {
		t.Fatal("departed object still blocks")
	}
	if err := ApplyEffect(g, RemovePermanentFromCombatAndUnblock(This), second.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.combat.GroupFor(attacker.ID()).Blocked {
		t.Fatal("departure erased blocker history")
	}
	g.BouncePermanentToHand(attacker)
	if g.combat.IsAttacking(attacker.ID()) {
		t.Fatal("departed object still attacks")
	}
	if lki := g.LKI(attacker.ID()); lki == nil || !lki.Attacking {
		t.Fatal("combat removal erased departure information")
	}
}

func TestCombatExitMultipleGroupsAndCloneHistory(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 8), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 8), a.PlayerID())
	blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 8), b.PlayerID())
	for _, attacker := range []*Permanent{first, second} {
		g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
		g.combat.AddBlockerIncarnation(blocker.ID(), blocker.incarnationID, attacker.ID())
	}
	clone := g.Clone()
	clone.combat.AddBlockerIncarnation(other.ID(), other.incarnationID, second.ID())
	clone.RemoveFromCombat(other.ID())
	if err := ApplyEffect(clone, RemovePermanentFromCombatAndUnblock(This), blocker.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if clone.combat.GroupFor(first.ID()).Blocked || !clone.combat.GroupFor(second.ID()).Blocked {
		t.Fatal("wrong per-attacker history")
	}
	if err := ApplyEffect(g, RemovePermanentFromCombatAndUnblock(This), blocker.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.combat.GroupFor(first.ID()).Blocked || g.combat.GroupFor(second.ID()).Blocked {
		t.Fatal("clone changed original history")
	}
}

func TestCombatExitDepartedOpponentCompatibility(t *testing.T) {
	for _, role := range []string{"attacker", "blocker"} {
		for _, change := range []string{"none", "source-return", "opponent-return", "opponent-phase", "clone"} {
			t.Run(role+"/"+change, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				source := g.PutOnBattlefield(NewCreature("Source", "", 1, 8), a.PlayerID())
				opponent := g.PutOnBattlefield(NewCreature("Opponent", "", 1, 8), b.PlayerID())
				saved := source.incarnationID
				if role == "attacker" {
					g.combat.AddAttackerIncarnation(source.ID(), source.incarnationID, b.PlayerID())
					g.combat.AddBlockerIncarnation(opponent.ID(), opponent.incarnationID, source.ID())
				} else {
					g.combat.AddAttackerIncarnation(opponent.ID(), opponent.incarnationID, a.PlayerID())
					g.combat.AddBlockerIncarnation(source.ID(), source.incarnationID, opponent.ID())
				}
				g.BouncePermanentToHand(source)
				switch change {
				case "source-return":
					g.PutOnBattlefield(source.Card, a.PlayerID())
				case "opponent-return":
					g.BouncePermanentToHand(opponent)
					opponent = g.PutOnBattlefield(opponent.Card, b.PlayerID())
				case "opponent-phase":
					g.PhaseOut(opponent.ID())
				case "clone":
					g = g.Clone()
				}
				ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: saved}
				if err := ForEachCombatOpponent(Tap(), "tap prior opponents").Apply(ctx); err != nil {
					t.Fatal(err)
				}
				want := change != "opponent-return" && change != "opponent-phase"
				if g.FindPermanentIncludingPhased(opponent.ID()).Tapped != want {
					t.Fatal("wrong departed-source opponents or identity")
				}
			})
		}
	}
}
