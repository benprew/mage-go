package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func landTapWord(t *testing.T, from, to core.BasicLandType) mage.TextChange {
	t.Helper()
	change, err := mage.ReplaceBasicLandWord(from, to)
	if err != nil {
		t.Fatal(err)
	}
	return change
}

func TestRONTextLandTapLayerOrderingExpiry(t *testing.T) {
	for _, tc := range opponentLandTapCards {
		t.Run(tc.set+"/"+tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			p := g.PutOnBattlefield(card, a.PlayerID())
			id := p.RuntimeAbilities[0].AbilityID()
			word := core.ForestLandType
			if tc.land == "Mountain" {
				word = core.MountainLandType
			}
			old := g.PutOnBattlefield(repositoryCard(t, "LEA", tc.land), b.PlayerID())
			island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
			swamp := g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), b.PlayerID())
			first := landTapWord(t, word, core.IslandLandType)
			second := landTapWord(t, core.IslandLandType, core.SwampLandType)
			g.TapPermanent(old)
			if err := g.AddPrintedLandTapTextEffect(p, id, core.EndOfTurn, first); err != nil {
				t.Fatal(err)
			}
			if err := g.AddPrintedLandTapTextEffect(g.FindPermanent(p.ID()), id, core.Indefinite, second); err != nil {
				t.Fatal(err)
			}
			for range 3 {
				g.ApplyContinuousEffects()
			}
			if err := g.RevisePrintedLandTapTrigger(g.FindPermanent(p.ID()), id, first); err == nil {
				t.Fatal("direct revision changed a layered result")
			}
			g.TapPermanent(island)
			g.TapPermanent(swamp)
			g.PutTriggersOnStack()
			if g.StackSize() != 2 {
				t.Fatalf("ordered event text: stack=%d", g.StackSize())
			}
			clone := g.Clone()
			clone.GetEffects().RemoveEndOfTurn()
			clone.ApplyContinuousEffects()
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 22 {
				t.Fatal("expiry changed pending bodies")
			}
			for _, land := range []*mage.Permanent{old, island, swamp} {
				clone.UntapPermanent(clone.FindPermanent(land.ID()))
				clone.TapPermanent(clone.FindPermanent(land.ID()))
			}
			clone.PutTriggersOnStack()
			if clone.StackSize() != 1 {
				t.Fatal("expiry did not recompute surviving changes from baseline")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 23 {
				t.Fatal("restored trigger did not resolve")
			}
			g.ResolveStack()
			if a.Life() != 22 {
				t.Fatal("clone changed original pending bodies")
			}
			g.UntapPermanent(swamp)
			g.TapPermanent(swamp)
			g.ResolveStack()
			if a.Life() != 23 {
				t.Fatal("clone expiry changed original text")
			}
			phased := g.PhaseOut(p.ID())
			g.RunStepWithPriority(core.Cleanup)
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			g.UntapPermanent(old)
			g.TapPermanent(old)
			g.ResolveStack()
			if a.Life() != 24 {
				t.Fatal("expiry while phased lost baseline")
			}
			if err := g.AddPrintedLandTapTextEffect(g.FindPermanent(p.ID()), id, core.EndOfTurn, first); err != nil {
				t.Fatal(err)
			}
			remover := g.PutOnBattlefield(mage.NewArtifact("Remover", ""), a.PlayerID())
			removal := mage.RemoveAllAbilities(p.ID(), core.WhileOnBattlefield)
			removal.SetSourceID(remover.ID())
			g.AddContinuousEffect(removal)
			g.ApplyContinuousEffects()
			g.UntapPermanent(swamp)
			g.TapPermanent(swamp)
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("removed declaration detected a tap")
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
			g.ApplyContinuousEffects()
			g.UntapPermanent(old)
			g.TapPermanent(old)
			g.ResolveStack()
			if a.Life() != 25 {
				t.Fatal("restoration retained expired text")
			}
			if err := g.AddPrintedLandTapTextEffect(g.FindPermanent(p.ID()), id, core.Indefinite, first); err != nil {
				t.Fatal(err)
			}
			stale := g.FindPermanent(p.ID())
			g.BouncePermanentToHand(stale)
			if g.LKIAbilities(p.ID())[0] != stale.RuntimeAbilities[0] {
				t.Fatal("departure lost final rule")
			}
			a.RemoveFromHand(card.ID())
			returned := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.AddPrintedLandTapTextEffect(stale, id, core.EndOfTurn, first); err == nil {
				t.Fatal("former source visit accepted")
			}
			g.UntapPermanent(old)
			g.TapPermanent(old)
			g.ResolveStack()
			if a.Life() != 26 || returned.RuntimeAbilities[0].AbilityID() == uuid.Nil {
				t.Fatal("returned source lost printed rules")
			}
		})
	}
}

