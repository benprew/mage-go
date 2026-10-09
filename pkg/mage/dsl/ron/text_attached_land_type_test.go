package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextAttachedLandTypeRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Evil Presence", "Phantasmal Terrain"} {
			t.Run(set+"/"+name, func(t *testing.T) {
				a := &ronBasicLandChooser{BasePlayer: mage.NewBasePlayer("A"), choice: 2}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				host := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Strip Mine"), b.PlayerID())
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
					t.Fatal("unpaid casting succeeded")
				}
				color, amount := core.Black, 1
				if name == "Phantasmal Terrain" {
					color, amount = core.Blue, 2
				}
				a.ManaPool().Add(color, amount)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err == nil || a.ManaPool().TotalMana() != amount {
					t.Fatal("illegal target spent mana")
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				if a.entryCalls != 0 || g.FindPermanent(host.ID()).HasSubType("Swamp") {
					t.Fatal("entry rule applied at announcement")
				}
				g.ResolveStack()
				source := g.FindPermanent(card.ID())
				id := textStaticID(t, source)
				change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
				if err := g.RevisePrintedStaticLandType(source, id, change); err != nil {
					t.Fatal(err)
				}
				want, wantColor := "Island", core.Blue
				if name == "Phantasmal Terrain" {
					want, wantColor = "Swamp", core.Black
					if a.entryCalls != 1 || source.ChosenBasicLandType != core.SwampLandType {
						t.Fatal("revision changed entry choice")
					}
				}
				textLandCheck(t, g, host, want, wantColor)
				if textStaticID(t, g.FindPermanent(source.ID())) != id {
					t.Fatal("revision changed declaration ID")
				}
				clone := g.Clone()
				next, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
				if err := clone.RevisePrintedStaticLandType(clone.FindPermanent(source.ID()), id, next); err != nil {
					t.Fatal(err)
				}
				cloneWant, cloneColor := "Forest", core.Green
				if name == "Phantasmal Terrain" {
					cloneWant, cloneColor = "Swamp", core.Black
				}
				textLandCheck(t, clone, host, cloneWant, cloneColor)
				textLandCheck(t, g, host, want, wantColor)
				second := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
				g.Attach(source.ID(), second.ID())
				textLandCheck(t, g, second, want, wantColor)
				if len(g.FindPermanent(host.ID()).SubTypes()) != 0 {
					t.Fatal("old host retained effect")
				}
				phased := g.PhaseOut(source.ID())
				g.ApplyContinuousEffects()
				textLandCheck(t, g, second, "Mountain", core.Red)
				if err := g.RevisePrintedStaticLandType(source, id, next); err == nil {
					t.Fatal("phased source accepted")
				}
				g.PhaseIn(phased)
				g.ApplyContinuousEffects()
				textLandCheck(t, g, second, want, wantColor)
				g.PhaseIn(g.PhaseOut(second.ID()))
				g.ApplyContinuousEffects()
				textLandCheck(t, g, second, want, wantColor)
				remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
				removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
				removal.SetSourceID(remover.ID())
				g.AddContinuousEffect(removal)
				textLandCheck(t, g, second, want, wantColor)
				if err := g.RevisePrintedStaticLandType(g.FindPermanent(source.ID()), id, next); err == nil {
					t.Fatal("removed declaration accepted")
				}
				g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
				if err := g.RevisePrintedStaticLandType(g.FindPermanent(source.ID()), id, next); err != nil {
					t.Fatal(err)
				}
				textLandCheck(t, g, second, cloneWant, cloneColor)
				g.BouncePermanentToHand(g.FindPermanent(second.ID()))
				if lki := g.LKI(second.ID()); lki == nil || !lki.Snapshot.HasSubType(cloneWant) {
					t.Fatal("LKI lost revised host type")
				}
				returnedHost := g.PutOnBattlefield(second.Card, b.PlayerID())
				g.ApplyContinuousEffects()
				textLandCheck(t, g, returnedHost, "Mountain", core.Red)
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if err := g.RevisePrintedStaticLandType(source, id, next); err == nil {
					t.Fatal("departed source accepted")
				}
				returned := g.PutOnBattlefield(card, a.PlayerID())
				g.Attach(returned.ID(), host.ID())
				textLandCheck(t, g, host, "Swamp", core.Black)
				if err := g.RevisePrintedStaticLandType(source, id, next); err == nil {
					t.Fatal("old incarnation accepted")
				}
				calls := 0
				if name == "Phantasmal Terrain" {
					calls = 2
				}
				if a.entryCalls != calls {
					t.Fatal("unexpected choices", a.entryCalls)
				}
			})
		}
	}
}

func TestRONTextAttachedLandTypeBoundaries(t *testing.T) {
	TestRONBasicLandChoiceInvalid(t)
	TestRONStaticLandTypeInvalidAndInventory(t)
	record := staticBaseRecord(t, "LEA", "Evil Presence")
	layer := record.Fields["Rules"].Args[1]
	layer.Args[1].Args = append(layer.Args[1].Args, Node{Kind: Tuple, Name: "AddAbility", Args: []Node{{Kind: List, Args: []Node{{Kind: Symbol, Name: "Flying"}}}}})
	record.Fields["Rules"].Args[1] = layer
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.RevisePrintedStaticLandType(source, textStaticID(t, source), change); err == nil {
		t.Fatal("mixed declaration accepted")
	}
}

func TestRONTextAttachedLandTypeDeclarations(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(repositoryCard(t, "LEA", "Evil Presence"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEA", "Evil Presence"), a.PlayerID())
	host := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	other := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
	g.Attach(first.ID(), host.ID())
	g.Attach(second.ID(), other.ID())
	id := textStaticID(t, first)
	change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.RevisePrintedStaticLandType(first, textStaticID(t, second), change); err == nil {
		t.Fatal("foreign declaration accepted")
	}
	if err := g.RevisePrintedStaticLandType(first, id, change); err != nil {
		t.Fatal(err)
	}
	textLandCheck(t, g, host, "Island", core.Blue)
	textLandCheck(t, g, other, "Swamp", core.Black)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: first.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	next, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	if err := g.RevisePrintedStaticLandType(g.FindPermanent(first.ID()), id, next); err != nil {
		t.Fatal(err)
	}
	textLandCheck(t, g, host, "Forest", core.Green)
	textLandCheck(t, g, other, "Swamp", core.Black)
	g.Attach(second.ID(), host.ID())
	change, _ = mage.ReplaceBasicLandWord(core.ForestLandType, core.PlainsLandType)
	if err := g.RevisePrintedStaticLandType(g.FindPermanent(first.ID()), id, change); err != nil {
		t.Fatal(err)
	}
	textLandCheck(t, g, host, "Swamp", core.Black)
}
