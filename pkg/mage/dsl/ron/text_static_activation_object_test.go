package ron

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func castStaticActivationWordBranch(t *testing.T, g *mage.Game, a mage.Player, source *mage.Permanent, name string) {
	t.Helper()
	record := staticBaseRecord(t, "LEA", name)
	record.Fields["Rules"].Args[0].Args[0] = record.Fields["Rules"].Args[0].Args[0].Args[0].Args[1]
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{source.ID()}, 0); err != nil {
		t.Fatal(err)
	}
}

func TestRONTextStaticActivationObjectRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set+"/Sedge Troll", func(t *testing.T) {
			g, base, b := ronGame()
			a := &intrinsicWordTester{Player: base, words: []string{"Swamp", "Island"}}
			g.SetPlayerAt(0, a)
			card := repositoryCard(t, set, "Sedge Troll")
			castRONSpell(t, g, base, card)
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			ptID, actionID := textStaticID(t, p), p.RuntimeAbilities[1].AbilityID()
			swamp := g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), a.PlayerID())
			staticPTCheck(t, g, p, 3, 3)
			castStaticActivationWordBranch(t, g, a, p, "Magical Hack")
			if a.calls != 0 {
				t.Fatal("early word choice")
			}
			g.ResolveStack()
			if a.calls != 2 {
				t.Fatal("missing word choice")
			}
			staticPTCheck(t, g, p, 2, 2)
			g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
			staticPTCheck(t, g, p, 2, 2)
			island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), a.PlayerID())
			staticPTCheck(t, g, p, 3, 3)
			if textStaticID(t, g.FindPermanent(p.ID())) != ptID || g.FindPermanent(p.ID()).RuntimeAbilities[1].AbilityID() != actionID {
				t.Fatal("sibling identity changed")
			}
			g.FindPermanent(swamp.ID()).Tapped = true
			g.FindPermanent(island.ID()).Tapped = true
			a.ManaPool().Clear()
			a.ManaPool().Add(core.Green, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err == nil || a.ManaPool().TotalMana() != 1 {
				t.Fatal("unpaid activation changed state")
			}
			a.ManaPool().Add(core.Black, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			g.DestroyPermanent(g.FindPermanent(p.ID()))
			if g.FindPermanent(p.ID()) == nil || !g.FindPermanent(p.ID()).Tapped {
				t.Fatal("changed text lost regeneration")
			}
			second, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
			if err := g.AddPrintedStaticActivationTextEffect(g.FindPermanent(p.ID()), core.EndOfTurn, second); err != nil {
				t.Fatal(err)
			}
			staticPTCheck(t, g, p, 2, 2)
			clone := g.Clone()
			clone.GetEffects().RemoveEndOfTurn()
			clone.ApplyContinuousEffects()
			staticPTCheck(t, clone, p, 3, 3)
			staticPTCheck(t, g, p, 2, 2)
			phased := g.PhaseOut(p.ID())
			g.GetEffects().RemoveEndOfTurn()
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, p, 3, 3)
			g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
			staticPTCheck(t, g, p, 2, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err == nil {
				t.Fatal("removed action activated")
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, p, 3, 3)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
			staticPTCheck(t, g, p, 3, 3)
			g.PhaseOut(g.FindPermanentIncludingPhased(island.ID()).ID())
			staticPTCheck(t, g, p, 3, 3)
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, p, 2, 2)
			g.PhaseIn([]uuid.UUID{island.ID()})
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, p, 3, 3)
			values, err := g.CaptureCopiableValues(g.FindPermanent(p.ID()))
			if err != nil {
				t.Fatal(err)
			}
			copyCard, err := values.Build()
			if err != nil {
				t.Fatal(err)
			}
			copied := g.PutOnBattlefield(copyCard, a.PlayerID())
			staticPTCheck(t, g, copied, 3, 3)
			g.RemoveFromBattlefield(g.FindPermanent(swamp.ID()))
			staticPTCheck(t, g, copied, 2, 2)
			staticPTCheck(t, g, p, 3, 3)
			stale := g.FindPermanent(p.ID())
			g.BouncePermanentToHand(stale)
			if g.LKI(p.ID()) == nil || g.LKI(p.ID()).Power != 3 {
				t.Fatal("departure lost final P/T")
			}
			castRONSpell(t, g, base, a.Hand()[0])
			g.ResolveStack()
			staticPTCheck(t, g, g.FindPermanent(card.ID()), 2, 2)
			if err := g.AddPrintedStaticActivationTextEffect(stale, core.Indefinite, second); err == nil {
				t.Fatal("returned visit inherited text")
			}
		})
	}
	t.Run("FEM/Thrull Champion", func(t *testing.T) {
		g, base, b := ronGame()
		a := &intrinsicWordTester{Player: base, words: []string{"Black", "Green"}}
		g.SetPlayerAt(0, a)
		card := repositoryCard(t, "FEM", "Thrull Champion")
		castRONSpell(t, g, base, card)
		g.ResolveStack()
		p := g.FindPermanent(card.ID())
		castStaticActivationWordBranch(t, g, a, p, "Sleight of Mind")
		g.ResolveStack()
		change, _ := mage.ReplaceColorWord(core.Black, core.Green)
		if err := g.AddPrintedStaticActivationTextEffect(g.FindPermanent(p.ID()), core.EndOfTurn, change); err != nil {
			t.Fatal(err)
		}
		target := g.PutOnBattlefield(repositoryCard(t, "FEM", "Basal Thrull"), b.PlayerID())
		other := g.PutOnBattlefield(repositoryCard(t, "LEA", "Grizzly Bears"), b.PlayerID())
		staticPTCheck(t, g, p, 3, 3)
		staticPTCheck(t, g, target, 2, 3)
		staticPTCheck(t, g, other, 2, 2)
		g.FindPermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{other.ID()}); err == nil || g.FindPermanent(p.ID()).Tapped {
			t.Fatal("illegal target consumed tap")
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
			t.Fatal(err)
		}
		clone := g.Clone()
		g.ResolveStack()
		clone.ResolveStack()
		if g.FindPermanent(target.ID()).ControllerID() != a.PlayerID() || clone.FindPermanent(target.ID()).ControllerID() != a.PlayerID() {
			t.Fatal("pending clone lost target")
		}
		g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
		if g.FindPermanent(target.ID()).ControllerID() != b.PlayerID() || clone.FindPermanent(target.ID()).ControllerID() != a.PlayerID() {
			t.Fatal("source-control duration crossed clones")
		}
	})
}

