package ron

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextStaticPTLayerOrderedExpiry(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Bad Moon")
	source := g.PutOnBattlefield(card, a.PlayerID())
	black := g.PutOnBattlefield(textColorCard(core.Black), b.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), a.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	id := textStaticID(t, source)
	first, _ := mage.ReplaceColorWord(core.Black, core.Red)
	second, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.AddPrintedStaticPTTextEffect(source, id, core.EndOfTurn, first); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedStaticPTTextEffect(g.FindPermanent(source.ID()), id, core.Indefinite, second); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
		staticPTCheck(t, g, black, 1, 1)
		staticPTCheck(t, g, red, 1, 1)
		staticPTCheck(t, g, green, 2, 2)
	}
	if err := g.RevisePrintedStaticPTAbility(g.FindPermanent(source.ID()), id, first); err == nil {
		t.Fatal("direct revision replaced layered text")
	}
	clone := g.Clone()
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	staticPTCheck(t, clone, black, 2, 2)
	staticPTCheck(t, clone, green, 1, 1)
	staticPTCheck(t, g, green, 2, 2)
	phased := g.PhaseOut(source.ID())
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, green, 1, 1)
	if err := g.AddPrintedStaticPTTextEffect(source, id, core.EndOfTurn, first); err == nil {
		t.Fatal("phased source accepted")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, black, 2, 2)
	staticPTCheck(t, g, green, 1, 1)
	if err := g.AddPrintedStaticPTTextEffect(g.FindPermanent(source.ID()), id, core.EndOfTurn, first); err != nil {
		t.Fatal(err)
	}
	remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
	removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
	removal.SetSourceID(remover.ID())
	g.AddContinuousEffect(removal)
	staticPTCheck(t, g, green, 1, 1)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
	staticPTCheck(t, g, black, 2, 2)
	staticPTCheck(t, g, green, 1, 1)
	if textStaticID(t, g.FindPermanent(source.ID())) != id {
		t.Fatal("identity changed")
	}
	if err := g.AddPrintedStaticPTTextEffect(g.FindPermanent(source.ID()), id, core.Indefinite, first); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(g.FindPermanent(red.ID()))
	if lki := g.LKI(red.ID()); lki == nil || lki.Power != 2 || lki.Toughness != 2 {
		t.Fatal("LKI lost layered P/T")
	}
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.AddPrintedStaticPTTextEffect(source, id, core.EndOfTurn, first); err == nil {
		t.Fatal("returned visit accepted")
	}
	staticPTCheck(t, g, black, 2, 2)
	staticPTCheck(t, g, green, 1, 1)
	if textStaticID(t, returned) != id {
		t.Fatal("printed declaration changed")
	}
}

func TestRONTextStaticPTLayerRecords(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	records := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			eligible := false
			for _, rule := range node.Fields["Rules"].Args {
				eligible = eligible || (rule.Name == "EachPermanentLayerEffect" && len(rule.Args) == 2 && requiresCreature(rule.Args[0]) && len(rule.Args[1].Args) == 1 && rule.Args[1].Args[0].Name == "AdjustPT")
			}
			if !eligible {
				continue
			}
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			t.Run(filepath.Base(file)+"/"+node.Fields["Name"].String, func(t *testing.T) { testStaticPTLayerRecord(t, node.Fields["Name"].String, factory) })
			names[node.Fields["Name"].String] = true
			records++
		}
	}
	if len(names) != 13 || records != 37 {
		t.Fatalf("unmapped records: %d / %v", records, names)
	}
}

