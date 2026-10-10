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

func TestRONTextMixedStaticEventSpellRecords(t *testing.T) {
	for _, name := range []string{"Gauntlet of Might", "Conversion", "Sunken City"} {
		sets := []string{"LEA", "LEB", "2ED", "CED", "CEI"}
		if name == "Conversion" {
			sets = append(sets, "3ED")
		}
		if name == "Sunken City" {
			sets = []string{"DRK"}
		}
		for _, set := range sets {
			t.Run(set+"/"+name, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, set, name)
				obj := castRONSpell(t, g, a, card)
				paid := a.ManaPool().TotalMana()
				change, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.IslandLandType)
				if name == "Gauntlet of Might" {
					color, _ := mage.ReplaceColorWord(core.Red, core.Green)
					if err := g.RevisePrintedSpellText(obj.ID, color); err != nil {
						t.Fatal(err)
					}
				}
				if name == "Sunken City" {
					change, _ = mage.ReplaceColorWord(core.Blue, core.Green)
				}
				if err := g.RevisePrintedSpellText(obj.ID, change); err != nil {
					t.Fatal(err)
				}
				if g.StackPeek().Card != card || a.ManaPool().TotalMana() != paid {
					t.Fatal("revision changed physical card or casting payment")
				}
				clone := g.Clone()
				g.ResolveStack()
				clone.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					source := game.FindPermanent(card.ID())
					if source == nil || source.Card.Owner() != a.PlayerID() || source.ControllerID() != a.PlayerID() {
						t.Fatal("entry lost identity")
					}
					if name == "Conversion" {
						island := game.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
						textLandCheck(t, game, island, "Plains", core.White)
						mountain := game.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
						textLandCheck(t, game, mountain, "Mountain", core.Red)
					} else {
						green := game.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
						red := game.PutOnBattlefield(textColorCard(core.Red), a.PlayerID())
						if green.CurrentPower(game) != 2 || green.CurrentToughness(game) != 2 || red.CurrentPower(game) != 1 {
							t.Fatal("static rule did not keep revised color")
						}
					}
					if name == "Gauntlet of Might" {
						land := game.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
						game.GetPlayer(a.PlayerID()).ManaPool().Clear()
						game.GetPlayer(b.PlayerID()).ManaPool().Clear()
						if err := game.TapForMana(b.PlayerID(), land.ID()); err != nil {
							t.Fatal(err)
						}
						game.PutTriggersOnStack()
						if game.GetPlayer(b.PlayerID()).ManaPool().Count(core.Blue) != 1 || game.GetPlayer(b.PlayerID()).ManaPool().Count(core.Red) != 1 || game.StackSize() != 0 || game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
							t.Fatal("bonus lost mana symbol, recipient or immediate timing")
						}
					} else {
						queueRONUpkeep(game, game.GetPlayer(b.PlayerID()))
						if game.StackSize() != 0 {
							t.Fatal("opponent upkeep triggered payment")
						}
						queueRONUpkeep(game, game.GetPlayer(a.PlayerID()))
						if game.StackSize() != 1 || game.FindPermanent(card.ID()) == nil {
							t.Fatal("upkeep payment did not wait for resolution")
						}
						game.GetPlayer(a.PlayerID()).ManaPool().Clear()
						game.ResolveStack()
						if game.FindPermanent(card.ID()) != nil || game.LKI(card.ID()) == nil {
							t.Fatal("unpaid upkeep lost self sacrifice or LKI")
						}
					}
				}
			})
		}
	}
}

func TestRONTextMixedStaticEventSpellInventory(t *testing.T) {
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
			if !mixedStaticEventSpellTextRules(record.Fields["Rules"].Args) {
				continue
			}
			if _, err := Compile(record); err != nil {
				t.Fatal(err)
			}
			counts[record.Fields["Name"].String]++
		}
	}
	if !maps.Equal(counts, map[string]int{"Gauntlet of Might": 5, "Conversion": 6, "Sunken City": 1}) {
		t.Fatalf("unmapped records: %v", counts)
	}
}

