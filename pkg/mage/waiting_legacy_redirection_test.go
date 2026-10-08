package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func legacyRedirectionFixture(t *testing.T, kind string) (*Game, *BasePlayer, *BasePlayer, *Permanent, ReplacementEffect) {
	t.Helper()
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Destination", "{W}", 1, 20)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	switch kind {
	case "bodyguard":
		e := BodyguardContinuous()
		e.SetSourceID(p.ID())
		if err := e.ApplyContinuous(&EffectContext{Game: g}); err != nil {
			t.Fatal(err)
		}
	case "player":
		e := PersonalIncarnationRedirect()
		e.SetSourceID(p.ID())
		if err := e.ApplyContinuous(&EffectContext{Game: g}); err != nil {
			t.Fatal(err)
		}
	case "artifact":
		g.SetArtifactDamageRedirect(a.PlayerID(), p.ID())
	}
	return g, a, b, p, g.effects.cycleReplacements[len(g.effects.cycleReplacements)-1].Clone()
}

func TestWaitingLegacyRedirectionReferences(t *testing.T) {
	for _, kind := range []string{"bodyguard", "player", "artifact"} {
		for _, state := range []string{"live", "tap", "phase", "phase-cycle", "depart", "return", "control", "cleanup"} {
			t.Run(kind+"/"+state, func(t *testing.T) {
				g, a, b, p, rule := legacyRedirectionFixture(t, kind)
				saved := g.CaptureDamageSource(p.ID())
				clone := g.Clone()
				want := 1
				switch state {
				case "tap":
					g.TapPermanent(p)
					if kind == "bodyguard" {
						want = 0
					}
				case "phase":
					g.PhaseOut(p.ID())
					want = 0
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "depart", "return":
					g.RemoveFromBattlefield(p)
					g.zones.ClearLKI()
					want = 0
					if state == "return" {
						g.PutOnBattlefield(p.Card, b.PlayerID())
					}
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "cleanup":
					g.effects.ClearReplacementsEndOfTurn()
				}
				g.effects.cycleReplacements = []ReplacementEffect{rule}
				source := g.PutOnBattlefield(NewArtifact("Artifact damage", ""), b.PlayerID())
				g.effects.cycleReplacements = []ReplacementEffect{rule}
				action := NewDamageToPlayerAction(source.ID(), a.PlayerID(), 2, true).withDamageSource(g.CaptureDamageSource(source.ID()))
				if rule.IsActive(g) != (want == 1) {
					t.Fatal("active rule did not check destination")
				}
				if rule.Matches(action, g) != (want == 1) {
					t.Fatal("match did not check destination")
				}
				result := rule.Replace(action, g)
				if want == 0 && result != action {
					t.Fatal("replace followed an inactive destination")
				}
				if want == 1 {
					redirected, ok := result.(*DamageToCreatureAction)
					if !ok || redirected.PermanentID() != p.ID() || redirected.Amount() != 2 || !redirected.IsCombatDamage() || !redirected.DamageSource().SameObject(action.DamageSource()) {
						t.Fatal("redirect changed damage facts")
					}
				}
				for _, game := range []*Game{g, g.Clone()} {
					refs := game.WaitingRedirectionPermanentSources()
					if len(refs) != want {
						t.Fatalf("references=%d want=%d", len(refs), want)
					}
					if want == 1 {
						if !refs[0].SameObject(saved) {
							t.Fatal("destination incarnation changed")
						}
						if state == "control" && refs[0].Controller() != b.PlayerID() {
							t.Fatal("stale source information")
						}
						refs[0].card.SetOwner(b.PlayerID())
						if game.WaitingRedirectionPermanentSources()[0].Owner() != a.PlayerID() {
							t.Fatal("query shared card state")
						}
					}
				}
				if len(clone.WaitingRedirectionPermanentSources()) != 1 {
					t.Fatal("mutation changed clone")
				}
			})
		}
	}
}

