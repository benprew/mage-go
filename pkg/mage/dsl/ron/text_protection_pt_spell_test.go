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

func TestRONTextProtectionPTSpellRecords(t *testing.T) {
	for _, name := range []string{"Beasts of Bogardan", "Ivory Guardians"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", name)
			obj := castRONSpell(t, g, a, card)
			paid := a.ManaPool().TotalMana()
			blue, _ := mage.ReplaceColorWord(core.Red, core.Blue)
			white, _ := mage.ReplaceColorWord(core.White, core.Blue)
			if err := g.RevisePrintedSpellText(obj.ID, blue, white); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil || g.StackPeek().Card != card || a.ManaPool().TotalMana() != paid {
				t.Fatal("revision changed identity, payment or timing")
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p == nil || p.Card.Owner() != a.PlayerID() || p.Card.ManaCost().String() != card.ManaCost().String() {
				t.Fatal("entry lost identity or printed mana")
			}
			staticPTCheck(t, g, p, 3, 3)
			qualifier := g.PutOnBattlefield(mage.NewEnchantment("Blue condition", "{U}"), b.PlayerID())
			staticPTCheck(t, g, p, 4, 4)
			blueSource := g.PutOnBattlefield(mage.NewCreature("Blue source", "{U}", 1, 1), a.PlayerID())
			g.DealDamageToPermanent(g.FindPermanent(p.ID()), 1, blueSource.ID())
			if g.FindPermanent(p.ID()).Damage != 0 {
				t.Fatal("continued protection did not prevent damage")
			}
			g.RemoveFromBattlefield(qualifier)
			staticPTCheck(t, g, p, 3, 3)
		})
	}
}

func TestRONTextProtectionPTSpellLifecycle(t *testing.T) {
	for _, name := range []string{"Beasts of Bogardan", "Ivory Guardians"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", name)
			obj := castRONSpell(t, g, a, card)
			blue, _ := mage.ReplaceColorWord(core.Red, core.Blue)
			white, _ := mage.ReplaceColorWord(core.White, core.Blue)
			green, _ := mage.ReplaceColorWord(core.Blue, core.Green)
			if err := g.RevisePrintedSpellText(obj.ID, blue, white); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
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
			if err := g.AddPrintedStaticPTTextEffect(p, ptID, core.EndOfTurn, green); err != nil {
				t.Fatal(err)
			}
			if err := g.AddPrintedProtectionTextEffect(g.FindPermanent(card.ID()), protectionID, core.EndOfTurn, green); err != nil {
				t.Fatal(err)
			}
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
			savedGame, savedA, savedB := ronGame()
			saved := savedGame.PutOnBattlefield(lki.Snapshot.Card.Copy(), savedA.PlayerID())
			savedSource := savedGame.PutOnBattlefield(textColorCard(core.Blue), savedB.PlayerID())
			staticPTCheck(t, savedGame, saved, 4, 4)
			savedGame.DealDamageToPermanent(saved, 1, savedSource.ID())
			if savedGame.FindPermanent(saved.ID()).Damage != 0 {
				t.Fatal("LKI lost continued protection")
			}
			redSource := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
			if err := g.AddPrintedStaticPTTextEffect(p, ptID, core.Indefinite, green); err == nil {
				t.Fatal("departed visit accepted")
			}
			castRONSpell(t, g, a, a.Hand()[0])
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

func TestRONTextProtectionPTSpellCopies(t *testing.T) {
	for _, name := range []string{"Beasts of Bogardan", "Ivory Guardians"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", name)
			obj := castRONSpell(t, g, a, card)
			blue, _ := mage.ReplaceColorWord(core.Red, core.Blue)
			white, _ := mage.ReplaceColorWord(core.White, core.Blue)
			green, _ := mage.ReplaceColorWord(core.Blue, core.Green)
			black, _ := mage.ReplaceColorWord(core.Green, core.Black)
			if err := g.RevisePrintedSpellText(obj.ID, blue, white); err != nil {
				t.Fatal(err)
			}
			copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
			if copied == nil {
				t.Fatal("copy failed")
			}
			if err := g.RevisePrintedSpellText(copied.ID, green); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			if err := clone.RevisePrintedSpellText(copied.ID, black); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			clone.ResolveStack()
			token := g.FindPermanent(copied.SourceID)
			if token == nil || !token.Card.IsToken() || token.ControllerID() != b.PlayerID() {
				t.Fatal("copy lost token identity")
			}
			qualifier := g.PutOnBattlefield(textColorCard(core.Green), a.PlayerID())
			clone.PutOnBattlefield(textColorCard(core.Green), a.PlayerID())
			staticPTCheck(t, g, token, 4, 4)
			staticPTCheck(t, clone, token, 3, 3)
			want := 3
			if name == "Ivory Guardians" {
				want = 4
			}
			staticPTCheck(t, g, g.FindPermanent(card.ID()), want, want)
			staticPTCheck(t, clone, clone.FindPermanent(card.ID()), 3, 3)
			g.DealDamageToPermanent(g.FindPermanent(token.ID()), 1, qualifier.ID())
			if g.FindPermanent(token.ID()).Damage != 0 {
				t.Fatal("copy protection lost changed color")
			}
		})
	}
}

