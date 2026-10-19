package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRetainedProtectionLifecycle(t *testing.T) {
	for _, change := range []string{"none", "tap", "remove", "phase", "phase-cycle", "leave", "return", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			host := g.PutOnBattlefield(NewArtifact("Subject", ""), a.PlayerID())
			old := g.PutOnBattlefield(NewAura("Old", "", WithCastTarget(TargetArtifact())), b.PlayerID())
			g.Attach(old.ID(), host.ID())
			subjects := ControlledPermanents(IsArtifact)
			condition := PermanentIsUntapped(This)
			card := NewCreature("Source", "", 2, 4, WithAbility(Static(
				PreventNewAuraAttachmentsIf(subjects, condition),
				PreventControlGainsIf(subjects, condition),
				While(condition, GrantEachKeyword(subjects, core.Indestructible)),
			)))
			source := g.PutOnBattlefield(card, a.PlayerID())
			clone := g.Clone()
			switch change {
			case "tap":
				g.MutablePermanent(source.ID()).Tapped = true
				g.ApplyContinuousEffects()
			case "remove":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "leave", "return":
				g.RemoveFromBattlefield(source)
				if change == "return" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			protected := change == "none" || change == "phase-cycle" || change == "return"
			if auraHostIsLegal(NewAura("New", "", WithCastTarget(TargetArtifact())), g.FindPermanent(host.ID()), g) == protected {
				t.Fatal("wrong new Aura permission")
			}
			if !auraHostIsLegal(old.Card, g.FindPermanent(host.ID()), g) {
				t.Fatal("existing Aura lost permission")
			}
			if auraHostIsLegal(NewAura("Clone New", "", WithCastTarget(TargetArtifact())), clone.FindPermanent(host.ID()), clone) {
				t.Fatal("clone lost rule")
			}
			g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			if (g.FindPermanent(host.ID()).ControllerID() == a.PlayerID()) != protected {
				t.Fatal("wrong control permission")
			}
			if g.FindPermanent(host.ID()).HasKeyword(core.Indestructible) != (protected || change == "control") {
				t.Fatal("later layer used rejected controller")
			}
		})
	}
}

func TestRetainedProtectionDefinitions(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "valid", Card: NewArtifact("Good", "", WithAbility(Static(PreventNewAuraAttachmentsIf(MatchingPermanents(IsArtifact), PermanentIsUntapped(This)), PreventControlGainsIf(MatchingPermanents(IsArtifact), PermanentIsUntapped(This)))))},
		{Name: "nil set", Card: NewArtifact("Bad", "", WithAbility(Static(PreventControlGainsIf(nil, PermanentIsUntapped(This))))), WantErr: true},
		{Name: "nil condition", Card: NewArtifact("Bad", "", WithAbility(Static(PreventNewAuraAttachmentsIf(MatchingPermanents(IsArtifact), nil)))), WantErr: true},
	})
}

