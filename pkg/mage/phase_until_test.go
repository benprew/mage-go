package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPhaseUntilLifecycle(t *testing.T) {
	for _, change := range []string{"leave", "phase-source", "external-phase-in", "external-cycle", "source-control", "source-abilities", "return-subject"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewEnchantment("Source", ""), a.PlayerID())
			card := NewCreature("Subject", "", 2, 3)
			subject := g.PutOnBattlefield(card, b.PlayerID())
			subject.Counters[core.P1P1] = 2
			aura := g.PutOnBattlefield(NewAura("Aura", ""), a.PlayerID())
			aura.AttachedTo = subject.ID()
			subject.Attachments = append(subject.Attachments, aura.ID())
			ref := DeclarePermanentTarget("subject", AnyPermanent)
			g.resolution.targetSpecs = []Target{ref.targetSpec()}
			if err := ApplyEffect(g, PhaseOutUntilSourceLeaves(ref, true), source.ID(), a.PlayerID(), []uuid.UUID{subject.ID()}); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(subject.ID()) != nil || g.FindPermanent(aura.ID()) != nil {
				t.Fatal("group did not phase out")
			}
			g.SetActivePlayerIndex(1)
			g.doUntap()
			if g.FindPermanent(subject.ID()) != nil {
				t.Fatal("untap step ended the wait")
			}
			clone := g.Clone()
			switch change {
			case "phase-source":
				g.PhaseOut(source.ID())
				g.doUntap()
				if g.FindPermanent(subject.ID()) != nil {
					t.Fatal("source phasing ended the wait")
				}
				g.PhaseIn([]uuid.UUID{source.ID()})
			case "external-phase-in":
				g.PhaseIn([]uuid.UUID{subject.ID()})
			case "external-cycle":
				g.PhaseIn([]uuid.UUID{subject.ID()})
				g.PhaseOut(subject.ID())
			case "source-control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "source-abilities":
				g.GetEffects().Add(RemoveAllAbilities(source.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
			case "return-subject":
				g.RemoveFromBattlefield(g.FindPermanentIncludingPhased(subject.ID()))
				g.PutOnBattlefield(card, b.PlayerID())
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			if clone.FindPermanent(subject.ID()) != nil {
				t.Fatal("clone shared wait completion")
			}
			clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
			if p := clone.FindPermanent(subject.ID()); p == nil || !p.Tapped {
				t.Fatal("clone did not complete its own wait")
			}
			p := g.FindPermanentIncludingPhased(subject.ID())
			if change == "external-cycle" {
				if !p.PhasedOut {
					t.Fatal("old wait ended a later phase cycle")
				}
				g.doUntap()
				if g.FindPermanent(subject.ID()) == nil {
					t.Fatal("expired wait blocked normal phase-in")
				}
				return
			}
			if p.PhasedOut {
				t.Fatal("source departure did not phase in subject")
			}
			wantTap := change != "external-phase-in" && change != "return-subject"
			if p.Tapped != wantTap {
				t.Fatal("wrong phase-in tap status")
			}
			if change != "return-subject" && (p.Counters[core.P1P1] != 2 || g.FindPermanent(aura.ID()) == nil || g.FindPermanent(aura.ID()).AttachedTo != subject.ID()) {
				t.Fatal("phase-in lost counters or attachment")
			}
		})
	}
}

func TestPhaseUntilDefinition(t *testing.T) {
	for _, ref := range []PermanentRef{nil, DeclarePermanentTarget("unbound", AnyPermanent)} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), PhaseOutUntilSourceLeaves(ref, true))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid reference accepted")
		}
	}
}

func TestPhaseUntilIndirectAttachments(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewEnchantment("Source", ""), a.PlayerID())
	subject := g.PutOnBattlefield(NewCreature("Subject", "", 2, 3), b.PlayerID())
	equipment := g.PutOnBattlefield(NewCreature("Equipment", "", 1, 1, WithCardType(core.TypeArtifact), WithSubTypes("Equipment")), a.PlayerID())
	fortification := g.PutOnBattlefield(NewArtifact("Fortification", "", WithSubTypes("Fortification")), a.PlayerID())
	nested := g.PutOnBattlefield(NewAura("Nested Aura", ""), a.PlayerID())
	direct := g.PutOnBattlefield(NewAura("Direct Aura", ""), a.PlayerID())
	equipment.AttachedTo = subject.ID()
	fortification.AttachedTo = subject.ID()
	nested.AttachedTo = equipment.ID()
	direct.AttachedTo = subject.ID()
	g.PhaseOut(direct.ID())
	ref := DeclarePermanentTarget("subject", AnyPermanent)
	g.resolution.targetSpecs = []Target{ref.targetSpec()}
	if err := ApplyEffect(g, PhaseOutUntilSourceLeaves(ref, true), source.ID(), a.PlayerID(), []uuid.UUID{subject.ID()}); err != nil {
		t.Fatal(err)
	}
	for _, p := range []*Permanent{subject, equipment, fortification, nested} {
		if g.FindPermanent(p.ID()) != nil {
			t.Fatal("indirect group member remained present", p.Name())
		}
	}
	g.PhaseIn([]uuid.UUID{equipment.ID()})
	if g.FindPermanent(equipment.ID()) != nil {
		t.Fatal("indirect attachment phased in by itself")
	}
	g.SetActivePlayerIndex(1)
	g.doUntap()
	if g.FindPermanent(equipment.ID()) != nil || g.FindPermanent(nested.ID()) != nil {
		t.Fatal("indirect group phased in without host")
	}
	g.RemoveFromBattlefield(source)
	for _, p := range []*Permanent{subject, equipment, fortification, nested} {
		if g.FindPermanent(p.ID()) == nil {
			t.Fatal("indirect group member did not return", p.Name())
		}
	}
	if g.FindPermanent(direct.ID()) != nil {
		t.Fatal("direct attachment inherited group phase-in")
	}
	if g.FindPermanent(equipment.ID()).Tapped || g.FindPermanent(nested.ID()).Tapped {
		t.Fatal("phase-in tapped attachments")
	}
}

