package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func manaLayerID(t *testing.T, p *mage.Permanent) uuid.UUID {
	t.Helper()
	for _, a := range p.RuntimeAbilities {
		switch a.(type) {
		case *mage.ManaBonusAbility, *mage.GenericTriggered:
			return a.AbilityID()
		}
	}
	t.Fatal("missing mana declaration")
	return uuid.Nil
}

func TestRONTextManaTapLayerRecords(t *testing.T) {
	records := 0
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Gauntlet of Might", "Mana Flare", "Wild Growth", "Manabarbs"} {
			if set == "3ED" && name == "Gauntlet of Might" {
				continue
			}
			records++
			t.Run(set+"/"+name, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, set, name)
				source := g.PutOnBattlefield(card, a.PlayerID())
				island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
				if name == "Wild Growth" {
					g.Attach(source.ID(), island.ID())
				}
				id := manaLayerID(t, source)
				first, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.IslandLandType)
				second, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
				color, _ := mage.ReplaceColorWord(core.Green, core.Blue)
				if err := g.AddPrintedManaTapTextEffect(source, id, core.EndOfTurn, first, color); err != nil {
					t.Fatal(err)
				}
				if err := g.AddPrintedManaTapTextEffect(g.FindPermanent(source.ID()), id, core.Indefinite, second); err != nil {
					t.Fatal(err)
				}
				for range 3 {
					g.ApplyContinuousEffects()
				}
				if manaLayerID(t, g.FindPermanent(source.ID())) != id {
					t.Fatal("declaration identity changed")
				}
				if err := g.RevisePrintedManaTapTrigger(g.FindPermanent(source.ID()), id, first); err == nil {
					t.Fatal("direct revision changed layered result")
				}
				if err := g.TapForMana(b.PlayerID(), island.ID()); err != nil {
					t.Fatal(err)
				}
				blue, green := 1, 0
				if name == "Mana Flare" {
					blue = 2
				}
				if name == "Wild Growth" {
					green = 1
				}
				if b.ManaPool().Count(core.Blue) != blue || b.ManaPool().Count(core.Green) != green || b.ManaPool().Count(core.Red) != 0 {
					t.Fatal("text changed symbols or ignored ordered land words")
				}
				g.PutTriggersOnStack()
				size := 0
				if name == "Manabarbs" {
					size = 1
				}
				if g.StackSize() != size {
					t.Fatal("mana event timing changed")
				}
				g.ResolveStack()
				life := 20
				if name == "Manabarbs" {
					life = 19
				}
				if b.Life() != life || a.Life() != 20 {
					t.Fatal("event player changed")
				}
				clone := g.Clone()
				clone.GetEffects().RemoveEndOfTurn()
				clone.ApplyContinuousEffects()
				g.ApplyContinuousEffects()
				if name == "Gauntlet of Might" {
					red := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mons's Goblin Raiders"), a.PlayerID())
					staticPTCheck(t, g, red, 2, 2)
					forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
					b.ManaPool().Clear()
					if err := g.AutoTapForCost(b.PlayerID(), core.ManaCost{Green: 1, Red: 1}); err != nil {
						t.Fatal(err)
					}
					if !g.FindPermanent(forest.ID()).Tapped || b.ManaPool().Count(core.Red) != 1 {
						t.Fatal("revised planning and production disagree")
					}
					cloneLand := clone.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
					clone.GetPlayer(b.PlayerID()).ManaPool().Clear()
					if err := clone.TapForMana(b.PlayerID(), cloneLand.ID()); err != nil {
						t.Fatal(err)
					}
					if clone.GetPlayer(b.PlayerID()).ManaPool().Count(core.Red) != 2 {
						t.Fatal("expiry did not recompute surviving changes from baseline")
					}
				}
				phased := g.PhaseOut(source.ID())
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				if err := g.AddPrintedManaTapTextEffect(source, id, core.EndOfTurn, first); err == nil {
					t.Fatal("phased declaration accepted")
				}
				g.PhaseIn(phased)
				g.ApplyContinuousEffects()
				remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), a.PlayerID())
				remove := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
				remove.SetSourceID(remover.ID())
				g.AddContinuousEffect(remove)
				g.ApplyContinuousEffects()
				if err := g.AddPrintedManaTapTextEffect(g.FindPermanent(source.ID()), id, core.EndOfTurn, first); err == nil {
					t.Fatal("removed declaration accepted")
				}
				g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
				g.ApplyContinuousEffects()
				if manaLayerID(t, g.FindPermanent(source.ID())) != id {
					t.Fatal("restoration lost declaration")
				}
				if err := g.AddPrintedManaTapTextEffect(g.FindPermanent(source.ID()), id, core.Indefinite, first); err != nil {
					t.Fatal(err)
				}
				if err := g.TapForMana(b.PlayerID(), island.ID()); err == nil {
					t.Fatal("tapped subject became available")
				}
				stale := g.FindPermanent(source.ID())
				g.BouncePermanentToHand(stale)
				a.RemoveFromHand(card.ID())
				returned := g.PutOnBattlefield(card, a.PlayerID())
				if err := g.AddPrintedManaTapTextEffect(stale, id, core.Indefinite, first); err == nil {
					t.Fatal("returned object accepted old visit")
				}
				if name == "Wild Growth" {
					g.Attach(returned.ID(), island.ID())
				}
				g.UntapPermanent(island)
				b.ManaPool().Clear()
				if err := g.TapForMana(b.PlayerID(), island.ID()); err != nil {
					t.Fatal(err)
				}
				if name == "Gauntlet of Might" && b.ManaPool().Count(core.Red) != 0 {
					t.Fatal("return retained text change")
				}
			})
		}
	}
	if records != 23 {
		t.Fatalf("tested %d records", records)
	}
}

