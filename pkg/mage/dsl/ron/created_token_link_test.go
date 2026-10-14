package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func linkedHiveRegistry(t *testing.T) *mage.Registry {
	t.Helper()
	data, err := os.ReadFile("../../../../cards/mtgish/LEB.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "The Hive"`) {
			record = line
		}
	}
	if record == "" {
		t.Fatal("missing real producer")
	}
	leave := `,TriggerA(WhenAPermanentLeavesTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([Exile([EachPermanent(TheCreatedTokens)],[])]))`
	i := strings.LastIndex(record, "])")
	record = record[:i] + leave + record[i:]
	registry, err := LoadFS(fstest.MapFS{"test.ron": {Data: []byte(record)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func TestRONCreatedTokenLinkGameplay(t *testing.T) {
	registry := linkedHiveRegistry(t)
	g, a, b := ronGame()
	makeSource := func() *mage.Permanent {
		card, err := registry.CreateCard("The Hive")
		if err != nil {
			t.Fatal(err)
		}
		card.SetOwner(a.PlayerID())
		return g.PutOnBattlefield(card, a.PlayerID())
	}
	first, second := makeSource(), makeSource()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, nil); err == nil {
		t.Fatal("unpaid creation accepted")
	}
	if g.StackSize() != 0 || len(g.FilterBattlefield(mage.IsToken)) != 0 {
		t.Fatal("failed payment created tokens")
	}
	for _, source := range []*mage.Permanent{first, second} {
		a.ManaPool().Add(core.Colorless, 5)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	tokens := g.FilterBattlefield(mage.IsToken)
	if len(tokens) != 2 {
		t.Fatal("wrong token count")
	}
	g.BouncePermanentToHand(first)
	a.RemoveFromHand(first.ID())
	returned := g.PutOnBattlefield(first.Card, b.PlayerID())
	a.ManaPool().Add(core.Colorless, 5)
	second.Tapped = false
	if err := g.ActivateAbilityByIndex(a.PlayerID(), second.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		if len(game.FilterBattlefield(mage.IsToken)) != 3 {
			t.Fatal("old source cleanup crossed source or copy links")
		}
		game.BouncePermanentToHand(game.FindPermanent(returned.ID()))
		game.ResolveStack()
		if len(game.FilterBattlefield(mage.IsToken)) != 3 {
			t.Fatal("returned source inherited old link")
		}
		game.BouncePermanentToHand(game.FindPermanent(second.ID()))
		game.ResolveStack()
		if len(game.FilterBattlefield(mage.IsToken)) != 0 {
			t.Fatal("copy or repeated creation was not linked")
		}
	}
}

func TestRONCreatedTokenLinkInvalid(t *testing.T) {
	create := `CreateTokens([` + saprolingRON + `],[])`
	consume := `Exile([EachPermanent(TheCreatedTokens)],[])`
	for _, rules := range []string{
		`Activated(PayMana([]),ActionList([` + consume + `]))`,
		`Activated(PayMana([]),ActionList([` + create + `,` + create + `,` + consume + `]))`,
		`SpellActions(ActionList([` + create + `,` + consume + `]))`,
		`Activated(PayMana([]),ActionList([` + create + `,` + strings.Replace(consume, "TheCreatedTokens", "TheCreatedTokens()", 1) + `]))`,
	} {
		kind := "Artifact"
		if strings.HasPrefix(rules, "SpellActions") {
			kind = "Sorcery"
		}
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[` + kind + `],Subtypes:[]),Rules:[` + rules + `])`
		if registry, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(src)}}, "*.ron"); err == nil || registry != nil {
			t.Fatalf("accepted %s", rules)
		}
	}
	for _, tc := range []struct{ set, name string }{{"DRK", "Dance of Many"}, {"ATQ", "Tetravus"}} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "`+tc.name+`"`) {
				if registry, err := LoadFS(fstest.MapFS{"real.ron": {Data: []byte(line)}}, "*.ron"); err == nil || registry != nil {
					t.Fatalf("published incomplete %s", tc.name)
				}
			}
		}
	}
}

func TestRONCreatedTokenLinkDepartedCreation(t *testing.T) {
	registry := linkedHiveRegistry(t)
	g, a, b := ronGame()
	card, err := registry.CreateCard("The Hive")
	if err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	a.ManaPool().Add(core.Colorless, 5)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.BouncePermanentToHand(source)
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, b.PlayerID())
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.ResolveStack()
		tokens := game.FilterBattlefield(mage.IsToken)
		if len(tokens) != 2 {
			t.Fatal("departure suppressed pending creation")
		}
		owners := map[string]int{}
		for _, token := range tokens {
			owners[token.Card.Owner().String()]++
		}
		if owners[a.PlayerID().String()] != 1 || owners[b.PlayerID().String()] != 1 {
			t.Fatal("copy lost its creating player")
		}
		game.BouncePermanentToHand(game.FindPermanent(returned.ID()))
		game.ResolveStack()
		if len(game.FilterBattlefield(mage.IsToken)) != 2 {
			t.Fatal("pending creation linked to returned source")
		}
	}
}