func TestRONTextProtectionPTSpellInventory(t *testing.T) {
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
			if !protectionPTSpellTextRules(record.Fields["Rules"].Args) {
				continue
			}
			if _, err := Compile(record); err != nil {
				t.Fatal(err)
			}
			counts[record.Fields["Name"].String]++
		}
	}
	if !maps.Equal(counts, map[string]int{"Beasts of Bogardan": 1, "Ivory Guardians": 1}) {
		t.Fatalf("unmapped inventory: %v", counts)
	}
}

func TestRONTextProtectionPTSpellBoundaries(t *testing.T) {
	protection := `Protection(FromColor(Colors([Red])))`
	conditional := `If(APlayerPassesFilter(Opponent, ControlsA(And([IsNonToken, IsColor(Red)]))), [PermanentLayerEffect(ThisPermanent, [AdjustPT(+1,+1)])])`
	for _, rules := range []string{
		`Protection(FromColor(Colors([]))),` + conditional,
		protection + `,` + strings.ReplaceAll(conditional, "ThisPermanent", "Ref_TargetPermanent"),
		protection + `,` + strings.ReplaceAll(conditional, "AdjustPT(+1,+1)", "AdjustPT(+1)"),
	} {
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[`+rules+`],CardPT:(Power:3,Toughness:3))`)
		if _, err := Compile(node); err == nil {
			t.Fatal("invalid declaration accepted", rules)
		}
	}
	for _, rules := range []string{
		protection + `,` + conditional + `,Flying`,
		protection + `,` + conditional + `,Activated(PayMana([ManaCostB]),ActionList([RegeneratePermanent(ThisPermanent)]))`,
		protection + `,If(APlayerPassesFilter(Opponent,ControlsA(IsLandType(Swamp))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(+1,+1)])])`,
	} {
		node := nameScopeNode(t, `Card(Name:"Broader",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[`+rules+`],CardPT:(Power:3,Toughness:3))`)
		factory, err := Compile(node)
		if err != nil {
			t.Fatal(err)
		}
		g, a, _ := ronGame()
		obj := castRONSpell(t, g, a, factory())
		change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
		if err := g.RevisePrintedSpellText(obj.ID, change); err == nil || g.StackPeek() != obj {
			t.Fatal("broader context accepted or mutated")
		}
	}
}

func TestRONTextProtectionPTSpellCopyExceptions(t *testing.T) {
	for _, name := range []string{"Beasts of Bogardan", "Ivory Guardians"} {
		g, a, b := ronGame()
		source := g.PutOnBattlefield(repositoryCard(t, "LEG", name), a.PlayerID())
		values, err := g.CaptureCopiableValues(source)
		if err != nil {
			t.Fatal(err)
		}
		modified, err := values.WithModifications(mage.CopyAddCardTypes(core.TypeArtifact))
		if err != nil {
			t.Fatal(err)
		}
		card, err := modified.Build()
		if err != nil {
			t.Fatal(err)
		}
		blue, _ := mage.ReplaceColorWord(core.Red, core.Blue)
		white, _ := mage.ReplaceColorWord(core.White, core.Blue)
		revised, err := mage.BuildTextDefinitionRevision(card, blue, white)
		if err != nil {
			t.Fatal(err)
		}
		if !revised.Definition().HasType(core.TypeArtifact) {
			t.Fatal("copy exception lost")
		}
		g.RemoveFromBattlefield(source)
		p := g.PutOnBattlefield(revised.Definition(), a.PlayerID())
		g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
		staticPTCheck(t, g, p, 4, 4)
	}
}
