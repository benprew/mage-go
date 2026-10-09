package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTemporaryManaBonusContract(t *testing.T) {
	for _, path := range []string{"manual", "ability", "planned"} {
		t.Run(path, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			land := g.PutOnBattlefield(NewLand("Mixed", WithSubTypes("Desert"), WithMultiManaAbility(ManaProduction{Color: core.Green, Amount: 2}, ManaProduction{Color: core.White, Amount: 1})), b.PlayerID())
			for range 2 {
				if err := ApplyEffect(g, LandManaBonusThisTurn(HasSubType("Desert"), core.Blue), uuid.New(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if err := g.AutoTapForCost(b.PlayerID(), core.ManaCost{Blue: 3}); err == nil {
				t.Fatal("impossible payment succeeded")
			}
			if land.Tapped || b.ManaPool().TotalMana() != 0 {
				t.Fatal("failed payment leaked state")
			}
			var err error
			switch path {
			case "manual":
				err = g.TapForMana(b.PlayerID(), land.ID())
			case "ability":
				err = g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 0, nil)
			case "planned":
				err = g.AutoTapForCost(b.PlayerID(), core.ManaCost{Blue: 2, Green: 2, White: 1})
			}
			if err != nil {
				t.Fatal(err)
			}
			if b.ManaPool().Count(core.Blue) != 2 || b.ManaPool().Count(core.Green) != 2 || b.ManaPool().Count(core.White) != 1 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("bonus amount or actor is wrong")
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("bonus used the stack")
			}
			clone := g.Clone()
			clone.doCleanupActions()
			if len(clone.mana.ManaBonuses(clone, land.ID())) != 0 || len(g.mana.ManaBonuses(g, land.ID())) != 2 {
				t.Fatal("cleanup shared registrations")
			}
		})
	}
}

func TestTemporaryManaBonusDeclarations(t *testing.T) {
	for _, color := range []core.Color{core.AnyColor, core.Color(-1)} {
		c := NewInstant("Invalid", "", WithAbility(Spell(LandManaBonusThisTurn(IsLand, color))))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("accepted invalid bonus color")
		}
	}
}

func TestTemporaryManaBonusMembershipAndTiming(t *testing.T) {
	for _, state := range []string{"later-entry", "nonland", "wrong-type", "zero", "no-tap", "tap-only", "source-left", "source-phased", "subject-phased", "subject-returned", "control", "removed-abilities", "type-change"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			if err := ApplyEffect(g, LandManaBonusThisTurn(HasSubType("Desert"), core.Blue), source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			card := NewLand("Subject", WithSubTypes("Desert"), WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(core.Green, Fixed(1)))))
			want := 1
			switch state {
			case "nonland":
				card = NewArtifact("Subject", "", WithSubTypes("Desert"), WithManaAbility(core.Green))
				want = 0
			case "wrong-type":
				card = NewLand("Subject", WithSubTypes("Locus"), WithManaAbility(core.Green))
				want = 0
			case "zero":
				card = NewLand("Subject", WithSubTypes("Desert"), WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(core.Green, Fixed(0)))))
				want = 0
			case "no-tap":
				card = NewLand("Subject", WithSubTypes("Desert"), WithAbility(Activate(Costs(), AddManaAmount(core.Green, Fixed(1)))))
				want = 0
			case "tap-only":
				want = 0
			}
			land := g.PutOnBattlefield(card, b.PlayerID())
			actor := b.PlayerID()
			switch state {
			case "source-left":
				g.RemoveFromBattlefield(source)
			case "source-phased":
				g.PhaseOut(source.ID())
			case "subject-phased":
				g.PhaseOut(land.ID())
				if len(g.mana.ManaBonuses(g, land.ID())) != 0 {
					t.Fatal("phased subject matched")
				}
				g.PhaseIn([]uuid.UUID{land.ID()})
			case "subject-returned":
				g.RemoveFromBattlefield(land)
				land = g.PutOnBattlefield(card, b.PlayerID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: land.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				g.ApplyContinuousEffects()
				actor = a.PlayerID()
			case "removed-abilities":
				g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			case "type-change":
				g.MutablePermanent(land.ID()).SubTypeOverride = []string{"Locus"}
				want = 0
			}
			if state == "tap-only" {
				g.TapPermanent(land)
			} else if err := g.ActivateAbilityByIndex(actor, land.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.GetPlayer(actor).ManaPool().Count(core.Blue) != want {
				t.Fatalf("bonus=%d want=%d", g.GetPlayer(actor).ManaPool().Count(core.Blue), want)
			}
		})
	}
}

func TestTemporaryManaBonusReplacementAndCastingRollback(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("Desert", WithSubTypes("Desert"), WithManaAbility(core.Green)), a.PlayerID())
	if err := ApplyEffect(g, LandManaBonusThisTurn(HasSubType("Desert"), core.Blue), uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, ReplaceTappedManaTypeForPlayerThisTurn(You, IsLand, core.White), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	spell := NewInstant("Unpayable", "{W}{U}{B}", WithAbility(Spell(GainLifeFor(You, Fixed(1)))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err == nil {
		t.Fatal("impossible cast succeeded")
	}
	if g.FindPermanent(land.ID()).Tapped || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 || len(a.Hand()) != 1 {
		t.Fatal("failed cast leaked state")
	}
	if err := g.AutoTapForCost(a.PlayerID(), core.ManaCost{White: 1, Blue: 1}); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(core.White) != 1 || a.ManaPool().Count(core.Blue) != 1 || a.ManaPool().Count(core.Green) != 0 {
		t.Fatal("replacement and fixed bonus disagree")
	}
}

func TestControllerManaBonusDeclarations(t *testing.T) {
	for _, color := range []core.Color{core.AnyColor, core.Color(-1)} {
		c := NewArtifact("Invalid", "", WithAbility(NewControllerManaBonusAbility(IsLand, color)))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("accepted invalid controller bonus color")
		}
	}
}

func TestControllerManaBonusRecipients(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewArtifact("Controller bonus", "", WithAbility(NewControllerManaBonusAbility(IsLand, core.Red))), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Legacy bonus", "", WithAbility(NewManaBonusAbility(IsLand, core.White))), a.PlayerID())
	land := g.PutOnBattlefield(NewLand("Shared Desert", WithSubTypes("Desert"), WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(core.Green, Fixed(1)), AnyPlayerMayActivate()))), b.PlayerID())
	if err := ApplyEffect(g, LandManaBonusThisTurn(HasSubType("Desert"), core.Blue), uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(core.Blue) != 1 || a.ManaPool().Count(core.Green) != 1 || a.ManaPool().Count(core.White) != 1 || a.ManaPool().Count(core.Red) != 0 || b.ManaPool().Count(core.Red) != 1 {
		t.Fatal("recipient contract changed")
	}
}
