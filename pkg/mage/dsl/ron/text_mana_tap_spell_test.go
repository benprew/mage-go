package ron

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextManaTapSpellRecords(t *testing.T) {
	for _, name := range []string{"Mana Flare", "Manabarbs"} {
		for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
			t.Run(set+"/"+name, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, set, name)
				obj := castRONSpell(t, g, a, card)
				paid := a.ManaPool().TotalMana()
				change, _ := mage.ReplaceColorWord(core.Red, core.Green)
				if err := g.RevisePrintedSpellText(obj.ID, change); err != nil {
					t.Fatal(err)
				}
				if g.StackPeek().Card != card || a.ManaPool().TotalMana() != paid || g.FindPermanent(card.ID()) != nil {
					t.Fatal("revision changed casting identity or payment")
				}
				g.ResolveStack()
				source := g.FindPermanent(card.ID())
				if source == nil || source.Card.Owner() != a.PlayerID() || source.ControllerID() != a.PlayerID() {
					t.Fatal("entry lost identity")
				}
				if !slices.Contains(source.Colors(), core.Red) {
					t.Fatal("mana symbols changed color")
				}
				land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
				a.ManaPool().Clear()
				b.ManaPool().Clear()
				if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
					t.Fatal(err)
				}
				mana, stack := 1, 1
				if name == "Mana Flare" {
					mana, stack = 2, 0
				}
				g.PutTriggersOnStack()
				if b.ManaPool().Count(core.Blue) != mana || a.ManaPool().TotalMana() != 0 || b.Life() != 20 || g.StackSize() != stack {
					t.Fatal("continued mana event changed player or timing")
				}
				clone := g.Clone()
				g.BouncePermanentToHand(source)
				if g.LKI(card.ID()) == nil || a.Hand()[0] != card {
					t.Fatal("departure lost printed card or LKI")
				}
				g.ResolveStack()
				life := 20
				if name == "Manabarbs" {
					life = 19
				}
				if b.Life() != life || a.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("pending event or clone state lost")
				}
				clone.ResolveStack()
				if clone.GetPlayer(b.PlayerID()).Life() != life {
					t.Fatal("clone lost pending event")
				}
				castRONSpell(t, g, a, card)
				g.ResolveStack()
				g.UntapPermanent(land)
				b.ManaPool().Clear()
				if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
					t.Fatal(err)
				}
				g.PutTriggersOnStack()
				if b.ManaPool().Count(core.Blue) != mana || g.StackSize() != stack {
					t.Fatal("recast lost printed declaration")
				}
			})
		}
	}
}

func TestRONTextManaTapSpellInventory(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		records, err := Parse(strings.NewReader(string(data)), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, record := range records {
			if !manaTapSpellTextRules(record.Fields["Rules"].Args) {
				continue
			}
			if _, err := Compile(record); err != nil {
				t.Fatal(err)
			}
			counts[record.Fields["Name"].String]++
		}
	}
	if !maps.Equal(counts, map[string]int{"Mana Flare": 6, "Manabarbs": 6}) {
		t.Fatalf("unmapped records: %v", counts)
	}
}