func TestRONTextManaTapLayerQueuedSnapshots(t *testing.T) {
	rule := `TriggerA(WhenAPlayerTapsAPermanentForMana(AnyPlayer,IsCardtype(Land)),Targeted([TargetPermanent(IsColor(Red))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`
	g, a, b := ronModalGame(0)
	source := g.PutOnBattlefield(permanentScopeFactory(t, "Enchantment", rule)(), a.PlayerID())
	land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	red := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mons's Goblin Raiders"), b.PlayerID())
	green := g.PutOnBattlefield(repositoryCard(t, "LEA", "Llanowar Elves"), b.PlayerID())
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.AddPrintedManaTapTextEffect(source, manaLayerID(t, source), core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{red.ID()}
	g.PutTriggersOnStack()
	g.UntapPermanent(land)
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	a.targets = []uuid.UUID{green.ID()}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("pending targets did not retain prior text")
	}
	clone := g.Clone()
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.BouncePermanentToHand(source)
	g.ResolveStack()
	if g.FindPermanent(red.ID()) != nil || g.FindPermanent(green.ID()) != nil {
		t.Fatal("queued snapshots lost")
	}
	if clone.FindPermanent(red.ID()) == nil || clone.FindPermanent(green.ID()) == nil {
		t.Fatal("clone shared resolution")
	}
	clone.ResolveStack()
	if clone.FindPermanent(red.ID()) != nil || clone.FindPermanent(green.ID()) != nil {
		t.Fatal("clone lost pending snapshots")
	}
}

func TestRONTextManaTapLayerPaymentRollback(t *testing.T) {
	g, a, _ := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Gauntlet of Might"), a.PlayerID())
	land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), a.PlayerID())
	change, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.IslandLandType)
	if err := g.AddPrintedManaTapTextEffect(source, manaLayerID(t, source), core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	if err := g.AutoTapForCost(a.PlayerID(), core.ManaCost{Blue: 1, Red: 1, Black: 1}); err == nil {
		t.Fatal("impossible payment succeeded")
	}
	if g.FindPermanent(land.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
		t.Fatal("failed payment leaked resources")
	}
	if err := g.AutoTapForCost(a.PlayerID(), core.ManaCost{Blue: 1, Red: 1}); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(core.Red) != 1 {
		t.Fatal("revised bonus missing")
	}
	g.RunStepWithPriority(core.Cleanup)
	g.UntapPermanent(land)
	if g.CanAfford(a.PlayerID(), core.ManaCost{Blue: 1, Red: 1}, nil) {
		t.Fatal("expired bonus remained in payment planning")
	}
}

