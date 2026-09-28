package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestPlayerControlsStateUsesCurrentController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	island := NewLand("Island", WithSubTypes("Island"))
	island.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(island, b.PlayerID())
	ctx := StateContext{ControllerID: a.PlayerID()}
	condition := PlayerControls(ActivePlayer, HasSubType("Island"))
	g.SetActivePlayerIndex(1)
	if !condition.checkState(g, ctx) {
		t.Fatal("active controller's Island did not match")
	}
	if PlayerControls(You, HasSubType("Island")).checkState(g, ctx) {
		t.Fatal("owner matched in place of controller")
	}
	g.PhaseOut(p.ID())
	if condition.checkState(g, ctx) {
		t.Fatal("phased Island matched")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(p.ID()))
	if condition.checkState(clone, ctx) || !condition.checkState(g, ctx) {
		t.Fatal("clone changed the original battlefield")
	}
	g.RemoveFromBattlefield(p)
	if condition.checkState(g, ctx) {
		t.Fatal("departed Island matched")
	}
	g.PutOnBattlefield(island, a.PlayerID())
	if condition.checkState(g, ctx) || !PlayerControls(You, HasSubType("Island")).checkState(g, ctx) {
		t.Fatal("returned Island kept its former controller")
	}
}

func TestPlayerControlsDeclaration(t *testing.T) {
	condition := PlayerControls(ActivePlayer, IsLand)
	card := NewInstant("Watch", "", WithAbility(Spell(IfState(condition, GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if meta := InspectDefinition(condition); meta.Role != "state-condition" || meta.Text == "" {
		t.Fatalf("inspection = %+v", meta)
	}
	static := NewArtifact("Static", "", WithAbility(Static(While(PlayerControls(You, IsLand), ModifyPT(This, Fixed(1), Fixed(0))))))
	if err := ValidateCardDefinition(static.Name(), static); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []StateCondition{PlayerControls(nil, IsLand), PlayerControls(DeclarePlayerTarget("player"), IsLand)} {
		invalid := NewInstant("Invalid", "", WithAbility(Spell(IfState(bad, GainLifeFor(You, Fixed(1))))))
		if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
			t.Fatalf("invalid condition accepted: %+v", InspectDefinition(bad))
		}
	}
}

func TestPlayerControlsResolutionReadsCurrentState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Land")
	g.PutOnBattlefield(land, b.PlayerID())
	effect := IfState(PlayerControls(ActivePlayer, IsLand), GainLifeFor(You, Fixed(2)))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	g.SetActivePlayerIndex(0)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 {
		t.Fatal("inactive player's land satisfied active-player condition")
	}
	g.SetActivePlayerIndex(1)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatalf("life totals = %d, %d", a.Life(), b.Life())
	}
}
