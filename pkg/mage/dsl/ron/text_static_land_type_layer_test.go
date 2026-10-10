package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextStaticLandTypeLayerRecords(t *testing.T) {
	records := 0
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI", "DRK"} {
		for _, name := range []string{"Blood Moon", "Conversion", "Evil Presence", "Phantasmal Terrain"} {
			if (set == "DRK") != (name == "Blood Moon") {
				continue
			}
			records++
			t.Run(set+"/"+name, func(t *testing.T) {
				a := &ronBasicLandChooser{BasePlayer: mage.NewBasePlayer("A"), choice: 2}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				aura := name == "Evil Presence" || name == "Phantasmal Terrain"
				landCard := repositoryCard(t, "ATQ", "Strip Mine")
				if name == "Conversion" || aura {
					landCard = repositoryCard(t, "LEA", "Mountain")
				}
				host := g.PutOnBattlefield(landCard, b.PlayerID())
				card := repositoryCard(t, set, name)
				source := g.PutOnBattlefield(card, a.PlayerID())
				if aura {
					g.Attach(source.ID(), host.ID())
				}
				id := textStaticID(t, source)
				printed := core.MountainLandType
				if name == "Conversion" {
					printed = core.PlainsLandType
				} else if aura {
					printed = core.SwampLandType
				}
				first, _ := mage.ReplaceBasicLandWord(printed, core.IslandLandType)
				second, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
				if err := g.AddPrintedStaticLandTypeTextEffect(source, id, core.EndOfTurn, first); err != nil {
					t.Fatal(err)
				}
				if err := g.AddPrintedStaticLandTypeTextEffect(g.FindPermanent(source.ID()), id, core.Indefinite, second); err != nil {
					t.Fatal(err)
				}
				chosen := name == "Phantasmal Terrain"
				want, color := "Forest", core.Green
				if chosen {
					want, color = "Swamp", core.Black
				}
				for range 3 {
					g.ApplyContinuousEffects()
					textLandCheck(t, g, host, want, color)
				}
				if err := g.RevisePrintedStaticLandType(g.FindPermanent(source.ID()), id, first); err == nil {
					t.Fatal("direct revision changed layered text")
				}
				clone := g.Clone()
				clone.GetEffects().RemoveEndOfTurn()
				clone.ApplyContinuousEffects()
				printedColor := map[core.BasicLandType]core.Color{core.MountainLandType: core.Red, core.PlainsLandType: core.White, core.SwampLandType: core.Black}[printed]
				textLandCheck(t, clone, host, printed.String(), printedColor)
				textLandCheck(t, g, host, want, color)
				landSet := "LEA"
				if name == "Blood Moon" {
					landSet = "ATQ"
				}
				late := g.PutOnBattlefield(repositoryCard(t, landSet, landCard.Name()), b.PlayerID())
				if aura {
					g.Attach(source.ID(), late.ID())
					textLandCheck(t, g, host, "Mountain", core.Red)
				}
				textLandCheck(t, g, late, want, color)
				phasedHost := g.PhaseOut(late.ID())
				g.PhaseIn(phasedHost)
				g.ApplyContinuousEffects()
				textLandCheck(t, g, late, want, color)
				phased := g.PhaseOut(source.ID())
				g.ApplyContinuousEffects()
				if err := g.AddPrintedStaticLandTypeTextEffect(source, id, core.EndOfTurn, first); err == nil {
					t.Fatal("phased source accepted")
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				g.PhaseIn(phased)
				g.ApplyContinuousEffects()
				textLandCheck(t, g, late, printed.String(), printedColor)
				if !chosen {
					want, color = "Island", core.Blue
				}
				if err := g.AddPrintedStaticLandTypeTextEffect(g.FindPermanent(source.ID()), id, core.EndOfTurn, first); err != nil {
					t.Fatal(err)
				}
				remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
				removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
				removal.SetSourceID(remover.ID())
				g.AddContinuousEffect(removal)
				textLandCheck(t, g, late, want, color)
				if err := g.AddPrintedStaticLandTypeTextEffect(g.FindPermanent(source.ID()), id, core.EndOfTurn, first); err == nil {
					t.Fatal("removed declaration accepted")
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
				textLandCheck(t, g, late, printed.String(), printedColor)
				if textStaticID(t, g.FindPermanent(source.ID())) != id {
					t.Fatal("declaration identity changed")
				}
				if err := g.AddPrintedStaticLandTypeTextEffect(g.FindPermanent(source.ID()), id, core.Indefinite, first); err != nil {
					t.Fatal(err)
				}
				g.BouncePermanentToHand(g.FindPermanent(late.ID()))
				if lki := g.LKI(late.ID()); lki == nil || !lki.Snapshot.HasSubType(want) {
					t.Fatal("departure lost layered land type")
				}
				returnedHost := g.PutOnBattlefield(late.Card, b.PlayerID())
				if aura {
					textLandCheck(t, g, returnedHost, "Mountain", core.Red)
				}
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				returned := g.PutOnBattlefield(card, a.PlayerID())
				if aura {
					g.Attach(returned.ID(), host.ID())
				}
				if err := g.AddPrintedStaticLandTypeTextEffect(source, id, core.EndOfTurn, first); err == nil {
					t.Fatal("old visit accepted")
				}
				textLandCheck(t, g, host, printed.String(), printedColor)
				calls := 0
				if chosen {
					calls = 2
				}
				if a.entryCalls != calls {
					t.Fatal("text layer repeated entry choice", a.entryCalls)
				}
			})
		}
	}
	if records != 19 {
		t.Fatal("record coverage", records)
	}
}

func TestRONTextStaticLandTypeLayerDependencies(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		g, a, b := ronGame()
		land := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Strip Mine"), b.PlayerID())
		basic := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
		var conversion, moon *mage.Permanent
		addConversion := func() { conversion = g.PutOnBattlefield(repositoryCard(t, "LEA", "Conversion"), a.PlayerID()) }
		addMoon := func() { moon = g.PutOnBattlefield(repositoryCard(t, "DRK", "Blood Moon"), a.PlayerID()) }
		if reverse {
			addMoon()
			addConversion()
		} else {
			addConversion()
			addMoon()
		}
		change, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.IslandLandType)
		if err := g.AddPrintedStaticLandTypeTextEffect(conversion, textStaticID(t, conversion), core.EndOfTurn, change); err != nil {
			t.Fatal(err)
		}
		textLandCheck(t, g, land, "Mountain", core.Red)
		textLandCheck(t, g, basic, "Mountain", core.Red)
		if err := g.AddPrintedStaticLandTypeTextEffect(g.FindPermanent(moon.ID()), textStaticID(t, moon), core.Indefinite, change); err != nil {
			t.Fatal(err)
		}
		for range 3 {
			g.ApplyContinuousEffects()
			textLandCheck(t, g, land, "Plains", core.White)
			textLandCheck(t, g, basic, "Mountain", core.Red)
		}
		g.GetEffects().RemoveEndOfTurn()
		g.ApplyContinuousEffects()
		textLandCheck(t, g, land, "Island", core.Blue)
		textLandCheck(t, g, basic, "Plains", core.White)
	}
}

