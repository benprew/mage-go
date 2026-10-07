package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestStaticSourceTypePrevention(t *testing.T) {
	for _, change := range []string{"live", "nonmatching", "lose-type", "gain-type", "depart", "return", "phase", "spell", "nonmatching-spell", "unknown", "unpreventable"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			victim := g.PutOnBattlefield(NewCreature("Protected", "", 2, 10, WithAbility(Static(PreventDamageFromCardType(This, core.TypeArtifact)))), a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 2, 10), a.PlayerID())
			card := NewArtifact("Source", "")
			if change == "nonmatching" || change == "gain-type" {
				card = NewCreature("Source", "", 1, 10)
			}
			source := g.PutOnBattlefield(card, b.PlayerID())
			saved := g.CaptureDamageSource(source.ID())
			want := 0
			switch change {
			case "nonmatching":
				want = 1
			case "lose-type":
				g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrIsArtifact)
				want = 1
			case "gain-type":
				g.MutablePermanent(source.ID()).GrantBaseAttr(core.AttrIsArtifact)
			case "depart", "return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "return" {
					returned := g.PutOnBattlefield(card, b.PlayerID())
					g.MutablePermanent(returned.ID()).RevokeBaseAttr(core.AttrIsArtifact)
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "spell", "nonmatching-spell":
				spell := NewInstant("Spell", "", nil)
				if change == "spell" {
					spell.AddType(core.TypeArtifact)
				} else {
					want = 1
				}
				saved = DamageSource{id: spell.ID(), card: spell}
			case "unknown":
				saved = DamageSource{id: source.ID(), incarnation: other.incarnationID}
				want = 1
			case "unpreventable":
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
				want = 1
			}
			for _, combat := range []bool{false, true} {
				g.damage.SetResolvingCombatDamage(combat)
				g.DealSimultaneousDamage(PermanentDamageAssignment(g.FindPermanent(victim.ID()), 1, saved), PermanentDamageAssignment(g.FindPermanent(other.ID()), 1, saved), PlayerDamageAssignment(a, 1, saved))
			}
			if g.FindPermanent(victim.ID()).Damage != 2*want || g.FindPermanent(other.ID()).Damage != 2 || a.Life() != 18 {
				t.Fatal("source matching or packet scope")
			}
		})
	}
}

func TestStaticSourceTypePreventionDeclarations(t *testing.T) {
	for _, rule := range []StaticRule{PreventDamageFromCardType(nil, core.TypeArtifact), PreventDamageFromCardType(DeclarePermanentTarget("target", IsCreature), core.TypeArtifact), PreventDamageFromCardType(From(BindPermanent("unbound")), core.TypeArtifact), PreventDamageFromCardType(This, core.CardType(-1)), PreventDamageFromCardType(This, core.CardType(99))} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(Static(rule)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid rule accepted")
		}
	}
}

func TestStaticSourceTypePreventionLiveHost(t *testing.T) {
	for _, change := range []string{"none", "reattach", "source-leaves", "source-phase", "source-phase-cycle", "host-return", "control", "remove-source-abilities", "remove-host-abilities", "clone"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			hostCard := NewCreature("Host", "", 2, 10)
			host := g.PutOnBattlefield(hostCard, a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 2, 10), b.PlayerID())
			aura := g.PutOnBattlefield(NewAura("Rule", "", WithAbility(Static(PreventDamageFromCardType(AttachedPermanent, core.TypeArtifact)))), a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			neutral := g.PutOnBattlefield(NewArtifact("Neutral", ""), b.PlayerID())
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
				clone.DealDamageToPermanentFromSource(clone.FindPermanent(host.ID()), 1, clone.CaptureDamageSource(neutral.ID()))
				if clone.FindPermanent(host.ID()).Damage != 1 || g.FindPermanent(host.ID()).Damage != 0 {
					t.Fatal("clone retained departed rule")
				}
			}
			g.damage.SetResolvingCombatDamage(true)
			g.DealDamageToPlayerFromSource(b, 1, saved)
			g.DealDamageToPermanentFromSource(g.FindPermanent(host.ID()), 1, g.CaptureDamageSource(neutral.ID()))
			want := 19
			damage := 1
			if protected {
				damage = 0
			}
			if b.Life() != want || g.FindPermanent(host.ID()).Damage != damage {
				t.Fatalf("life=%d damage=%d", b.Life(), g.FindPermanent(host.ID()).Damage)
			}
			g.damage.SetResolvingCombatDamage(false)
			g.DealDamageToPlayerFromSource(b, 1, saved)
			if b.Life() != want-1 {
				t.Fatal("outgoing damage changed")
			}
			g.DealDamageToPermanentFromSource(g.FindPermanent(host.ID()), 1, g.CaptureDamageSource(neutral.ID()))
			if g.FindPermanent(host.ID()).Damage != 2*damage {
				t.Fatal("noncombat prevention")
			}
		})
	}
}

func TestStaticSourceTypePreventionRecheck(t *testing.T) {
	for _, change := range []string{"lose-type", "gain-type", "rule-phase", "rule-return", "redirect"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Protected", "", 2, 10, WithAbility(Static(PreventDamageFromCardType(This, core.TypeArtifact))))
			victim := g.PutOnBattlefield(card, a.PlayerID())
			source := g.PutOnBattlefield(NewArtifact("Source", ""), b.PlayerID())
			if change == "gain-type" {
				source.RevokeBaseAttr(core.AttrIsArtifact)
			}
			g.AddReplacementEffect(&testPrioritizedReplacement{
				replacementBase: replacementBase{duration: core.Indefinite, priority: ReplacementPrioritySelf},
				matchFunc:       func(action Action) bool { _, ok := action.(*DamageToCreatureAction); return ok },
				replaceFunc: func(action Action, game *Game) Action {
					switch change {
					case "lose-type":
						game.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrIsArtifact)
					case "gain-type":
						game.MutablePermanent(source.ID()).GrantBaseAttr(core.AttrIsArtifact)
					case "rule-phase":
						game.PhaseOut(victim.ID())
					case "rule-return":
						game.RemoveFromBattlefield(game.FindPermanent(victim.ID()))
						game.PutOnBattlefield(card, a.PlayerID())
					case "redirect":
						damage := action.(*DamageToCreatureAction)
						return NewDamageToPlayerAction(damage.ActionSource(), a.PlayerID(), damage.Amount(), damage.IsCombatDamage()).withDamageSource(damage.DamageSource())
					}
					return action
				},
			})
			g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, source.ID())
			want := 0
			if change == "lose-type" {
				want = 1
			}
			if g.FindPermanentIncludingPhased(victim.ID()).Damage != want {
				t.Fatal("replacement recheck")
			}
			life := 20
			if change == "redirect" {
				life = 19
			}
			if a.Life() != life {
				t.Fatal("redirected damage")
			}
		})
	}
}
