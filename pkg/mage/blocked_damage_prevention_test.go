package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestBlockedDamagePrevention(t *testing.T) {
	for _, change := range []string{"live", "tap", "attacker-control", "blocker-control", "attacker-depart", "attacker-phase", "unrelated", "reverse", "remove-attacker", "remove-blocker", "attacker-return", "blocker-return", "attacker-phase-cycle", "blocker-phase-cycle", "lose-creature", "ability-removal", "unpreventable", "clone"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Blocker", "", 0, 10, WithAbility(Static(PreventDamageFromBlockedCreatures(This))))
			wall := g.PutOnBattlefield(card, a.PlayerID())
			attackerCard := NewCreature("Attacker", "", 2, 10)
			attacker := g.PutOnBattlefield(attackerCard, b.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 2, 10), a.PlayerID())
			g.GetCombat().AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, a.PlayerID())
			g.GetCombat().AddBlockerIncarnation(wall.ID(), wall.incarnationID, attacker.ID())
			saved := g.CaptureDamageSource(attacker.ID())
			want := 1
			switch change {
			case "live":
				want = 0
			case "tap":
				g.TapPermanent(g.FindPermanent(wall.ID()))
				want = 0
			case "attacker-control":
				g.AddControlEffect(ControlEffectSpec{SourceID: other.ID(), TargetID: attacker.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "blocker-control":
				g.AddControlEffect(ControlEffectSpec{SourceID: other.ID(), TargetID: wall.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "attacker-depart":
				g.RemoveFromBattlefield(g.FindPermanent(attacker.ID()))
			case "attacker-phase":
				g.PhaseOut(attacker.ID())
			case "unrelated":
				saved = g.CaptureDamageSource(other.ID())
			case "reverse":
				g.GetCombat().Reset()
				g.GetCombat().AddAttackerIncarnation(wall.ID(), wall.incarnationID, b.PlayerID())
				g.GetCombat().AddBlockerIncarnation(attacker.ID(), attacker.incarnationID, wall.ID())
			case "remove-attacker":
				g.GetCombat().RemoveFromCombat(attacker.ID())
			case "remove-blocker":
				g.GetCombat().RemoveFromCombat(wall.ID())
			case "attacker-return":
				g.RemoveFromBattlefield(g.FindPermanent(attacker.ID()))
				attacker = g.PutOnBattlefield(attackerCard, b.PlayerID())
				g.GetCombat().AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, a.PlayerID())
				g.GetCombat().AddBlockerIncarnation(wall.ID(), wall.incarnationID, attacker.ID())
			case "blocker-return":
				g.RemoveFromBattlefield(g.FindPermanent(wall.ID()))
				wall = g.PutOnBattlefield(card, a.PlayerID())
			case "attacker-phase-cycle":
				g.PhaseIn(g.PhaseOut(attacker.ID()))
			case "blocker-phase-cycle":
				g.PhaseIn(g.PhaseOut(wall.ID()))
			case "lose-creature":
				g.MutablePermanent(attacker.ID()).RevokeBaseAttr(core.AttrIsCreature)
			case "ability-removal":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), wall.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "unpreventable":
				g.MutablePermanent(wall.ID()).GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
			case "clone":
				clone := g.Clone()
				clone.GetCombat().RemoveFromCombat(attacker.ID())
				clone.DealDamageToPermanentFromSource(clone.FindPermanent(wall.ID()), 1, saved)
				if clone.FindPermanent(wall.ID()).Damage != 1 || g.FindPermanent(wall.ID()).Damage != 0 {
					t.Fatal("clone changed pair or damage")
				}
				want = 0
			}
			for _, combat := range []bool{false, true} {
				g.damage.SetResolvingCombatDamage(combat)
				g.DealSimultaneousDamage(PermanentDamageAssignment(g.FindPermanent(wall.ID()), 1, saved), PermanentDamageAssignment(g.FindPermanent(other.ID()), 1, saved), PlayerDamageAssignment(a, 1, saved))
			}
			if g.FindPermanent(wall.ID()).Damage != 2*want || g.FindPermanent(other.ID()).Damage != 2 || a.Life() != 18 {
				t.Fatalf("damage=%d want=%d", g.FindPermanent(wall.ID()).Damage, 2*want)
			}
		})
	}
}