func TestRONTextMixedStaticEventSpellPayments(t *testing.T) {
	for _, name := range []string{"Conversion", "Sunken City"} {
		for _, scenario := range []string{"paid", "declined", "rollback", "control", "returned", "phased", "copied-trigger"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "declined"}
				b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				set, color := "LEA", core.White
				change, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.IslandLandType)
				if name == "Sunken City" {
					set, color = "DRK", core.Blue
					change, _ = mage.ReplaceColorWord(core.Blue, core.Green)
				}
				card := repositoryCard(t, set, name)
				obj := castRONSpell(t, g, a.BasePlayer, card)
				if err := g.RevisePrintedSpellText(obj.ID, change); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				source := g.FindPermanent(card.ID())
				a.ManaPool().Clear()
				b.ManaPool().Clear()
				land := g.PutOnBattlefield(mage.NewLand("Payment", mage.WithMultiManaAbility(mage.ManaProduction{Color: color, Amount: 1})), a.PlayerID())
				if scenario != "rollback" {
					a.ManaPool().Add(color, 1)
				}
				queueRONUpkeep(g, a)
				if g.StackSize() != 1 {
					t.Fatal("missing upkeep trigger")
				}
				switch scenario {
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "returned":
					g.BouncePermanentToHand(source)
					g.PutOnBattlefield(card, b.PlayerID())
				case "phased":
					g.PhaseOut(source.ID())
				case "copied-trigger":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				if b.offers != 0 {
					t.Fatal("pending payment followed new controller")
				}
				paid := scenario != "declined" && scenario != "rollback"
				if g.FindPermanent(land.ID()).Tapped != paid {
					t.Fatal("payment did not use unchanged mana symbols or rollback")
				}
				if clone.FindPermanent(land.ID()).Tapped || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != map[bool]int{true: 1, false: 0}[scenario != "rollback"] {
					t.Fatal("clone shared payment state")
				}
				if scenario == "declined" || scenario == "rollback" {
					if g.FindPermanent(card.ID()) != nil {
						t.Fatal("unpaid exact source was not sacrificed")
					}
				} else if scenario != "phased" && g.FindPermanent(card.ID()) == nil {
					t.Fatal("paid or returned source was lost")
				}
				if scenario == "rollback" && (a.offers != 0 || a.ManaPool().TotalMana() != 0) {
					t.Fatal("unaffordable payment offered a choice or leaked mana")
				}
				if scenario == "declined" && (a.offers != 1 || a.ManaPool().Count(color) != 1) {
					t.Fatal("declined payment spent mana")
				}
			})
		}
	}
}

func TestRONTextMixedStaticEventSpellLifecycle(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Gauntlet of Might")
	obj := castRONSpell(t, g, a, card)
	blue, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	island, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.IslandLandType)
	if err := g.RevisePrintedSpellText(obj.ID, blue, island); err != nil {
		t.Fatal(err)
	}
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("spell copy failed")
	}
	green, _ := mage.ReplaceColorWord(core.Blue, core.Green)
	if err := g.RevisePrintedSpellText(copied.ID, green); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	white, _ := mage.ReplaceColorWord(core.Green, core.White)
	if err := clone.RevisePrintedSpellText(copied.ID, white); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	clone.ResolveStack()
	token := g.FindPermanent(copied.SourceID)
	if token == nil || !token.Card.IsToken() || token.ControllerID() != b.PlayerID() {
		t.Fatal("token entry lost identity")
	}
	for _, game := range []*mage.Game{g, clone} {
		for _, color := range []core.Color{core.Blue, core.Green, core.White} {
			p := game.PutOnBattlefield(textColorCard(color), a.PlayerID())
			power := 1
			if color == core.Blue || (game == g && color == core.Green) || (game == clone && color == core.White) {
				power = 2
			}
			if p.CurrentPower(game) != power {
				t.Fatal("spell copies or clones shared rule state")
			}
		}
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
	redProbe := g.PutOnBattlefield(textColorCard(core.Red), a.PlayerID())
	if g.FindPermanent(redProbe.ID()).CurrentPower(g) != 2 {
		t.Fatal("copy values retained ordinary word changes")
	}
	g.RemoveFromBattlefield(sibling)
	g.RemoveFromBattlefield(token)
	source := g.FindPermanent(card.ID())
	probe := g.PutOnBattlefield(textColorCard(core.Green), a.PlayerID())
	if err := g.AddPrintedStaticPTTextEffect(source, textStaticID(t, source), core.EndOfTurn, green); err != nil {
		t.Fatal(err)
	}
	forest, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	if err := g.AddPrintedManaTapTextEffect(source, manaLayerID(t, source), core.EndOfTurn, forest); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	g.ApplyContinuousEffects()
	if g.FindPermanent(probe.ID()).CurrentPower(g) != 2 {
		t.Fatal("later layer lost continued baseline")
	}
	land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	a.ManaPool().Clear()
	b.ManaPool().Clear()
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Red) != 1 {
		t.Fatal("later mana layer lost continued baseline")
	}
	phase := g.PhaseOut(source.ID())
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.PhaseIn(phase)
	g.ApplyContinuousEffects()
	if g.FindPermanent(probe.ID()).CurrentPower(g) != 1 {
		t.Fatal("phase expiry did not restore continued color")
	}
	g.UntapPermanent(land)
	b.ManaPool().Clear()
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Red) != 0 {
		t.Fatal("phase expiry did not restore continued subtype")
	}
	remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), a.PlayerID())
	removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
	removal.SetSourceID(remover.ID())
	g.AddContinuousEffect(removal)
	g.ApplyContinuousEffects()
	blueProbe := g.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
	islandLand := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	b.ManaPool().Clear()
	if err := g.TapForMana(b.PlayerID(), islandLand.ID()); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(blueProbe.ID()).CurrentPower(g) != 1 || b.ManaPool().Count(core.Red) != 0 {
		t.Fatal("removed rules remained active")
	}
	g.RemoveFromBattlefield(remover)
	g.ApplyContinuousEffects()
	if g.FindPermanent(blueProbe.ID()).CurrentPower(g) != 2 {
		t.Fatal("restoration lost continued rule")
	}
	g.UntapPermanent(islandLand)
	b.ManaPool().Clear()
	if err := g.TapForMana(b.PlayerID(), islandLand.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Red) != 1 {
		t.Fatal("restoration lost continued bonus")
	}
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	if a.Hand()[0] != card || g.FindPermanent(blueProbe.ID()).CurrentPower(g) != 1 || g.LKI(card.ID()) == nil {
		t.Fatal("departure lost printed rules or LKI")
	}
	lki := g.PutOnBattlefield(g.LKI(card.ID()).Snapshot.Card.Copy(), b.PlayerID())
	if g.FindPermanent(blueProbe.ID()).CurrentPower(g) != 2 {
		t.Fatal("LKI lost continued static rule")
	}
	g.RemoveFromBattlefield(lki)
	castRONSpell(t, g, a, card)
	g.ResolveStack()
	if g.FindPermanent(blueProbe.ID()).CurrentPower(g) != 1 || g.FindPermanent(redProbe.ID()).CurrentPower(g) != 2 {
		t.Fatal("recast retained revised color")
	}
	g.UntapPermanent(islandLand)
	b.ManaPool().Clear()
	if err := g.TapForMana(b.PlayerID(), islandLand.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Red) != 0 {
		t.Fatal("recast retained revised subtype")
	}
}

