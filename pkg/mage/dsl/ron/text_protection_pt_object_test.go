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

func TestRONTextProtectionPTObjectRecords(t *testing.T) {
	for _, name := range []string{"Beasts of Bogardan", "Ivory Guardians"} {
		t.Run(name, func(t *testing.T) {
			g, base, b := ronGame()
			a := &intrinsicWordTester{Player: base, words: []string{"Red", "Blue"}}
			if name == "Beasts of Bogardan" {
				a.words = []string{"White", "Blue", "Red", "Blue"}
			}
			g.SetPlayerAt(0, a)
			card := repositoryCard(t, "LEG", name)
			castRONSpell(t, g, base, card)
			g.ResolveStack()
			original := g.FindPermanent(card.ID())
			castStaticActivationWordBranch(t, g, a, original, "Sleight of Mind")
			if a.calls != 0 {
				t.Fatal("early word choice")
			}
			g.ResolveStack()
			if a.calls != 2 {
				t.Fatal("missing word choices")
			}
			if name == "Beasts of Bogardan" {
				castStaticActivationWordBranch(t, g, a, g.FindPermanent(card.ID()), "Sleight of Mind")
				g.ResolveStack()
			}
			green, _ := mage.ReplaceColorWord(core.Blue, core.Green)
			p := g.FindPermanent(card.ID())
			ptID := textStaticID(t, p)
			protectionID := p.RuntimeAbilities[0].AbilityID()
			oldColor := core.Red
			if name == "Beasts of Bogardan" {
				oldColor = core.White
			}
			old := g.PutOnBattlefield(textColorCard(oldColor), b.PlayerID())
			staticPTCheck(t, g, p, 3, 3)
			g.RemoveFromBattlefield(old)
			tokenCard := mage.NewToken("Token condition", 1, 1, []core.CardType{core.TypeCreature}, nil)
			tokenCard.SetColorOverride([]core.Color{core.Blue})
			token := g.PutOnBattlefield(tokenCard, b.PlayerID())
			staticPTCheck(t, g, p, 3, 3)
			g.RemoveFromBattlefield(token)
			wrong := g.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
			staticPTCheck(t, g, p, 3, 3)
			g.RemoveFromBattlefield(wrong)
			qualifier := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
			staticPTCheck(t, g, p, 4, 4)
			if name == "Ivory Guardians" {
				sibling := g.PutOnBattlefield(repositoryCard(t, "LEG", name), b.PlayerID())
				staticPTCheck(t, g, sibling, 4, 4)
				unrelated := g.PutOnBattlefield(mage.NewCreature("Other name", "", 3, 3), b.PlayerID())
				staticPTCheck(t, g, unrelated, 3, 3)
				g.RemoveFromBattlefield(sibling)
				g.RemoveFromBattlefield(unrelated)
			}
			if err := g.AddPrintedProtectionPTTextEffect(p, core.EndOfTurn, green); err != nil {
				t.Fatal(err)
			}
			independent := g.Clone()
			independent.GetEffects().RemoveEndOfTurn()
			independent.ApplyContinuousEffects()
			staticPTCheck(t, independent, p, 4, 4)
			staticPTCheck(t, g, p, 3, 3)
			source := g.PutOnBattlefield(textColorCard(core.Green), a.PlayerID())
			g.DealDamageToPermanent(g.FindPermanent(p.ID()), 1, source.ID())
			if g.FindPermanent(p.ID()).Damage != 0 {
				t.Fatal("later protection layer did not apply")
			}
			g.RemoveFromBattlefield(source)
			phase := g.PhaseOut(card.ID())
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			g.PhaseIn(phase)
			g.ApplyContinuousEffects()
			p = g.FindPermanent(card.ID())
			staticPTCheck(t, g, p, 4, 4)
			if textStaticID(t, p) != ptID || p.RuntimeAbilities[0].AbilityID() != protectionID {
				t.Fatal("phasing changed declaration identity")
			}
			source = g.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
			g.DealDamageToPermanent(p, 1, source.ID())
			if g.FindPermanent(p.ID()).Damage != 0 {
				t.Fatal("expired layer lost continued protection")
			}
			g.RemoveFromBattlefield(source)
			phase = g.PhaseOut(qualifier.ID())
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, p, 3, 3)
			g.PhaseIn(phase)
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, p, 4, 4)
			g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
			staticPTCheck(t, g, p, 3, 3)
			source = g.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
			g.DealDamageToPermanent(g.FindPermanent(p.ID()), 1, source.ID())
			if g.FindPermanent(p.ID()).Damage != 1 {
				t.Fatal("removed protection still active")
			}
			g.RemoveFromBattlefield(source)
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, p, 4, 4)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
			staticPTCheck(t, g, p, 3, 3)
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, p, 4, 4)
			values, err := g.CaptureCopiableValues(g.FindPermanent(p.ID()))
			if err != nil {
				t.Fatal(err)
			}
			printed, err := values.Build()
			if err != nil {
				t.Fatal(err)
			}
			copyGame, copyA, copyB := ronGame()
			copyGame.PutOnBattlefield(textColorCard(core.Blue), copyB.PlayerID())
			sibling := copyGame.PutOnBattlefield(printed, copyA.PlayerID())
			staticPTCheck(t, copyGame, sibling, 3, 3)
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			lki := g.LKI(p.ID())
			if lki == nil || lki.Power != 4 || lki.Toughness != 4 {
				t.Fatal("departure lost final LKI")
			}
			if !lki.Snapshot.HasProtectionFrom(textColorCard(core.Blue)) {
				t.Fatal("LKI lost changed protection")
			}
			redSource := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
			if err := g.AddPrintedProtectionPTTextEffect(p, core.Indefinite, green); err == nil {
				t.Fatal("departed visit accepted")
			}
			castRONSpell(t, g, base, a.Hand()[0])
			g.ResolveStack()
			expected := 3
			if name == "Ivory Guardians" {
				expected = 4
			}
			staticPTCheck(t, g, g.FindPermanent(card.ID()), expected, expected)
			g.DealDamageToPermanent(g.FindPermanent(card.ID()), 1, redSource.ID())
			if g.FindPermanent(card.ID()).Damage != 0 {
				t.Fatal("recast lost printed protection")
			}
		})
	}
}