func TestRONTextLandTapLayerDirectBaseline(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "LEA", "Lifetap"), a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	if err := g.RevisePrintedLandTapTrigger(p, id, landTapWord(t, core.ForestLandType, core.IslandLandType)); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedLandTapTextEffect(g.FindPermanent(p.ID()), id, core.EndOfTurn, landTapWord(t, core.IslandLandType, core.SwampLandType)); err != nil {
		t.Fatal(err)
	}
	g.RunStepWithPriority(core.Cleanup)
	island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	g.TapPermanent(island)
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("cleanup lost prior direct baseline")
	}
}

func TestRONTextLandTapLayerPendingTargets(t *testing.T) {
	g, a, b := ronModalGame(0)
	rule := `TriggerA(WhenAPermanentBecomesTapped(And([IsLandType(Forest),ControlledByAPlayer(Opponent)])),Targeted([TargetPermanent(IsLandType(Forest))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`
	p := g.PutOnBattlefield(permanentScopeFactory(t, "Enchantment", rule)(), a.PlayerID())
	forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	g.TapPermanent(forest)
	if err := g.AddPrintedLandTapTextEffect(p, p.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, landTapWord(t, core.ForestLandType, core.IslandLandType)); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{forest.ID()}
	g.PutTriggersOnStack()
	g.TapPermanent(island)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	a.targets = []uuid.UUID{island.ID()}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("expiry changed pending target text")
	}
	g.ResolveStack()
	if g.FindPermanent(forest.ID()) != nil || g.FindPermanent(island.ID()) != nil {
		t.Fatal("queued targets lost saved text")
	}
}

func TestRONTextLandTapLayerControllerIsolation(t *testing.T) {
	g, a, b := ronGame()
	rule := `TriggerA(WhenAPermanentBecomesTapped(And([IsLandType(Forest),ControlledByAPlayer(Opponent)])),ActionList([PlayerAction(ControllerOfPermanent(ThisPermanent),GainLife(Integer(1)))]))`
	p := g.PutOnBattlefield(permanentScopeFactory(t, "Enchantment", rule)(), a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	if err := g.AddPrintedLandTapTextEffect(p, id, core.EndOfTurn, landTapWord(t, core.ForestLandType, core.IslandLandType)); err != nil {
		t.Fatal(err)
	}
	island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	g.TapPermanent(island)
	clone := g.Clone()
	clone.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	clone.ApplyContinuousEffects()
	clone.ResolveStack()
	if clone.GetPlayer(b.PlayerID()).Life() != 21 {
		t.Fatal("pending source read lost live controller")
	}
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	forest := clone.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
	clone.TapPermanent(forest)
	clone.ResolveStack()
	if clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatal("restored event lost current controller")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("clone changed original controller")
	}
	g.UntapPermanent(island)
	g.TapPermanent(island)
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("departure lost retained source controller")
	}
}

