package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestControlledSinceTurnStartState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Launcher", "")
	card.SetOwner(a.PlayerID())
	g.SetTurn(2)
	p := g.PutOnBattlefield(card, a.PlayerID())
	ctx := StateContext{SourceID: p.ID(), SourceIncarnation: p.incarnationID, ControllerID: a.PlayerID()}
	condition := ControlledSinceTurnStart(This)
	if condition.checkState(g, ctx) {
		t.Fatal("new permanent passed continuous control")
	}
	g.SetTurn(3)
	if !condition.checkState(g, ctx) || !condition.checkState(g.Clone(), ctx) {
		t.Fatal("continuous control was not retained")
	}
	if condition.checkState(g, StateContext{SourceID: p.ID(), SourceIncarnation: p.incarnationID, ControllerID: b.PlayerID()}) {
		t.Fatal("another controller passed continuous control")
	}
	p.PhasedOut = true
	if condition.checkState(g, ctx) {
		t.Fatal("phased source passed continuous control")
	}
	p.PhasedOut = false
	if err := ApplyEffect(g, GainControl(), b.PlayerID(), b.PlayerID(), []uuid.UUID{p.ID()}); err != nil {
		t.Fatal(err)
	}
	if condition.checkState(g, ctx) {
		t.Fatal("former controller passed after control change")
	}
	g.RemoveFromBattlefield(p)
	g.PutOnBattlefield(card, a.PlayerID())
	if condition.checkState(g, ctx) || condition.checkState(g, StateContext{SourceID: card.ID(), ControllerID: a.PlayerID()}) {
		t.Fatal("returned permanent inherited continuous control")
	}
}

func TestControlledSinceTurnStartDefinition(t *testing.T) {
	condition := ControlledSinceTurnStart(This)
	card := NewArtifact("Launcher", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), ActivationIf(condition))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if meta := InspectDefinition(condition); meta.Role != "state-condition" || meta.Text == "" {
		t.Fatalf("metadata = %+v", meta)
	}
	for _, bad := range []StateCondition{ControlledSinceTurnStart(nil), ControlledSinceTurnStart(From(BindPermanent("missing")))} {
		invalid := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), ActivationIf(bad))))
		if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
			t.Fatalf("invalid condition accepted: %+v", InspectDefinition(bad))
		}
	}
	spell := NewInstant("Invalid spell", "", Spell(IfState(condition, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell validation = %v", err)
	}
}

func TestControlledSinceTurnStartActivation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(2)
	card := NewArtifact("Launcher", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), ActivationIf(ControlledSinceTurnStart(This)))))
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err == nil || g.StackSize() != 0 {
		t.Fatal("new permanent allowed activation")
	}
	g.SetTurn(3)
	clone := g.Clone()
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 {
		t.Fatal("activation or clone isolation failed")
	}
	if err := ApplyEffect(g, GainControl(), b.PlayerID(), b.PlayerID(), []uuid.UUID{card.ID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), card.ID(), 0, nil); err == nil || g.StackSize() != 0 {
		t.Fatal("new controller allowed activation")
	}
}

func TestControlledSinceTurnStartAttachedPermanent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(2)
	host := NewCreature("Host", "", 2, 2)
	aura := NewAura("Watcher", "")
	g.PutOnBattlefield(host, a.PlayerID())
	g.PutOnBattlefield(aura, a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	ctx := StateContext{SourceID: aura.ID(), SourceIncarnation: g.FindPermanent(aura.ID()).incarnationID, ControllerID: a.PlayerID()}
	condition := ControlledSinceTurnStart(AttachedPermanent)
	if condition.checkState(g, ctx) {
		t.Fatal("new host passed continuous control")
	}
	g.SetTurn(3)
	if !condition.checkState(g, ctx) {
		t.Fatal("controlled host did not match")
	}
	if err := ApplyEffect(g, GainControl(), b.PlayerID(), b.PlayerID(), []uuid.UUID{host.ID()}); err != nil {
		t.Fatal(err)
	}
	if condition.checkState(g, ctx) {
		t.Fatal("opponent-controlled host matched")
	}
	valid := NewAura("Valid", "", WithAbility(Static(While(condition, GrantKeywordModification(This, core.Shroud)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
}