func TestRONTextProtectionPTObjectOrderAndContinuation(t *testing.T) {
	for _, name := range []string{"Beasts of Bogardan", "Ivory Guardians"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			obj := castRONSpell(t, g, a, repositoryCard(t, "LEG", name))
			redBlue, _ := mage.ReplaceColorWord(core.Red, core.Blue)
			whiteBlue, _ := mage.ReplaceColorWord(core.White, core.Blue)
			if err := g.RevisePrintedSpellText(obj.ID, redBlue, whiteBlue); err != nil {
				t.Fatal(err)
			}
			copied := g.CopySpellOnStack(obj.Card.ID(), b.PlayerID(), false)
			if copied == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			p, token := g.FindPermanent(obj.Card.ID()), g.FindPermanent(copied.SourceID)
			if !token.Card.IsToken() || token.RuntimeAbilities[0].AbilityID() == p.RuntimeAbilities[0].AbilityID() {
				t.Fatal("copy shared identity")
			}
			blueGreen, _ := mage.ReplaceColorWord(core.Blue, core.Green)
			greenBlack, _ := mage.ReplaceColorWord(core.Green, core.Black)
			land, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
			if err := g.AddPrintedProtectionPTTextEffect(p, core.EndOfTurn, land, blueGreen); err != nil {
				t.Fatal(err)
			}
			if err := g.AddPrintedProtectionPTTextEffect(g.FindPermanent(p.ID()), core.Indefinite, greenBlack); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(textColorCard(core.Black), b.PlayerID())
			staticPTCheck(t, g, p, 4, 4)
			if !g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Black)) {
				t.Fatal("ordered protection failed")
			}
			if !g.FindPermanent(token.ID()).HasProtectionFrom(textColorCard(core.Blue)) {
				t.Fatal("copy inherited object layer")
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, p, 3, 3)
			if !g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.Blue)) {
				t.Fatal("expiry lost entry baseline")
			}
			g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
			staticPTCheck(t, g, p, 4, 4)
		})
	}
}

func TestRONTextProtectionPTObjectTargetVisit(t *testing.T) {
	for _, scenario := range []string{"returned", "shroud", "phased"} {
		t.Run(scenario, func(t *testing.T) {
			g, base, b := ronGame()
			a := &intrinsicWordTester{Player: base, words: []string{"Red", "Blue"}}
			g.SetPlayerAt(0, a)
			card := repositoryCard(t, "LEG", "Ivory Guardians")
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			castStaticActivationWordBranch(t, g, a, p, "Sleight of Mind")
			switch scenario {
			case "returned":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				p = g.PutOnBattlefield(b.Hand()[0], b.PlayerID())
				b.RemoveFromHand(card.ID())
			case "shroud":
				g.FindPermanent(p.ID()).GrantBaseAttr(core.Shroud)
			case "phased":
				g.PhaseOut(p.ID())
			}
			g.ResolveStack()
			if a.calls != 0 {
				t.Fatal("illegal target made choices")
			}
			if !g.FindPermanentIncludingPhased(p.ID()).HasProtectionFrom(textColorCard(core.Red)) {
				t.Fatal("illegal target changed words")
			}
		})
	}
}