func TestRONTextStaticActivationObjectBothWords(t *testing.T) {
	node := nameScopeNode(t, `Card(Name:"Mixed",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[EachPermanentLayerEffect(And([IsCardtype(Creature),IsColor(Black)]),[AdjustPT(1,1)]),Activated(PayMana([ManaCostB]),Targeted([TargetPermanent(And([IsCardtype(Creature),IsColor(Black)]))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))],CardPT:(Power:2,Toughness:2))`)
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	green, _ := mage.ReplaceColorWord(core.Black, core.Green)
	if err := g.AddPrintedStaticActivationTextEffect(p, core.Indefinite, green); err != nil {
		t.Fatal(err)
	}
	blackTarget := g.PutOnBattlefield(textColorCard(core.Black), b.PlayerID())
	greenTarget := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	staticPTCheck(t, g, blackTarget, 1, 1)
	staticPTCheck(t, g, greenTarget, 2, 2)
	a.ManaPool().Add(core.Black, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{blackTarget.ID()}); err == nil || a.ManaPool().TotalMana() != 2 {
		t.Fatal("old word remained or failed payment spent mana")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{greenTarget.ID()}); err != nil {
		t.Fatal(err)
	}
	blue, _ := mage.ReplaceColorWord(core.Green, core.Blue)
	if err := g.AddPrintedStaticActivationTextEffect(g.FindPermanent(p.ID()), core.EndOfTurn, blue); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, greenTarget, 1, 1)
	g.ResolveStack()
	if g.FindPermanent(greenTarget.ID()) != nil {
		t.Fatal("later text changed pending body")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	greenTarget = g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{greenTarget.ID()}); err != nil {
		t.Fatal(err)
	}
	greenTarget.GrantBaseAttr(core.Shroud)
	g.ResolveStack()
	if g.FindPermanent(greenTarget.ID()) == nil {
		t.Fatal("resolution ignored target legality")
	}
}

func TestRONTextStaticActivationObjectInventory(t *testing.T) {
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
			if !staticActivationTextRules(record.Fields["Rules"].Args) {
				continue
			}
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			change, _ := mage.ReplaceColorWord(core.Black, core.Green)
			if err := g.AddPrintedStaticActivationTextEffect(p, core.Indefinite, change); err != nil {
				t.Fatal(file, record.Fields["Name"].String, err)
			}
			counts[record.Fields["Name"].String]++
		}
	}
	if !maps.Equal(counts, map[string]int{"Sedge Troll": 6, "Thrull Champion": 1}) {
		t.Fatalf("unmapped complete records: %v", counts)
	}
}

