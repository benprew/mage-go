package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextStaticAnimationLayerRecords(t *testing.T) {
	records := 0
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI", "LEG"} {
		for _, name := range []string{"Kormus Bell", "Living Lands", "Living Plane"} {
			if (set == "LEG") != (name == "Living Plane") {
				continue
			}
			records++
			t.Run(set+"/"+name, func(t *testing.T) {
				g, a, b := ronGame()
				printed, from := "Forest", core.ForestLandType
				if name == "Kormus Bell" {
					printed, from = "Swamp", core.SwampLandType
				}
				old := g.PutOnBattlefield(repositoryCard(t, "LEA", printed), a.PlayerID())
				island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
				mountain := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
				card := repositoryCard(t, set, name)
				source := g.PutOnBattlefield(card, a.PlayerID())
				id := textStaticID(t, source)
				first, _ := mage.ReplaceBasicLandWord(from, core.IslandLandType)
				second, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.MountainLandType)
				changes := []mage.TextChange{first}
				if name == "Kormus Bell" {
					color, _ := mage.ReplaceColorWord(core.Black, core.Red)
					changes = append(changes, color)
				}
				if err := g.AddPrintedStaticAnimationTextEffect(source, id, core.EndOfTurn, changes...); err != nil {
					t.Fatal(err)
				}
				if err := g.AddPrintedStaticAnimationTextEffect(g.FindPermanent(source.ID()), id, core.Indefinite, second); err != nil {
					t.Fatal(err)
				}
				all := name == "Living Plane"
				for range 3 {
					g.ApplyContinuousEffects()
					animationCheck(t, g, old, all)
					animationCheck(t, g, island, all)
					animationCheck(t, g, mountain, true)
					if name == "Kormus Bell" {
						animationCheck(t, g, mountain, true, core.Red)
					}
				}
				if textStaticID(t, g.FindPermanent(source.ID())) != id {
					t.Fatal("text changed declaration identity")
				}
				if err := g.RevisePrintedStaticAnimation(g.FindPermanent(source.ID()), id, first); err == nil {
					t.Fatal("direct revision changed layered text")
				}
				clone := g.Clone()
				clone.GetEffects().RemoveEndOfTurn()
				clone.ApplyContinuousEffects()
				animationCheck(t, clone, old, true)
				animationCheck(t, clone, mountain, all)
				animationCheck(t, g, mountain, true)
				if name == "Kormus Bell" {
					animationCheck(t, clone, old, true, core.Black)
				}
				late := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), a.PlayerID())
				animationCheck(t, g, late, true)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), late.ID(), 0, nil); err == nil {
					t.Fatal("animated land ignored summoning sickness")
				}
				phased := g.PhaseOut(late.ID())
				g.PhaseIn(phased)
				g.ApplyContinuousEffects()
				animationCheck(t, g, late, true)
				phased = g.PhaseOut(source.ID())
				g.ApplyContinuousEffects()
				animationCheck(t, g, mountain, false)
				if err := g.AddPrintedStaticAnimationTextEffect(source, id, core.EndOfTurn, first); err == nil {
					t.Fatal("phased source accepted")
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				g.PhaseIn(phased)
				g.ApplyContinuousEffects()
				animationCheck(t, g, old, true)
				animationCheck(t, g, mountain, all)
				if err := g.AddPrintedStaticAnimationTextEffect(g.FindPermanent(source.ID()), id, core.EndOfTurn, changes...); err != nil {
					t.Fatal(err)
				}
				animationCheck(t, g, island, true)
				remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Plains"), a.PlayerID())
				removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
				removal.SetSourceID(remover.ID())
				g.AddContinuousEffect(removal)
				animationCheck(t, g, island, true)
				if name == "Kormus Bell" {
					animationCheck(t, g, island, true, core.Red)
				}
				if err := g.AddPrintedStaticAnimationTextEffect(g.FindPermanent(source.ID()), id, core.EndOfTurn, first); err == nil {
					t.Fatal("removed declaration accepted")
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				animationCheck(t, g, old, true)
				g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
				animationCheck(t, g, old, true)
				if err := g.AddPrintedStaticAnimationTextEffect(g.FindPermanent(source.ID()), id, core.Indefinite, changes...); err != nil {
					t.Fatal(err)
				}
				g.BouncePermanentToHand(g.FindPermanent(island.ID()))
				if lki := g.LKI(island.ID()); lki == nil || lki.Power != 1 || lki.Toughness != 1 || !lki.Snapshot.HasType(core.TypeCreature) {
					t.Fatal("departure lost layered animation")
				}
				returnedLand := g.PutOnBattlefield(island.Card, b.PlayerID())
				animationCheck(t, g, returnedLand, true)
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				animationCheck(t, g, returnedLand, false)
				g.PutOnBattlefield(card, a.PlayerID())
				animationCheck(t, g, old, true)
				animationCheck(t, g, returnedLand, all)
				if name == "Kormus Bell" {
					animationCheck(t, g, old, true, core.Black)
				}
				if err := g.AddPrintedStaticAnimationTextEffect(source, id, core.EndOfTurn, first); err == nil {
					t.Fatal("returned source inherited old visit")
				}
			})
		}
	}
	if records != 13 {
		t.Fatal("record coverage", records)
	}
}