func TestRONTextProtectionPTObjectBoundaries(t *testing.T) {
	prefix := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([`
	for _, body := range []string{
		`CreatePermanentLayerEffect(Ref_TargetPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheSecondChosenColorWord)])`,
		`ChooseTwoColorWords,CreatePermanentLayerEffect(Ref_TargetPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheFirstChosenColorWord)])`,
		`ChooseTwoColorWords,CreatePermanentLayerEffect(Ref_TargetPermanent,[ReplaceLandTypeVariableWithNewLandTypeVariable(TheFirstChosenColorWord,TheSecondChosenColorWord)])`,
		`ChooseTwoColorWords,CreatePermanentLayerEffect(Ref_TargetPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheSecondChosenColorWord),AdjustPT(1,1)])`,
	} {
		if _, err := Compile(nameScopeNode(t, prefix+body+`])))])`)); err == nil {
			t.Fatal("invalid declaration accepted", body)
		}
	}
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		if _, err := Compile(staticBaseRecord(t, "LEA", name)); err == nil || !strings.Contains(err.Error(), "CreateSpellEffect") {
			t.Fatal("complete consumer boundary changed", name, err)
		}
	}
	for _, tc := range []struct{ set, name string }{{"LEA", "Evil Presence"}, {"LEA", "Phantasmal Terrain"}, {"LEA", "Shivan Dragon"}, {"LEA", "Sedge Troll"}} {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
		before := g.FindPermanent(p.ID())
		change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
		if err := g.AddPrintedProtectionPTTextEffect(p, core.Indefinite, change); err == nil || g.FindPermanent(p.ID()) != before {
			t.Fatal("excluded context accepted or changed", tc.name)
		}
	}
}

func TestRONTextProtectionPTObjectInventory(t *testing.T) {
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
			if !protectionPTTextRules(record.Fields["Rules"].Args) {
				continue
			}
			for _, reverse := range []bool{false, true} {
				if reverse {
					slices.Reverse(record.Fields["Rules"].Args)
				}
				factory, err := Compile(record)
				if err != nil {
					t.Fatal(err)
				}
				g, a, _ := ronGame()
				p := g.PutOnBattlefield(factory(), a.PlayerID())
				change, _ := mage.ReplaceColorWord(core.Black, core.Green)
				if err := g.AddPrintedProtectionPTTextEffect(p, core.Indefinite, change); err != nil {
					t.Fatal(file, record.Fields["Name"].String, err)
				}
			}
			counts[record.Fields["Name"].String]++
		}
	}
	if !maps.Equal(counts, map[string]int{"Beasts of Bogardan": 1, "Ivory Guardians": 1}) {
		t.Fatal("unmapped records", counts)
	}
}

func TestRONTextProtectionPTObjectPendingBindings(t *testing.T) {
	g, base, b := ronGame()
	a := &intrinsicWordTester{Player: base, words: []string{"Red", "Green", "Red", "Black"}}
	g.SetPlayerAt(0, a)
	first := g.PutOnBattlefield(repositoryCard(t, "LEG", "Beasts of Bogardan"), b.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEG", "Ivory Guardians"), b.PlayerID())
	castStaticActivationWordBranch(t, g, a, first, "Sleight of Mind")
	castStaticActivationWordBranch(t, g, a, second, "Sleight of Mind")
	clone := g.Clone()
	clonedPlayer := &intrinsicWordTester{Player: clone.PlayerAt(0), words: []string{"Red", "White", "Red", "Green"}}
	clone.SetPlayerAt(0, clonedPlayer)
	g.ResolveStack()
	clone.ResolveStack()
	check := func(game *mage.Game, p *mage.Permanent, color core.Color) {
		t.Helper()
		if !game.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(color)) {
			t.Fatal("pending choice binding crossed declarations or clones")
		}
	}
	check(g, second, core.Green)
	check(clone, second, core.White)
	check(g, first, core.Black)
	check(clone, first, core.Green)
	check(g, second, core.Green)
	if a.calls != 4 || clonedPlayer.calls != 4 {
		t.Fatal("choices did not belong to each resolution")
	}
}