func TestRetainedProtectionAttachmentsAndIdentity(t *testing.T) {
	g, a, b := newSpellCopyGame()
	card := NewArtifact("Subject", "")
	host := g.PutOnBattlefield(card, a.PlayerID())
	other := g.PutOnBattlefield(NewArtifact("Other", ""), b.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Aura", "", WithCastTarget(TargetArtifact())), b.PlayerID())
	g.Attach(aura.ID(), host.ID())
	source := g.PutOnBattlefield(NewCreature("Source", "", 2, 4, WithAbility(Static(PreventNewAuraAttachmentsIf(ControlledPermanents(IsArtifact), PermanentIsUntapped(This)), PreventControlGainsIf(ControlledPermanents(IsArtifact), PermanentIsUntapped(This))))), a.PlayerID())
	if g.ExchangeControl(host.ID(), other.ID(), source.ID()) {
		t.Fatal("prohibited exchange succeeded")
	}
	if g.FindPermanent(other.ID()).ControllerID() != b.PlayerID() {
		t.Fatal("exchange changed the unprotected half")
	}
	g.CheckStateBasedActions()
	if g.FindPermanent(aura.ID()) == nil {
		t.Fatal("existing Aura was removed")
	}
	g.Attach(aura.ID(), other.ID())
	g.Attach(aura.ID(), host.ID())
	if g.FindPermanent(aura.ID()).AttachedTo != other.ID() {
		t.Fatal("former Aura reattached")
	}
	g.RemoveFromBattlefield(host)
	host = g.PutOnBattlefield(card, a.PlayerID())
	if auraHostIsLegal(aura.Card, host, g) {
		t.Fatal("returned visit inherited an attachment exception")
	}
	g.PhaseIn(g.PhaseOut(host.ID()))
	if auraHostIsLegal(aura.Card, g.FindPermanent(host.ID()), g) {
		t.Fatal("phasing removed the restriction")
	}
	g.MutablePermanent(source.ID()).Tapped = true
	g.ApplyContinuousEffects()
	g.Attach(aura.ID(), host.ID())
	g.MutablePermanent(source.ID()).Tapped = false
	g.ApplyContinuousEffects()
	g.CheckStateBasedActions()
	if g.FindPermanent(aura.ID()) == nil || !auraHostIsLegal(aura.Card, g.FindPermanent(host.ID()), g) {
		t.Fatal("reactivation removed a legal Aura")
	}
}

func TestRetainedProtectionEstablishedControl(t *testing.T) {
	g, a, b := newSpellCopyGame()
	host := g.PutOnBattlefield(NewArtifact("Borrowed", ""), b.PlayerID())
	g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	source := g.PutOnBattlefield(NewCreature("Source", "", 2, 4, WithAbility(Static(PreventControlGainsIf(ControlledPermanents(IsArtifact), PermanentIsUntapped(This))))), a.PlayerID())
	for range 3 {
		g.ApplyContinuousEffects()
	}
	if g.FindPermanent(host.ID()).ControllerID() != a.PlayerID() {
		t.Fatal("established control was removed")
	}
	clone := g.Clone()
	clone.MutablePermanent(source.ID()).Tapped = true
	clone.ApplyContinuousEffects()
	clone.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if clone.FindPermanent(host.ID()).ControllerID() != b.PlayerID() || g.FindPermanent(host.ID()).ControllerID() != a.PlayerID() {
		t.Fatal("clone control state is not separate")
	}
}

func TestRetainedProtectionControlExpiry(t *testing.T) {
	for _, underlying := range []bool{false, true} {
		g, a, b := newSpellCopyGame()
		host := g.PutOnBattlefield(NewArtifact("Borrowed", ""), b.PlayerID())
		if underlying {
			g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
		}
		g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.EndOfTurn})
		g.PutOnBattlefield(NewCreature("Source", "", 2, 4, WithAbility(Static(PreventControlGainsIf(ControlledPermanents(IsArtifact), PermanentIsUntapped(This))))), a.PlayerID())
		clone := g.Clone()
		clone.effects.RemoveEndOfTurn()
		clone.ApplyContinuousEffects()
		if clone.FindPermanent(host.ID()).ControllerID() != b.PlayerID() || g.FindPermanent(host.ID()).ControllerID() != a.PlayerID() {
			t.Fatal("clone lost control provenance or changed the original")
		}
		g.effects.RemoveEndOfTurn()
		g.ApplyContinuousEffects()
		if g.FindPermanent(host.ID()).ControllerID() != b.PlayerID() {
			t.Fatal("restriction extended an ended control effect")
		}
	}
}

func TestRetainedProtectionControlForms(t *testing.T) {
	for _, form := range []string{"layer", "aura"} {
		t.Run(form, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			host := g.PutOnBattlefield(NewArtifact("Borrowed", ""), b.PlayerID())
			var aura *Permanent
			if form == "layer" {
				if err := ApplyEffect(g, Until(core.EndOfTurn, ChangeController(This, You)), host.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			} else {
				aura = g.PutOnBattlefield(NewAura("Control", "", WithCastTarget(TargetArtifact()), WithStaticAbility(ControlAttached())), a.PlayerID())
				g.Attach(aura.ID(), host.ID())
			}
			g.PutOnBattlefield(NewCreature("Source", "", 2, 4, WithAbility(Static(PreventControlGainsIf(ControlledPermanents(IsArtifact), PermanentIsUntapped(This))))), a.PlayerID())
			if g.FindPermanent(host.ID()).ControllerID() != a.PlayerID() {
				t.Fatal("established control was lost")
			}
			g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			if g.FindPermanent(host.ID()).ControllerID() != a.PlayerID() {
				t.Fatal("new gain was permitted")
			}
			clone := g.Clone()
			for _, game := range []*Game{clone, g} {
				if aura == nil {
					game.effects.RemoveEndOfTurn()
				} else {
					game.RemoveFromBattlefield(game.FindPermanent(aura.ID()))
				}
				game.ApplyContinuousEffects()
				if game.FindPermanent(host.ID()).ControllerID() != b.PlayerID() {
					t.Fatal("ended effect retained control")
				}
			}
		})
	}
}