func TestBlockedDamagePreventionDeclarations(t *testing.T) {
	for _, subject := range []PermanentRef{nil, From(BindPermanent("unbound")), DeclarePermanentTarget("target", IsCreature)} {
		card := NewCreature("Invalid", "", 0, 1, WithAbility(Static(PreventDamageFromBlockedCreatures(subject))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid subject accepted")
		}
	}
}

func TestBlockedDamagePreventionRecheck(t *testing.T) {
	for _, change := range []string{"remove-pair", "add-pair", "redirect"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			wall := g.PutOnBattlefield(NewCreature("Blocker", "", 0, 10, WithAbility(Static(PreventDamageFromBlockedCreatures(This)))), a.PlayerID())
			attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 10), b.PlayerID())
			g.GetCombat().AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, a.PlayerID())
			if change != "add-pair" {
				g.GetCombat().AddBlockerIncarnation(wall.ID(), wall.incarnationID, attacker.ID())
			}
			g.AddReplacementEffect(&testPrioritizedReplacement{
				replacementBase: replacementBase{duration: core.Indefinite, priority: ReplacementPrioritySelf},
				matchFunc:       func(action Action) bool { _, ok := action.(*DamageToCreatureAction); return ok },
				replaceFunc: func(action Action, game *Game) Action {
					switch change {
					case "remove-pair":
						game.GetCombat().RemoveFromCombat(attacker.ID())
					case "add-pair":
						game.GetCombat().AddBlockerIncarnation(wall.ID(), wall.incarnationID, attacker.ID())
					case "redirect":
						damage := action.(*DamageToCreatureAction)
						return NewDamageToPlayerAction(damage.ActionSource(), a.PlayerID(), damage.Amount(), damage.IsCombatDamage()).withDamageSource(damage.DamageSource())
					}
					return action
				},
			})
			g.DealDamageToPermanent(g.FindPermanent(wall.ID()), 1, attacker.ID())
			want := 0
			if change == "remove-pair" {
				want = 1
			}
			if g.FindPermanent(wall.ID()).Damage != want {
				t.Fatal("pair was not rechecked")
			}
			life := 20
			if change == "redirect" {
				life = 19
			}
			if a.Life() != life {
				t.Fatal("recipient was not rechecked")
			}
		})
	}
}

func TestBlockedDamagePreventionHost(t *testing.T) {
	for _, change := range []string{"live", "reattach", "remove-source", "phase-source", "phase-cycle-source", "remove-host-abilities", "remove-source-abilities"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			host := g.PutOnBattlefield(NewCreature("Host", "", 0, 10), a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 0, 10), a.PlayerID())
			aura := g.PutOnBattlefield(NewAura("Rule", "", WithAbility(Static(PreventDamageFromBlockedCreatures(AttachedPermanent)))), a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 10), b.PlayerID())
			g.GetCombat().AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, a.PlayerID())
			g.GetCombat().AddBlockerIncarnation(host.ID(), host.incarnationID, attacker.ID())
			active := true
			switch change {
			case "reattach":
				g.Attach(aura.ID(), other.ID())
				active = false
			case "remove-source":
				g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
				active = false
			case "phase-source":
				g.PhaseOut(aura.ID())
				active = false
			case "phase-cycle-source":
				g.PhaseIn(g.PhaseOut(aura.ID()))
			case "remove-host-abilities":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), host.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "remove-source-abilities":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), aura.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				active = false
			}
			g.DealDamageToPermanent(g.FindPermanent(host.ID()), 1, attacker.ID())
			want := 1
			if active {
				want = 0
			}
			if g.FindPermanent(host.ID()).Damage != want {
				t.Fatal("live attachment or source lifecycle")
			}
		})
	}
}
