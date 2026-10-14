package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestConditionalStaticBodyPrevention(t *testing.T) {
	for _, change := range []string{"untapped", "tap", "phase", "phase-cycle", "remove", "clone", "return", "unpreventable", "recheck-tap", "recheck-untap"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Rule", "", 1, 20, WithAbility(Static(PreventDamageFromMatchingPermanentsIf(This, IsArtifact, PermanentIsUntapped(This)))))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			source := g.PutOnBattlefield(NewArtifact("Source", ""), b.PlayerID())
			want := 0
			switch change {
			case "tap":
				g.MutablePermanent(p.ID()).Tapped = true
				want = 1
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "remove":
				g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.EndOfTurn))
				want = 1
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				p = g.PutOnBattlefield(card, a.PlayerID())
			case "clone":
				clone := g.Clone()
				clone.MutablePermanent(p.ID()).Tapped = true
				clone.DealDamageToPermanent(clone.FindPermanent(p.ID()), 1, source.ID())
				if clone.FindPermanent(p.ID()).Damage != 1 || g.FindPermanent(p.ID()).Damage != 0 {
					t.Fatal("clone state shared")
				}
			case "unpreventable":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
				want = 1
			case "recheck-tap", "recheck-untap":
				g.MutablePermanent(p.ID()).Tapped = change == "recheck-untap"
				if change == "recheck-tap" {
					want = 1
				}
				g.AddReplacementEffect(&testPrioritizedReplacement{
					replacementBase: replacementBase{duration: core.Indefinite, priority: ReplacementPrioritySelf},
					matchFunc:       func(action Action) bool { _, ok := action.(*DamageToCreatureAction); return ok },
					replaceFunc: func(action Action, game *Game) Action {
						game.MutablePermanent(p.ID()).Tapped = change == "recheck-tap"
						return action
					},
				})
			}
			g.DealDamageToPermanentFromSource(g.FindPermanentIncludingPhased(p.ID()), 1, g.CaptureDamageSource(source.ID()))
			if got := g.FindPermanentIncludingPhased(p.ID()).Damage; got != want {
				t.Fatalf("damage %d want %d", got, want)
			}
		})
	}
}

func TestConditionalStaticBodyDefinitions(t *testing.T) {
	for _, tc := range []struct {
		ref       PermanentRef
		condition StateCondition
	}{
		{nil, PermanentIsUntapped(This)}, {This, nil}, {This, PermanentIsUntapped(From(BindPermanent("missing")))},
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(Static(PreventDamageFromMatchingPermanentsIf(tc.ref, IsArtifact, tc.condition))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestConditionalStaticBodyWaitingAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewCreature("Host", "", 1, 20), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 20), b.PlayerID())
	card := NewAura("Rule", "", WithAbility(Static(PreventDamageFromMatchingPermanentsIf(AttachedPermanent, IsCreature, PermanentIsUntapped(This)))))
	source := g.PutOnBattlefield(card, a.PlayerID())
	g.Attach(source.ID(), host.ID())
	var saved *staticDamagePreventionReplacement
	for _, replacement := range g.effects.cycleReplacements {
		if rule, ok := replacement.(*staticDamagePreventionReplacement); ok {
			saved = rule.Clone().(*staticDamagePreventionReplacement)
		}
	}
	if saved == nil {
		t.Fatal("missing rule")
	}
	check := func(want *Permanent) {
		t.Helper()
		g.effects.cycleReplacements = []ReplacementEffect{saved}
		got := g.WaitingPreventionPermanentSources()
		if want == nil {
			if len(got) != 0 {
				t.Fatal("inactive reference")
			}
		} else if len(got) != 1 || !got[0].SameObject(g.CaptureDamageSource(want.ID())) {
			t.Fatal("wrong visit")
		}
	}
	check(host)
	g.MutablePermanent(source.ID()).Tapped = true
	check(nil)
	g.MutablePermanent(source.ID()).Tapped = false
	check(host)
	g.Attach(source.ID(), other.ID())
	check(other)
	phased := g.PhaseOut(source.ID())
	check(nil)
	g.PhaseIn(phased)
	check(other)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	returned := g.PutOnBattlefield(card, a.PlayerID())
	g.Attach(returned.ID(), host.ID())
	check(nil)
}

func TestConditionalStaticBodyControllerAndBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := PlayerControls(You, IsLand)
	p := g.PutOnBattlefield(NewCreature("Rule", "", 1, 20, WithAbility(Static(PreventDamageFromMatchingPermanentsIf(This, IsArtifact, condition)))), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 20), a.PlayerID())
	source := g.PutOnBattlefield(NewArtifact("Source", ""), b.PlayerID())
	land := g.PutOnBattlefield(NewLand("Land"), a.PlayerID())
	captured := g.CaptureDamageSource(source.ID())
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.DealSimultaneousDamage(PermanentDamageAssignment(g.FindPermanent(p.ID()), 2, captured), PermanentDamageAssignment(other, 2, captured), PlayerDamageAssignment(a, 2, captured))
	if g.FindPermanent(p.ID()).Damage != 0 || g.FindPermanent(other.ID()).Damage != 2 || a.Life() != 18 {
		t.Fatal("batch or departure source")
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: other.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if len(g.WaitingPreventionPermanentSources()) != 0 {
		t.Fatal("condition kept old controller")
	}
	g.DealDamageToPermanentFromSource(g.FindPermanent(p.ID()), 1, captured)
	if g.FindPermanent(p.ID()).Damage != 1 {
		t.Fatal("old controller supplied condition")
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: other.ID(), TargetID: land.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.DealDamageToPermanentFromSource(g.FindPermanent(p.ID()), 1, captured)
	if g.FindPermanent(p.ID()).Damage != 1 {
		t.Fatal("new controller condition did not start")
	}
}