func TestRONTextLandTapLayerPaymentAndBatch(t *testing.T) {
	for _, tc := range []struct{ set, name, land string }{{"LEB", "Lifetap", "Forest"}, {"LEG", "Lifeblood", "Mountain"}} {
		for _, scenario := range []string{"mana", "paid-cast", "failed-cast", "batch", "instances"} {
			t.Run(tc.name+"/"+scenario, func(t *testing.T) {
				g, a, b := ronGame()
				source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
				word := core.ForestLandType
				if tc.land == "Mountain" {
					word = core.MountainLandType
				}
				if err := g.AddPrintedLandTapTextEffect(source, source.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, landTapWord(t, word, core.IslandLandType)); err != nil {
					t.Fatal(err)
				}
				land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
				want := 1
				switch scenario {
				case "mana":
					if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
						t.Fatal(err)
					}
					if b.ManaPool().TotalMana() != 1 || g.StackSize() != 0 || a.Life() != 20 {
						t.Fatal("mana trigger resolved immediately")
					}
				case "paid-cast", "failed-cast":
					g.SetActivePlayerIndex(1)
					cost := "{1}"
					if scenario == "failed-cast" {
						cost = "{2}"
						want = 0
					}
					spell := mage.NewSorcery("Payment fixture", cost, mage.WithAbility(mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1)))))
					spell.SetOwner(b.PlayerID())
					b.AddToHand(spell)
					err := g.CastSpellByID(b.PlayerID(), spell.ID(), nil, 0)
					if scenario == "failed-cast" {
						if err == nil || g.FindPermanent(land.ID()).Tapped || len(b.Hand()) != 1 || b.ManaPool().TotalMana() != 0 {
							t.Fatal("failed payment did not roll back")
						}
					} else {
						if err != nil {
							t.Fatal(err)
						}
						if !g.FindPermanent(land.ID()).Tapped || g.StackSize() != 1 {
							t.Fatal("cast did not pay with land")
						}
					}
				case "batch":
					g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
					g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
					if err := mage.ApplyEffect(g, mage.TapEach(mage.MatchingPermanents(mage.IsLand)), a.PlayerID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					want = 2
				case "instances":

					for _, player := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
						sibling := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), player)
						if err := g.AddPrintedLandTapTextEffect(sibling, sibling.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, landTapWord(t, word, core.IslandLandType)); err != nil {
							t.Fatal(err)
						}
					}
					g.TapPermanent(land)
					want = 2
				}
				g.PutTriggersOnStack()
				stack := want
				if scenario == "paid-cast" {
					stack++
				}
				if g.StackSize() != stack || a.Life() != 20 {
					t.Fatalf("stack=%d want=%d", g.StackSize(), stack)
				}
				for range want {
					g.ResolveTopOfStack()
				}
				if a.Life() != 20+want || b.Life() != 20 {
					t.Fatal("wrong trigger recipient or count")
				}
			})
		}
	}
}

func TestRONTextLandTapLayerColorTargetsAndSibling(t *testing.T) {
	g, a, b := ronModalGame(0)
	event := `WhenAPermanentBecomesTapped(And([IsLandType(Forest),ControlledByAPlayer(Opponent)]))`
	rule := `TriggerA(` + event + `,Targeted([TargetPermanent(IsColor(Red))],ActionList([DestroyPermanent(Ref_TargetPermanent)]))),TriggerA(` + event + `,ActionList([GainLife(Integer(2))]))`
	p := g.PutOnBattlefield(permanentScopeFactory(t, "Enchantment", rule)(), a.PlayerID())
	forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	red := g.PutOnBattlefield(mage.NewCreature("Red", "{R}", 1, 1), b.PlayerID())
	blue := g.PutOnBattlefield(mage.NewCreature("Blue", "{U}", 1, 1), b.PlayerID())
	g.TapPermanent(forest)
	change, err := mage.ReplaceColorWord(core.Red, core.Blue)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedLandTapTextEffect(p, p.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{red.ID()}
	g.PutTriggersOnStack()
	g.UntapPermanent(forest)
	g.TapPermanent(forest)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	a.targets = []uuid.UUID{blue.ID()}
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatal("color target text or sibling event changed")
	}
	g.ResolveStack()
	if g.FindPermanent(red.ID()) != nil || g.FindPermanent(blue.ID()) != nil || a.Life() != 24 {
		t.Fatal("pending target text or sibling body changed")
	}
}