func TestRONTextManaTapSpellBoundaries(t *testing.T) {
	for _, rule := range []string{
		`TriggerA(WhenAPlayerTapsAPermanentForMana(AnyPlayer(),IsCardtype(Land)),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerTapsAPermanentForMana(Opponent,IsCardtype(Land)),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerTapsAPermanentForMana(AnyPlayer,IsCardtype(Creature)),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerTapsAPermanentForMana(AnyPlayer,IsCardtype(Land)),ActionList([PlayerAction(Trigger_ThatPermanent,GainLife(Integer(1)))]))`,
		`TriggerA(WhenAPlayerTapsAPermanentForMana(AnyPlayer,IsCardtype(Land)),ActionList([If(CostWasPaid,[GainLife(Integer(1))])]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", rule)
		}
	}
	for _, name := range []string{"Gauntlet of Might", "Wild Growth", "Conversion"} {
		g, a, _ := ronGame()
		card := repositoryCard(t, "LEA", name)
		var obj *mage.StackObject
		if name == "Wild Growth" {
			land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
			obj = castRONSpell(t, g, a, card, land.ID())
		} else {
			obj = castRONSpell(t, g, a, card)
		}
		change, _ := mage.ReplaceColorWord(core.Red, core.Green)
		if err := g.RevisePrintedSpellText(obj.ID, change); err == nil || g.StackPeek() != obj {
			t.Fatal("unsupported context mutated", name)
		}
	}
}

func TestRONTextManaTapSpellLifecycle(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerTapsAPermanentForMana(AnyPlayer,IsCardtype(Land)),Targeted([TargetPermanent(IsColor(Red))],ActionList([DestroyPermanent(Ref_TargetPermanent),PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))])))`)
	g, a, b := ronGame()
	card := factory()
	obj := castRONSpell(t, g, a, card)
	first, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	second, _ := mage.ReplaceColorWord(core.Blue, core.Green)
	third, _ := mage.ReplaceColorWord(core.Green, core.White)
	if err := g.RevisePrintedSpellText(obj.ID, first); err != nil {
		t.Fatal(err)
	}
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("spell copy failed")
	}
	if err := g.RevisePrintedSpellText(copied.ID, second); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := clone.RevisePrintedSpellText(copied.ID, third); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	clone.ResolveStack()
	token := g.FindPermanent(copied.SourceID)
	if token == nil || !token.Card.IsToken() || token.ControllerID() != b.PlayerID() {
		t.Fatal("copy entry identity lost")
	}
	land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	blue := g.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), a.PlayerID())
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("copy and original declarations lost")
	}
	g.ResolveStack()
	if g.FindPermanent(blue.ID()) != nil || g.FindPermanent(green.ID()) != nil || b.Life() != 22 || a.Life() != 20 {
		t.Fatal("copy targets or event player lost")
	}
	cloneLand := clone.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	cloneBlue := clone.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
	cloneWhite := clone.PutOnBattlefield(textColorCard(core.White), a.PlayerID())
	if err := clone.TapForMana(b.PlayerID(), cloneLand.ID()); err != nil {
		t.Fatal(err)
	}
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if clone.FindPermanent(cloneBlue.ID()) != nil || clone.FindPermanent(cloneWhite.ID()) != nil || clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatal("clone shared stack text")
	}
	values, err := g.CaptureCopiableValues(token)
	if err != nil {
		t.Fatal(err)
	}
	printed, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	sibling := g.PutOnBattlefield(printed, a.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), a.PlayerID())
	g.UntapPermanent(land)
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("copiable values retained ordinary changes")
	}
	g.ResolveStack()
	if g.FindPermanent(red.ID()) != nil {
		t.Fatal("printed copy did not use red")
	}
	g.RemoveFromBattlefield(g.FindPermanent(sibling.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(token.ID()))
	source := g.FindPermanent(card.ID())
	id := manaLayerID(t, source)
	if err := g.AddPrintedManaTapTextEffect(source, id, core.EndOfTurn, second); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
	}
	green = g.PutOnBattlefield(textColorCard(core.Green), a.PlayerID())
	g.UntapPermanent(land)
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.StackPeek().Targets[0] != green.ID() {
		t.Fatal("later layer lost continued baseline")
	}
	g.ResolveStack()
	phased := g.PhaseOut(source.ID())
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	if manaLayerID(t, g.FindPermanent(source.ID())) != id {
		t.Fatal("phase cycle lost declaration")
	}
	remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), a.PlayerID())
	removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
	removal.SetSourceID(remover.ID())
	g.AddContinuousEffect(removal)
	g.ApplyContinuousEffects()
	blue = g.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
	g.UntapPermanent(land)
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("removed rule detected event")
	}
	g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
	g.ApplyContinuousEffects()
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.UntapPermanent(land)
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.StackPeek().Targets[0] != blue.ID() || g.StackPeek().Controller != b.PlayerID() {
		t.Fatal("restoration lost baseline or controller")
	}
	pendingClone := g.Clone()
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	g.ResolveStack()
	if g.FindPermanent(blue.ID()) != nil || pendingClone.FindPermanent(blue.ID()) == nil {
		t.Fatal("pending event shared clone or followed source")
	}
	pendingClone.ResolveStack()
	if pendingClone.FindPermanent(blue.ID()) != nil || g.LKI(card.ID()) == nil {
		t.Fatal("departure lost pending rules or LKI")
	}
	lki := g.PutOnBattlefield(g.LKI(card.ID()).Snapshot.Card.Copy(), b.PlayerID())
	blue = g.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
	g.UntapPermanent(land)
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.StackPeek().Targets[0] != blue.ID() {
		t.Fatal("LKI lost continued rule")
	}
	g.ResolveStack()
	g.RemoveFromBattlefield(g.FindPermanent(lki.ID()))
	castRONSpell(t, g, a, card)
	g.ResolveStack()
	red = g.PutOnBattlefield(textColorCard(core.Red), a.PlayerID())
	g.UntapPermanent(land)
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.StackPeek().Targets[0] != red.ID() {
		t.Fatal("recast retained changed words")
	}
}

