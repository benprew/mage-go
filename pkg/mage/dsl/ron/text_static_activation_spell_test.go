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

func TestRONTextStaticActivationSpellInventory(t *testing.T) {
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
			if _, err := Compile(record); err != nil {
				t.Fatal(err)
			}
			counts[record.Fields["Name"].String]++
		}
	}
	if !maps.Equal(counts, map[string]int{"Sedge Troll": 6, "Thrull Champion": 1}) {
		t.Fatalf("unmapped inventory: %v", counts)
	}
}

func TestRONTextStaticActivationSpellBoundaries(t *testing.T) {
	static := `If(PlayerPassesFilter(You,ControlsA(IsLandType(Swamp))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,1)])])`
	action := `Activated(PayMana([ManaCostB]),ActionList([RegeneratePermanent(ThisPermanent)]))`
	for _, rules := range []string{
		strings.ReplaceAll(static, "AdjustPT(1,1)", "AdjustPT(1)") + `,` + action,
		strings.ReplaceAll(static, "ThisPermanent", "Ref_TargetPermanent") + `,` + action,
		static + `,` + strings.ReplaceAll(action, "ThisPermanent", "Ref_TargetPermanent"),
		static + `,` + strings.ReplaceAll(action, "PayMana([ManaCostB])", "PayMana([InvalidMana])"),
	} {
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[`+rules+`],CardPT:(Power:2,Toughness:2))`)
		if _, err := Compile(node); err == nil {
			t.Fatal("invalid declaration accepted", rules)
		}
	}
	for _, rules := range []string{
		static + `,` + action + `,Flying`,
		static + `,` + action + `,` + action,
		static + `,ActivatedWithModifiers(PayMana([ManaCostB]),ActionList([RegeneratePermanent(ThisPermanent)]),ActivateOnlyAsASorcery)`,
		static + `,Activated(PayMana([ManaCostB]),ActionList([If(TimesThisAbilityHasBeenActivatedThisTurnIs(GreaterThanOrEqualTo(Integer(2))),[RegeneratePermanent(ThisPermanent)])]))`,
	} {
		node := nameScopeNode(t, `Card(Name:"Broader",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[`+rules+`],CardPT:(Power:2,Toughness:2))`)
		factory, err := Compile(node)
		if err != nil {
			t.Fatal(err)
		}
		g, a, _ := ronGame()
		obj := castRONSpell(t, g, a, factory())
		change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
		if err := g.RevisePrintedSpellText(obj.ID, change); err == nil || g.StackPeek() != obj {
			t.Fatal("broader context accepted or mutated")
		}
	}
}

func TestRONTextStaticActivationSpellRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set+"/Sedge Troll", func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, set, "Sedge Troll")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Red, 10)
			a.ManaPool().Add(core.Black, 10)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			paid := a.ManaPool().TotalMana()
			change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
			color, _ := mage.ReplaceColorWord(core.Black, core.Green)
			if err := g.RevisePrintedSpellText(g.StackPeek().ID, change, color); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != paid || g.StackPeek().Card != card {
				t.Fatal("revision changed payment or physical card")
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), a.PlayerID())
			staticPTCheck(t, g, p, 2, 2)
			g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
			staticPTCheck(t, g, p, 2, 2)
			g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), a.PlayerID())
			staticPTCheck(t, g, p, 3, 3)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != paid-1 {
				t.Fatal("printed regeneration cost changed")
			}
			g.ResolveStack()
			g.DestroyPermanent(g.FindPermanent(p.ID()))
			if g.FindPermanent(p.ID()) == nil || !g.FindPermanent(p.ID()).Tapped {
				t.Fatal("fresh regeneration body lost its source")
			}
		})
	}
	t.Run("FEM/Thrull Champion", func(t *testing.T) {
		g, a, b := ronGame()
		card := repositoryCard(t, "FEM", "Thrull Champion")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Black, 10)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		change, _ := mage.ReplaceColorWord(core.Black, core.Green)
		if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		p := g.FindPermanent(card.ID())
		staticPTCheck(t, g, p, 3, 3)
		p.RevokeBaseAttr(core.AttrSummonSick)
		target := g.PutOnBattlefield(repositoryCard(t, "FEM", "Basal Thrull"), b.PlayerID())
		staticPTCheck(t, g, target, 2, 3)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if g.FindPermanent(target.ID()).ControllerID() != a.PlayerID() {
			t.Fatal("fresh activation lost its target binding")
		}
		g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
		g.ApplyContinuousEffects()
		if g.FindPermanent(target.ID()).ControllerID() != b.PlayerID() {
			t.Fatal("source control change did not end the stolen Thrull rule")
		}
	})
}

func TestRONTextStaticActivationSpellLifecycle(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Sedge Troll")
	obj := castRONSpell(t, g, a, card)
	change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.RevisePrintedSpellText(obj.ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	ptID, activationID := textStaticID(t, p), p.RuntimeAbilities[1].AbilityID()
	swamp := g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), a.PlayerID())
	island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), a.PlayerID())
	staticPTCheck(t, g, p, 3, 3)
	second, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.MountainLandType)
	if err := g.AddPrintedStaticPTTextEffect(p, ptID, core.EndOfTurn, second); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedActivatedTextEffect(p, activationID, core.EndOfTurn, second); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, p, 2, 2)
	phased := g.PhaseOut(p.ID())
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	p = g.FindPermanent(p.ID())
	staticPTCheck(t, g, p, 3, 3)
	if textStaticID(t, p) != ptID || p.RuntimeAbilities[1].AbilityID() != activationID {
		t.Fatal("phase cycle changed sibling identity")
	}
	phased = g.PhaseOut(island.ID())
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, p, 2, 2)
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, p, 3, 3)
	g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
	staticPTCheck(t, g, p, 2, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err == nil {
		t.Fatal("removed activation remained available")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, p, 3, 3)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
	staticPTCheck(t, g, p, 2, 2)
	b.ManaPool().Add(core.Black, 1)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.DestroyPermanent(g.FindPermanent(p.ID()))
	if g.FindPermanent(p.ID()) == nil {
		t.Fatal("restored activation lost its new controller or source")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	values, err := g.CaptureCopiableValues(g.FindPermanent(p.ID()))
	if err != nil {
		t.Fatal(err)
	}
	printed, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	copyGame, copyA, _ := ronGame()
	printedPermanent := copyGame.PutOnBattlefield(printed, copyA.PlayerID())
	copyGame.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), copyA.PlayerID())
	staticPTCheck(t, copyGame, printedPermanent, 2, 2)
	copyGame.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), copyA.PlayerID())
	staticPTCheck(t, copyGame, printedPermanent, 3, 3)
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	lki := g.LKI(p.ID())
	if lki == nil || lki.Power != 3 || lki.Toughness != 3 {
		t.Fatal("departure lost changed P/T")
	}
	savedGame, savedA, _ := ronGame()
	saved := savedGame.PutOnBattlefield(lki.Snapshot.Card.Copy(), savedA.PlayerID())
	savedGame.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), savedA.PlayerID())
	staticPTCheck(t, savedGame, saved, 3, 3)
	if err := g.AddPrintedActivatedTextEffect(p, activationID, core.Indefinite, second); err == nil {
		t.Fatal("old source visit accepted")
	}
	g.RemoveFromBattlefield(g.FindPermanent(swamp.ID()))
	castRONSpell(t, g, a, a.Hand()[0])
	g.ResolveStack()
	returned := g.FindPermanent(card.ID())
	staticPTCheck(t, g, returned, 2, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), returned.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(g.FindPermanent(returned.ID()))
	newVisit := g.PutOnBattlefield(a.Hand()[0], a.PlayerID())
	a.RemoveFromHand(newVisit.Card.ID())
	g.ResolveStack()
	g.DestroyPermanent(g.FindPermanent(newVisit.ID()))
	if g.FindPermanent(newVisit.ID()) != nil {
		t.Fatal("pending regeneration followed a returned source")
	}
}

func TestRONTextStaticActivationSpellCopies(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Sedge Troll")
	obj := castRONSpell(t, g, a, card)
	first, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	second, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.PlainsLandType)
	third, _ := mage.ReplaceBasicLandWord(core.PlainsLandType, core.ForestLandType)
	if err := g.RevisePrintedSpellText(obj.ID, first); err != nil {
		t.Fatal(err)
	}
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy failed")
	}
	if err := g.RevisePrintedSpellText(copied.ID, second); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := clone.RevisePrintedSpellText(copied.ID, third); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	clone.ResolveStack()
	p, token := g.FindPermanent(card.ID()), g.FindPermanent(copied.SourceID)
	if token == nil || !token.Card.IsToken() || token.ControllerID() != b.PlayerID() || token.RuntimeAbilities[1].AbilityID() == p.RuntimeAbilities[1].AbilityID() {
		t.Fatal("copy shared activation or physical identity")
	}
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), a.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Plains"), b.PlayerID())
	clone.PutOnBattlefield(repositoryCard(t, "LEA", "Plains"), b.PlayerID())
	staticPTCheck(t, g, p, 3, 3)
	staticPTCheck(t, g, token, 3, 3)
	staticPTCheck(t, clone, token, 2, 2)
	clone.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	staticPTCheck(t, clone, token, 3, 3)
	b.ManaPool().Add(core.Black, 1)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), token.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.DestroyPermanent(g.FindPermanent(p.ID()))
	g.DestroyPermanent(g.FindPermanent(token.ID()))
	if g.FindPermanent(p.ID()) != nil || g.FindPermanent(token.ID()) == nil || !g.FindPermanent(token.ID()).Tapped {
		t.Fatal("copy regeneration used the original source")
	}
	if clone.FindPermanent(p.ID()) == nil || clone.FindPermanent(token.ID()).Tapped {
		t.Fatal("clone shared destruction or activation state")
	}
}

func TestRONTextStaticActivationSpellBothWords(t *testing.T) {
	static := `EachPermanentLayerEffect(And([IsCardtype(Creature),IsColor(Black)]),[AdjustPT(1,1)])`
	action := `Activated(PayMana([ManaCostB]),Targeted([TargetPermanent(And([IsCardtype(Creature),IsColor(Black)]))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`
	node := nameScopeNode(t, `Card(Name:"Mixed",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),ManaCost:[],Rules:[`+static+`,`+action+`],CardPT:(Power:2,Toughness:2))`)
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	obj := castRONSpell(t, g, a, factory())
	green, _ := mage.ReplaceColorWord(core.Black, core.Green)
	if err := g.RevisePrintedSpellText(obj.ID, green); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(obj.Card.ID())
	blackTarget := g.PutOnBattlefield(textColorCard(core.Black), b.PlayerID())
	greenTarget := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	staticPTCheck(t, g, blackTarget, 1, 1)
	staticPTCheck(t, g, greenTarget, 2, 2)
	paid := a.ManaPool().TotalMana()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{blackTarget.ID()}); err == nil {
		t.Fatal("old target word remained")
	}
	if a.ManaPool().TotalMana() != paid {
		t.Fatal("illegal target consumed payment")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{greenTarget.ID()}); err != nil {
		t.Fatal(err)
	}
	blue, _ := mage.ReplaceColorWord(core.Green, core.Blue)
	if err := g.AddPrintedStaticPTTextEffect(p, textStaticID(t, p), core.EndOfTurn, blue); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedActivatedTextEffect(p, p.RuntimeAbilities[1].AbilityID(), core.EndOfTurn, blue); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, greenTarget, 1, 1)
	g.ResolveStack()
	if g.FindPermanent(greenTarget.ID()) != nil {
		t.Fatal("later text changed the pending activation")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	greenTarget = g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	staticPTCheck(t, g, greenTarget, 2, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{greenTarget.ID()}); err != nil {
		t.Fatal("expiry lost the continued activation", err)
	}
	greenTarget.GrantBaseAttr(core.Shroud)
	g.ResolveStack()
	if g.FindPermanent(greenTarget.ID()) == nil {
		t.Fatal("revised targets did not recheck legality")
	}
}

func TestRONTextStaticActivationSpellThrullLifecycle(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "FEM", "Thrull Champion")
	obj := castRONSpell(t, g, a, card)
	change, _ := mage.ReplaceColorWord(core.Black, core.Green)
	if err := g.RevisePrintedSpellText(obj.ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	ptID, actionID := textStaticID(t, p), p.RuntimeAbilities[1].AbilityID()
	target := g.PutOnBattlefield(repositoryCard(t, "FEM", "Basal Thrull"), b.PlayerID())
	other := g.PutOnBattlefield(repositoryCard(t, "LEA", "Grizzly Bears"), b.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{other.ID()}); err == nil || g.FindPermanent(p.ID()).Tapped {
		t.Fatal("invalid target consumed tap payment")
	}
	if err := g.AddPrintedStaticPTTextEffect(p, ptID, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedActivatedTextEffect(p, actionID, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	phased := g.PhaseOut(p.ID())
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, target, 1, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{target.ID()}); err == nil {
		t.Fatal("phased source activated")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	p = g.FindPermanent(p.ID())
	if p.RuntimeAbilities[1].AbilityID() != actionID || textStaticID(t, p) != ptID {
		t.Fatal("phase cycle changed declaration identity")
	}
	staticPTCheck(t, g, target, 2, 3)
	staticPTCheck(t, g, other, 2, 2)
	g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
	staticPTCheck(t, g, target, 1, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{target.ID()}); err == nil {
		t.Fatal("removed source activated")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	if g.FindPermanent(target.ID()).ControllerID() != a.PlayerID() || clone.FindPermanent(target.ID()).ControllerID() != a.PlayerID() {
		t.Fatal("clone lost fresh target binding")
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.ApplyContinuousEffects()
	if g.FindPermanent(target.ID()).ControllerID() != b.PlayerID() || clone.FindPermanent(target.ID()).ControllerID() != a.PlayerID() {
		t.Fatal("clone shared source control duration")
	}
	clone.BouncePermanentToHand(clone.FindPermanent(p.ID()))
	if clone.FindPermanent(target.ID()).ControllerID() != b.PlayerID() {
		t.Fatal("source departure retained stolen control")
	}
	lki := clone.LKI(p.ID())
	if lki == nil || lki.Power != 3 || lki.Toughness != 3 {
		t.Fatal("departure lost static rule information")
	}
	if err := clone.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	clone.ResolveStack()
	staticPTCheck(t, clone, clone.FindPermanent(card.ID()), 3, 3)
}

func TestRONTextStaticActivationSpellCopyExceptions(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEA", "Sedge Troll"}, {"FEM", "Thrull Champion"}} {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
		values, err := g.CaptureCopiableValues(p)
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
		change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
		revision, err := mage.BuildTextDefinitionRevision(card, change)
		if err != nil {
			t.Fatal(err)
		}
		if !revision.Definition().HasType(core.TypeArtifact) || len(revision.Definition().Abilities()) != 2 {
			t.Fatal("copy exception lost paired rules or added type")
		}
	}
}