func TestPhaseUntilSimultaneousAndEvents(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	sources := make([]*Permanent, 0, 2)
	subjects := make([]*Permanent, 0, 2)
	for range 2 {
		source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
		subject := g.PutOnBattlefield(NewCreature("Subject", "", 2, 3), b.PlayerID())
		sources = append(sources, source)
		subjects = append(subjects, subject)
		ref := DeclarePermanentTarget("subject", AnyPermanent)
		g.resolution.targetSpecs = []Target{ref.targetSpec()}
		if err := ApplyEffect(g, PhaseOutUntilSourceLeaves(ref, true), source.ID(), a.PlayerID(), []uuid.UUID{subject.ID()}); err != nil {
			t.Fatal(err)
		}
	}
	events := 0
	watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(event *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if event.FromZone != core.ZoneBattlefield {
			return false
		}
		events++
		for _, p := range subjects {
			if live := g.FindPermanent(p.ID()); live == nil || !live.Tapped {
				t.Error("leave event observed incomplete phase-in group")
			}
		}
		for _, p := range sources {
			if g.FindPermanent(p.ID()) != nil {
				t.Error("phase-in preceded simultaneous source departures")
			}
		}
		return false
	})
	g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(watcher)), a.PlayerID())
	clone := g.Clone()
	g.DestroyPermanents(sources)
	if events != 2 {
		t.Fatal("wrong leave event count", events)
	}
	for _, p := range subjects {
		if clone.FindPermanent(p.ID()) != nil {
			t.Fatal("clone shared phase-in group")
		}
	}
}

func TestPhaseUntilNoTapOrZoneEvents(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	subject := g.PutOnBattlefield(NewCreature("Subject", "", 2, 3, WithAbility(WhenEvent(BecomesTapped(This), GainLifeFor(You, Fixed(1)))), WithAbility(WhenEvent(Enters(This), GainLifeFor(You, Fixed(1))))), b.PlayerID())
	g.ResolveStack()
	before := b.Life()
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	ref := DeclarePermanentTarget("subject", AnyPermanent)
	g.resolution.targetSpecs = []Target{ref.targetSpec()}
	if err := ApplyEffect(g, PhaseOutUntilSourceLeaves(ref, true), source.ID(), a.PlayerID(), []uuid.UUID{subject.ID()}); err != nil {
		t.Fatal(err)
	}
	g.AddRegenerationShield(source.ID())
	g.DestroyPermanent(source)
	if g.FindPermanent(subject.ID()) != nil {
		t.Fatal("regeneration ended the wait")
	}
	g.ExilePermanent(g.FindPermanent(source.ID()))
	g.ResolveStack()
	if b.Life() != before {
		t.Fatal("phase-in emitted a tap or entry event")
	}
	if p := g.FindPermanent(subject.ID()); p == nil || !p.Tapped {
		t.Fatal("exile did not complete wait")
	}
}

func TestPhaseUntilIndependentWaits(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	sources := make([]*Permanent, 0, 2)
	subjects := make([]*Permanent, 0, 2)
	for range 2 {
		source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
		subject := g.PutOnBattlefield(NewCreature("Subject", "", 2, 3), b.PlayerID())
		ref := DeclarePermanentTarget("subject", AnyPermanent)
		g.resolution.targetSpecs = []Target{ref.targetSpec()}
		if err := ApplyEffect(g, PhaseOutUntilSourceLeaves(ref, false), source.ID(), a.PlayerID(), []uuid.UUID{subject.ID()}); err != nil {
			t.Fatal(err)
		}
		sources = append(sources, source)
		subjects = append(subjects, subject)
	}
	g.BouncePermanentToHand(sources[0])
	if p := g.FindPermanent(subjects[0].ID()); p == nil || p.Tapped {
		t.Fatal("wrong untapped phase-in")
	}
	if g.FindPermanent(subjects[1].ID()) != nil {
		t.Fatal("first source released second wait")
	}
	g.RemoveFromBattlefield(sources[1])
	if g.FindPermanent(subjects[1].ID()) == nil {
		t.Fatal("second wait did not complete")
	}
}
