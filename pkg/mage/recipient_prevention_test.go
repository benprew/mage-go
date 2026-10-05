package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPreventDamageThisTurnIdentity(t *testing.T) {
	for _, state := range []string{"live", "control", "phase", "return", "absent-at-resolution", "phased-at-resolution"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Recipient", "", 1, 20)
			p := g.PutOnBattlefield(card, a.PlayerID())
			if state == "absent-at-resolution" {
				g.RemoveFromBattlefield(p)
			}
			if state == "phased-at-resolution" {
				g.PhaseOut(p.ID())
			}
			if err := ApplyEffect(g, PreventDamageThisTurn(This, PermanentFilter{}), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "phase", "phased-at-resolution":
				if state == "phase" {
					g.PhaseOut(p.ID())
				}
				clone := g.Clone()
				clone.PhaseIn([]uuid.UUID{p.ID()})
				clone.DealDamageToPermanent(clone.FindPermanent(p.ID()), 2, uuid.New())
				want := 0
				if state == "phased-at-resolution" {
					want = 2
				}
				if clone.FindPermanent(p.ID()).Damage != want || g.FindPermanent(p.ID()) != nil {
					t.Fatal("phasing or clone identity")
				}
				g.PhaseIn([]uuid.UUID{p.ID()})
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, a.PlayerID())
			case "absent-at-resolution":
				g.PutOnBattlefield(card, a.PlayerID())
			}
			p = g.FindPermanent(p.ID())
			g.DealDamageToPermanent(p, 2, uuid.New())
			g.DealDamageToPermanent(p, 3, uuid.New())
			want := 0
			if state == "return" || state == "absent-at-resolution" || state == "phased-at-resolution" {
				want = 5
			}
			if p.Damage != want {
				t.Fatalf("damage=%d want=%d", p.Damage, want)
			}
			g.DealDamageToPlayer(a, 1, uuid.New())
			if a.Life() != 19 {
				t.Fatal("permanent prevention protected player")
			}
			clone := g.Clone()
			g.GetEffects().ClearReplacementsEndOfTurn()
			g.DealDamageToPermanent(p, 1, uuid.New())
			clone.DealDamageToPermanent(clone.FindPermanent(p.ID()), 1, uuid.New())
			if g.FindPermanent(p.ID()).Damage != want+1 || (want == 0 && clone.FindPermanent(p.ID()).Damage != 0) {
				t.Fatalf("expiry or clone isolation: original=%d clone=%d want=%d", g.FindPermanent(p.ID()).Damage, clone.FindPermanent(p.ID()).Damage, want)
			}
		})
	}
}

func TestPreventDamageThisTurnSourceAndBatch(t *testing.T) {
	for _, state := range []string{"live", "lost-flying", "departed", "returned", "phased"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Flying source", "", 1, 10, WithKeyword(Flying))
			p := g.PutOnBattlefield(card, b.PlayerID())
			source := g.CaptureDamageSource(p.ID())
			victim := g.PutOnBattlefield(NewCreature("Other recipient", "", 1, 10), a.PlayerID())
			if err := ApplyEffect(g, PreventDamageThisTurn(You, And(IsCreature, HasKeywordFilter(Flying))), victim.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "lost-flying":
				p.RevokeBaseAttr(Flying)
			case "departed", "returned":
				g.RemoveFromBattlefield(p)
				if state == "returned" {
					g.PutOnBattlefield(card, a.PlayerID()).RevokeBaseAttr(Flying)
				}
			case "phased":
				g.PhaseOut(p.ID())
			}
			g.DealSimultaneousDamage(PlayerDamageAssignment(a, 2, source), PlayerDamageAssignment(a, 3, source), PlayerDamageAssignment(b, 1, source), PermanentDamageAssignment(victim, 1, source))
			want := 20
			if state == "lost-flying" {
				want = 15
			}
			if a.Life() != want || b.Life() != 19 || victim.Damage != 1 {
				t.Fatalf("scope or source filter: life=%d/%d damage=%d", a.Life(), b.Life(), victim.Damage)
			}
			g.DealDamageToPlayerFromSource(a, 2, source)
			if state != "lost-flying" && a.Life() != 20 {
				t.Fatal("unlimited prevention was consumed")
			}
		})
	}
}

func TestPreventDamageThisTurnDeclarations(t *testing.T) {
	invalid := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), PreventDamageThisTurn(nil, IsCreature))))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
		t.Fatal("nil recipient accepted")
	}
	valid := NewArtifact("Valid", "", WithAbility(Activate(Costs(), PreventDamageThisTurn(You, PermanentFilter{}))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
}

func TestPreventDamageThisTurnAttackingSourceLKI(t *testing.T) {
	for _, state := range []string{"live", "removed", "departed", "returned", "phased", "phase-cycle"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Attacker", "", 1, 20)
			p := g.PutOnBattlefield(card, b.PlayerID())
			g.GetCombat().AddAttackerIncarnation(p.ID(), p.incarnationID, a.PlayerID())
			source := g.CaptureDamageSource(p.ID())
			if err := ApplyEffect(g, PreventDamageThisTurn(You, And(IsCreature, DamageSourceIsAttacking())), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "removed":
				g.RemoveFromCombat(p.ID())
			case "departed", "returned":
				g.RemoveFromBattlefield(p)
				if state == "returned" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
				g.zones.ClearLKI()
			case "phased":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			clone := g.Clone()
			g.DealDamageToPlayerFromSource(a, 2, source)
			clone.DealDamageToPlayerFromSource(clone.GetPlayer(a.PlayerID()), 2, source)
			want := 20
			if state == "removed" || state == "phase-cycle" {
				want = 18
			}
			if a.Life() != want || clone.GetPlayer(a.PlayerID()).Life() != want {
				t.Fatalf("life=%d want=%d", a.Life(), want)
			}
		})
	}
}

func TestPreventDamageThisTurnReplacementOrder(t *testing.T) {
	for choice := range 2 {
		a := NewBasePlayer("A")
		b := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("B"), choice: choice}
		g := NewGame(a, b)
		victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
		if err := ApplyEffect(g, PreventDamageThisTurn(This, PermanentFilter{}), victim.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.SetCreatureDamageRedirect(victim.ID(), b.PlayerID())
		g.DealDamageToPermanent(victim, 2, uuid.New())
		want := 20
		if choice == 1 {
			want = 18
		}
		if b.Life() != want || g.FindPermanent(victim.ID()).Damage != 0 || b.prompts != 1 {
			t.Fatalf("choice=%d life=%d prompts=%d", choice, b.Life(), b.prompts)
		}
		g.MutablePermanent(victim.ID()).GrantBaseAttr(AttrDamageCantBePreventedOrRedirected)
		g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 2, uuid.New())
		if g.FindPermanent(victim.ID()).Damage != 2 {
			t.Fatal("unpreventable damage was prevented")
		}
	}
}
