package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextGlobalStaticObjectRecords(t *testing.T) {
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
				for _, c := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green, core.Colorless} {
					a.ManaPool().Add(c, 10)
				}
				source := g.PutOnBattlefield(card, a.PlayerID())
				name, from, to := "Sleight of Mind", tc.color.String(), "Blue"
				if tc.color == core.Colorless {
					name, from, to = "Magical Hack", "Mountain", "Island"
				}
				chooser := &intrinsicWordTester{Player: a, words: []string{from, to}}
				g.SetPlayerAt(0, chooser)
				record := staticBaseRecord(t, "LEA", name)
				record.Fields["Rules"].Args[0].Args[0] = record.Fields["Rules"].Args[0].Args[0].Args[0].Args[1]
				factory, err := Compile(record)
				if err != nil {
					t.Fatal(err)
				}
				spell := factory()
				spell.SetOwner(a.PlayerID())
				chooser.AddToHand(spell)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{source.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				if chooser.calls != 0 {
					t.Fatal("word choices ran before resolution")
				}
				g.ResolveStack()
				if chooser.calls != 2 {
					t.Fatal("missing word choices")
				}

				clone := g.Clone()
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

func TestRONTextGlobalStaticObjectPTLifecycle(t *testing.T) {
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
	if err := g.AddPrintedGlobalStaticTextEffect(source, core.EndOfTurn, second); err != nil {
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

func TestRONTextGlobalStaticObjectLandLifecycle(t *testing.T) {
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
	if err := g.AddPrintedGlobalStaticTextEffect(source, core.EndOfTurn, second); err != nil {
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
	if err := g.AddPrintedGlobalStaticTextEffect(source, core.EndOfTurn, second); err == nil {
		t.Fatal("departed visit accepted")
	}
}

func TestRONTextGlobalStaticObjectControl(t *testing.T) {
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
	g.ResolveStack()
	if err := g.AddPrintedGlobalStaticTextEffect(g.FindPermanent(card.ID()), core.Indefinite, change); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, own, 1, 3)
	staticPTCheck(t, g, other, 1, 1)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	staticPTCheck(t, g, own, 1, 1)
	staticPTCheck(t, g, other, 1, 3)
}

func TestRONTextGlobalStaticObjectAtomic(t *testing.T) {
	change, _ := mage.ReplaceColorWord(core.Black, core.Blue)
	for _, tc := range []struct{ set, name string }{{"LEA", "Zombie Master"}, {"LEG", "Kobold Overlord"}, {"LEA", "Gauntlet of Might"}, {"LEA", "Conversion"}, {"ARN", "Kird Ape"}, {"LEA", "Serra Angel"}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, _ := ronGame()
			p := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
			before := g.FindPermanent(p.ID())
			if err := g.AddPrintedGlobalStaticTextEffect(p, core.Indefinite, change); err == nil {
				t.Fatal("unsupported context accepted")
			}
			if g.FindPermanent(p.ID()) != before {
				t.Fatal("rejection changed state")
			}
		})
	}
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "LEA", "Bad Moon"), a.PlayerID())
	for _, duration := range []core.Duration{core.WhileOnBattlefield, core.Indefinite} {
		changes := []mage.TextChange{change}
		if duration == core.Indefinite {
			changes = append(changes, mage.TextChange{})
		}
		before := g.FindPermanent(p.ID())
		if err := g.AddPrintedGlobalStaticTextEffect(p, duration, changes...); err == nil {
			t.Fatal("invalid layer accepted")
		}
		if before != g.FindPermanent(p.ID()) {
			t.Fatal("failed preflight changed state")
		}
	}
	stale := g.FindPermanent(p.ID())
	g.BouncePermanentToHand(stale)
	returned := g.PutOnBattlefield(a.Hand()[0], a.PlayerID())
	if err := g.AddPrintedGlobalStaticTextEffect(stale, core.Indefinite, change); err == nil {
		t.Fatal("returned visit accepted")
	}
	g.PhaseOut(returned.ID())
	if err := g.AddPrintedGlobalStaticTextEffect(returned, core.Indefinite, change); err == nil {
		t.Fatal("phased subject accepted")
	}
}

func TestRONTextGlobalStaticObjectPendingIsolation(t *testing.T) {
	g, base, b := ronGame()
	a := &intrinsicWordTester{Player: base, words: []string{"White", "Blue", "Black", "Green"}}
	g.SetPlayerAt(0, a)
	first := g.PutOnBattlefield(repositoryCard(t, "LEA", "Bad Moon"), b.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEA", "Crusade"), b.PlayerID())
	record := staticBaseRecord(t, "LEA", "Sleight of Mind")
	record.Fields["Rules"].Args[0].Args[0] = record.Fields["Rules"].Args[0].Args[0].Args[0].Args[1]
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	a.ManaPool().Add(core.Blue, 3)
	for _, p := range []*mage.Permanent{first, second} {
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
			t.Fatal(err)
		}
	}
	clone := g.Clone()
	clone.SetPlayerAt(0, &intrinsicWordTester{Player: clone.GetPlayer(a.PlayerID()), words: []string{"White", "Red", "Black", "Red"}})
	g.ResolveStack()
	g.ResolveStack()
	clone.ResolveStack()
	clone.ResolveStack()
	for _, tc := range []struct {
		game  *mage.Game
		color core.Color
		power int
	}{{g, core.Blue, 2}, {g, core.Green, 2}, {g, core.Red, 1}, {clone, core.Red, 3}, {clone, core.Blue, 1}} {
		p := tc.game.PutOnBattlefield(textColorCard(tc.color), b.PlayerID())
		staticPTCheck(t, tc.game, p, tc.power, tc.power)
	}
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(g.FindPermanent(first.ID()))
	returned := g.PutOnBattlefield(b.Hand()[0], b.PlayerID())
	calls := a.calls
	g.ResolveStack()
	if a.calls != calls {
		t.Fatal("returned target made word choices")
	}
	black := g.PutOnBattlefield(textColorCard(core.Black), b.PlayerID())
	staticPTCheck(t, g, black, 2, 2)
	if returned.Card.Name() != first.Card.Name() {
		t.Fatal("return lost printed identity")
	}
}
