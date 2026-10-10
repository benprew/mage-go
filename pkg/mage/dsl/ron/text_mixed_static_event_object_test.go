package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextMixedStaticEventObjectRecords(t *testing.T) {
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
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				printed := append([]mage.Ability(nil), source.RuntimeAbilities...)
				if name == "Gauntlet of Might" {
					castActivatedObjectWords(t, g, a, source, "Sleight of Mind", "Red", "Green")
				}
				consumer, from, to := "Magical Hack", "Mountain", "Island"
				if name == "Sunken City" {
					consumer, from, to = "Sleight of Mind", "Blue", "Green"
				}
				castActivatedObjectWords(t, g, a, source, consumer, from, to)
				clone := g.Clone()
				g.ResolveStack()
				clone.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					source := game.FindPermanent(card.ID())
					if source == nil || source.Card.Owner() != a.PlayerID() || source.ControllerID() != a.PlayerID() {
						t.Fatal("entry lost identity")
					}
					for i, rule := range source.RuntimeAbilities {
						if rule.AbilityID() != printed[i].AbilityID() {
							t.Fatal("object text changed a declaration ID")
						}
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

func TestRONTextMixedStaticEventObjectPayments(t *testing.T) {
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
				card.SetOwner(a.PlayerID())
				subject := g.PutOnBattlefield(card, a.PlayerID())
				if err := g.AddPrintedMixedStaticEventTextEffect(subject, core.Indefinite, change); err != nil {
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
				pending := g.StackPeek()
				later, _ := mage.ReplaceColorWord(core.Green, core.Black)
				if name == "Conversion" {
					later, _ = mage.ReplaceBasicLandWord(core.PlainsLandType, core.SwampLandType)
				}
				if err := g.AddPrintedMixedStaticEventTextEffect(g.FindPermanent(source.ID()), core.Indefinite, later); err != nil {
					t.Fatal(err)
				}
				if g.StackPeek() != pending {
					t.Fatal("object text replaced an earlier pending upkeep")
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
func TestRONTextMixedStaticEventObjectLifecycle(t *testing.T) {
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
	forest, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	if err := g.AddPrintedMixedStaticEventTextEffect(source, core.EndOfTurn, green, forest); err != nil {
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

func TestRONTextMixedStaticEventObjectOrdering(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Gauntlet of Might"), a.PlayerID())
	redGreen, _ := mage.ReplaceColorWord(core.Red, core.Green)
	greenBlue, _ := mage.ReplaceColorWord(core.Green, core.Blue)
	mountainForest, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.ForestLandType)
	forestIsland, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	if err := g.AddPrintedMixedStaticEventTextEffect(source, core.EndOfTurn, redGreen, mountainForest); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedMixedStaticEventTextEffect(g.FindPermanent(source.ID()), core.Indefinite, greenBlue, forestIsland); err != nil {
		t.Fatal(err)
	}
	blue := g.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	mountain := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
	for range 3 {
		g.ApplyContinuousEffects()
	}
	staticPTCheck(t, g, blue, 2, 2)
	staticPTCheck(t, g, red, 1, 1)
	b.ManaPool().Clear()
	if err := g.TapForMana(b.PlayerID(), island.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Red) != 1 {
		t.Fatal("ordered event words did not reach production")
	}
	clone := g.Clone()
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, blue, 1, 1)
	staticPTCheck(t, g, red, 2, 2)
	staticPTCheck(t, clone, blue, 2, 2)
	staticPTCheck(t, clone, red, 1, 1)
	b.ManaPool().Clear()
	if err := g.TapForMana(b.PlayerID(), mountain.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Red) != 2 {
		t.Fatal("expiry did not restore the event baseline")
	}
}

func TestRONTextMixedStaticEventObjectBoundaries(t *testing.T) {
	for _, name := range []string{"Firebreathing", "Green Ward", "Goblin King", "Northern Paladin"} {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(repositoryCard(t, "LEA", name), a.PlayerID())
		change, _ := mage.ReplaceColorWord(core.Red, core.Green)
		if err := g.AddPrintedMixedStaticEventTextEffect(p, core.Indefinite, change); err == nil {
			t.Fatal("unrelated context accepted")
		}
	}
	consumer := staticBaseRecord(t, "LEA", "Sleight of Mind")
	consumer.Fields["Rules"].Args[0].Args[0] = consumer.Fields["Rules"].Args[0].Args[0].Args[0].Args[1]
	body := consumer.Fields["Rules"].Args[0].Args[0].Args[1].Args[0]
	body.Args[0] = Node{Kind: Symbol, Name: "DrawACard"}
	if _, err := Compile(consumer); err == nil {
		t.Fatal("unbound word producer accepted")
	}
}
