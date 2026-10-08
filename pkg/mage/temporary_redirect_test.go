package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTemporaryRedirectIdentity(t *testing.T) {
	for _, state := range []string{"live", "attacker-depart", "attacker-return", "attacker-phase", "absorber-return", "absorber-phase", "phase-cycle", "control", "noncreature", "cleanup"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewCreature("Source", "{R}", 1, 20), b.PlayerID())
			absorber := g.PutOnBattlefield(NewCreature("Absorber", "{B}", 1, 20), a.PlayerID())
			saved := g.CaptureDamageSource(source.ID())
			target := g.CaptureDamageSource(absorber.ID())
			binding := BindPermanent("source")
			g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{binding: {source.ID(), source.incarnationID}}
			if err := RedirectDamageFromPermanentThisTurn(From(binding), You, This).Apply(&EffectContext{Game: g, SourceID: absorber.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: absorber.incarnationID}); err != nil {
				t.Fatal(err)
			}
			baseline := g.Clone()
			switch state {
			case "attacker-depart", "attacker-return":
				g.RemoveFromBattlefield(source)
				if state == "attacker-return" {
					g.PutOnBattlefield(source.Card, b.PlayerID())
				}
				g.zones.ClearLKI()
			case "attacker-phase":
				g.PhaseOut(source.ID())
			case "absorber-return":
				g.RemoveFromBattlefield(absorber)
				g.PutOnBattlefield(absorber.Card, a.PlayerID())
			case "absorber-phase":
				g.PhaseOut(absorber.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(absorber.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: absorber.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "noncreature":
				g.AddContinuousEffect(FuncContinuousEffect(core.LayerType, core.EndOfTurn, func(game *Game, _ uuid.UUID) error { game.RevokeAttr(absorber.ID(), core.AttrIsCreature); return nil }))
				g.ApplyContinuousEffects()
			case "cleanup":
				g.effects.ClearReplacementsEndOfTurn()
			}
			for _, game := range []*Game{g, g.Clone()} {
				refs := game.WaitingRedirectionPermanentSources()
				if state == "cleanup" {
					if len(refs) != 0 {
						t.Fatal("cleanup kept references")
					}
				} else if len(refs) != 2 || !refs[0].SameObject(saved) || !refs[1].SameObject(target) {
					t.Fatal("lost exact waiting identities")
				}
				if state == "attacker-return" {
					game.DealDamageToPlayer(game.GetPlayer(a.PlayerID()), 1, source.ID())
					if game.GetPlayer(a.PlayerID()).Life() != 19 {
						t.Fatal("returned source inherited rule")
					}
				}
				life := game.GetPlayer(a.PlayerID()).Life()
				game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 2, saved)
				canRedirect := state != "absorber-return" && state != "absorber-phase" && state != "noncreature" && state != "cleanup"
				if canRedirect {
					if game.GetPlayer(a.PlayerID()).Life() != life || game.FindPermanent(absorber.ID()).Damage != 2 {
						t.Fatal("exact damage was not redirected")
					}
					game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 1, saved)
					if game.FindPermanent(absorber.ID()).Damage != 3 {
						t.Fatal("unlimited rule consumed")
					}
				} else if game.GetPlayer(a.PlayerID()).Life() != life-2 {
					t.Fatal("invalid destination redirected")
				}
				game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 1, saved)
				if game.GetPlayer(b.PlayerID()).Life() != 19 {
					t.Fatal("wrong player redirected")
				}
			}
			if baseline.FindPermanent(absorber.ID()).Damage != 0 || len(baseline.WaitingRedirectionPermanentSources()) != 2 {
				t.Fatal("clone shared mutation")
			}
		})
	}
}

func TestTemporaryRedirectBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "{R}", 1, 20), a.PlayerID())
	if err := ApplyEffect(g, RedirectDamageFromPermanentThisTurn(This, You, This), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	rule := g.effects.replacements[0]
	for _, amount := range []int{0, -1} {
		if rule.Matches(NewDamageToPlayerAction(p.ID(), a.PlayerID(), amount, false).withDamageSource(g.CaptureDamageSource(p.ID())), g) {
			t.Fatal("nonpositive damage matches")
		}
	}
	g.effects.cycleReplacements = append(g.effects.cycleReplacements, rule.Clone())
	if len(g.WaitingRedirectionPermanentSources()) != 1 {
		t.Fatal("same source and destination not deduplicated")
	}
	for _, effect := range []Effect{RedirectDamageFromPermanentThisTurn(nil, You, This), RedirectDamageFromPermanentThisTurn(This, nil, This), RedirectDamageFromPermanentThisTurn(This, You, nil)} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("missing reference accepted")
		}
	}
	if err := ApplyEffect(g, RedirectDamageFromPermanentThisTurn(This, You, This), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.effects.replacements) != 1 {
		t.Fatal("absent source installed rule")
	}
}

func TestTemporaryRedirectDamageBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "{R}", 1, 20, WithKeyword(core.Lifelink)), b.PlayerID())
	absorber := g.PutOnBattlefield(NewCreature("Absorber", "{B}", 1, 20), a.PlayerID())
	binding := BindPermanent("source")
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{binding: {source.ID(), source.incarnationID}}
	effect := RedirectDamageFromPermanentThisTurn(From(binding), You, This)
	if err := effect.Apply(&EffectContext{Game: g, SourceID: absorber.ID(), Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	captured := g.CaptureDamageSource(source.ID())
	g.AddPreventionShield(absorber.ID(), 1)
	outcomes := g.dealSimultaneousDamageOutcomes(PlayerDamageAssignment(a, 2, captured), PlayerDamageAssignment(a, 3, captured))
	total := 0
	for _, outcome := range outcomes {
		if outcome.playerID != uuid.Nil || outcome.permanent.id != absorber.ID() || !outcome.source.SameObject(captured) {
			t.Fatal("redirection lost packet identity")
		}
		total += outcome.amount
	}
	if total != 4 || absorber.Damage != 4 || a.Life() != 20 || b.Life() != 24 {
		t.Fatal("replacement recheck, simultaneous damage or lifelink failed")
	}
	action := NewDamageToPlayerAction(source.ID(), a.PlayerID(), 2, true).withDamageSource(captured)
	redirected := g.effects.replacements[0].Replace(action, g).(*DamageToCreatureAction)
	if !redirected.IsCombatDamage() || !redirected.DamageSource().SameObject(captured) || redirected.Amount() != 2 {
		t.Fatal("redirection changed source, amount or combat flag")
	}
	g.PhaseOut(absorber.ID())
	if g.effects.replacements[0].Replace(action, g) != action {
		t.Fatal("replacement did not recheck destination")
	}
}

func TestTemporaryRedirectPlaneswalker(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 20), b.PlayerID())
	destination := g.PutOnBattlefield(NewPlaneswalker("Destination", "", 5), a.PlayerID())
	binding := BindPermanent("source")
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{binding: {source.ID(), source.incarnationID}}
	if err := RedirectDamageFromPermanentThisTurn(From(binding), You, This).Apply(&EffectContext{Game: g, SourceID: destination.ID(), Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPlayer(a, 2, source.ID())
	if a.Life() != 20 || destination.Counters[core.Loyalty] != 3 {
		t.Fatal("planeswalker did not receive redirected damage")
	}
}