func TestRONTextMixedStaticEventSpellBoundaries(t *testing.T) {
	static := `EachPermanentLayerEffect(IsCardtype(Creature),[AdjustPT(1,1)])`
	for _, event := range []string{
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([MayCost(PayMana([ManaCostX])),Unless(CostWasPaid,[SacrificePermanent(ThisPermanent)])]))`,
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([Unless(CostWasPaid,[SacrificePermanent(ThisPermanent)])]))`,
		`TriggerA(WhenAPermanentIsTappedForMana(IsLandType(Mountain)),ActionList([PlayerAction(Trigger_ThatPlayer,AddMana(ManaProduceR))]))`,
	} {
		records, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+static+`,`+event+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(records[0]); err == nil {
			t.Fatal("invalid paired declaration accepted")
		}
	}
	for _, rules := range []string{
		static + `,TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),ActionList([GainLife(Integer(1))]))`,
		static + `,TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([GainLife(Integer(1))]))`,
		static + `,TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([MayCost(PayMana([])),Unless(CostWasPaid,[SacrificePermanent(ThisPermanent)])])),Flying`,
	} {
		g, a, _ := ronGame()
		card := permanentScopeFactory(t, "Artifact", rules)()
		obj := castRONSpell(t, g, a, card)
		change, _ := mage.ReplaceColorWord(core.Red, core.Green)
		if err := g.RevisePrintedSpellText(obj.ID, change); err == nil || g.StackPeek() != obj {
			t.Fatal("unsupported mixed context mutated")
		}
	}
}

func TestRONTextMixedStaticEventSpellCopyExceptions(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Gauntlet of Might"), a.PlayerID())
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
	change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	revision, err := mage.BuildTextDefinitionRevision(card, change)
	if err != nil {
		t.Fatal(err)
	}
	if !revision.Definition().HasType(core.TypeLand) {
		t.Fatal("copy lost added type")
	}
	g.RemoveFromBattlefield(source)
	g.PutOnBattlefield(revision.Definition(), a.PlayerID())
	probe := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
	if probe.CurrentPower(g) != 2 {
		t.Fatal("Land copy lost revised static rule")
	}
	land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Red) != 2 {
		t.Fatal("Land copy lost unchanged bonus mana symbol")
	}
}
