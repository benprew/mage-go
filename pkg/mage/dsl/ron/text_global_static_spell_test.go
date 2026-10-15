package ron

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type globalStaticSpellCase struct {
	name                            string
	color                           core.Color
	power, toughness                int
	controlled, attacking, untapped bool
}

func globalStaticSpellCases() []globalStaticSpellCase {
	return []globalStaticSpellCase{
		{name: "Blood Moon"},
		{name: "Bad Moon", color: core.Black, power: 1, toughness: 1},
		{name: "Crusade", color: core.White, power: 1, toughness: 1},
		{name: "Castle", toughness: 2, controlled: true, untapped: true},
		{name: "Orcish Oriflamme", power: 1, controlled: true, attacking: true},
		{name: "Jacques le Vert", color: core.Green, toughness: 2, controlled: true},
		{name: "Kobold Taskmaster", power: 1, controlled: true},
		{name: "Mightstone", power: 1, attacking: true},
		{name: "Weakstone", power: -1, attacking: true},
	}
}

func TestRONTextGlobalStaticSpellRecords(t *testing.T) {
	for _, tc := range globalStaticSpellCases() {
		sets := []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"}
		switch tc.name {
		case "Blood Moon":
			sets = []string{"DRK"}
		case "Jacques le Vert", "Kobold Taskmaster":
			sets = []string{"LEG"}
		case "Mightstone", "Weakstone":
			sets = []string{"ATQ"}
		}
		for _, set := range sets {
			t.Run(set+"/"+tc.name, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, set, tc.name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				for _, c := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green, core.Colorless} {
					a.ManaPool().Add(c, 10)
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				paid := a.ManaPool().TotalMana()
				change, _ := mage.ReplaceColorWord(tc.color, core.Blue)
				if tc.color == core.Colorless {
					change, _ = mage.ReplaceBasicLandWord(core.MountainLandType, core.IslandLandType)
				}
				if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(card.ID()) != nil || g.StackPeek().Card != card || a.ManaPool().TotalMana() != paid {
					t.Fatal("revision changed timing, physical card or payment")
				}
				clone := g.Clone()
				g.ResolveStack()
				clone.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					source := game.FindPermanent(card.ID())
					if source == nil || source.ControllerID() != a.PlayerID() || source.Card.Owner() != a.PlayerID() {
						t.Fatal("entry lost identity")
					}
					if tc.name == "Blood Moon" {
						land := game.PutOnBattlefield(repositoryCard(t, "ATQ", "Strip Mine"), b.PlayerID())
						textLandCheck(t, game, land, "Island", core.Blue)
						basic := game.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
						textLandCheck(t, game, basic, "Mountain", core.Red)
					} else {
						for _, player := range []mage.Player{a, b} {
							for _, color := range []core.Color{core.Blue, tc.color} {
								probe := textColorCard(color)
								mage.WithSubTypes("Kobold")(probe.(*mage.BaseCard))
								p := game.PutOnBattlefield(probe, player.PlayerID())
								if tc.attacking {
									game.GetCombat().AddAttacker(p.ID(), b.PlayerID())
								}
								game.ApplyContinuousEffects()
								active := (!tc.controlled || player == a) && (tc.color == core.Colorless || color == core.Blue)
								power, toughness := 1, 1
								if active {
									power += tc.power
									toughness += tc.toughness
								}
								staticPTCheck(t, game, p, power, toughness)
								if tc.untapped {
									game.TapPermanent(game.FindPermanent(p.ID()))
									game.ApplyContinuousEffects()
									staticPTCheck(t, game, p, 1, 1)
								}
								if tc.attacking {
									game.GetCombat().RemoveFromCombat(p.ID())
									game.ApplyContinuousEffects()
									staticPTCheck(t, game, p, 1, 1)
								}
							}
						}
						if tc.name == "Kobold Taskmaster" {
							staticPTCheck(t, game, source, card.Power(), card.Toughness())
						}
					}
					values, err := game.CaptureCopiableValues(game.FindPermanent(card.ID()))
					if err != nil {
						t.Fatal(err)
					}
					printed, err := values.Build()
					if err != nil {
						t.Fatal(err)
					}
					printedRevision, err := mage.BuildTextChangedDefinition(card)
					if err != nil {
						t.Fatal(err)
					}
					if printed.Name() != card.Name() || printed.ManaCost().String() != printedRevision.ManaCost().String() {
						t.Fatal("copy values changed printed metadata")
					}
				}
				g.PhaseIn(g.PhaseOut(card.ID()))
				g.ApplyContinuousEffects()
				g.BouncePermanentToHand(g.FindPermanent(card.ID()))
				if g.LKI(card.ID()) == nil || len(a.Hand()) != 1 || a.Hand()[0] != card {
					t.Fatal("departure lost LKI or printed physical card")
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if tc.name == "Blood Moon" {
					land := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Strip Mine"), b.PlayerID())
					textLandCheck(t, g, land, "Mountain", core.Red)
				} else if tc.color != core.Colorless {
					blue := g.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
					staticPTCheck(t, g, blue, 1, 1)
				}
			})
		}
	}
}

