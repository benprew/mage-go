package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const deathReturnRule = `ReplaceWouldPutIntoGraveyard(APermanentWouldDie(SinglePermanent(ThisPermanent)),[PutItInOwnersHandInstead,CreatePlayerEffectUntil(OwnerOfTheCardReturnedToHandThisWay,[PlaysWithCardInHandRevealed(TheCardReturnedToHandThisWay),CantPlayCardInHand(TheCardReturnedToHandThisWay)],UntilPlayersNextTurn(OwnerOfTheCardReturnedToHandThisWay))])`

func TestRONDeathReturnHandGameplay(t *testing.T) {
	g, a, _ := ronGame()
	card := repositoryCard(t, "LEG", "Firestorm Phoenix")
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("failed payment changed state")
	}
	a.ManaPool().Add(core.Red, 6)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil || !p.HasKeyword(core.Flying) || p.CurrentPower(g) != 3 || p.CurrentToughness(g) != 2 {
		t.Fatal("complete record lost creature behavior")
	}
	g.DestroyPermanent(p)
	if len(a.Hand()) != 1 || len(a.Graveyard()) != 0 || !g.IsHandCardRevealed(a.PlayerID(), card.ID()) || g.CreatureDeaths() != 0 || g.StackSize() != 0 {
		t.Fatal("death was not replaced with the restricted hand visit")
	}
	a.ManaPool().Add(core.Red, 6)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || a.ManaPool().TotalMana() != 6 {
		t.Fatal("restricted cast spent mana or succeeded")
	}
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(core.Untap)
	if !g.IsSpellCastProhibited(a.PlayerID(), card) {
		t.Fatal("opponent's turn ended restriction")
	}
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(core.PrecombatMain)
	if g.IsHandCardRevealed(a.PlayerID(), card.ID()) {
		t.Fatal("next actual turn retained reveal")
	}
	a.ManaPool().Add(core.Red, 6)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.DoSacrifice(g.FindPermanent(card.ID()))
	if !g.IsSpellCastProhibited(a.PlayerID(), card) || g.CreatureDeaths() != 0 {
		t.Fatal("recast did not create a fresh replacement and lock")
	}
}

func TestRONDeathReturnHandCopiesAndOwner(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Firestorm Phoenix")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 6)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	stackCopy := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	if stackCopy == nil {
		t.Fatal("permanent spell copy failed")
	}
	g.ResolveStack()
	var token *mage.Permanent
	for _, p := range g.AllBattlefield() {
		if p.IsToken {
			token = p
		}
	}
	if token == nil {
		t.Fatal("copied spell did not enter as a token")
	}
	g.DestroyPermanent(token)
	if len(b.Hand()) != 0 || g.CreatureDeaths() != 0 {
		t.Fatal("token return created a hand card or death")
	}
	p := g.FindPermanent(card.ID())
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	clone := g.Clone()
	g.DoSacrifice(g.FindPermanent(p.ID()))
	if !g.IsHandCardRevealed(a.PlayerID(), card.ID()) || len(b.Hand()) != 0 || clone.FindPermanent(p.ID()) == nil {
		t.Fatal("return used controller or changed clone")
	}
	clone.PutPermanentIntoGraveyard(clone.FindPermanent(p.ID()))
	if !clone.IsSpellCastProhibited(a.PlayerID(), card) || len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 0 {
		t.Fatal("clone lost replacement declaration")
	}
}

func TestRONDeathReturnHandLoadAndInventory(t *testing.T) {
	node := repositoryRecord(t, "LEG", "Firestorm Phoenix", "phoenix.ron")
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	first, second := factory(), factory()
	if first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("factories shared declarations")
	}
	file, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(file), "\n") {
		if strings.Contains(line, `Name: "Firestorm Phoenix"`) {
			record = line
		}
	}
	files := fstest.MapFS{"phoenix.ron": {Data: []byte(record)}}
	if registry, err := LoadFS(files, "*.ron"); err != nil || registry == nil {
		t.Fatalf("strict load: %v", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory: %v", err)
	}
	for name, role := range map[string]string{"APermanentWouldDie": "temporary_death_replacement_event", "ThisPermanent": "permanent_reference", "PutItInOwnersHandInstead": "death_return_operation", "OwnerOfTheCardReturnedToHandThisWay": "player_reference", "TheCardReturnedToHandThisWay": "card_reference", "CantPlayCardInHand": "returned_hand_rule", "UntilPlayersNextTurn": "duration"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role %s", name, role)
		}
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(record, "ThisPermanent", "ThisPermanent()", 1))}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("invalid record published a partial registry")
	}
}

func TestRONDeathReturnHandDeclarations(t *testing.T) {
	for _, rule := range []string{
		strings.Replace(deathReturnRule, "ThisPermanent", "Ref_TargetPermanent", 1),
		strings.Replace(deathReturnRule, "ThisPermanent", "ThisPermanent()", 1),
		strings.Replace(deathReturnRule, "PutItInOwnersHandInstead", "PutItInOwnersHandInstead()", 1),
		strings.Replace(deathReturnRule, "OwnerOfTheCardReturnedToHandThisWay,", "You,", 1),
		strings.Replace(deathReturnRule, "TheCardReturnedToHandThisWay)", "ThisCardInHand)", 1),
		strings.Replace(deathReturnRule, "UntilPlayersNextTurn(OwnerOfTheCardReturnedToHandThisWay)", "UntilEndOfTurn", 1),
		strings.Replace(deathReturnRule, "CantPlayCardInHand(TheCardReturnedToHandThisWay)", "Unknown", 1),
		strings.Replace(deathReturnRule, "APermanentWouldDie", "APermanentWouldLeave", 1),
		strings.Replace(deathReturnRule, "PutItInOwnersHandInstead,", "", 1),
		`SpellActions(ActionList([` + deathReturnRule + `]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}