func TestRONTextManaTapLayerBoundaries(t *testing.T) {
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mana Flare"), a.PlayerID())
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	for _, duration := range []core.Duration{core.WhileOnBattlefield, core.EndOfCombat} {
		if err := g.AddPrintedManaTapTextEffect(p, manaLayerID(t, p), duration, change); err == nil {
			t.Fatal("unsupported duration accepted")
		}
	}
	if err := g.AddPrintedManaTapTextEffect(p, manaLayerID(t, p), core.EndOfTurn); err == nil {
		t.Fatal("empty changes accepted")
	}
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		if _, err := Compile(staticBaseRecord(t, "LEA", name)); err != nil {
			t.Fatal("complete text consumer failed")
		}
	}
}

func TestRONTextManaTapLayerInstancesAndLiveTypes(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Gauntlet of Might")
	source := g.PutOnBattlefield(card, a.PlayerID())
	sibling := g.PutOnBattlefield(repositoryCard(t, "LEA", "Gauntlet of Might"), b.PlayerID())
	id := manaLayerID(t, source)
	change, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.IslandLandType)
	if err := g.AddPrintedManaTapTextEffect(source, manaLayerID(t, sibling), core.EndOfTurn, change); err == nil {
		t.Fatal("foreign declaration accepted")
	}
	if err := g.AddPrintedManaTapTextEffect(source, textStaticID(t, source), core.EndOfTurn, change); err == nil {
		t.Fatal("static declaration accepted as mana event")
	}
	if err := g.AddPrintedManaTapTextEffect(source, id, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Underground Sea"), b.PlayerID())
	if err := g.AutoTapForCost(b.PlayerID(), core.ManaCost{Blue: 1, Red: 1}); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Red) != 1 {
		t.Fatal("nonbasic Island did not match live text")
	}
	g.UntapPermanent(land)
	b.ManaPool().Clear()
	g.AddContinuousEffect(mage.BecomesBasicLandTargetEffect(land.ID(), core.Indefinite, "Mountain"))
	g.ApplyContinuousEffects()
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Red) != 2 {
		t.Fatal("current subtype or independent sibling ignored")
	}
	shared := g.PutOnBattlefield(mage.NewLand("Shared", mage.WithSubTypes("Island"), mage.WithAbility(mage.Activate(mage.Costs(mage.TapCost(mage.This)), mage.AddManaAmount(core.Green, mage.Fixed(1)), mage.AnyPlayerMayActivate()))), b.PlayerID())
	a.ManaPool().Clear()
	b.ManaPool().Clear()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), shared.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(core.Green) != 1 || a.ManaPool().Count(core.Red) != 0 || b.ManaPool().Count(core.Red) != 1 {
		t.Fatal("bonus lost event permanent controller")
	}
	g.PhaseOut(shared.ID())
	if g.CanAfford(b.PlayerID(), core.ManaCost{Red: 2}, nil) {
		t.Fatal("phased or tapped subjects supplied mana")
	}
}

func TestRONTextManaTapLayerProducedReplacement(t *testing.T) {
	for _, planned := range []bool{false, true} {
		t.Run(map[bool]string{false: "manual", true: "planned"}[planned], func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mana Flare"), a.PlayerID())
			change, _ := mage.ReplaceColorWord(core.Red, core.Green)
			if err := g.AddPrintedManaTapTextEffect(source, manaLayerID(t, source), core.EndOfTurn, change); err != nil {
				t.Fatal(err)
			}
			land := g.PutOnBattlefield(mage.NewLand("Mixed", mage.WithMultiManaAbility(mage.ManaProduction{Color: core.White, Amount: 1}, mage.ManaProduction{Color: core.Red, Amount: 2})), b.PlayerID())
			if err := mage.ApplyEffect(g, mage.ReplaceTappedManaTypeForPlayerThisTurn(mage.You, mage.IsLand, core.Blue), source.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			var err error
			if planned {
				err = g.AutoTapForCost(b.PlayerID(), core.ManaCost{Blue: 4})
			} else {
				err = g.TapForMana(b.PlayerID(), land.ID())
			}
			if err != nil {
				t.Fatal(err)
			}
			if b.ManaPool().Count(core.Blue) != 4 || b.ManaPool().TotalMana() != 4 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("bonus did not use actual replaced type once")
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("bonus used stack")
			}
		})
	}
}