func TestRONTextGlobalStaticSpellPTLifecycle(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Bad Moon")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 10)
	blue := g.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	black := g.PutOnBattlefield(textColorCard(core.Black), a.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	first, _ := mage.ReplaceColorWord(core.Black, core.Blue)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, first); err != nil {
		t.Fatal(err)
	}
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	second, _ := mage.ReplaceColorWord(core.Blue, core.Green)
	if err := g.RevisePrintedSpellText(copied.ID, second); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	third, _ := mage.ReplaceColorWord(core.Green, core.Red)
	if err := clone.RevisePrintedSpellText(copied.ID, third); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	clone.ResolveStack()
	staticPTCheck(t, g, blue, 2, 2)
	staticPTCheck(t, g, green, 2, 2)
	staticPTCheck(t, clone, green, 1, 1)
	token := g.FindPermanent(copied.SourceID)
	if token == nil || !token.Card.IsToken() || token.ControllerID() != b.PlayerID() {
		t.Fatal("spell copy lost token identity")
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
	staticPTCheck(t, g, black, 2, 2)
	staticPTCheck(t, g, green, 2, 2)
	g.RemoveFromBattlefield(g.FindPermanent(sibling.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(token.ID()))
	if len(b.Graveyard()) != 0 {
		t.Fatal("spell copy left a physical card")
	}
	source := g.FindPermanent(card.ID())
	id := textStaticID(t, source)
	if err := g.AddPrintedStaticPTTextEffect(source, id, core.EndOfTurn, second); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
		staticPTCheck(t, g, blue, 1, 1)
		staticPTCheck(t, g, green, 2, 2)
	}
	g.AddContinuousEffect(mage.RemoveAllAbilities(card.ID(), core.EndOfTurn))
	staticPTCheck(t, g, green, 1, 1)
	phased := g.PhaseOut(card.ID())
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, blue, 2, 2)
	staticPTCheck(t, g, green, 1, 1)
	if textStaticID(t, g.FindPermanent(card.ID())) != id {
		t.Fatal("restoration changed declaration identity")
	}
	g.BouncePermanentToHand(g.FindPermanent(card.ID()))
	staticPTCheck(t, g, blue, 1, 1)
	lki := g.LKI(card.ID())
	if lki == nil {
		t.Fatal("missing departure information")
	}
	revision, err := mage.BuildTextDefinitionRevision(lki.Snapshot.Card, second)
	if err != nil || revision.Definition() == nil {
		t.Fatal("LKI lost continued definition", err)
	}
	lkiSource := g.PutOnBattlefield(lki.Snapshot.Card.Copy(), a.PlayerID())
	staticPTCheck(t, g, blue, 2, 2)
	staticPTCheck(t, g, black, 1, 1)
	g.RemoveFromBattlefield(g.FindPermanent(lkiSource.ID()))
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	staticPTCheck(t, g, black, 2, 2)
	staticPTCheck(t, g, blue, 1, 1)
}

