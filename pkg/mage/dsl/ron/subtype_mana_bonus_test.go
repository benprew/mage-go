package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSubtypeManaBonusRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, set, "Gauntlet of Might")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast succeeded")
			}
			a.ManaPool().Add(core.Colorless, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			red := g.PutOnBattlefield(mage.NewCreature("Red", "{R}", 2, 2), b.PlayerID())
			green := g.PutOnBattlefield(mage.NewCreature("Green", "{G}", 2, 2), a.PlayerID())
			staticPTCheck(t, g, red, 3, 3)
			staticPTCheck(t, g, green, 2, 2)
			for _, player := range []*mage.BasePlayer{a, b} {
				land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), player.PlayerID())
				if err := g.TapForMana(player.PlayerID(), land.ID()); err != nil {
					t.Fatal(err)
				}
				if player.ManaPool().Count(core.Red) != 2 {
					t.Fatal("wrong recipient or amount")
				}
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("mana bonus used stack")
			}
		})
	}
	t.Run("High Tide", func(t *testing.T) {
		g, a, b := ronGame()
		card := repositoryCard(t, "FEM", "High Tide")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
			t.Fatal("unpaid cast succeeded")
		}
		a.ManaPool().Add(core.Blue, 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		before := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
		if err := g.TapForMana(b.PlayerID(), before.ID()); err != nil {
			t.Fatal(err)
		}
		if b.ManaPool().Count(core.Blue) != 1 {
			t.Fatal("bonus started before resolution")
		}
		g.ResolveStack()
		g.UntapPermanent(before)
		b.ManaPool().Clear()
		for _, player := range []*mage.BasePlayer{a, b} {
			land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), player.PlayerID())
			if err := g.AutoTapForCost(player.PlayerID(), core.ManaCost{Blue: 2}); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(land.ID()).Tapped && !before.Tapped {
				t.Fatal("payment did not tap a land")
			}
			if player.ManaPool().Count(core.Blue) != 2 {
				t.Fatal("temporary planning and production disagree")
			}
		}
		g.PutTriggersOnStack()
		if g.StackSize() != 0 {
			t.Fatal("temporary mana bonus used stack")
		}
		clone := g.Clone()
		clone.RunStepWithPriority(core.Cleanup)
		g.UntapPermanent(before)
		b.ManaPool().Clear()
		if err := g.TapForMana(b.PlayerID(), before.ID()); err != nil {
			t.Fatal(err)
		}
		if b.ManaPool().Count(core.Blue) != 2 {
			t.Fatal("clone cleanup changed original")
		}
		g.RunStepWithPriority(core.Cleanup)
		g.UntapPermanent(before)
		if err := g.TapForMana(b.PlayerID(), before.ID()); err != nil {
			t.Fatal(err)
		}
		if b.ManaPool().Count(core.Blue) != 1 {
			t.Fatal("bonus survived cleanup")
		}
	})
}

func TestRONSubtypeManaBonusInvalid(t *testing.T) {
	valid := `TriggerA(WhenAPermanentIsTappedForMana(IsLandType(Mountain)),ActionList([PlayerAction(ControllerOfPermanent(Trigger_ThatPermanent),AddMana(ManaProduceR))]))`
	temporary := `SpellActions(ActionList([CreateTriggerUntil(WhenAPlayerTapsAPermanentForMana(AnyPlayer,IsLandType(Island)),ActionList([AddMana(ManaProduceU)]),UntilEndOfTurn)]))`
	for _, rule := range []string{
		strings.Replace(valid, "Mountain", "UnknownLand", 1), strings.Replace(valid, "Mountain", "Goblin", 1), strings.Replace(valid, "Mountain", "Mountain()", 1), strings.Replace(valid, "IsLandType(Mountain)", "IsLandType()", 1), strings.Replace(valid, "IsLandType(Mountain)", "IsLandType(Mountain,Island)", 1), strings.Replace(valid, "ControllerOfPermanent(Trigger_ThatPermanent)", "You", 1), strings.Replace(valid, "ManaProduceR", "AnyManaColor", 1), strings.Replace(valid, "AddMana(ManaProduceR)", "AddManaRepeated(ManaProduceR,Integer(2))", 1), strings.Replace(valid, "PlayerAction(ControllerOfPermanent(Trigger_ThatPermanent),AddMana(ManaProduceR))", "AddMana(ManaProduceR)", 1), strings.Replace(temporary, "AnyPlayer,", "Opponent,", 1), strings.Replace(temporary, "UntilEndOfTurn", "UntilEndOfCombat", 1), strings.Replace(temporary, "ManaProduceU", "Trigger_AnyManaTypeProduced", 1), strings.Replace(temporary, "AddMana(ManaProduceU)", "GainLife(Integer(1))", 1), strings.Replace(temporary, "IsLandType(Island)", "IsLandType(Goblin)", 1), strings.Replace(temporary, "AddMana(ManaProduceU)", "AddMana(ManaProduceU),DrawACard", 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(rule, "TriggerA") {
			nodes, err = Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONSubtypeManaBonusPrintedRevision(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Gauntlet of Might"), a.PlayerID())
	sibling := g.PutOnBattlefield(repositoryCard(t, "LEA", "Gauntlet of Might"), a.PlayerID())
	var id = source.RuntimeAbilities[1].AbilityID()
	change, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.IslandLandType)
	if err := g.RevisePrintedManaTapTrigger(source, sibling.RuntimeAbilities[1].AbilityID(), change); err == nil {
		t.Fatal("foreign declaration accepted")
	}
	if err := g.RevisePrintedManaTapTrigger(source, id, change); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	next, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	if err := clone.RevisePrintedManaTapTrigger(clone.FindPermanent(source.ID()), id, next); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(source.ID()).RuntimeAbilities[1].AbilityID() != id {
		t.Fatal("active identity changed")
	}
	island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	if err := g.TapForMana(b.PlayerID(), island.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Blue) != 1 || b.ManaPool().Count(core.Red) != 1 {
		t.Fatal("revised subtype or fixed mana symbol changed")
	}
	mountain := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
	b.ManaPool().Clear()
	if err := g.TapForMana(b.PlayerID(), mountain.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Red) != 2 {
		t.Fatal("sibling declaration changed")
	}
	phased := g.PhaseOut(source.ID())
	g.UntapPermanent(island)
	b.ManaPool().Clear()
	if err := g.TapForMana(b.PlayerID(), island.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Red) != 0 {
		t.Fatal("phased bonus remained")
	}
	g.PhaseIn(phased)
	remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), a.PlayerID())
	removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
	removal.SetSourceID(remover.ID())
	g.AddContinuousEffect(removal)
	g.ApplyContinuousEffects()
	g.UntapPermanent(island)
	b.ManaPool().Clear()
	if err := g.TapForMana(b.PlayerID(), island.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Red) != 0 {
		t.Fatal("removed bonus remained")
	}
	g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
	g.ApplyContinuousEffects()
	g.UntapPermanent(island)
	b.ManaPool().Clear()
	if err := g.AutoTapForCost(b.PlayerID(), core.ManaCost{Blue: 1, Red: 1}); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Red) != 1 {
		t.Fatal("restoration lost revised bonus")
	}
	card := source.Card
	g.RemoveFromBattlefield(source)
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.RevisePrintedManaTapTrigger(source, id, change); err == nil {
		t.Fatal("old visit accepted")
	}
	g.UntapPermanent(island)
	b.ManaPool().Clear()
	if err := g.TapForMana(b.PlayerID(), island.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Red) != 0 {
		t.Fatal("returned source kept revision")
	}
	if returned.ID() != source.ID() {
		t.Fatal("physical card identity changed")
	}
}

