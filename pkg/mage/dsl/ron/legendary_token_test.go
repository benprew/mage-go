package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLegendaryTokenGameplay(t *testing.T) {
	record := hostUntapRecord(t, "LEG", "Stangg")
	rules := record.Fields["Rules"]
	body := rules.Args[0].Args[1]
	creation := body.Args[0].Args[0]
	spell := `Card(Name:"Token declaration probe",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),ManaCost:[ManaCostG],Rules:[SpellActions(ActionList([]))])`
	probe := nameScopeNode(t, spell)
	probe.Fields["Rules"].Args[0].Args[0].Args[0].Args = []Node{creation}
	factory, err := Compile(probe)
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast accepted")
	}
	a.ManaPool().Add(core.Green, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		if len(game.FilterBattlefield(mage.IsToken)) != 0 {
			t.Fatal("token entered before resolution")
		}
		game.ResolveStack()
		tokens := game.FilterBattlefield(mage.IsToken)
		if len(tokens) != 1 || tokens[0].Name() != "Stangg Twin" || !tokens[0].Card.HasSuperType(core.SuperLegendary) || !tokens[0].HasSubType("Human") || !tokens[0].HasSubType("Warrior") || tokens[0].CurrentPower(game) != 3 || tokens[0].CurrentToughness(game) != 4 {
			t.Fatal("real RON token characteristics")
		}
	}
	if _, err := Compile(record); err != nil {
		t.Fatalf("complete Stangg rejected: %v", err)
	}
}

func TestRONLegendaryTokenInvalid(t *testing.T) {
	for _, supers := range []string{`[Legendary()]`, `[Legendary,Legendary]`, `[Snow]`, `[World]`, `[Basic]`, `[Unknown]`, `Legendary`, `[Legendary, Snow]`} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([CreateTokens([TokenWithPT(PT(1,1),Colorless,`+supers+`,[Creature],SubtypesList([Human]),[])],[])]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatal("accepted", supers)
		}
	}
}