func TestRONTextStaticAnimationLayerDependencies(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		g, a, b := ronGame()
		land := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Strip Mine"), b.PlayerID())
		basic := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
		var animation, moon *mage.Permanent
		addAnimation := func() { animation = g.PutOnBattlefield(repositoryCard(t, "LEA", "Kormus Bell"), a.PlayerID()) }
		addMoon := func() { moon = g.PutOnBattlefield(repositoryCard(t, "DRK", "Blood Moon"), a.PlayerID()) }
		if reverse {
			addMoon()
			addAnimation()
		} else {
			addAnimation()
			addMoon()
		}
		change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
		if err := g.AddPrintedStaticAnimationTextEffect(animation, textStaticID(t, animation), core.EndOfTurn, change); err != nil {
			t.Fatal(err)
		}
		animationCheck(t, g, land, false)
		animationCheck(t, g, basic, true, core.Black)
		change, _ = mage.ReplaceBasicLandWord(core.MountainLandType, core.IslandLandType)
		if err := g.AddPrintedStaticLandTypeTextEffect(moon, textStaticID(t, moon), core.EndOfTurn, change); err != nil {
			t.Fatal(err)
		}
		for range 3 {
			g.ApplyContinuousEffects()
			animationCheck(t, g, land, true, core.Black)
			animationCheck(t, g, basic, true, core.Black)
		}
		g.GetEffects().RemoveEndOfTurn()
		g.ApplyContinuousEffects()
		animationCheck(t, g, land, false)
		animationCheck(t, g, basic, false)
	}
}

func TestRONTextStaticAnimationLayerBindings(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `EachPermanentLayerEffect(And([IsLandType(Forest),ControlledByAPlayer(SinglePlayer(You)),Other(ThisPermanent)]),[SetPT(PT(1,1)),AddCardtype(Creature)]),EachPermanentLayerEffect(IsLandType(Forest),[SetPT(PT(2,3)),AddCardtype(Creature)])`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	own := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), a.PlayerID())
	other := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	first, second := source.RuntimeAbilities[0].AbilityID(), source.RuntimeAbilities[1].AbilityID()
	change, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	if err := g.RevisePrintedStaticAnimation(source, first, change); err != nil {
		t.Fatal(err)
	}
	next, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.MountainLandType)
	if err := g.AddPrintedStaticAnimationTextEffect(g.FindPermanent(source.ID()), first, core.EndOfTurn, next); err != nil {
		t.Fatal(err)
	}
	animationCheck(t, g, own, false)
	clone := g.Clone()
	clone.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	animationCheck(t, clone, own, false)
	animationCheck(t, clone, other, true)
	animationCheck(t, g, other, false)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	animationCheck(t, g, own, true)
	animationCheck(t, g, other, false)
	if clone.FindPermanent(source.ID()).RuntimeAbilities[0].Controller() != b.PlayerID() || g.FindPermanent(source.ID()).RuntimeAbilities[0].Controller() != a.PlayerID() {
		t.Fatal("clone shared declaration context")
	}
	if err := g.AddPrintedStaticAnimationTextEffect(g.FindPermanent(source.ID()), second, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, own, 2, 3)
	staticPTCheck(t, g, other, 2, 3)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	animationCheck(t, g, own, true)
	animationCheck(t, g, other, false)
	foreign := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.AddPrintedStaticAnimationTextEffect(source, foreign.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, change); err == nil {
		t.Fatal("foreign declaration accepted")
	}
}

func TestRONTextStaticAnimationLayerBoundaries(t *testing.T) {
	TestRONTextStaticAnimationBoundaries(t)
	change, _ := mage.ReplaceColorWord(core.Black, core.Red)
	for _, name := range []string{"Bad Moon", "Holy Strength", "Blood Moon"} {
		set := "LEA"
		if name == "Blood Moon" {
			set = "DRK"
		}
		g, a, _ := ronGame()
		source := g.PutOnBattlefield(repositoryCard(t, set, name), a.PlayerID())
		if err := g.AddPrintedStaticAnimationTextEffect(source, textStaticID(t, source), core.EndOfTurn, change); err == nil {
			t.Fatal("unsupported context accepted", name)
		}
	}
	g, a, _ := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Kormus Bell"), a.PlayerID())
	for _, duration := range []core.Duration{core.WhileOnBattlefield, core.EndOfCombat} {
		if err := g.AddPrintedStaticAnimationTextEffect(source, textStaticID(t, source), duration, change); err == nil {
			t.Fatal("unsupported duration accepted")
		}
	}
	if err := g.AddPrintedStaticAnimationTextEffect(source, textStaticID(t, source), core.EndOfTurn); err == nil {
		t.Fatal("empty changes accepted")
	}
}