func TestRONTextGlobalStaticSpellLandLifecycle(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "DRK", "Blood Moon")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 10)
	host := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Strip Mine"), b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	first, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.IslandLandType)
	second, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, first); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	textLandCheck(t, g, host, "Island", core.Blue)
	source := g.FindPermanent(card.ID())
	id := textStaticID(t, source)
	if err := g.AddPrintedStaticLandTypeTextEffect(source, id, core.EndOfTurn, second); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
		textLandCheck(t, g, host, "Forest", core.Green)
	}
	clone := g.Clone()
	g.AddContinuousEffect(mage.RemoveAllAbilities(card.ID(), core.EndOfTurn))
	textLandCheck(t, g, host, "Forest", core.Green)
	phased := g.PhaseOut(card.ID())
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(host.ID()).RuntimeAbilities) != 2 {
		t.Fatal("source phasing did not restore land abilities")
	}
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	textLandCheck(t, g, host, "Island", core.Blue)
	textLandCheck(t, clone, host, "Forest", core.Green)
	values, err := g.CaptureCopiableValues(g.FindPermanent(card.ID()))
	if err != nil {
		t.Fatal(err)
	}
	printed, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(g.FindPermanent(card.ID()))
	source = g.PutOnBattlefield(printed, a.PlayerID())
	textLandCheck(t, g, host, "Mountain", core.Red)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	if err := g.AddPrintedStaticLandTypeTextEffect(source, id, core.EndOfTurn, second); err == nil {
		t.Fatal("departed visit accepted")
	}
}

func TestRONTextGlobalStaticSpellControl(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Jacques le Vert")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	for _, c := range []core.Color{core.White, core.Red, core.Green} {
		a.ManaPool().Add(c, 10)
	}
	own := g.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
	other := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceColorWord(core.Green, core.Blue)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	staticPTCheck(t, g, own, 1, 3)
	staticPTCheck(t, g, other, 1, 1)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	staticPTCheck(t, g, own, 1, 1)
	staticPTCheck(t, g, other, 1, 3)
}

func TestRONTextGlobalStaticSpellBoundaries(t *testing.T) {
	for _, body := range []string{
		`EachPermanentLayerEffect(IsCardtype(Creature), [AdjustPT(1)])`,
		`EachPermanentLayerEffect(IsCardtype(Creature), [SetLandType(Desert)])`,
		`EachPermanentLayerEffect(IsCardtype(Creature), [AdjustPT(ValueX, 1)])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[`+body+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid static declaration accepted", body)
		}
	}
	g, a, _ := ronGame()
	card := repositoryCard(t, "ATQ", "Titania's Song")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Green, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	before := g.StackPeek()
	change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.RevisePrintedSpellText(before.ID, change); err == nil || before != g.StackPeek() {
		t.Fatal("unmarked sticky spell text changed")
	}
	for _, tc := range []struct{ set, name string }{{"LEA", "Gauntlet of Might"}, {"LEA", "Conversion"}} {
		g, a, _ := ronGame()
		card := repositoryCard(t, tc.set, tc.name)
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		for _, c := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green, core.Colorless} {
			a.ManaPool().Add(c, 10)
		}
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		before := g.StackPeek()
		change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
		if err := g.RevisePrintedSpellText(before.ID, change); err != nil || before == g.StackPeek() {
			t.Fatal("supported mixed context rejected", tc.name)
		}
	}
}

func TestRONTextGlobalStaticSpellInventory(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]int{}
	for _, tc := range globalStaticSpellCases() {
		expected[tc.name] = 1
	}
	for _, name := range []string{"Bad Moon", "Castle", "Crusade", "Orcish Oriflamme"} {
		expected[name] = 6
	}
	expected["Kird Ape"] = 2
	expected["Water Wurm"] = 1
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
			if !globalStaticSpellTextRules(record.Fields["Rules"].Args) {
				continue
			}
			if _, err := Compile(record); err != nil {
				t.Fatal(err)
			}
			counts[record.Fields["Name"].String]++
		}
	}
	if !maps.Equal(counts, expected) {
		t.Fatalf("unmapped inventory: got %v, want %v", counts, expected)
	}
}

func TestRONTextGlobalStaticSpellCopyExceptions(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Bad Moon"), a.PlayerID())
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
	change, _ := mage.ReplaceColorWord(core.Black, core.Blue)
	revised, err := mage.BuildTextDefinitionRevision(card, change)
	if err != nil {
		t.Fatal(err)
	}
	if !revised.Definition().HasType(core.TypeLand) {
		t.Fatal("text lost copied land exception")
	}
	copied := g.PutOnBattlefield(revised.Definition(), a.PlayerID())
	blue := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
	black := g.PutOnBattlefield(textColorCard(core.Black), b.PlayerID())
	staticPTCheck(t, g, blue, 2, 2)
	staticPTCheck(t, g, black, 2, 2)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	staticPTCheck(t, g, blue, 2, 2)
	staticPTCheck(t, g, black, 1, 1)
	if !g.FindPermanent(copied.ID()).Card.HasType(core.TypeLand) {
		t.Fatal("copy lost added type")
	}
}