func TestRONTextManaTapSpellManaProduction(t *testing.T) {
	for _, planned := range []bool{false, true} {
		t.Run(map[bool]string{false: "manual", true: "planned"}[planned], func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEA", "Mana Flare")
			obj := castRONSpell(t, g, a, card)
			change, _ := mage.ReplaceColorWord(core.Red, core.Green)
			if err := g.RevisePrintedSpellText(obj.ID, change); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			land := g.PutOnBattlefield(mage.NewLand("Mixed", mage.WithMultiManaAbility(mage.ManaProduction{Color: core.White, Amount: 1}, mage.ManaProduction{Color: core.Red, Amount: 2})), b.PlayerID())
			a.ManaPool().Clear()
			b.ManaPool().Clear()
			if err := mage.ApplyEffect(g, mage.ReplaceTappedManaTypeForPlayerThisTurn(mage.You, mage.IsLand, core.Blue), source.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if err := g.AutoTapForCost(b.PlayerID(), core.ManaCost{Blue: 4, Black: 1}); err == nil {
				t.Fatal("impossible payment succeeded")
			}
			if g.FindPermanent(land.ID()).Tapped || b.ManaPool().TotalMana() != 0 || g.StackSize() != 0 {
				t.Fatal("failed payment leaked mana, taps or triggers")
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
			g.PutTriggersOnStack()
			if b.ManaPool().Count(core.Blue) != 4 || b.ManaPool().TotalMana() != 4 || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 {
				t.Fatal("continued bonus disagreed with actual replaced production")
			}
			clone := g.Clone()
			g.PhaseOut(source.ID())
			g.UntapPermanent(land)
			b.ManaPool().Clear()
			if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
				t.Fatal(err)
			}
			if b.ManaPool().Count(core.Blue) != 3 {
				t.Fatal("phased bonus remained active")
			}
			clone.UntapPermanent(clone.FindPermanent(land.ID()))
			clone.GetPlayer(b.PlayerID()).ManaPool().Clear()
			if err := clone.TapForMana(b.PlayerID(), land.ID()); err != nil {
				t.Fatal(err)
			}
			if clone.GetPlayer(b.PlayerID()).ManaPool().Count(core.Blue) != 4 {
				t.Fatal("clone shared phase or bonus state")
			}
		})
	}
}

func TestRONTextManaTapSpellIllegalTargets(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerTapsAPermanentForMana(AnyPlayer,IsCardtype(Land)),Targeted([TargetPermanent(IsColor(Red))],ActionList([DestroyPermanent(Ref_TargetPermanent),PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))])))`)
	g, a, b := ronGame()
	card := factory()
	obj := castRONSpell(t, g, a, card)
	change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.RevisePrintedSpellText(obj.ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	target := g.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.StackPeek().Targets[0] != target.ID() {
		t.Fatal("revised target not selected")
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.Red)), target.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(target.ID()) == nil || b.Life() != 20 {
		t.Fatal("illegal target did not stop body")
	}
}

func TestRONTextManaTapSpellPendingEvents(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Manabarbs")
	obj := castRONSpell(t, g, a, card)
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedSpellText(obj.ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	landA := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), a.PlayerID())
	landB := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	if err := g.TapForMana(a.PlayerID(), landA.ID()); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if err := g.TapForMana(b.PlayerID(), landB.ID()); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("separate events lost")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	clone := g.Clone()
	g.BouncePermanentToHand(g.FindPermanent(card.ID()))
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 18 || clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("event players shared a binding or clone")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 19 || clone.GetPlayer(b.PlayerID()).Life() != 18 {
		t.Fatal("clone lost pending events")
	}
}

func TestRONTextManaTapSpellCopyExceptions(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mana Flare"), a.PlayerID())
	values, err := g.CaptureCopiableValues(source)
	if err != nil {
		t.Fatal(err)
	}
	modified, err := values.WithModifications(mage.CopyAddCardTypes(core.TypeLand))
	if err != nil {
		t.Fatal(err)
	}
	card, err := modified.Build()
	if err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	revision, err := mage.BuildTextDefinitionRevision(card, change)
	if err != nil {
		t.Fatal(err)
	}
	if !revision.Definition().HasType(core.TypeLand) {
		t.Fatal("copy lost added type")
	}
	g.RemoveFromBattlefield(source)
	g.PutOnBattlefield(revision.Definition(), a.PlayerID())
	land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Blue) != 2 {
		t.Fatal("copied Land lost bonus")
	}
}