func testStaticPTLayerRecord(t *testing.T, name string, factory mage.CardFactory) {
	t.Helper()
	g, a, b := ronGame()
	color := core.Black
	power, toughness := 1, 1
	controlled, attacking := false, false
	switch name {
	case "Bad Moon":
	case "Gauntlet of Might":
		color = core.Red
	case "Crusade":
		color = core.White
	case "Sunken City":
		color = core.Blue
	case "Jacques le Vert":
		color = core.Green
		power, toughness = 0, 2
		controlled = true
	case "Castle", "Arcades Sabboth":
		power, toughness = 0, 2
		controlled = true
	case "Orcish Oriflamme":
		power, toughness = 1, 0
		controlled, attacking = true, true
	case "Mightstone":
		power, toughness = 1, 0
		attacking = true
	case "Weakstone":
		power, toughness = -1, 0
		attacking = true
	case "Kobold Taskmaster":
		power, toughness = 1, 0
		controlled = true
	case "Rohgahh of Kher Keep":
		power, toughness = 2, 2
		controlled = true
	case "Thrull Champion":
	default:
		t.Fatal("missing gameplay mapping", name)
	}
	subject := func(player mage.Player, c core.Color) *mage.Permanent {
		card := textColorCard(c)
		if name == "Rohgahh of Kher Keep" {
			named := mage.NewCreature("Kobolds of Kher Keep", "", 1, 1)
			named.SetColorOverride([]core.Color{c})
			card = named
		}
		mage.WithSubTypes("Thrull", "Kobold")(card.(*mage.BaseCard))
		return g.PutOnBattlefield(card, player.PlayerID())
	}
	own, other := subject(a, color), subject(b, color)
	destination := core.Red
	if color == core.Red {
		destination = core.Blue
	}
	red := subject(a, destination)
	if attacking {
		g.GetCombat().AddAttacker(own.ID(), b.PlayerID())
		g.GetCombat().AddAttacker(other.ID(), a.PlayerID())
		g.GetCombat().AddAttacker(red.ID(), b.PlayerID())
	}
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	id := textStaticID(t, source)
	change, _ := mage.ReplaceColorWord(color, destination)
	if err := g.AddPrintedStaticPTTextEffect(source, id, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	hasColor := name == "Gauntlet of Might" || name == "Bad Moon" || name == "Crusade" || name == "Sunken City" || name == "Jacques le Vert"
	check := func(revised bool, controller mage.Player) {
		t.Helper()
		for _, p := range []*mage.Permanent{own, other, red} {
			active := !controlled || g.FindPermanent(p.ID()).ControllerID() == controller.PlayerID()
			if hasColor {
				active = active && ((revised && p.ID() == red.ID()) || (!revised && p.ID() != red.ID()))
			}
			wantP, wantT := 1, 1
			if active {
				wantP += power
				wantT += toughness
			}
			staticPTCheck(t, g, p, wantP, wantT)
		}
	}
	check(true, a)
	for range 2 {
		g.ApplyContinuousEffects()
		check(true, a)
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	check(true, b)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	check(false, b)
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	for _, p := range []*mage.Permanent{own, other, red} {
		staticPTCheck(t, g, p, 1, 1)
	}
}

func TestRONTextStaticPTLayerIndependentDeclarations(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `EachPermanentLayerEffect(And([IsColor(Red),IsCardtype(Creature)]),[AdjustPT(1,1)]),EachPermanentLayerEffect(And([IsColor(Red),IsCardtype(Creature)]),[AdjustPT(2,2)])`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	sibling := g.PutOnBattlefield(factory(), a.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	blue := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
	firstID, secondID := source.RuntimeAbilities[0].AbilityID(), source.RuntimeAbilities[1].AbilityID()
	first, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.AddPrintedStaticPTTextEffect(source, textStaticID(t, sibling), core.EndOfTurn, first); err == nil {
		t.Fatal("foreign declaration accepted")
	}
	if err := g.AddPrintedStaticPTTextEffect(source, firstID, core.EndOfTurn, first); err != nil {
		t.Fatal(err)
	}
	second, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.AddPrintedStaticPTTextEffect(g.FindPermanent(source.ID()), secondID, core.Indefinite, second); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, red, 4, 4)
	staticPTCheck(t, g, green, 2, 2)
	staticPTCheck(t, g, blue, 3, 3)
	clone := g.Clone()
	next, _ := mage.ReplaceColorWord(core.Blue, core.Green)
	if err := clone.AddPrintedStaticPTTextEffect(clone.FindPermanent(source.ID()), secondID, core.Indefinite, next); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, clone, green, 4, 4)
	staticPTCheck(t, g, green, 2, 2)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, red, 5, 5)
	staticPTCheck(t, g, green, 1, 1)
	staticPTCheck(t, g, blue, 3, 3)
}

