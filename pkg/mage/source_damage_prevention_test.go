package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceDamagePreventionIdentity(t *testing.T) {
	for _, state := range []string{"live", "depart", "return", "phase", "phase-in", "control"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "{R}", 2, 20, WithKeyword(core.Lifelink))
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			if err := ApplyEffect(g, PreventDamageByThisTurn(This), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "depart", "return":
				g.RemoveFromBattlefield(p)
				if state == "return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phase", "phase-in":
				ids := g.PhaseOut(p.ID())
				if state == "phase-in" {
					g.PhaseIn(ids)
				}
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			for _, game := range []*Game{g, g.Clone()} {
				for _, combat := range []bool{false, true} {
					game.damage.SetResolvingCombatDamage(combat)
					game.DealSimultaneousDamage(PlayerDamageAssignment(game.GetPlayer(b.PlayerID()), 2, saved), PermanentDamageAssignment(game.FindPermanent(victim.ID()), 3, saved))
				}
				if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 || game.FindPermanent(victim.ID()).Damage != 0 {
					t.Fatal("damage or lifelink escaped prevention")
				}
				refs := game.WaitingPreventionPermanentSources()
				if len(refs) != 1 || !refs[0].SameObject(saved) {
					t.Fatal("waiting reference lost identity")
				}
				if state == "return" {
					game.damage.SetResolvingCombatDamage(false)
					game.DealDamageToPlayer(game.GetPlayer(a.PlayerID()), 1, p.ID())
					if game.GetPlayer(a.PlayerID()).Life() != 19 {
						t.Fatal("returned source inherited prevention")
					}
				}
			}
			clone := g.Clone()
			before := clone.GetPlayer(b.PlayerID()).Life()
			clone.effects.ClearReplacementsEndOfTurn()
			clone.DealDamageToPlayerFromSource(clone.GetPlayer(b.PlayerID()), 1, saved)
			if clone.GetPlayer(b.PlayerID()).Life() != before-1 || len(clone.WaitingPreventionPermanentSources()) != 0 || len(g.WaitingPreventionPermanentSources()) != 1 {
				t.Fatal("clone expiry shared or missing")
			}
		})
	}
}

func TestSourceDamagePreventionDeclarations(t *testing.T) {
	target := DeclarePermanentTarget("source", IsCreature)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "valid", Card: NewInstant("Valid", "", Spell(PreventDamageByThisTurn(target), WithTargets(target)))},
		{Name: "nil", Card: NewInstant("Bad", "", Spell(PreventDamageByThisTurn(nil))), WantErr: true},
		{Name: "unbound", Card: NewInstant("Bad", "", Spell(PreventDamageByThisTurn(target))), WantErr: true},
	})
}

func TestSourceDamagePreventionDelayedLKI(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 1, 20)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	saved := g.CaptureDamageSource(p.ID())
	if err := ApplyEffect(g, AtNext(EachEndStep, PreventDamageByThisTurn(This)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, b.PlayerID())
	for range 3 {
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(saved) || len(g.effects.replacements) != 0 {
			t.Fatal("query lost source or installed rule")
		}
	}
	for _, game := range []*Game{g, g.Clone()} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.ResolveStack()
		game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 2, saved)
		if game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatal("delayed body lost absent source identity")
		}
		game.DealDamageToPlayer(game.GetPlayer(a.PlayerID()), 1, p.ID())
		if game.GetPlayer(a.PlayerID()).Life() != 19 {
			t.Fatal("delayed body followed return")
		}
	}
}

func TestSourceDamagePreventionOrderingAndUnpreventable(t *testing.T) {
	for choice := range 2 {
		a := NewBasePlayer("A")
		b := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("B"), choice: choice}
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewCreature("Source", "", 1, 20), a.PlayerID())
		victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
		if err := ApplyEffect(g, PreventDamageByThisTurn(This), source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.SetCreatureDamageRedirect(victim.ID(), b.PlayerID())
		g.DealDamageToPermanent(victim, 2, source.ID())
		if b.Life() != 20 || victim.Damage != 0 || b.prompts != 1 {
			t.Fatalf("choice=%d life=%d damage=%d prompts=%d", choice, b.Life(), victim.Damage, b.prompts)
		}
		g.MutablePermanent(victim.ID()).GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
		g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 2, source.ID())
		if g.FindPermanent(victim.ID()).Damage != 2 {
			t.Fatal("unpreventable damage prevented")
		}
		g.DealDamageToPlayer(b, 1, source.ID())
		if b.Life() != 20 || len(g.WaitingPreventionPermanentSources()) != 1 {
			t.Fatal("unpreventable event consumed rule")
		}
	}
}