func TestRONTextStaticLandTypeLayerPriorRevisionAndDeclarations(t *testing.T) {
	g, a, b := ronGame()
	host := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	other := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
	first := g.PutOnBattlefield(repositoryCard(t, "LEA", "Evil Presence"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEA", "Evil Presence"), a.PlayerID())
	g.Attach(first.ID(), host.ID())
	g.Attach(second.ID(), other.ID())
	id := textStaticID(t, first)
	change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.AddPrintedStaticLandTypeTextEffect(first, textStaticID(t, second), core.EndOfTurn, change); err == nil {
		t.Fatal("foreign declaration accepted")
	}
	if err := g.RevisePrintedStaticLandType(first, id, change); err != nil {
		t.Fatal(err)
	}
	next, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	if err := g.AddPrintedStaticLandTypeTextEffect(g.FindPermanent(first.ID()), id, core.EndOfTurn, next); err != nil {
		t.Fatal(err)
	}
	textLandCheck(t, g, host, "Forest", core.Green)
	textLandCheck(t, g, other, "Swamp", core.Black)
	clone := g.Clone()
	clone.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: first.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	textLandCheck(t, clone, host, "Island", core.Blue)
	textLandCheck(t, g, host, "Forest", core.Green)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	textLandCheck(t, g, host, "Island", core.Blue)
	if clone.FindPermanent(first.ID()).RuntimeAbilities[0].Controller() != b.PlayerID() || g.FindPermanent(first.ID()).RuntimeAbilities[0].Controller() != a.PlayerID() {
		t.Fatal("clone shared declaration context")
	}
}

func TestRONTextStaticLandTypeLayerBoundaries(t *testing.T) {
	TestRONTextAttachedLandTypeBoundaries(t)
	TestRONStaticLandTypeInvalidAndInventory(t)
	change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	for _, name := range []string{"Living Lands", "Holy Strength"} {
		g, a, _ := ronGame()
		source := g.PutOnBattlefield(repositoryCard(t, "LEA", name), a.PlayerID())
		if err := g.AddPrintedStaticLandTypeTextEffect(source, textStaticID(t, source), core.EndOfTurn, change); err == nil {
			t.Fatal("unsupported context accepted", name)
		}
	}
	g, a, _ := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Evil Presence"), a.PlayerID())
	for _, duration := range []core.Duration{core.WhileOnBattlefield, core.EndOfCombat} {
		if err := g.AddPrintedStaticLandTypeTextEffect(source, textStaticID(t, source), duration, change); err == nil {
			t.Fatal("unsupported duration accepted")
		}
	}
	if err := g.AddPrintedStaticLandTypeTextEffect(source, textStaticID(t, source), core.EndOfTurn); err == nil {
		t.Fatal("empty changes accepted")
	}
}

func TestRONTextStaticLandTypeLayerUpkeep(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	for range 2 {
		g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), a.PlayerID())
	}
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Conversion"), a.PlayerID())
	id := textStaticID(t, source)
	change, _ := mage.ReplaceBasicLandWord(core.PlainsLandType, core.IslandLandType)
	if err := g.AddPrintedStaticLandTypeTextEffect(source, id, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	queueRONUpkeep(g, b)
	g.ResolveStack()
	if a.offers != 0 {
		t.Fatal("opponent upkeep offered payment")
	}
	a.ManaPool().Add(core.White, 2)
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if a.offers != 1 || g.FindPermanent(source.ID()) == nil || a.ManaPool().TotalMana() != 0 {
		t.Fatal("unrelated upkeep declaration changed")
	}
	a.accept = false
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if g.FindPermanent(source.ID()) != nil {
		t.Fatal("unpaid revised source survived")
	}
	for _, land := range g.FilterBattlefield(mage.AnyPermanent) {
		if !land.HasSubType("Mountain") {
			t.Fatal("sacrifice retained revised rule")
		}
	}
}
