package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestLifeGainedThisTurnValueActualOutcomeAndPlayer(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewInstant("Gain", "")
	value := LifeGainedThisTurn(You)
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}
	if got := value.resolveValue(ctx); got != 0 {
		t.Fatalf("initial gain = %d", got)
	}
	g.effects.AddReplacement(&lifeOutcomeReplacement{consume: true})
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(3)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := value.resolveValue(ctx); got != 0 {
		t.Fatalf("consumed gain = %d", got)
	}
	g.effects.replacements = nil
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(0)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.effects.AddReplacement(&lifeOutcomeReplacement{amount: 5})
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(2)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, LoseLifeFor(You, Fixed(4)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.effects.replacements = nil
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(2)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := value.resolveValue(&EffectContext{Game: g, SourceID: uuid.New(), Controller: a.PlayerID()}); got != 7 {
		t.Fatalf("controller gain after source departure = %d, want 7", got)
	}
	if got := value.resolveValue(&EffectContext{Game: g, Controller: b.PlayerID()}); got != 0 {
		t.Fatalf("opponent gain = %d", got)
	}
	if got := value.resolveValue(&EffectContext{Game: g}); got != 0 {
		t.Fatalf("absent player gain = %d", got)
	}
	clone := g.Clone()
	clone.trackers.Turn.ResetForNewTurn(b.PlayerID())
	if got := value.resolveValue(&EffectContext{Game: clone, Controller: a.PlayerID()}); got != 0 {
		t.Fatalf("clone gain after reset = %d", got)
	}
	if got := value.resolveValue(ctx); got != 7 {
		t.Fatalf("original gain after clone reset = %d", got)
	}
}

func TestLifeGainedThisTurnValueResolutionAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewInstant("Synthetic life history", "", WithAbility(Spell(GainLifeFor(You, LifeGainedThisTurn(You)))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	target := DeclarePlayerTarget("recipient")
	targetAbility := Spell(GainLifeFor(You, LifeGainedThisTurn(target)), WithTargets(target))
	targeted := NewInstant("Target history", "", WithAbility(targetAbility))
	if err := ValidateCardDefinition(targeted.Name(), targeted); err != nil {
		t.Fatal(err)
	}
	for _, subject := range []PlayerRef{nil, DeclarePlayerTarget("undeclared"), FromPlayer(BindPlayer("missing"))} {
		bad := NewInstant("Invalid", "", WithAbility(Spell(GainLifeFor(You, LifeGainedThisTurn(subject)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatalf("accepted invalid player reference %T", subject)
		}
	}
	info := InspectDefinition(LifeGainedThisTurn(You))
	if info.Role != "resolution-value" || len(info.Children) != 1 {
		t.Fatalf("inspection = %+v", info)
	}
	source := NewCreature("History source", "", 1, 1, WithAbility(At(EachEndStep, GainLifeFor(You, LifeGainedThisTurn(You)))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("missing triggered ability")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	g.ExilePermanent(g.FindPermanent(source.ID()))
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(3)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(2)), source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := LifeGainedThisTurn(target).resolveValue(&EffectContext{
		Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{
			target.targetDeclaration: {{id: b.PlayerID()}},
		},
	}); got != 2 {
		t.Fatalf("target player's gain = %d, want 2", got)
	}
	if got := LifeGainedThisTurn(target).resolveValue(&EffectContext{Game: g, Controller: a.PlayerID()}); got != 0 {
		t.Fatalf("absent target gain = %d", got)
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 26 || clone.GetPlayer(b.PlayerID()).Life() != 24 {
		t.Fatalf("clone life = %d, %d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life())
	}
	if a.Life() != 23 || b.Life() != 22 {
		t.Fatal("clone changed original game")
	}
	g.ResolveStack()
	if a.Life() != 26 || b.Life() != 24 {
		t.Fatalf("original life = %d, %d", a.Life(), b.Life())
	}
}