func TestWaitingLegacyRedirectionChoices(t *testing.T) {
	g, a, b, p, rule := legacyRedirectionFixture(t, "artifact")
	g.effects.AddReplacement(rule.Clone())
	g.effects.AddReplacement(&artifactDamageRedirectReplacement{controllerID: a.PlayerID(), redirectPermID: p.ID()})
	g.SetArtifactDamageRedirect(a.PlayerID(), uuid.New())
	if len(g.WaitingRedirectionPermanentSources()) != 1 {
		t.Fatal("duplicate or invalid identity")
	}
	first, second := BindDamageSource("first"), BindDamageSource("second")
	ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
	for _, binding := range []*DamageSourceBinding{first, second} {
		if err := ChooseDamageSourceFromWaitingRedirectionReferences(You, DamageSourceHasColor(core.White), binding).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if !g.resolution.damageSourceBindings[binding].SameObject(g.CaptureDamageSource(p.ID())) {
			t.Fatal("choice identity")
		}
	}
	if err := ChooseDamageSourceFromWaitingRedirectionReferences(You, DamageSourceHasColor(core.Blue), first).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[first].Valid() || !g.resolution.damageSourceBindings[second].Valid() {
		t.Fatal("binding isolation")
	}
	for _, effect := range []ResolvingEffect{ChooseDamageSourceFromWaitingRedirectionReferences(nil, DamageSourceFilter{}, first), ChooseDamageSourceFromWaitingRedirectionReferences(You, DamageSourceFilter{}, nil), Sequence(DealDamageFrom(FromDamageSource(first), You, Fixed(1)), ChooseDamageSourceFromWaitingRedirectionReferences(You, DamageSourceFilter{}, first))} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestWaitingLegacyRedirectionDamageAndCycle(t *testing.T) {
	for _, kind := range []string{"bodyguard", "player", "artifact"} {
		t.Run(kind, func(t *testing.T) {
			g, a, b, p, rule := legacyRedirectionFixture(t, kind)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), b.PlayerID())
			saved := g.CaptureDamageSource(source.ID())
			g.effects.cycleReplacements = []ReplacementEffect{rule}
			other := NewDamageToPlayerAction(source.ID(), b.PlayerID(), 2, true).withDamageSource(saved)
			if rule.Matches(other, g) || rule.Replace(other, g) != other {
				t.Fatal("wrong player redirected")
			}
			noncombat := NewDamageToPlayerAction(source.ID(), a.PlayerID(), 2, false).withDamageSource(saved)
			if rule.Matches(noncombat, g) != (kind != "bodyguard") {
				t.Fatal("combat restriction changed")
			}
			wrong := NewDamageToCreatureAction(source.ID(), p.ID(), 2, true).withDamageSource(saved)
			if rule.Matches(wrong, g) || rule.Replace(wrong, g) != wrong {
				t.Fatal("wrong action replaced")
			}
			g.executeDamageAction(g.effects.ApplyReplacements(NewDamageToPlayerAction(source.ID(), a.PlayerID(), 2, true).withDamageSource(saved), g))
			if a.Life() != 20 || g.FindPermanent(p.ID()).Damage != 2 || !g.HasDealtDamageToPermanentFromSource(saved, g.FindPermanent(p.ID())) {
				t.Fatal("redirect execution or attribution")
			}
			if len(g.WaitingRedirectionPermanentSources()) != 1 {
				t.Fatal("persistent rule consumed")
			}
			g.ApplyContinuousEffects()
			if len(g.WaitingRedirectionPermanentSources()) != 0 {
				t.Fatal("cycle rebuild retained an unowned registration")
			}
			g.SetArtifactDamageRedirect(a.PlayerID(), p.ID())
			before := g.CaptureDamageSource(p.ID())
			g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
			returned := g.PutOnBattlefield(p.Card, b.PlayerID())
			g.effects.cycleReplacements = []ReplacementEffect{rule}
			if len(g.WaitingRedirectionPermanentSources()) != 0 {
				t.Fatal("returned destination inherited rule")
			}
			g.SetArtifactDamageRedirect(a.PlayerID(), returned.ID())
			refs := g.WaitingRedirectionPermanentSources()
			if len(refs) != 1 || refs[0].SameObject(before) || !refs[0].SameObject(g.CaptureDamageSource(returned.ID())) {
				t.Fatal("new registration did not bind returned destination")
			}
		})
	}
}

func TestWaitingLegacyRedirectionReplacementOrder(t *testing.T) {
	for choice := range 2 {
		t.Run(string(rune('0'+choice)), func(t *testing.T) {
			a := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: choice}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), b.PlayerID())
			p := g.PutOnBattlefield(NewCreature("Destination", "{W}", 1, 20), a.PlayerID())
			g.SetArtifactDamageRedirect(a.PlayerID(), p.ID())
			g.effects.AddReplacement(&damagePreventionRuleReplacement{from: IsArtifact, toPlayerID: a.PlayerID(), playerOnly: true})
			g.DealSimultaneousDamage(PlayerDamageAssignment(a, 2, g.CaptureDamageSource(source.ID())), PlayerDamageAssignment(a, 3, g.CaptureDamageSource(source.ID())))
			want := 0
			if choice == 1 {
				want = 5
			}
			if a.Life() != 20 || g.FindPermanent(p.ID()).Damage != want {
				t.Fatal("replacement ordering or simultaneous damage changed")
			}
			if len(g.WaitingRedirectionPermanentSources()) != 1 {
				t.Fatal("damage changed persistent reference lifetime")
			}
		})
	}
}
