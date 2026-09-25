package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestUnionDestructionRegeneratesOnce(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Artifact enchantment creature", "", 2, 2)
	card.AddType(core.TypeArtifact)
	card.AddType(core.TypeEnchantment)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.AddRegenerationShield(p.ID())
	clone := g.Clone()
	if err := DestroyEach(MatchingPermanents(Or(IsArtifact, IsCreature, IsEnchantment))).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	got := clone.FindPermanent(p.ID())
	if got == nil || !got.Tapped {
		t.Fatal("one shield must replace the entire destruction of a permanent with three matching types")
	}
	if g.FindPermanent(p.ID()) == nil || p.Tapped {
		t.Fatal("destruction changed the original game")
	}
	clone.DestroyPermanent(got)
	if clone.FindPermanent(p.ID()) != nil {
		t.Fatal("regeneration shield was not consumed")
	}
}

func TestUnionDestructionIsSimultaneous(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	creature := NewCreature("Creature", "", 2, 2)
	artifact := NewArtifact("Artifact", "")
	observer := NewLand("Observer", WithAbility(NewStateTriggered(false, Sacrifice(This)).SetCondition(
		func(_ *core.GameEvent, game GameReader, _, _ uuid.UUID) bool {
			return game.FindPermanent(creature.ID()) == nil && game.FindPermanent(artifact.ID()) != nil
		})))
	for _, card := range []Card{creature, artifact, observer} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	if err := DestroyEach(MatchingPermanents(Or(IsArtifact, IsCreature, IsEnchantment))).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(creature.ID()) != nil || g.FindPermanent(artifact.ID()) != nil {
		t.Fatal("matching permanents survived")
	}
	if g.FindPermanent(observer.ID()) == nil {
		t.Fatal("state trigger observed separate destruction batches")
	}
}

func TestUnionDestructionMembership(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	effect := DestroyEach(MatchingPermanents(Or(IsArtifact, IsCreature, IsEnchantment)))
	if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	var destroyed, survivors []*Permanent
	for _, owner := range []*BasePlayer{a, b} {
		for _, typ := range []core.CardType{core.TypeArtifact, core.TypeCreature, core.TypeEnchantment, core.TypeLand, core.TypePlaneswalker} {
			for _, status := range []string{"ordinary", "indestructible", "phased"} {
				card := NewArtifact(status, "")
				card.types = []core.CardType{typ}
				card.power, card.toughness = 2, 2
				card.SetOwner(owner.PlayerID())
				p := g.PutOnBattlefield(card, owner.PlayerID())
				if status == "indestructible" {
					p.GrantBaseAttr(core.Indestructible)
				}
				if status == "phased" {
					p.PhasedOut = true
				}
				if status == "ordinary" && typ != core.TypeLand && typ != core.TypePlaneswalker {
					destroyed = append(destroyed, p)
				} else {
					survivors = append(survivors, p)
				}
			}
		}
	}
	clone := g.Clone()
	if err := effect.Apply(&EffectContext{Game: clone, Controller: b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	for _, p := range destroyed {
		if clone.FindPermanent(p.ID()) != nil || g.FindPermanent(p.ID()) == nil {
			t.Fatal("wrong membership or clone isolation")
		}
	}
	for _, p := range survivors {
		if clone.FindPermanentIncludingPhased(p.ID()) == nil {
			t.Fatal("destroyed an excluded permanent")
		}
	}
}
