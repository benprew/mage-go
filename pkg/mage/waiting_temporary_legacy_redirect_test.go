package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func temporaryLegacyRedirectFixture(kind string) (*Game, *BasePlayer, *BasePlayer, *Permanent, *Permanent, ReplacementEffect) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "{R}", 1, 20), b.PlayerID())
	destination := g.PutOnBattlefield(NewCreature("Destination", "{W}", 1, 20), a.PlayerID())
	if kind == "creature" {
		g.SetCreatureDamageRedirect(destination.ID(), b.PlayerID())
	} else {
		g.SetAttackerDamageRedirect(source.ID(), destination.ID())
	}
	return g, a, b, source, destination, g.effects.replacements[len(g.effects.replacements)-1]
}

func TestWaitingTemporaryLegacyRedirectReferences(t *testing.T) {
	for _, kind := range []string{"creature", "attacker"} {
		for _, state := range []string{"live", "phase", "phase-cycle", "depart", "return", "cleanup"} {
			t.Run(kind+"/"+state, func(t *testing.T) {
				g, _, b, source, destination, _ := temporaryLegacyRedirectFixture(kind)
				saved := g.CaptureDamageSource(destination.ID())
				savedSource := g.CaptureDamageSource(source.ID())
				baseline := g.Clone()
				switch state {
				case "phase":
					g.PhaseOut(destination.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(destination.ID()))
				case "depart", "return":
					g.RemoveFromBattlefield(destination)
					if state == "return" {
						g.PutOnBattlefield(destination.Card, b.PlayerID())
					}
					g.zones.ClearLKI()
				case "cleanup":
					g.effects.ClearReplacementsEndOfTurn()
				}
				want := 1
				if kind == "attacker" {
					want = 2
				}
				for _, game := range []*Game{g, g.Clone()} {
					refs := game.WaitingRedirectionPermanentSources()
					if state == "cleanup" {
						if len(refs) != 0 {
							t.Fatal("cleanup kept references")
						}
						continue
					}
					if len(refs) != want {
						t.Fatalf("references=%d want=%d", len(refs), want)
					}
					foundDestination, foundSource := false, false
					for _, ref := range refs {
						foundDestination = foundDestination || ref.SameObject(saved)
						foundSource = foundSource || ref.SameObject(savedSource)
					}
					if !foundDestination || (kind == "attacker" && !foundSource) {
						t.Fatal("saved incarnation missing")
					}
					refs[0].card.SetOwner(uuid.New())
					if game.WaitingRedirectionPermanentSources()[0].card.Owner() == refs[0].card.Owner() {
						t.Fatal("query shared card state")
					}
				}
				if len(baseline.WaitingRedirectionPermanentSources()) != want {
					t.Fatal("mutation changed clone")
				}
			})
		}
	}
}

func TestWaitingTemporaryLegacyRedirectRechecks(t *testing.T) {
	for _, kind := range []string{"creature", "attacker"} {
		for _, state := range []string{"live", "phase", "return", "type", "zero", "wrong-kind", "consumed"} {
			t.Run(kind+"/"+state, func(t *testing.T) {
				g, a, b, source, destination, rule := temporaryLegacyRedirectFixture(kind)
				saved := g.CaptureDamageSource(source.ID())
				var action Action = NewDamageToCreatureAction(source.ID(), destination.ID(), 2, true).withDamageSource(saved)
				if kind == "attacker" {
					action = NewDamageToPlayerAction(source.ID(), a.PlayerID(), 2, true).withDamageSource(saved)
				}
				valid := state == "live"
				switch state {
				case "phase":
					g.PhaseOut(destination.ID())
				case "return":
					g.RemoveFromBattlefield(destination)
					g.PutOnBattlefield(destination.Card, b.PlayerID())
				case "type":
					g.AddContinuousEffect(FuncContinuousEffect(core.LayerType, core.EndOfTurn, func(game *Game, _ uuid.UUID) error {
						game.RevokeAttr(destination.ID(), core.AttrIsCreature)
						return nil
					}))
					g.ApplyContinuousEffects()
				case "zero":
					action = NewDamageToCreatureAction(source.ID(), destination.ID(), 0, true).withDamageSource(saved)
					if kind == "attacker" {
						action = NewDamageToPlayerAction(source.ID(), a.PlayerID(), 0, true).withDamageSource(saved)
					}
				case "wrong-kind":
					action = NewLifeGainAction(source.ID(), a.PlayerID(), 2)
				case "consumed":
					rule.Replace(action, g)
					valid = kind == "attacker"
				}
				if rule.Matches(action, g) != valid {
					t.Fatal("match did not recheck applicability")
				}
				result := rule.Replace(action, g)
				if !valid && result != action {
					t.Fatal("direct replacement changed inapplicable action")
				}
				if valid && result == action {
					t.Fatal("applicable damage not redirected")
				}
			})
		}
	}
}

