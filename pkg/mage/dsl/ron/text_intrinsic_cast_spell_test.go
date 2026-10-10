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

func TestRONTextIntrinsicCastSpellRecords(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Sol'kanar the Swamp King")
	obj := castRONSpell(t, g, a, card)
	paid := a.ManaPool().TotalMana()
	land, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	color, _ := mage.ReplaceColorWord(core.Black, core.Blue)
	if err := g.RevisePrintedSpellText(obj.ID, land, color); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil || a.ManaPool().TotalMana() != paid || g.StackPeek().Card != card {
		t.Fatal("revision changed casting or payment")
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		p := game.FindPermanent(card.ID())
		if p == nil || p.Card.Owner() != a.PlayerID() || p.ControllerID() != a.PlayerID() || p.CurrentPower(game) != 5 || p.CurrentToughness(game) != 5 || !p.HasKeyword(core.Islandwalk) || p.HasKeyword(core.Swampwalk) {
			t.Fatal("entry lost identity or revised intrinsic rules")
		}
		game.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
		if !mage.HasLandwalkEvasion(game.FindPermanent(card.ID()), b.PlayerID(), game) {
			t.Fatal("continued landwalk has no evasion")
		}
		player := game.GetPlayer(b.PlayerID())
		bp := b
		if game == clone {
			bp = player.(*mage.SearchPlayer).BasePlayer
		}
		castRONSpell(t, game, bp, textTriggerSpell(core.Blue))
		game.PutTriggersOnStack()
		if game.StackSize() != 2 || game.GetPlayer(a.PlayerID()).Life() != 20 {
			t.Fatal("cast event lost timing or revised color")
		}
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 21 {
			t.Fatal("continued trigger lost controller")
		}
		castRONSpell(t, game, bp, textTriggerSpell(core.Black))
		game.PutTriggersOnStack()
		if game.StackSize() != 1 {
			t.Fatal("old color still triggered")
		}
		game.ResolveStack()
	}
}

func TestRONTextIntrinsicCastSpellLifecycle(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Sol'kanar the Swamp King")
	obj := castRONSpell(t, g, a, card)
	blue, _ := mage.ReplaceColorWord(core.Black, core.Blue)
	island, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	green, _ := mage.ReplaceColorWord(core.Blue, core.Green)
	forest, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	if err := g.RevisePrintedSpellText(obj.ID, blue, island); err != nil {
		t.Fatal(err)
	}
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("spell copy failed")
	}
	if err := g.RevisePrintedSpellText(copied.ID, green, forest); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	red, _ := mage.ReplaceColorWord(core.Green, core.Red)
	if err := clone.RevisePrintedSpellText(copied.ID, red); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	clone.ResolveStack()
	token := g.FindPermanent(copied.SourceID)
	if token == nil || !token.Card.IsToken() || token.ControllerID() != b.PlayerID() || !token.HasKeyword(core.Forestwalk) || !g.FindPermanent(card.ID()).HasKeyword(core.Islandwalk) {
		t.Fatal("copy did not keep separate intrinsic rules")
	}
	castRONSpell(t, g, a, textTriggerSpell(core.Green))
	g.PutTriggersOnStack()
	castRONSpell(t, clone, clone.GetPlayer(a.PlayerID()).(*mage.SearchPlayer).BasePlayer, textTriggerSpell(core.Green))
	clone.PutTriggersOnStack()
	if g.StackSize() != 2 || clone.StackSize() != 1 {
		t.Fatal("clones shared continued event rules")
	}
	g.ResolveStack()
	clone.ResolveStack()
	if b.Life() != 21 || a.Life() != 20 {
		t.Fatal("copy trigger used wrong controller")
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
	if !sibling.HasKeyword(core.Swampwalk) || sibling.HasKeyword(core.Forestwalk) {
		t.Fatal("ordinary text entered copy values")
	}
	castRONSpell(t, g, b, textTriggerSpell(core.Black))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("copiable event lost printed color")
	}
	g.ResolveStack()
	g.RemoveFromBattlefield(g.FindPermanent(sibling.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(token.ID()))
	p := g.FindPermanent(card.ID())
	var landID, eventID = p.RuntimeAbilities[0].AbilityID(), p.RuntimeAbilities[1].AbilityID()
	castRONSpell(t, g, b, textTriggerSpell(core.Blue))
	g.PutTriggersOnStack()
	if err := g.AddPrintedBasicLandwalkTextEffect(p, landID, core.EndOfTurn, forest); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedSpellCastTextEffect(g.FindPermanent(card.ID()), eventID, core.EndOfTurn, green); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
	}
	if !g.FindPermanent(card.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("later layer lost continued intrinsic baseline")
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 21 {
		t.Fatal("pending trigger lost detection controller")
	}
	castRONSpell(t, g, a, textTriggerSpell(core.Green))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("later text layer lost event rule")
	}
	g.ResolveStack()
	if b.Life() != 22 {
		t.Fatal("live trigger did not use current controller")
	}
	g.AddContinuousEffect(mage.RemoveAllAbilities(card.ID(), core.EndOfTurn))
	if g.FindPermanent(card.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("ability removal kept intrinsic rule")
	}
	castRONSpell(t, g, a, textTriggerSpell(core.Green))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("removed event detected cast")
	}
	g.ResolveStack()
	phased := g.PhaseOut(card.ID())
	castRONSpell(t, g, a, textTriggerSpell(core.Green))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("phased source detected cast")
	}
	g.ResolveStack()
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	p = g.FindPermanent(card.ID())
	if !p.HasKeyword(core.Islandwalk) || p.HasKeyword(core.Forestwalk) || p.RuntimeAbilities[0].AbilityID() != landID || p.RuntimeAbilities[1].AbilityID() != eventID {
		t.Fatal("phased expiry lost continued baseline or identity")
	}
	castRONSpell(t, g, a, textTriggerSpell(core.Blue))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("expiry did not restore continued event")
	}
	g.BouncePermanentToHand(p)
	g.ResolveStack()
	if b.Life() != 23 {
		t.Fatal("departure lost pending trigger")
	}
	lki := g.LKI(card.ID())
	if lki == nil || !lki.Snapshot.HasKeyword(core.Islandwalk) || len(a.Hand()) != 1 || a.Hand()[0].AttrSeeds()[core.Swampwalk] == 0 {
		t.Fatal("departure lost LKI or printed rules")
	}
	g.SetActivePlayerIndex(0)
	castRONSpell(t, g, a, a.Hand()[0])
	g.ResolveStack()
	if !g.FindPermanent(card.ID()).HasKeyword(core.Swampwalk) || g.FindPermanent(card.ID()).HasKeyword(core.Islandwalk) {
		t.Fatal("recast inherited old intrinsic text")
	}
	castRONSpell(t, g, b, textTriggerSpell(core.Black))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("recast inherited old event text")
	}
}

