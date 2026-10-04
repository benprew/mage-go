package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTemporaryCardTypeCapture(t *testing.T) {
	for _, change := range []string{"none", "phase", "return", "absent", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewLand("Subject")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			id := p.ID()
			if change == "absent" {
				g.PhaseOut(id)
			}
			body := Until(core.EndOfTurn, AddCardType(This, core.TypeArtifact), AddCardType(This, core.TypeCreature), AddSubtypeFamily(This, SubtypeCreature, "Assembly-Worker"), SetBasePowerToughness(This, Fixed(2), Fixed(2)), ModifyPT(This, Fixed(1), Fixed(1)))
			if err := ApplyEffect(g, body, id, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "phase", "absent":
				g.PhaseIn(g.PhaseOut(id))
				g.PhaseIn([]uuid.UUID{id})
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(id))
				a.RemoveFromHand(card.ID())
				id = g.PutOnBattlefield(card, a.PlayerID()).ID()
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: id, TargetID: id, ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.ApplyContinuousEffects()
			animated := change != "return" && change != "absent"
			p = g.FindPermanent(id)
			if !p.HasType(core.TypeLand) || p.HasType(core.TypeArtifact) != animated || p.HasType(core.TypeCreature) != animated || p.HasSubType("Assembly-Worker") != animated {
				t.Fatal("type addition lost identity or existing type")
			}
			if animated && (p.CurrentPower(g) != 3 || p.CurrentToughness(g) != 3 || !p.HasAttr(core.AttrHasPowerToughness)) {
				t.Fatal("animation did not apply across layers")
			}
			clone := g.Clone()
			clone.GetEffects().RemoveEndOfTurn()
			clone.ApplyContinuousEffects()
			if clone.FindPermanent(id).HasType(core.TypeCreature) || clone.FindPermanent(id).HasType(core.TypeArtifact) || g.FindPermanent(id).HasType(core.TypeCreature) != animated {
				t.Fatal("cleanup changed another game")
			}
		})
	}
}

func TestTemporaryCardTypeDefinitionValidation(t *testing.T) {
	target := DeclarePermanentTarget("subject", AnyPermanent)
	for _, kind := range []core.CardType{core.TypeCreature, core.TypeArtifact, core.TypeLand, core.TypeEnchantment} {
		body := Until(core.EndOfTurn, AddCardType(target, kind))
		if err := ValidateCardDefinition("Valid", NewInstant("Valid", "", WithAbility(Spell(body, WithTargets(target))))); err != nil {
			t.Fatal(err)
		}
		if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithAbility(Spell(body)))); err == nil {
			t.Fatal("undeclared type subject accepted")
		}
	}
	for _, kind := range []core.CardType{core.TypeInstant, core.TypeSorcery, core.TypePlaneswalker, core.CardType(99)} {
		if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithAbility(Spell(Until(core.EndOfTurn, AddCardType(This, kind)))))); err == nil {
			t.Fatal("nonpermanent type accepted")
		}
	}
}

func TestTemporaryCardTypeKinds(t *testing.T) {
	for _, kind := range []core.CardType{core.TypeCreature, core.TypeArtifact, core.TypeLand, core.TypeEnchantment} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 3, WithKeyword(core.Flying), WithSubTypes("Human")), a.PlayerID())
		if err := ApplyEffect(g, Until(core.EndOfTurn, AddCardType(This, kind)), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		p = g.FindPermanent(p.ID())
		if !p.HasType(kind) || !p.HasType(core.TypeCreature) || !p.HasSubType("Human") || !p.HasKeyword(core.Flying) || p.CurrentToughness(g) != 3 {
			t.Fatal("type addition replaced existing characteristics")
		}
		g.GetEffects().RemoveEndOfTurn()
		g.ApplyContinuousEffects()
		if kind != core.TypeCreature && g.FindPermanent(p.ID()).HasType(kind) {
			t.Fatal("type grant survived expiry")
		}
	}
}
