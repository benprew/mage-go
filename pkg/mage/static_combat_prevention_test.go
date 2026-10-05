package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestStaticCombatPreventionLiveHost(t *testing.T) {
	for _, change := range []string{"none", "reattach", "source-leaves", "source-phase", "source-phase-cycle", "host-return", "control", "remove-source-abilities", "remove-host-abilities", "clone"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			hostCard := NewCreature("Host", "", 2, 10)
			host := g.PutOnBattlefield(hostCard, a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 2, 10), b.PlayerID())
			aura := g.PutOnBattlefield(NewAura("Rule", "", WithAbility(Static(PreventCombatDamageFrom(AttachedPermanent), PreventCombatDamageTo(AttachedPermanent, AnyPermanent)))), a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			neutral := g.PutOnBattlefield(NewCreature("Neutral", "", 2, 10), b.PlayerID())
			saved := g.CaptureDamageSource(host.ID())
			protected := true
			switch change {
			case "reattach":
				g.Attach(aura.ID(), other.ID())
				protected = false
			case "source-leaves":
				g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
				protected = false
			case "source-phase":
				g.PhaseOut(aura.ID())
				protected = false
			case "source-phase-cycle":
				g.PhaseIn(g.PhaseOut(aura.ID()))
			case "host-return":
				g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
				host = g.PutOnBattlefield(hostCard, a.PlayerID())
				protected = false
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: other.ID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "remove-source-abilities":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), aura.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				protected = false
			case "remove-host-abilities":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), host.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "clone":
				clone := g.Clone()
				clone.RemoveFromBattlefield(clone.FindPermanent(aura.ID()))
				clone.damage.SetResolvingCombatDamage(true)
				clone.DealDamageToPlayerFromSource(clone.GetPlayer(b.PlayerID()), 1, saved)
				if clone.GetPlayer(b.PlayerID()).Life() != 19 {
					t.Fatal("clone retained departed rule")
				}
			}
			g.damage.SetResolvingCombatDamage(true)
			g.DealDamageToPlayerFromSource(b, 1, saved)
			g.DealDamageToPermanentFromSource(g.FindPermanent(host.ID()), 1, g.CaptureDamageSource(neutral.ID()))
			want := 19
			damage := 1
			if protected {
				want = 20
				damage = 0
			}
			if b.Life() != want || g.FindPermanent(host.ID()).Damage != damage {
				t.Fatalf("life=%d damage=%d", b.Life(), g.FindPermanent(host.ID()).Damage)
			}
			g.damage.SetResolvingCombatDamage(false)
			g.DealDamageToPlayerFromSource(b, 1, saved)
			if b.Life() != want-1 {
				t.Fatal("noncombat damage prevented")
			}
		})
	}
}

func TestStaticCombatPreventionFilterAndDeclarations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Protected", "", 2, 10, WithAbility(Static(PreventCombatDamageTo(This, And(IsCreature, IsEnchanted)))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	victim := g.PutOnBattlefield(card, a.PlayerID())
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 10), b.PlayerID())
	g.damage.SetResolvingCombatDamage(true)
	g.DealDamageToPermanent(victim, 1, attacker.ID())
	aura := g.PutOnBattlefield(NewAura("Aura", ""), b.PlayerID())
	g.Attach(aura.ID(), attacker.ID())
	for range 2 {
		g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, attacker.ID())
	}
	if g.FindPermanent(victim.ID()).Damage != 1 {
		t.Fatal("live enchanted filter or repeated prevention")
	}
	g.PhaseOut(aura.ID())
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, attacker.ID())
	if g.FindPermanent(victim.ID()).Damage != 2 {
		t.Fatal("phased aura matched")
	}
	for _, rule := range []StaticRule{PreventCombatDamageFrom(nil), PreventCombatDamageTo(nil, AnyPermanent), PreventCombatDamageFrom(From(BindPermanent("unbound"))), PreventCombatDamageTo(DeclarePermanentTarget("target", IsCreature), AnyPermanent)} {
		invalid := NewEnchantment("Invalid", "", WithAbility(Static(rule)))
		if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
			t.Fatal("invalid static reference accepted")
		}
	}
}

func TestStaticCombatPreventionSimultaneousAndUnpreventable(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 10, WithAbility(Static(PreventCombatDamageTo(This, AnyPermanent)))), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 10), a.PlayerID())
	source := g.PutOnBattlefield(NewCreature("Source", "", 2, 10), b.PlayerID())
	g.damage.SetResolvingCombatDamage(true)
	for range 2 {
		g.DealSimultaneousDamage(PermanentDamageAssignment(g.FindPermanent(first.ID()), 1, g.CaptureDamageSource(source.ID())), PermanentDamageAssignment(g.FindPermanent(second.ID()), 1, g.CaptureDamageSource(source.ID())), PlayerDamageAssignment(a, 1, g.CaptureDamageSource(source.ID())))
	}
	if g.FindPermanent(first.ID()).Damage != 0 || g.FindPermanent(second.ID()).Damage != 2 || a.Life() != 18 {
		t.Fatal("simultaneous packet scope or repeat")
	}
	g.MutablePermanent(first.ID()).GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
	g.DealDamageToPermanent(g.FindPermanent(first.ID()), 1, source.ID())
	if g.FindPermanent(first.ID()).Damage != 1 {
		t.Fatal("unpreventable damage prevented")
	}
}

func TestStaticCombatPreventionEnchantedSourceInformation(t *testing.T) {
	for _, change := range []string{"depart", "return", "phase", "phase-cycle", "aura-leaves-live"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			victim := g.PutOnBattlefield(NewCreature("Protected", "", 2, 10, WithAbility(Static(PreventCombatDamageTo(This, And(IsCreature, DamageSourceIsEnchanted()))))), a.PlayerID())
			card := NewCreature("Source", "", 2, 10)
			source := g.PutOnBattlefield(card, b.PlayerID())
			aura := g.PutOnBattlefield(NewAura("Aura", ""), b.PlayerID())
			g.Attach(aura.ID(), source.ID())
			saved := g.CaptureDamageSource(source.ID())
			switch change {
			case "depart", "return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
				if change == "return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "aura-leaves-live":
				g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
			}
			clone := g.Clone()
			for _, game := range []*Game{g, clone} {
				game.damage.SetResolvingCombatDamage(true)
				game.DealDamageToPermanentFromSource(game.FindPermanent(victim.ID()), 1, saved)
				want := 0
				if change == "aura-leaves-live" {
					want = 1
				}
				if game.FindPermanent(victim.ID()).Damage != want {
					t.Fatal("source enchanted information lost")
				}
				if change == "return" {
					game.DealDamageToPermanent(game.FindPermanent(victim.ID()), 1, source.ID())
					if game.FindPermanent(victim.ID()).Damage != 1 {
						t.Fatal("returned source inherited old enchanted state")
					}
				}
			}
		})
	}
}