func TestRONTextStaticPTLayerBoundaries(t *testing.T) {
	change, _ := mage.ReplaceColorWord(core.Black, core.Red)
	for _, tc := range []struct{ set, name string }{{"LEA", "Holy Strength"}, {"LEA", "Goblin King"}} {
		g, a, _ := ronGame()
		source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
		if err := g.AddPrintedStaticPTTextEffect(source, textStaticID(t, source), core.EndOfTurn, change); err == nil {
			t.Fatal("unsupported context accepted", tc.name)
		}
	}
	g, a, _ := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Bad Moon"), a.PlayerID())
	for _, duration := range []core.Duration{core.WhileOnBattlefield, core.EndOfCombat} {
		if err := g.AddPrintedStaticPTTextEffect(source, textStaticID(t, source), duration, change); err == nil {
			t.Fatal("unsupported duration accepted")
		}
	}
	if err := g.AddPrintedStaticPTTextEffect(source, textStaticID(t, source), core.EndOfTurn); err == nil {
		t.Fatal("empty changes accepted")
	}
	for _, rule := range []string{`EachPermanentLayerEffect(IsCardtype(Creature),[AdjustPT(1)])`, `EachPermanentLayerEffect(IsCardtype(Creature),[AdjustPT(ValueX,1)])`, `EachPermanentLayerEffect(ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer)),[AdjustPT(1,1)])`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),ManaCost:[],Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", rule)
		}
	}
}

func TestRONTextStaticPTLayerPriorRevisionAndCasting(t *testing.T) {
	g, a, b := ronGame()
	g.SetStep(core.PrecombatMain)
	black := g.PutOnBattlefield(textColorCard(core.Black), b.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), a.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	card := repositoryCard(t, "LEA", "Bad Moon")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, black, 1, 1)
	g.ResolveTopOfStack()
	source := g.FindPermanent(card.ID())
	id := textStaticID(t, source)
	staticPTCheck(t, g, black, 2, 2)
	if a.ManaPool().TotalMana() != 0 {
		t.Fatal("casting payment failed")
	}
	change, _ := mage.ReplaceColorWord(core.Black, core.Red)
	if err := g.RevisePrintedStaticPTAbility(source, id, change); err != nil {
		t.Fatal(err)
	}
	change, _ = mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.AddPrintedStaticPTTextEffect(g.FindPermanent(source.ID()), id, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, black, 1, 1)
	staticPTCheck(t, g, green, 2, 2)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, green, 1, 1)
	staticPTCheck(t, g, red, 2, 2)
}

func TestRONTextStaticPTLayerLiveMembership(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Castle"), a.PlayerID())
	own := g.PutOnBattlefield(textColorCard(core.Black), a.PlayerID())
	other := g.PutOnBattlefield(textColorCard(core.Black), b.PlayerID())
	change, _ := mage.ReplaceColorWord(core.Black, core.Red)
	if err := g.AddPrintedStaticPTTextEffect(source, textStaticID(t, source), core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, own, 1, 3)
	staticPTCheck(t, g, other, 1, 1)
	g.TapPermanent(g.FindPermanent(own.ID()))
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, own, 1, 1)
	late := g.PutOnBattlefield(textColorCard(core.Red), a.PlayerID())
	staticPTCheck(t, g, late, 1, 3)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: late.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	staticPTCheck(t, g, late, 1, 1)

	g, a, b = ronGame()
	source = g.PutOnBattlefield(repositoryCard(t, "LEA", "Bad Moon"), a.PlayerID())
	own = g.PutOnBattlefield(textColorCard(core.Black), a.PlayerID())
	change, _ = mage.ReplaceColorWord(core.Black, core.Red)
	if err := g.AddPrintedStaticPTTextEffect(source, textStaticID(t, source), core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.Red)), own.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, own, 2, 2)
	late = g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	staticPTCheck(t, g, late, 2, 2)
}

func TestRONTextStaticPTLayerSiblingMana(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Gauntlet of Might"), a.PlayerID())
	mountain := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
	forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	change, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.ForestLandType)
	if err := g.AddPrintedStaticPTTextEffect(source, textStaticID(t, source), core.Indefinite, change); err != nil {
		t.Fatal(err)
	}
	if err := g.TapForMana(b.PlayerID(), mountain.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().TotalMana() != 2 {
		t.Fatal("text change reached sibling mana declaration")
	}
	if err := g.TapForMana(b.PlayerID(), forest.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().TotalMana() != 3 {
		t.Fatal("sibling mana filter changed")
	}
}
