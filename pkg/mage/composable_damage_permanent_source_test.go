package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageSourceOfTargetKeepsDepartedIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	oldCard := NewCreature("Old source", "", 2, 2, WithKeyword(Lifelink))
	old := g.PutOnBattlefield(oldCard, a.PlayerID())
	target := DeclarePermanentTarget("source", IsCreature)
	recipient := BindPlayer("recipient")
	g.resolution.playerBindings = map[*PlayerBinding]uuid.UUID{recipient: b.PlayerID()}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{
		target.targetDeclaration: {{id: old.ID(), incarnation: old.incarnationID}},
	}}
	g.RemoveFromBattlefield(old)
	g.PutOnBattlefield(oldCard, b.PlayerID())
	g.zones.ClearLKI()
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		ctx.Game = game
		if err := DealDamageFrom(DamageSourceOf(target), FromPlayer(recipient), Fixed(2)).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if game.GetPlayer(a.PlayerID()).Life() != 22 {
			t.Fatalf("old source lifelink and controller were lost: %d", game.GetPlayer(a.PlayerID()).Life())
		}
	}
	if a.Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 22 {
		t.Fatal("clone changed the original result")
	}
}

func TestDamageSourceOfAbsentPhasedAndDeclaration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 2), a.PlayerID())
	ref := DeclarePermanentTarget("source", IsCreature)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{
		ref.targetDeclaration: {{id: p.ID(), incarnation: p.incarnationID}},
	}}
	g.PhaseOut(p.ID())
	if err := DealDamageFrom(DamageSourceOf(ref), You, Fixed(2)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 {
		t.Fatal("phased source dealt damage")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	if err := DealDamageFrom(DamageSourceOf(ref), You, Fixed(2)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 18 {
		t.Fatal("active source did not deal damage")
	}
	valid := NewInstant("Valid", "", Spell(DealDamageFrom(DamageSourceOf(ref), You, Fixed(1)), WithTargets(ref)))
	if err := ValidateCardDefinition("Valid", valid); err != nil {
		t.Fatal(err)
	}
	missing := NewInstant("Missing", "", Spell(DealDamageFrom(DamageSourceOf(ref), You, Fixed(1))))
	if err := ValidateCardDefinition("Missing", missing); err == nil || !strings.Contains(err.Error(), "target binding requires") {
		t.Fatalf("missing declaration: %v", err)
	}
	nilRef := NewInstant("Nil", "", Spell(DealDamageFrom(DamageSourceOf(nil), You, Fixed(1))))
	if err := ValidateCardDefinition("Nil", nilRef); err == nil {
		t.Fatal("accepted nil damage source permanent")
	}
}

func TestDamageSourceOfSpellTargetAndSavedSnapshot(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Borrowed source", "", 2, 3, WithKeyword(Lifelink)), b.PlayerID())
	target := DeclarePermanentTarget("source", IsCreature)
	spell := NewInstant("Borrowed damage", "", Spell(DealDamageFrom(DamageSourceOf(target), You, Fixed(2)), WithTargets(target)))
	spell.SetOwner(a.PlayerID())
	ability := spell.Abilities()[0].(*ActionDefinition)
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{source.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	clone := g.Clone()
	g.ResolveStackObject(obj)
	if a.Life() != 18 || b.Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("spell used its own controller instead of the referenced source")
	}
	clone.ResolveStackObject(clone.StackPeek())
	if clone.GetPlayer(a.PlayerID()).Life() != 18 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatal("clone lost target source identity")
	}

	saved := BindPermanent("saved source")
	value := capturedPermanent{id: source.ID(), incarnation: source.incarnationID}
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{saved: value}
	g.RemoveFromBattlefield(source)
	returned := g.PutOnBattlefield(source.Card, a.PlayerID())
	if err := DealDamageFrom(DamageSourceOf(From(saved)), You, Fixed(1)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 17 || b.Life() != 23 || returned.ControllerID() != a.PlayerID() {
		t.Fatal("saved damage source followed returned card")
	}
}
