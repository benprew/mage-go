package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestCombatRemovalKeepsAttackerBlocked(t *testing.T) {
	for _, kind := range []string{"sole blocker", "one of two blockers", "unrelated creature"} {
		t.Run(kind, func(t *testing.T) {
			combat := NewCombat()
			attacker, blocker, other := uuid.New(), uuid.New(), uuid.New()
			combat.AddAttacker(attacker, uuid.New())
			combat.AddBlocker(blocker, attacker)
			removed := blocker
			if kind == "one of two blockers" {
				combat.AddBlocker(other, attacker)
			}
			if kind == "unrelated creature" {
				removed = other
			}
			combat.RemoveFromCombat(removed)
			if !combat.GroupFor(attacker).Blocked {
				t.Fatal("removing a creature made the attacker unblocked")
			}
			if combat.IsBlocking(removed) {
				t.Fatal("removed creature still blocks")
			}
		})
	}
}

func TestTypedCombatRemovalLifetime(t *testing.T) {
	for _, role := range []string{"attacker", "blocker"} {
		for _, change := range []string{"none", "return", "absent", "phase", "phase cycle"} {
			t.Run(role+"/"+change, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Subject", "", 2, 2)
				p := g.PutOnBattlefield(card, a.PlayerID())
				saved := BindPermanent("saved")
				ctx := &EffectContext{Game: g, SourceID: card.ID(), Controller: a.PlayerID()}
				if err := Snapshot(This, saved).Apply(ctx); err != nil {
					t.Fatal(err)
				}
				switch change {
				case "return":
					g.RemoveFromBattlefield(p)
					g.PutOnBattlefield(card, a.PlayerID())
				case "absent":
					g.RemoveFromBattlefield(p)
				case "phase":
					g.PhaseOut(card.ID())
				case "phase cycle":
					g.PhaseIn(g.PhaseOut(card.ID()))
				}
				attacker := card.ID()
				if role == "blocker" {
					attacker = uuid.New()
				}
				g.combat.AddAttacker(attacker, b.PlayerID())
				if role == "blocker" {
					g.combat.AddBlocker(card.ID(), attacker)
				}
				clone := g.Clone()
				ctx.Game = clone
				if err := RemovePermanentFromCombat(From(saved)).Apply(ctx); err != nil {
					t.Fatal(err)
				}
				wantPresent := change != "none" && change != "phase cycle"
				present := clone.combat.IsAttacking(card.ID()) || clone.combat.IsBlocking(card.ID())
				if present != wantPresent {
					t.Fatalf("combat membership = %v, want %v", present, wantPresent)
				}
				if !g.combat.IsAttacking(card.ID()) && !g.combat.IsBlocking(card.ID()) {
					t.Fatal("clone changed original combat")
				}
				if role == "blocker" && !clone.combat.GroupFor(attacker).Blocked {
					t.Fatal("attacker became unblocked")
				}
			})
		}
	}
}

func TestTypedCombatRemovalValidation(t *testing.T) {
	saved := BindPermanent("saved")
	for _, subject := range []PermanentRef{nil, From(saved), DeclarePermanentTarget("undeclared", IsCreature)} {
		card := NewInstant("Invalid", "", WithAbility(Spell(RemovePermanentFromCombat(subject))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid subject")
		}
	}
}
