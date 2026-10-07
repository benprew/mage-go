package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestStaticDamageRedirectLifecycle(t *testing.T) {
	for _, change := range []string{"live", "tap", "control", "phase", "phase-cycle", "departure", "remove-abilities"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Redirector", "", 1, 20, WithAbility(Static(RedirectDamageFromCardTypeToPermanentIf(You, This, core.TypeArtifact, PermanentIsUntapped(This)))))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			source := g.PutOnBattlefield(NewArtifact("Source", ""), b.PlayerID())
			clone := g.Clone()
			switch change {
			case "tap":
				g.TapPermanent(p)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "departure":
				g.RemoveFromBattlefield(p)
			case "remove-abilities":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.DealDamageToPlayer(a, 2, source.ID())
			want := 20
			if change != "live" && change != "phase-cycle" {
				want = 18
			}
			if a.Life() != want {
				t.Fatalf("life=%d want=%d", a.Life(), want)
			}
			clone.DealDamageToPlayer(clone.GetPlayer(a.PlayerID()), 2, source.ID())
			if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.FindPermanent(p.ID()).Damage != 2 {
				t.Fatal("clone lost independent rule")
			}
			if change == "control" {
				g.DealDamageToPlayer(b, 2, source.ID())
				if b.Life() != 20 || g.FindPermanent(p.ID()).Damage != 2 {
					t.Fatal("rule ignored live controller")
				}
			}
		})
	}
}

func TestStaticDamageRedirectInvalid(t *testing.T) {
	for _, rule := range []StaticRule{
		RedirectDamageFromCardTypeToPermanentIf(nil, This, core.TypeArtifact, PermanentIsUntapped(This)),
		RedirectDamageFromCardTypeToPermanentIf(You, nil, core.TypeArtifact, PermanentIsUntapped(This)),
		RedirectDamageFromCardTypeToPermanentIf(You, This, core.TypeArtifact, nil),
		RedirectDamageFromCardTypeToPermanentIf(You, This, core.CardType(255), PermanentIsUntapped(This)),
		RedirectDamageToPermanentIf(You, This, IsCreature, AllConditions()),
	} {
		card := NewCreature("Invalid", "", 1, 3, WithAbility(Static(rule)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid rule accepted")
		}
	}
}

func TestStaticDamageRedirectSourceInformation(t *testing.T) {
	for _, mode := range []string{"artifact", "unblocked"} {
		for _, change := range []string{"live", "before-blockers", "blocked", "blocker-removed", "removed", "departed", "returned", "phased", "phase-cycle", "lost-type"} {
			t.Run(mode+"/"+change, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				var rule = RedirectDamageFromCardTypeToPermanentIf(You, This, core.TypeArtifact, PermanentIsUntapped(This))
				if mode == "unblocked" {
					rule = RedirectDamageToPermanentIf(You, This, And(IsCreature, DamageSourceIsUnblocked()), PermanentIsUntapped(This))
				}
				p := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 20, WithAbility(Static(rule))), a.PlayerID())
				card := NewCreature("Source", "", 1, 20, WithKeyword(core.AttrIsArtifact), WithKeyword(core.Lifelink))
				source := g.PutOnBattlefield(card, b.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{source.ID()})
				var blocker *Permanent
				if change == "blocked" || change == "blocker-removed" {
					blocker = g.PutOnBattlefield(NewCreature("Blocker", "", 0, 20), a.PlayerID())
					g.ExecuteBlockers([]BlockAssignment{{BlockerID: blocker.ID(), AttackerID: source.ID()}})
				} else if change != "before-blockers" {
					g.ExecuteBlockers(nil)
				}
				information := g.CaptureDamageSource(source.ID())
				switch change {
				case "blocker-removed":
					g.RemoveFromCombat(blocker.ID())
				case "removed":
					g.RemoveFromCombat(source.ID())
				case "departed", "returned":
					g.RemoveFromBattlefield(source)
					if change == "returned" {
						g.PutOnBattlefield(card, b.PlayerID()).RevokeBaseAttr(core.AttrIsArtifact)
					}
				case "phased":
					g.PhaseOut(source.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(source.ID()))
				case "lost-type":
					g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrIsArtifact)
				}
				clone := g.Clone()
				matches := mode == "artifact" && change != "lost-type" || mode == "unblocked" && change != "before-blockers" && change != "blocked" && change != "blocker-removed" && change != "removed" && change != "phase-cycle"
				for _, game := range []*Game{g, clone} {
					game.DealSimultaneousDamage(PlayerDamageAssignment(game.GetPlayer(a.PlayerID()), 2, information), PlayerDamageAssignment(game.GetPlayer(a.PlayerID()), 3, information))
					wantLife, wantDamage := 15, 0
					if matches {
						wantLife, wantDamage = 20, 5
					}
					if game.GetPlayer(a.PlayerID()).Life() != wantLife || game.FindPermanent(p.ID()).Damage != wantDamage || game.GetPlayer(b.PlayerID()).Life() != 25 {
						t.Fatalf("life=%d damage=%d sourceLife=%d", game.GetPlayer(a.PlayerID()).Life(), game.FindPermanent(p.ID()).Damage, game.GetPlayer(b.PlayerID()).Life())
					}
				}
			})
		}
	}
}