func TestWaitingTemporaryLegacyRedirectSourceIdentity(t *testing.T) {
	for _, state := range []string{"depart", "return", "phase", "phase-cycle", "control"} {
		t.Run(state, func(t *testing.T) {
			g, a, b, source, destination, rule := temporaryLegacyRedirectFixture("attacker")
			saved := g.CaptureDamageSource(source.ID())
			switch state {
			case "depart", "return":
				g.RemoveFromBattlefield(source)
				if state == "return" {
					g.PutOnBattlefield(source.Card, b.PlayerID())
				}
				g.zones.ClearLKI()
			case "phase":
				g.PhaseOut(source.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: source.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			for _, game := range []*Game{g, g.Clone()} {
				if len(game.WaitingRedirectionPermanentSources()) != 2 {
					t.Fatal("source reference lost")
				}
				if state == "return" {
					action := NewDamageToPlayerAction(source.ID(), a.PlayerID(), 1, false).withDamageSource(game.CaptureDamageSource(source.ID()))
					if rule.Matches(action, game) || rule.Replace(action, game) != action {
						t.Fatal("returned source inherited rule")
					}
				}
				game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 2, saved)
				if game.GetPlayer(a.PlayerID()).Life() != 20 || game.FindPermanent(destination.ID()).Damage != 2 {
					t.Fatal("saved source did not redirect")
				}
				if !game.HasDealtDamageToPermanentFromSource(saved, game.FindPermanent(destination.ID())) {
					t.Fatal("source attribution changed")
				}
			}
		})
	}
}

func TestWaitingTemporaryLegacyRedirectConsumptionAndBatch(t *testing.T) {
	for _, kind := range []string{"creature", "attacker"} {
		t.Run(kind, func(t *testing.T) {
			g, a, b, source, destination, _ := temporaryLegacyRedirectFixture(kind)
			saved := g.CaptureDamageSource(source.ID())
			baseline := g.Clone()
			if kind == "creature" {
				g.DealDamageToPermanent(destination, 2, source.ID())
				if b.Life() != 18 || g.FindPermanent(destination.ID()).Damage != 0 || len(g.WaitingRedirectionPermanentSources()) != 0 {
					t.Fatal("one-use lifetime or redirection")
				}
				g.DealDamageToPermanent(g.FindPermanent(destination.ID()), 1, source.ID())
				if g.FindPermanent(destination.ID()).Damage != 1 {
					t.Fatal("consumed rule reapplied")
				}
			} else {
				g.DealSimultaneousDamage(PlayerDamageAssignment(a, 2, saved), PlayerDamageAssignment(a, 3, saved))
				if a.Life() != 20 || g.FindPermanent(destination.ID()).Damage != 5 || len(g.WaitingRedirectionPermanentSources()) != 2 {
					t.Fatal("batch or persistent lifetime")
				}
			}
			want := 1
			if kind == "attacker" {
				want = 2
			}
			if len(baseline.WaitingRedirectionPermanentSources()) != want {
				t.Fatal("consumption changed clone")
			}
		})
	}
}

func TestWaitingTemporaryLegacyRedirectMissingAndDuplicate(t *testing.T) {
	g, a, _, source, destination, rule := temporaryLegacyRedirectFixture("attacker")
	g.effects.AddReplacement(rule.Clone())
	g.SetCreatureDamageRedirect(destination.ID(), a.PlayerID())
	g.SetAttackerDamageRedirect(uuid.New(), uuid.New())
	g.SetCreatureDamageRedirect(uuid.New(), a.PlayerID())
	g.effects.AddReplacement(&creatureDamageRedirectReplacement{creatureID: source.ID(), targetPlayerID: a.PlayerID()})
	if len(g.WaitingRedirectionPermanentSources()) != 2 {
		t.Fatal("duplicate or identity-free reference")
	}
	first, second := BindDamageSource("first"), BindDamageSource("second")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	for _, binding := range []*DamageSourceBinding{first, second} {
		if err := ChooseDamageSourceFromWaitingRedirectionReferences(You, DamageSourceHasColor(core.Red), binding).Apply(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if !g.resolution.damageSourceBindings[first].SameObject(g.CaptureDamageSource(source.ID())) {
		t.Fatal("filtered choice identity")
	}
	if err := ChooseDamageSourceFromWaitingRedirectionReferences(You, DamageSourceHasColor(core.Blue), first).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[first].Valid() || !g.resolution.damageSourceBindings[second].Valid() {
		t.Fatal("choice did not clear or isolate bindings")
	}
}

func TestWaitingTemporaryLegacyRedirectReplacementOrder(t *testing.T) {
	for choice := range 2 {
		t.Run(string(rune('0'+choice)), func(t *testing.T) {
			a := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: choice}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewCreature("Source", "{R}", 1, 20), b.PlayerID())
			destination := g.PutOnBattlefield(NewCreature("Destination", "{W}", 1, 20), a.PlayerID())
			g.SetAttackerDamageRedirect(source.ID(), destination.ID())
			g.effects.AddReplacement(&damagePreventionRuleReplacement{toPlayerID: a.PlayerID(), playerOnly: true})
			g.DealDamageToPlayerFromSource(a, 2, g.CaptureDamageSource(source.ID()))
			want := 2
			if choice == 1 {
				want = 0
			}
			if a.Life() != 20 || g.FindPermanent(destination.ID()).Damage != want {
				t.Fatal("affected-player ordering")
			}
			if len(g.WaitingRedirectionPermanentSources()) != 2 {
				t.Fatal("replacement consumed waiting references")
			}
		})
	}
}

func TestWaitingTemporaryLegacyRedirectAbsentInstallation(t *testing.T) {
	for _, kind := range []string{"creature", "attacker"} {
		t.Run(kind, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Absent", "{R}", 1, 20)
			source := g.PutOnBattlefield(NewCreature("Source", "{R}", 1, 20), b.PlayerID())
			if kind == "creature" {
				g.SetCreatureDamageRedirect(card.ID(), a.PlayerID())
			} else {
				g.SetAttackerDamageRedirect(source.ID(), card.ID())
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			if kind == "creature" {
				g.DealDamageToPermanent(p, 2, source.ID())
			} else {
				g.DealDamageToPlayer(a, 2, source.ID())
			}
			if kind == "creature" && g.FindPermanent(p.ID()).Damage != 2 {
				t.Fatal("absent installation followed a later permanent")
			}
			if kind == "attacker" && a.Life() != 18 {
				t.Fatal("absent installation followed a later destination")
			}
		})
	}
}
