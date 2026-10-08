package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingStaticPreventionReferences(t *testing.T) {
	for _, rule := range []StaticRule{PreventCombatDamageFrom(This), PreventCombatDamageTo(This, AnyPermanent), PreventDamageFromCardType(This, core.TypeArtifact), PreventDamageFromBlockedCreatures(This)} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		subject := g.PutOnBattlefield(NewCreature("Subject", "{R}", 1, 10, WithAbility(Static(rule))), a.PlayerID())
		saved := g.CaptureDamageSource(subject.ID())
		if got := g.WaitingPreventionPermanentSources(); len(got) != 1 || !got[0].SameObject(saved) {
			t.Fatalf("missing typed static subject: %v", got)
		}
		g.PhaseOut(subject.ID())
		if got := g.WaitingPreventionPermanentSources(); len(got) != 0 {
			t.Fatal("phased static source supplied subject")
		}
	}
}

func TestWaitingStaticPreventionChoiceBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	hostCard := NewCreature("Host", "{R}", 1, 10)
	hostCard.SetOwner(b.PlayerID())
	host := g.PutOnBattlefield(hostCard, b.PlayerID())
	for range 2 {
		aura := g.PutOnBattlefield(NewAura("Rule", "", WithAbility(Static(PreventCombatDamageFrom(AttachedPermanent), PreventCombatDamageTo(AttachedPermanent, AnyPermanent)))), a.PlayerID())
		g.Attach(aura.ID(), host.ID())
	}
	g.PutOnBattlefield(NewCreature("Unrelated", "{U}", 1, 10), a.PlayerID())
	g.SetPreventCombatDamage()
	if got := g.WaitingPreventionPermanentSources(); len(got) != 1 || !got[0].SameObject(g.CaptureDamageSource(host.ID())) {
		t.Fatal("duplicate subjects or registration sources")
	}
	first, second := BindDamageSource("first"), BindDamageSource("second")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	for _, binding := range []*DamageSourceBinding{first, second} {
		if err := ChooseDamageSourceFromWaitingPreventionReferences(You, DamageSourceHasColor(core.Red), binding).Apply(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err := ChooseDamageSourceFromWaitingPreventionReferences(You, DamageSourceHasColor(core.Blue), first).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[first].Valid() || !g.resolution.damageSourceBindings[second].SameObject(g.CaptureDamageSource(host.ID())) {
		t.Fatal("choice outputs were shared or included unrelated permanent")
	}
	got := g.WaitingPreventionPermanentSources()
	got[0].card.SetOwner(uuid.New())
	if g.WaitingPreventionPermanentSources()[0].card.Owner() != b.PlayerID() {
		t.Fatal("query exposed mutable snapshot")
	}
	g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
	g.PutOnBattlefield(hostCard, a.PlayerID())
	g.zones.ClearLKI()
	if len(g.WaitingPreventionPermanentSources()) != 0 {
		t.Fatal("returned host restored old static subject")
	}
}

func TestWaitingStaticPreventionStaleCycle(t *testing.T) {
	for _, state := range []string{"reattach", "source-phase", "subject-phase", "source-return", "host-return", "control"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			hostCard := NewCreature("Host", "{R}", 1, 10)
			host := g.PutOnBattlefield(hostCard, a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "{U}", 1, 10), b.PlayerID())
			card := NewAura("Rule", "", WithAbility(Static(PreventCombatDamageFrom(AttachedPermanent))))
			aura := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			var saved *staticDamagePreventionReplacement
			for _, replacement := range g.effects.cycleReplacements {
				if rule, ok := replacement.(*staticDamagePreventionReplacement); ok {
					saved = rule.Clone().(*staticDamagePreventionReplacement)
					break
				}
			}
			if saved == nil {
				t.Fatal("missing installed rule")
			}
			expected := host
			switch state {
			case "reattach":
				g.Attach(aura.ID(), other.ID())
				expected = other
			case "source-phase":
				g.PhaseOut(aura.ID())
				expected = nil
			case "subject-phase":
				g.PhaseOut(host.ID())
				expected = nil
			case "source-return":
				g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
				expected = nil
			case "host-return":
				g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
				g.PutOnBattlefield(hostCard, a.PlayerID())
				expected = nil
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: other.ID(), TargetID: aura.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.effects.cycleReplacements = []ReplacementEffect{saved}
			got := g.WaitingPreventionPermanentSources()
			if expected == nil {
				if len(got) != 0 {
					t.Fatal("stale cycle supplied inactive subject")
				}
			} else if len(got) != 1 || !got[0].SameObject(g.CaptureDamageSource(expected.ID())) {
				t.Fatal("stale cycle did not resolve live subject")
			}
		})
	}
}