func TestRONTextStaticActivationObjectBoundaries(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEG", "Beasts of Bogardan"}, {"LEG", "Ivory Guardians"}, {"LEA", "Evil Presence"}, {"LEA", "Phantasmal Terrain"}, {"LEA", "Shivan Dragon"}} {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
		before := g.FindPermanent(p.ID())
		change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
		if err := g.AddPrintedStaticActivationTextEffect(p, core.Indefinite, change); err == nil || g.FindPermanent(p.ID()) != before {
			t.Fatal("excluded context accepted or changed", tc.name)
		}
	}
	prefix := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([`
	for _, body := range []string{
		`CreatePermanentLayerEffect(Ref_TargetPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheSecondChosenColorWord)])`,
		`ChooseTwoColorWords,CreatePermanentLayerEffect(Ref_TargetPermanent,[ReplaceLandTypeVariableWithNewLandTypeVariable(TheFirstChosenColorWord,TheSecondChosenColorWord)])`,
		`ChooseTwoColorWords,CreatePermanentLayerEffect(Ref_TargetPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheFirstChosenColorWord)])`,
		`ChooseTwoColorWords,CreatePermanentLayerEffect(Ref_TargetPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheSecondChosenColorWord),AdjustPT(1,1)])`,
	} {
		if _, err := Compile(nameScopeNode(t, prefix+body+`])))])`)); err == nil {
			t.Fatal("invalid word declaration accepted", body)
		}
	}
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		if _, err := Compile(staticBaseRecord(t, "LEA", name)); err == nil || !strings.Contains(err.Error(), "CreateSpellEffect") {
			t.Fatal("complete consumer boundary changed", name, err)
		}
	}
}

func TestRONTextStaticActivationObjectTargetVisit(t *testing.T) {
	g, base, b := ronGame()
	a := &intrinsicWordTester{Player: base, words: []string{"Swamp", "Island"}}
	g.SetPlayerAt(0, a)
	card := repositoryCard(t, "LEA", "Sedge Troll")
	card.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(card, b.PlayerID())
	castStaticActivationWordBranch(t, g, a, p, "Magical Hack")
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	returned := g.PutOnBattlefield(b.Hand()[0], b.PlayerID())
	b.RemoveFromHand(card.ID())
	g.ResolveStack()
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	staticPTCheck(t, g, returned, 2, 2)
	if a.calls != 0 {
		t.Fatal("returned target caused word choices")
	}
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), b.PlayerID())
	staticPTCheck(t, g, returned, 3, 3)
}

func TestRONTextStaticActivationObjectPendingVisit(t *testing.T) {
	g, a, _ := ronGame()
	card := repositoryCard(t, "LEA", "Sedge Troll")
	p := g.PutOnBattlefield(card, a.PlayerID())
	change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.AddPrintedStaticActivationTextEffect(p, core.Indefinite, change); err != nil {
		t.Fatal(err)
	}
	a.ManaPool().Add(core.Black, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	returned := g.PutOnBattlefield(a.Hand()[0], a.PlayerID())
	a.RemoveFromHand(card.ID())
	g.ResolveStack()
	g.DestroyPermanent(g.FindPermanent(returned.ID()))
	if g.FindPermanent(returned.ID()) != nil {
		t.Fatal("pending regeneration followed a returned source")
	}
}

func TestRONTextStaticActivationObjectContinuedBaseline(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Sedge Troll")
	obj := castRONSpell(t, g, a, card)
	first, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.RevisePrintedSpellText(obj.ID, first); err != nil {
		t.Fatal(err)
	}
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	p, token := g.FindPermanent(card.ID()), g.FindPermanent(copied.SourceID)
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), a.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	staticPTCheck(t, g, p, 3, 3)
	staticPTCheck(t, g, token, 3, 3)
	second, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	if err := g.AddPrintedStaticActivationTextEffect(p, core.EndOfTurn, second); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, p, 2, 2)
	staticPTCheck(t, g, token, 3, 3)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, p, 3, 3)
	if !token.Card.IsToken() || token.RuntimeAbilities[1].AbilityID() == p.RuntimeAbilities[1].AbilityID() {
		t.Fatal("copied declarations shared identity")
	}
}
