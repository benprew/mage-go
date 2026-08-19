package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func newObjectColorGame() (*Game, *BasePlayer, *BasePlayer) {
	a := NewBasePlayer("A")
	b := NewBasePlayer("B")
	return NewGame(a, b), a, b
}

func pushCreatureSpell(g *Game, player Player, name, cost string) *StackObject {
	card := NewCreature(name, cost, 2, 2)
	card.SetOwner(player.PlayerID())
	obj := &StackObject{
		ID:         uuid.New(),
		Card:       card,
		Controller: player.PlayerID(),
		SourceID:   card.ID(),
	}
	g.pushStack(obj)
	return obj
}

func TestChangeColorEffectChangesSpellAndCarriesToPermanent(t *testing.T) {
	g, a, _ := newObjectColorGame()
	spell := pushCreatureSpell(g, a, "Green Spell", "{G}")

	if err := ApplyEffect(g, ChangeColorEffect(Blue), uuid.New(), a.PlayerID(), []uuid.UUID{spell.ID}); err != nil {
		t.Fatalf("ChangeColorEffect: %v", err)
	}
	if got := g.EffectiveColors(spell.SourceID); !slices.Equal(got, []Color{Blue}) {
		t.Fatalf("spell colors = %v, want [Blue]", got)
	}

	g.ResolveStack()
	permanent := g.FindPermanent(spell.SourceID)
	if permanent == nil {
		t.Fatal("creature spell did not resolve to a permanent")
	}
	if got := permanent.Colors(); !slices.Equal(got, []Color{Blue}) {
		t.Fatalf("resolved permanent colors = %v, want [Blue]", got)
	}

	g.ExilePermanent(permanent)
	card, ok := g.RemoveFromExile(spell.SourceID)
	if !ok {
		t.Fatal("remove resolved card from exile")
	}
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if got := returned.Colors(); !slices.Equal(got, []Color{Green}) {
		t.Fatalf("new battlefield object colors = %v, want printed [Green]", got)
	}
}

func TestSpellColorOverrideCloneAndCopyIsolation(t *testing.T) {
	g, a, _ := newObjectColorGame()
	spell := pushCreatureSpell(g, a, "Green Spell", "{G}")
	if !g.ChangeObjectColor(spell.ID, Blue) {
		t.Fatal("ChangeObjectColor did not find spell")
	}

	clone := g.Clone()
	if !clone.ChangeObjectColor(spell.ID, Red) {
		t.Fatal("clone ChangeObjectColor did not find spell")
	}
	if got := g.EffectiveColors(spell.SourceID); !slices.Equal(got, []Color{Blue}) {
		t.Fatalf("original spell colors after clone mutation = %v, want [Blue]", got)
	}
	if got := clone.EffectiveColors(spell.SourceID); !slices.Equal(got, []Color{Red}) {
		t.Fatalf("clone spell colors = %v, want [Red]", got)
	}

	spellCopy := g.CopyStackObjectDirect(spell, a.PlayerID(), false)
	if got := g.EffectiveColors(spellCopy.SourceID); !slices.Equal(got, []Color{Green}) {
		t.Fatalf("copied spell colors = %v, want printed [Green]", got)
	}
}

func TestPermanentColorOverrideCloneIsolation(t *testing.T) {
	g, a, _ := newObjectColorGame()
	card := NewCreature("Green Permanent", "{G}", 2, 2)
	card.SetOwner(a.PlayerID())
	permanent := g.PutOnBattlefield(card, a.PlayerID())
	g.ChangeObjectColor(permanent.ID(), Blue)

	clone := g.Clone()
	clone.ChangeObjectColor(permanent.ID(), Red)
	if got := g.EffectiveColors(permanent.ID()); !slices.Equal(got, []Color{Blue}) {
		t.Fatalf("original permanent colors after clone mutation = %v, want [Blue]", got)
	}
	if got := clone.EffectiveColors(permanent.ID()); !slices.Equal(got, []Color{Red}) {
		t.Fatalf("clone permanent colors = %v, want [Red]", got)
	}
}