func TestRONTextIntrinsicCastSpellInventory(t *testing.T) {
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
			if !intrinsicCastSpellTextRules(record.Fields["Rules"].Args) {
				continue
			}
			if _, err := Compile(record); err != nil {
				t.Fatal(err)
			}
			counts[record.Fields["Name"].String]++
		}
	}
	if !maps.Equal(counts, map[string]int{"Sol'kanar the Swamp King": 1}) {
		t.Fatalf("unmapped inventory: %v", counts)
	}
}

func TestRONTextIntrinsicCastSpellBoundaries(t *testing.T) {
	for _, rules := range []string{
		`Landwalk(IsLandType(Swamp)),TriggerA(WhenAPlayerCastsASpell(AnyPlayer(),IsColor(Black)),ActionList([GainLife(Integer(1))]))`,
		`Landwalk(IsLandType(Swamp)),TriggerA(WhenAPlayerCastsASpell(AnyPlayer,IsColor(Black)),ActionList([PlayerAction(Trigger_ThatSpell,GainLife(Integer(1)))]))`,
		`Landwalk(IsLandType(Swamp)),TriggerA(WhenAPlayerCastsASpell(AnyPlayer,IsColor(Black)),ActionList([If(CostWasPaid,[GainLife(Integer(1))])]))`,
	} {
		records, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[`+rules+`],CardPT:(Power:5,Toughness:5))`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(records[0]); err == nil {
			t.Fatal("invalid declaration accepted", rules)
		}
	}
	for _, rules := range []string{
		`Landwalk(IsLandType(Swamp)),TriggerA(WhenAPlayerCastsASpell(AnyPlayer,IsColor(Black)),ActionList([GainLife(Integer(1))])),TriggerA(WhenAPlayerCastsASpell(AnyPlayer,IsColor(Black)),ActionList([GainLife(Integer(1))]))`,
		`Landwalk(IsLandType(Swamp)),TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([GainLife(Integer(1))]))`,
	} {
		records, err := Parse(strings.NewReader(`Card(Name:"Broader",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[`+rules+`],CardPT:(Power:5,Toughness:5))`), "broader.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(records[0])
		if err != nil {
			t.Fatal(err)
		}
		g, a, _ := ronGame()
		obj := castRONSpell(t, g, a, factory())
		change, _ := mage.ReplaceColorWord(core.Black, core.Blue)
		if err := g.RevisePrintedSpellText(obj.ID, change); err == nil || g.StackPeek() != obj {
			t.Fatal("broader context mutated")
		}
	}
}

func TestRONTextIntrinsicCastSpellEventBindingsAndTargets(t *testing.T) {
	records, err := Parse(strings.NewReader(`Card(Name:"Mixed",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[Landwalk(IsLandType(Swamp)),TriggerA(WhenAPlayerCastsASpell(Opponent,IsColor(Black)),Targeted([TargetPermanent(IsColor(Black))],ActionList([DestroyPermanent(Ref_TargetPermanent),PlayerMayCost(Trigger_ThatPlayer,PayMana([ManaCostGeneric(1)])),Unless(CostWasPaid,[CounterSpell(Trigger_ThatSpell)])])))],CardPT:(Power:5,Toughness:5))`), "mixed.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(records[0])
	if err != nil {
		t.Fatal(err)
	}
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	obj := castRONSpell(t, g, a.BasePlayer, factory())
	change, _ := mage.ReplaceColorWord(core.Black, core.Blue)
	if err := g.RevisePrintedSpellText(obj.ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	target := g.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
	first := castRONSpell(t, g, b.BasePlayer, textTriggerSpell(core.Blue))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 || len(g.StackPeek().Targets) != 1 || g.StackPeek().Targets[0] != target.ID() {
		t.Fatal("fresh target lost revised filter")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(target.ID()) != nil || g.GetStack().FindByID(first.ID) != nil || b.offers != 1 || a.offers != 0 {
		t.Fatal("fresh bindings lost payer or spell")
	}
	target = g.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
	second := castRONSpell(t, g, b.BasePlayer, textTriggerSpell(core.Blue))
	g.PutTriggersOnStack()
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.Red)), target.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(target.ID()) == nil || g.GetStack().FindByID(second.ID) == nil || b.offers != 1 {
		t.Fatal("illegal target did not stop body")
	}
	g.ResolveStack()
	castRONSpell(t, g, a.BasePlayer, textTriggerSpell(core.Blue))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("opponent context changed")
	}
}

func TestRONTextIntrinsicCastSpellIntrinsicGroups(t *testing.T) {
	records, err := Parse(strings.NewReader(`Card(Name:"Mixed",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),ManaCost:[ManaCostB],Rules:[TriggerA(WhenAPlayerCastsASpell(AnyPlayer,IsColor(Black)),ActionList([GainLife(Integer(1))])),Landwalk(IsLandType(Swamp)),Protection(FromColor(Colors([Black]))),Landwalk(IsLandType(Swamp)),FirstStrike],CardPT:(Power:5,Toughness:5))`), "groups.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(records[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	card := factory()
	obj := castRONSpell(t, g, a, card)
	color, _ := mage.ReplaceColorWord(core.Black, core.Blue)
	land, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.RevisePrintedSpellText(obj.ID, color, land); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if !p.HasKeyword(core.FirstStrike) || p.Card.AttrSeeds()[core.Islandwalk] != 2 || p.HasKeyword(core.Swampwalk) || len(p.RuntimeAbilities) != 4 {
		t.Fatal("intrinsic declaration groups lost contributions")
	}
	if !p.HasProtectionFrom(textColorCard(core.Blue)) || p.HasProtectionFrom(textColorCard(core.Black)) {
		t.Fatal("entry lost changed protection")
	}
	if p.Card.ManaCost().String() != card.ManaCost().String() {
		t.Fatal("word change altered mana symbol")
	}
	castRONSpell(t, g, b, textTriggerSpell(core.Blue))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("reversed declaration order lost cast event")
	}
}

func TestRONTextIntrinsicCastSpellCopyExceptions(t *testing.T) {
	g, a, _ := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Sol'kanar the Swamp King"), a.PlayerID())
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
	land, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	color, _ := mage.ReplaceColorWord(core.Black, core.Blue)
	revision, err := mage.BuildTextDefinitionRevision(card, land, color)
	if err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(revision.Definition(), a.PlayerID())
	if !p.HasType(core.TypeLand) || !p.HasKeyword(core.Islandwalk) {
		t.Fatal("copy exception lost types or intrinsic rules")
	}
	castRONSpell(t, g, a, textTriggerSpell(core.Blue))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("copy exception lost event rule")
	}
}
