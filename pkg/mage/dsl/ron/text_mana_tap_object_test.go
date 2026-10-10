package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextManaTapObjectRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Mana Flare", "Manabarbs"} {
			for _, consumer := range []string{"Magical Hack", "Sleight of Mind"} {
				t.Run(set+"/"+name+"/"+consumer, func(t *testing.T) {
					g, a, b := ronGame()
					card := repositoryCard(t, set, name)
					castRONSpell(t, g, a, card)
					g.ResolveStack()
					source := g.FindPermanent(card.ID())
					id := manaLayerID(t, source)
					from, to := "Mountain", "Island"
					if consumer == "Sleight of Mind" {
						from, to = "Red", "Green"
					}
					castManaTapObjectWords(t, g, a, source, set, consumer, from, to)
					change, _ := mage.ReplaceColorWord(core.Red, core.Green)
					if err := g.RevisePrintedManaTapTrigger(g.FindPermanent(card.ID()), id, change); err == nil {
						t.Fatal("resolving consumer did not install a text layer")
					}
					if manaLayerID(t, g.FindPermanent(card.ID())) != id || g.FindPermanent(card.ID()).Card.ManaCost().String() != card.ManaCost().String() {
						t.Fatal("text changed declaration or mana symbols")
					}
					land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
					a.ManaPool().Clear()
					b.ManaPool().Clear()
					if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
						t.Fatal(err)
					}
					g.PutTriggersOnStack()
					mana, stack := 2, 0
					if name == "Manabarbs" {
						mana, stack = 1, 1
					}
					if b.ManaPool().Count(core.Blue) != mana || a.ManaPool().TotalMana() != 0 || b.Life() != 20 || g.StackSize() != stack {
						t.Fatal("event lost player or timing")
					}
					clone := g.Clone()
					g.BouncePermanentToHand(g.FindPermanent(source.ID()))
					g.ResolveStack()
					clone.ResolveStack()
					life := 20
					if name == "Manabarbs" {
						life = 19
					}
					if b.Life() != life || clone.GetPlayer(b.PlayerID()).Life() != life || a.Life() != 20 || g.LKI(card.ID()) == nil {
						t.Fatal("pending event lost facts or LKI")
					}
					castRONSpell(t, g, a, card)
					g.ResolveStack()
					g.UntapPermanent(land)
					b.ManaPool().Clear()
					if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
						t.Fatal(err)
					}
					g.PutTriggersOnStack()
					g.ResolveStack()
					if b.ManaPool().Count(core.Blue) != mana || b.Life() != 2*life-20 {
						t.Fatal("printed return lost rule")
					}
				})
			}
		}
	}
}

func TestRONTextManaTapObjectQueuedSnapshots(t *testing.T) {
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
	if err := g.AddPrintedManaTapObjectTextEffect(source, core.EndOfTurn, change); err != nil {
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

func TestRONTextManaTapObjectProducedReplacement(t *testing.T) {
	for _, planned := range []bool{false, true} {
		t.Run(map[bool]string{false: "manual", true: "planned"}[planned], func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mana Flare"), a.PlayerID())
			change, _ := mage.ReplaceColorWord(core.Red, core.Green)
			if err := g.AddPrintedManaTapObjectTextEffect(source, core.EndOfTurn, change); err != nil {
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

func castManaTapObjectWords(t *testing.T, g *mage.Game, base mage.Player, subject *mage.Permanent, set, name, from, to string) {
	t.Helper()
	a := &intrinsicWordTester{Player: base, words: []string{from, to}}
	g.SetPlayerAt(0, a)
	spell := repositoryCard(t, set, name)
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{subject.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	if a.calls != 0 {
		t.Fatal("announcement chose words")
	}
	g.ResolveTopOfStack()
	if a.calls != 2 {
		t.Fatal("resolution lost word choices")
	}
	g.SetPlayerAt(0, base)
}

func TestRONTextManaTapObjectPaymentRollback(t *testing.T) {
	g, a, _ := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mana Flare"), a.PlayerID())
	land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), a.PlayerID())
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.AddPrintedManaTapObjectTextEffect(source, core.Indefinite, change); err != nil {
		t.Fatal(err)
	}
	if err := g.AutoTapForCost(a.PlayerID(), core.ManaCost{Blue: 2, Black: 1}); err == nil {
		t.Fatal("impossible payment succeeded")
	}
	if g.FindPermanent(land.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
		t.Fatal("failed payment changed resources")
	}
	if err := g.AutoTapForCost(a.PlayerID(), core.ManaCost{Blue: 2}); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(land.ID()).Tapped || a.ManaPool().Count(core.Blue) != 2 {
		t.Fatal("planner lost immediate bonus")
	}
}

func TestRONTextManaTapObjectBoundaries(t *testing.T) {
	g, a, _ := ronGame()
	change, _ := mage.ReplaceColorWord(core.Green, core.Blue)
	for _, name := range []string{"Wild Growth", "Gauntlet of Might", "Crystal Rod"} {
		p := g.PutOnBattlefield(repositoryCard(t, "LEA", name), a.PlayerID())
		if err := g.AddPrintedManaTapObjectTextEffect(p, core.Indefinite, change); err == nil {
			t.Fatalf("accepted foreign whole-object context %s", name)
		}
	}
	for _, rule := range []string{
		`SpellActions(ActionList([CreatePermanentLayerEffect(ThisPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheSecondChosenColorWord)])]))`,
		`SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([ChooseTwoColorWords,CreatePermanentLayerEffect(Ref_TargetPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheFirstChosenColorWord)])])))`,
		`SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([ChooseTwoColorWords,CreatePermanentLayerEffect(Ref_TargetPermanent,[ReplaceLandTypeVariableWithNewLandTypeVariable(TheFirstChosenColorWord,TheSecondChosenColorWord)])])))`,
		`SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([ChooseTwoColorWords,CreatePermanentLayerEffect(Ref_TargetPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheSecondChosenColorWord),AdjustPT(1,1)])])))`,
	} {
		records, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(records[0]); err == nil {
			t.Fatal("invalid text declaration accepted")
		}
	}
}