func TestRONSubtypeManaBonusTemporaryStacking(t *testing.T) {
	g, a, b := ronGame()
	for range 2 {
		card := repositoryCard(t, "FEM", "High Tide")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Blue, 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Mana Flare"), a.PlayerID())
	land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	if err := g.AutoTapForCost(b.PlayerID(), core.ManaCost{Blue: 4}); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Blue) != 4 {
		t.Fatal("bonuses did not stack")
	}
	g.RunStepWithPriority(core.Cleanup)
	g.UntapPermanent(land)
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Blue) != 2 {
		t.Fatal("cleanup removed persistent bonus or kept temporary bonuses")
	}
}

func TestRONSubtypeManaBonusLiveTypes(t *testing.T) {
	for _, name := range []string{"Gauntlet of Might", "High Tide"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			subtype, color, set, landName := "Mountain", core.Red, "LEA", "Badlands"
			if name == "High Tide" {
				subtype, color, set, landName = "Island", core.Blue, "FEM", "Tropical Island"
			}
			card := repositoryCard(t, set, name)
			if name == "Gauntlet of Might" {
				g.PutOnBattlefield(card, a.PlayerID())
			} else {
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(core.Blue, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
			}
			landCard := repositoryCard(t, "LEA", landName)
			land := g.PutOnBattlefield(landCard, b.PlayerID())
			if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
				t.Fatal(err)
			}
			if b.ManaPool().Count(color) < 1 || b.ManaPool().TotalMana() != 2 {
				t.Fatal("nonbasic matching land did not get one bonus")
			}
			g.UntapPermanent(land)
			b.ManaPool().Clear()
			g.AddContinuousEffect(mage.BecomesBasicLandTargetEffect(land.ID(), core.Indefinite, "Forest"))
			g.ApplyContinuousEffects()
			if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
				t.Fatal(err)
			}
			if b.ManaPool().TotalMana() != 1 || b.ManaPool().Count(core.Green) != 1 {
				t.Fatal("bonus read printed subtype instead of current subtype")
			}
			other := g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), b.PlayerID())
			g.AddContinuousEffect(mage.BecomesBasicLandTargetEffect(other.ID(), core.Indefinite, subtype))
			g.ApplyContinuousEffects()
			b.ManaPool().Clear()
			if err := g.AutoTapForCost(b.PlayerID(), core.ManaCost{Generic: 2}); err != nil {
				t.Fatal(err)
			}
			if b.ManaPool().Count(color) != 2 {
				t.Fatal("new matching subtype did not supply planned bonus")
			}
			g.PhaseOut(other.ID())
			b.ManaPool().Clear()
			if g.CanAfford(b.PlayerID(), core.ManaCost{Generic: 2}, nil) {
				t.Fatal("phased subject supplied mana")
			}
		})
	}
}

func TestRONSubtypeManaBonusRecipients(t *testing.T) {
	g, a, b := ronGame()
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Gauntlet of Might"), a.PlayerID())
	card := repositoryCard(t, "FEM", "High Tide")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	land := g.PutOnBattlefield(mage.NewLand("Shared", mage.WithSubTypes("Mountain", "Island"), mage.WithAbility(mage.Activate(mage.Costs(mage.TapCost(mage.This)), mage.AddManaAmount(core.Green, mage.Fixed(1)), mage.AnyPlayerMayActivate()))), b.PlayerID())
	a.ManaPool().Clear()
	b.ManaPool().Clear()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(core.Blue) != 1 || a.ManaPool().Count(core.Green) != 1 || a.ManaPool().Count(core.Red) != 0 || b.ManaPool().Count(core.Red) != 1 {
		t.Fatal("controller and activating player bonuses used the same recipient")
	}
}
