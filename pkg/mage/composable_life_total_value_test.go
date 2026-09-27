package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestLifeTotalOfReadsCurrentPlayerAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	a.SetLife(7)
	b.SetLife(13)
	if got := LifeTotalOf(You).resolveValue(ctx); got != 7 {
		t.Fatalf("controller life = %d, want 7", got)
	}
	g.SetActivePlayerIndex(1)
	if got := LifeTotalOf(ActivePlayer).resolveValue(ctx); got != 13 {
		t.Fatalf("active player life = %d, want 13", got)
	}
	clone := g.Clone()
	clone.GetPlayer(b.PlayerID()).SetLife(4)
	if got := LifeTotalOf(ActivePlayer).resolveValue(&EffectContext{Game: clone, Controller: a.PlayerID()}); got != 4 {
		t.Fatalf("clone life = %d, want 4", got)
	}
	if got := LifeTotalOf(ActivePlayer).resolveValue(ctx); got != 13 {
		t.Fatalf("original life = %d, want 13", got)
	}
	if got := LifeTotalOf(ControllerOf(This)).resolveValue(&EffectContext{Game: g, SourceID: uuid.New(), Controller: a.PlayerID()}); got != 0 {
		t.Fatalf("absent player life = %d, want 0", got)
	}
}

func TestLifeTotalOfSequenceTargetAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
	ability := Spell(Sequence(LoseLifeFor(target, Fixed(3)), GainLifeFor(You, LifeTotalOf(target))), WithTargets(target))
	card := NewSorcery("Synthetic life total", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	info := InspectDefinition(LifeTotalOf(target))
	if info.Role != "resolution-value" || len(info.Children) != 1 {
		t.Fatalf("inspection = %+v", info)
	}
	obj := &StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	g.ResolveStack()
	if got := a.Life(); got != 37 {
		t.Fatalf("life after ordered actions = %d, want 37", got)
	}
	if got := b.Life(); got != 17 {
		t.Fatalf("target life = %d, want 17", got)
	}
	for _, subject := range []PlayerRef{nil, DeclarePlayerTarget("undeclared"), FromPlayer(BindPlayer("missing"))} {
		invalid := NewSorcery("Invalid", "", WithAbility(Spell(GainLifeFor(You, LifeTotalOf(subject)))))
		if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
			t.Fatalf("accepted invalid player reference %T", subject)
		}
	}
}