func TestStaticDamageRedirectOrdering(t *testing.T) {
	for choice := range 2 {
		a := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: choice}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		p := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 20, WithAbility(Static(RedirectDamageFromCardTypeToPermanentIf(You, This, core.TypeArtifact, PermanentIsUntapped(This))))), a.PlayerID())
		source := g.PutOnBattlefield(NewArtifact("Source", ""), b.PlayerID())
		if err := ApplyEffect(g, PreventDamageThisTurn(You, PermanentFilter{}), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.DealDamageToPlayer(a, 2, source.ID())
		want := 0
		if choice == 1 {
			want = 2
		}
		if a.Life() != 20 || g.FindPermanent(p.ID()).Damage != want || a.prompts != 1 {
			t.Fatalf("choice=%d damage=%d prompts=%d", choice, g.FindPermanent(p.ID()).Damage, a.prompts)
		}
	}
}

func TestStaticDamageRedirectDestinationAndRechecks(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewCreature("Host", "", 1, 20), b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 20), b.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Rule", "", WithAbility(Static(RedirectDamageFromCardTypeToPermanentIf(You, AttachedPermanent, core.TypeArtifact, PermanentIsUntapped(This))))), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	source := g.PutOnBattlefield(NewArtifact("Source", ""), b.PlayerID())
	g.DealDamageToPlayer(a, 1, source.ID())
	if g.FindPermanent(host.ID()).Damage != 1 {
		t.Fatal("current host")
	}
	clone := g.Clone()
	g.Attach(aura.ID(), other.ID())
	g.DealDamageToPlayer(a, 2, source.ID())
	if g.FindPermanent(host.ID()).Damage != 1 || g.FindPermanent(other.ID()).Damage != 2 {
		t.Fatal("reattachment")
	}
	clone.PhaseOut(host.ID())
	clone.DealDamageToPlayer(clone.GetPlayer(a.PlayerID()), 1, source.ID())
	if clone.GetPlayer(a.PlayerID()).Life() != 19 {
		t.Fatal("phased destination")
	}
	g.RemoveFromBattlefield(g.FindPermanent(other.ID()))
	g.PutOnBattlefield(other.Card, b.PlayerID())
	g.DealDamageToPlayer(a, 1, source.ID())
	if a.Life() != 19 {
		t.Fatal("returned host inherited reference")
	}
	g.Attach(aura.ID(), host.ID())
	g.MutablePermanent(host.ID()).RevokeBaseAttr(core.AttrIsCreature)
	g.DealDamageToPlayer(a, 1, source.ID())
	if a.Life() != 18 {
		t.Fatal("noncreature destination")
	}
	g.MutablePermanent(host.ID()).GrantBaseAttr(core.AttrIsCreature)
	if err := ApplyEffect(g, PreventDamageThisTurn(AttachedPermanent, PermanentFilter{}), aura.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPlayer(a, 2, source.ID())
	if a.Life() != 18 || g.FindPermanent(host.ID()).Damage != 1 {
		t.Fatal("recipient replacement was not rechecked")
	}
}

func TestStaticDamageRedirectOriginalInstance(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Rule", "", 1, 20, WithAbility(Static(RedirectDamageFromCardTypeToPermanentIf(You, This, core.TypeArtifact, PermanentIsUntapped(This)))))
	p := g.PutOnBattlefield(card, a.PlayerID())
	source := g.PutOnBattlefield(NewArtifact("Source", ""), b.PlayerID())
	old := g.effects.cycleReplacements[0]
	g.RemoveFromBattlefield(p)
	returned := g.PutOnBattlefield(card, a.PlayerID())
	action := NewDamageToPlayerAction(source.ID(), a.PlayerID(), 2, false).withDamageSource(g.CaptureDamageSource(source.ID()))
	if old.Matches(action, g) {
		t.Fatal("old rule used returned source")
	}
	g.DealDamageToPlayer(a, 2, source.ID())
	if a.Life() != 20 || g.FindPermanent(returned.ID()).Damage != 2 {
		t.Fatal("fresh instance did not redirect")
	}
	g.RunStepWithPriority(core.Cleanup)
	g.DealDamageToPlayer(a, 1, source.ID())
	if a.Life() != 20 || g.FindPermanent(returned.ID()).Damage != 1 {
		t.Fatal("printed rule expired at cleanup")
	}
}
