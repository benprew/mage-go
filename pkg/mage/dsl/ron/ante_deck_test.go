package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestRONAnteDeckDeclaration(t *testing.T) {
	for _, kind := range []string{"Artifact", "Creature", "Sorcery", "Instant"} {
		t.Run(kind, func(t *testing.T) {
			extra, body := "", ""
			if kind == "Creature" {
				extra = ",CardPT:(Power:1,Toughness:1)"
			}
			if kind == "Sorcery" || kind == "Instant" {
				body = ",SpellActions(ActionList([DrawACard]))"
			}
			nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Restricted",Typeline:(Supertypes:[],Cardtypes:[%s],Subtypes:[]),Rules:[DeckConstruction(RemoveFromDeckIfNotPlayingForAnte)%s]%s)`, kind, body, extra)), "ante.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			first, second := factory(), factory()
			if !first.(*mage.BaseCard).RequiresAnte() || !second.(*mage.BaseCard).RequiresAnte() || first.ID() == second.ID() {
				t.Fatal("missing or shared restriction")
			}
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			first.SetOwner(a.PlayerID())
			a.SetLibrary([]mage.Card{first})
			mage.NewGame(a, b)
			if len(a.Library()) != 0 {
				t.Fatal("RON declaration did not control game setup")
			}
			rebuilt, err := mage.RebuildCardDefinition(first)
			if err != nil || !rebuilt.(*mage.BaseCard).RequiresAnte() {
				t.Fatalf("rebuilt restriction: %v", err)
			}
		})
	}
}

func TestRONAnteDeckInvalid(t *testing.T) {
	for _, rule := range []string{
		`DeckConstruction`, `DeckConstruction()`, `DeckConstruction(RemoveFromDeckIfNotPlayingForAnte())`,
		`DeckConstruction(Unknown)`, `DeckConstruction([RemoveFromDeckIfNotPlayingForAnte])`,
		`DeckConstruction(RemoveFromDeckIfNotPlayingForAnte,RemoveFromDeckIfNotPlayingForAnte)`,
		`DeckConstruction(RemoveFromDeckIfNotPlayingForAnte),DeckConstruction(RemoveFromDeckIfNotPlayingForAnte)`,
		`Activated(PayMana([]),ActionList([DeckConstruction(RemoveFromDeckIfNotPlayingForAnte)]))`,
		`AsPermanentEnters(ThisPermanent,[DeckConstruction(RemoveFromDeckIfNotPlayingForAnte)])`,
		`Activated(DeckConstruction(RemoveFromDeckIfNotPlayingForAnte),ActionList([]))`,
		`If(PermanentPassesFilter(ThisPermanent,IsTapped),[DeckConstruction(RemoveFromDeckIfNotPlayingForAnte)])`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONAnteDeckCollectionDeclarations(t *testing.T) {
	expected := map[string]bool{"Contract from Below": true, "Darkpact": true, "Demonic Attorney": true, "Rebirth": true, "Bronze Tablet": true, "Jeweled Bird": true, "Tempest Efreet": true}
	seen := make(map[string]bool)
	paths, err := os.ReadDir("../../../../cards/mtgish")
	if err != nil {
		t.Fatal(err)
	}
	records := 0
	for _, path := range paths {
		if !strings.HasSuffix(path.Name(), ".ron") {
			continue
		}
		data, err := os.ReadFile("../../../../cards/mtgish/" + path.Name())
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), path.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			var declaration *Node
			for _, rule := range node.Fields["Rules"].Args {
				if rule.Name == "DeckConstruction" {
					saved := rule
					declaration = &saved
				}
			}
			if declaration == nil {
				continue
			}
			name := node.Fields["Name"].String
			if !expected[name] {
				t.Fatalf("unexpected deck rule on %s", name)
			}
			seen[name] = true
			records++
			_, compileErr := Compile(node)
			complete := name == "Demonic Attorney" || name == "Contract from Below" || name == "Jeweled Bird"
			if complete && compileErr != nil || !complete && (compileErr == nil || strings.Contains(compileErr.Error(), "DeckConstruction")) {
				t.Fatalf("%s: independent behavior support: %v", name, compileErr)
			}
			option, err := compileDeckConstruction(*declaration)
			if err != nil {
				t.Fatal(err)
			}
			card := mage.NewArtifact(name, "", option())
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			a.SetLibrary([]mage.Card{card})
			mage.NewGame(a, b)
			if len(a.Library()) != 0 {
				t.Fatalf("real declaration retained %s", name)
			}
			kept, removed := mage.PrepareCardsForAnte([]mage.Card{card}, true)
			if len(kept) != 1 || len(removed) != 0 {
				t.Fatalf("ante excluded %s", name)
			}
		}
	}
	if len(seen) != len(expected) || records != 22 {
		t.Fatalf("names=%d records=%d", len(seen), records)
	}
	t.Logf("%d unique names / %d real declarations; full record support is checked separately", len(seen), records)
}

func TestRONAnteDeckInventory(t *testing.T) {
	report, err := InspectFS(fstest.MapFS{"ante.ron": {Data: []byte(`Card(Name:"Restricted",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[DeckConstruction(RemoveFromDeckIfNotPlayingForAnte)])`)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range report.Signatures {
		if entry.Name == "RemoveFromDeckIfNotPlayingForAnte" {
			found = true
			if entry.Role != "deck_construction_operation" {
				t.Fatalf("operation role: %s", entry.Role)
			}
		}
	}
	if !found || report.Compiled != 1 {
		t.Fatal("missing compiled deck declaration")
	}
}