func TestProtectionUsesEffectiveSourceColor(t *testing.T) {
	g, a, b := newObjectColorGame()
	sourceCard := NewCreature("Source", "{R}", 3, 6)
	sourceCard.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	protectedCard := NewCreature("Protected", "{G}", 2, 10,
		WithAbility(ProtectionFromColor(Red)))
	protectedCard.SetOwner(b.PlayerID())
	protected := g.PutOnBattlefield(protectedCard, b.PlayerID())

	if protected.CanBeTargetedBy(source.Card, a.PlayerID(), g) {
		t.Fatal("protection allowed targeting by red source")
	}
	g.DealDamageToPermanent(protected, 3, source.ID())
	if protected.Damage != 0 {
		t.Fatalf("protected damage = %d, want 0", protected.Damage)
	}
	if CanBlock(source, protected, g) {
		t.Fatal("red source blocked creature with protection from red")
	}
	fightPermanents(&EffectContext{Game: g, Controller: a.PlayerID()}, source, protected)
	if protected.Damage != 0 {
		t.Fatalf("protected fight damage = %d, want 0", protected.Damage)
	}

	if !g.ChangeObjectColor(source.ID(), Blue) {
		t.Fatal("ChangeObjectColor did not find permanent")
	}
	if !protected.CanBeTargetedBy(source.Card, a.PlayerID(), g) {
		t.Fatal("protection from red blocked blue-recolored source")
	}
	if !CanBlock(source, protected, g) {
		t.Fatal("blue-recolored source could not block creature with protection from red")
	}
	g.DealDamageToPermanent(protected, 3, source.ID())
	if protected.Damage != 3 {
		t.Fatalf("damage from blue-recolored source = %d, want 3", protected.Damage)
	}
}

func TestProtectionUsesRecoloredSpellAtResolution(t *testing.T) {
	g, a, b := newObjectColorGame()
	protectedCard := NewCreature("Protected", "{G}", 2, 10,
		WithAbility(ProtectionFromColor(Red)))
	protectedCard.SetOwner(b.PlayerID())
	protected := g.PutOnBattlefield(protectedCard, b.PlayerID())
	spellCard := NewInstant("Blue Damage", "{U}", NewSpellAbility(DealDamage(Fixed(3))))
	spellCard.SetOwner(a.PlayerID())
	resolved := false
	spell := &StackObject{
		ID:         uuid.New(),
		Card:       spellCard,
		Controller: a.PlayerID(),
		SourceID:   spellCard.ID(),
		Effects: []Effect{
			DealDamage(Fixed(3)),
			FuncEffect("record resolution", EffectProperties{}, func(*Game, uuid.UUID, uuid.UUID, []uuid.UUID) error {
				resolved = true
				return nil
			}),
		},
		Targets:     []uuid.UUID{protected.ID()},
		TargetSpecs: []Target{TargetCreature()},
	}
	g.pushStack(spell)
	if !g.ChangeObjectColor(spell.ID, Red) {
		t.Fatal("ChangeObjectColor did not find damage spell")
	}
	g.ResolveStack()
	if protected.Damage != 0 {
		t.Fatalf("damage from red-recolored spell = %d, want 0", protected.Damage)
	}
	if resolved {
		t.Fatal("red-recolored spell resolved despite protection making its target illegal")
	}
}

func TestProtectionFromCardTypeStillUsesItsFilter(t *testing.T) {
	g, a, b := newObjectColorGame()
	sourceCard := NewCreature("Source", "{U}", 2, 2)
	sourceCard.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	protectedCard := NewCreature("Protected", "{G}", 2, 2,
		WithAbility(ProtectionFromCardType(TypeCreature)))
	protectedCard.SetOwner(b.PlayerID())
	protected := g.PutOnBattlefield(protectedCard, b.PlayerID())

	g.ChangeObjectColor(source.ID(), Red)
	if !protected.HasProtectionFromInGame(source.Card, g) {
		t.Fatal("protection from creatures stopped matching a recolored creature")
	}
}
