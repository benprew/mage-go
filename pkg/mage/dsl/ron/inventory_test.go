package ron

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestInventoryWalksRejectedDescendantsAndSeparatesRoles(t *testing.T) {
	src := `Card(Name: "Nested", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(TapPermanent(ThisPermanent), ActionList([TapPermanent(ThisPermanent), UnknownParent(HiddenChild(Ref_Missing))]))])`
	files := fstest.MapFS{"a.ron": {Data: []byte(src)}, "b.ron": {Data: []byte(src)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Records != 2 || report.Compiled != 0 || report.UniqueCards != 1 || len(report.Issues) != 2 {
		t.Fatalf("counts: %+v", report)
	}
	roles := map[string]bool{}
	hidden := false
	for _, entry := range report.Signatures {
		if entry.Name == "TapPermanent" {
			roles[entry.Role] = true
		}
		if entry.Name == "HiddenChild" {
			hidden = true
			if entry.Role != "unknown" || len(entry.Cards) != 1 || entry.Occurrences != 2 || len(entry.Examples) != 2 {
				t.Fatalf("hidden signature: %+v", entry)
			}
			example := entry.Examples[0]
			if example.Path != "Card.Rules[0].args[1].args[0][1].args[0]" || example.Source.Line != 1 || example.Source.Column == 0 || example.Card != "Nested" {
				t.Fatalf("location: %+v", example)
			}
		}
	}
	if !hidden || !roles["cost"] || !roles["action"] {
		t.Fatalf("missing nested roles: %v, hidden %v", roles, hidden)
	}
	first, _ := json.Marshal(report)
	again, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	second, _ := json.Marshal(again)
	if string(first) != string(second) {
		t.Fatal("inventory is not deterministic")
	}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("inspection weakened strict loading")
	}
}

func TestInventoryKeepsCompilationSeparateFromSupport(t *testing.T) {
	files := fstest.MapFS{"a.ron": {Data: []byte(testBolt)}, "b.ron": {Data: []byte(strings.Replace(testBolt, "Integer(3)", "Integer(4)", 1))}}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 || len(report.Issues) != 1 || !strings.Contains(report.Issues[0].Message, "conflicting definition") {
		t.Fatalf("reprint conflict: %+v", report)
	}
	for _, entry := range report.Signatures {
		if entry.Support != "unverified" {
			t.Fatalf("syntax implies support: %+v", entry)
		}
	}
}

func TestInventoryReportsMalformedAndEmptyFiles(t *testing.T) {
	files := fstest.MapFS{"empty.ron": {Data: []byte("// empty")}, "malformed.ron": {Data: []byte("Card(")}, "valid.ron": {Data: []byte(testBolt)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Records != 1 || report.Compiled != 1 || len(report.Issues) != 2 {
		t.Fatalf("file failures: %+v", report)
	}
	if _, err := InspectFS(files, "["); err == nil {
		t.Fatal("invalid pattern accepted")
	}
	if _, err := InspectFS(files, "*.missing"); err == nil {
		t.Fatal("missing collection accepted")
	}
}

func TestInventoryRepositoryCollection(t *testing.T) {
	report, err := InspectFS(os.DirFS("../../../../cards/mtgish"), "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Records != 2445 || report.UniqueCards != 986 || report.Compiled != 2142 || len(report.Issues) != 303 {
		t.Fatalf("collection counts: %d/%d/%d/%d", report.Records, report.UniqueCards, report.Compiled, len(report.Issues))
	}
	for _, issue := range report.Issues {
		if issue.Card == "Cuombajj Witches" {
			t.Fatalf("supported union-target card rejected: %v", issue)
		}
		if issue.Card == "Island Sanctuary" {
			t.Fatalf("supported player protection card rejected: %v", issue)
		}
		if issue.Card == "Serendib Djinn" {
			t.Fatalf("supported no-land state card rejected: %v", issue)
		}
		if issue.Card == "Chains of Mephistopheles" {
			t.Fatalf("supported draw replacement card rejected: %v", issue)
		}
		if issue.Card == "Mind Bomb" {
			t.Fatalf("supported per-player discard card rejected: %v", issue)
		}
		if issue.Card == "Golgothian Sylex" || issue.Card == "Voodoo Doll" || issue.Card == "Gauntlet of Might" || issue.Card == "High Tide" || issue.Card == "Artifact Ward" || issue.Card == "Backfire" {
			t.Fatalf("supported card rejected: %v", issue)
		}
		if issue.Card == "Master of the Hunt" {
			t.Fatalf("supported token quality rejected: %v", issue)
		}
		if issue.Card == "Glyph of Destruction" {
			t.Fatalf("supported combat duration rejected: %v", issue)
		}
		if issue.Card == "Primordial Ooze" {
			t.Fatalf("supported resolution snapshot rejected: %v", issue)
		}
		if issue.Card == "Animate Artifact" {
			t.Fatalf("supported conditional animation rejected: %v", issue)
		}
		if issue.Card == "Shimian Night Stalker" {
			t.Fatalf("supported temporary redirection card rejected: %v", issue)
		}
		if issue.Card == "Angry Mob" || issue.Card == "Lurker" || issue.Card == "Martyrs of Korlis" || issue.Card == "Veteran Bodyguard" {
			t.Fatalf("supported static branch card rejected: %v", issue)
		}
		if issue.Card == "Phantasmal Terrain" || issue.Card == "Giant Slug" {
			t.Fatalf("supported land choice card rejected: %v", issue)
		}
		if issue.Card == "Delif's Cube" {
			t.Fatalf("supported combat assignment card rejected: %v", issue)
		}
		if issue.Card == "Festival" {
			t.Fatalf("supported collection attack card rejected: %v", issue)
		}
		if issue.Card == "Orc General" || issue.Card == "Arcades Sabboth" || issue.Card == "Blood Lust" || issue.Card == "Erg Raiders" {
			t.Fatalf("supported card rejected: %v", issue)
		}
		if issue.Card == "Stasis" || issue.Card == "Winter Orb" {
			t.Fatalf("supported static untap card rejected: %v", issue)
		}
		if issue.Card == "Blaze of Glory" || issue.Card == "Hurr Jackal" {
			t.Fatalf("supported temporary rule card rejected: %v", issue)
		}
		if issue.Card == "Spiritual Sanctuary" || issue.Card == "Osai Vultures" {
			t.Fatalf("supported upkeep card rejected: %v", issue)
		}
		if issue.Card == "Hellfire" || issue.Card == "Volcanic Eruption" {
			t.Fatalf("supported destruction count card rejected: %v", issue)
		}
		if issue.Card == "Thelonite Druid" || issue.Card == "Xenic Poltergeist" {
			t.Fatalf("supported animation card rejected: %v", issue)
		}
		if issue.Card == "Lure" || issue.Card == "Marble Priest" || issue.Card == "Berserk" || issue.Card == "Fire and Brimstone" {
			t.Fatalf("supported card rejected: %v", issue)
		}
		if issue.Card == "Library of Alexandria" || issue.Card == "Rocket Launcher" {
			t.Fatalf("supported activation restriction card rejected: %v", issue)
		}
		if issue.Card == "War Barge" || issue.Card == "Sandals of Abdallah" {
			t.Fatalf("supported delayed departure card rejected: %v", issue)
		}
		if issue.Card == "Brainwash" || issue.Card == "Leviathan" || issue.Card == "Zombie Master" || issue.Card == "Life Matrix" || issue.Card == "Living Artifact" || issue.Card == "Spore Cloud" || issue.Card == "Telekinesis" || issue.Card == "Armageddon Clock" || issue.Card == "Spitting Slug" || issue.Card == "Rust" || issue.Card == "Ayesha Tanaka" || issue.Card == "Goblin Shrine" || issue.Card == "Hazezon Tamar" {
			t.Fatalf("supported card rejected: %v", issue)
		}
	}
	nested := false
	for _, entry := range report.Signatures {
		if entry.Name == "Ref_TargetPermanent" && entry.Role == "permanent_reference" {
			nested = true
		}
	}
	if !nested {
		t.Fatal("real RON target references missing")
	}
}

func TestInventoryLeavesRealRONGameplayAndRegistryUnchanged(t *testing.T) {
	old := mage.DefaultRegistry
	bolt := repositoryCard(t, "LEB", "Lightning Bolt")
	report, err := InspectFS(os.DirFS("../../../../cards/mtgish"), "*.ron")
	if err != nil || report.Compiled != 2142 {
		t.Fatalf("inspection: %v", err)
	}
	if mage.DefaultRegistry != old {
		t.Fatal("inspection changed the production registry")
	}
	g, player, opponent := ronGame()
	bolt.SetOwner(player.PlayerID())
	player.AddToHand(bolt)
	player.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(player.PlayerID(), bolt.ID(), []uuid.UUID{opponent.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if opponent.Life() != 17 || len(player.Graveyard()) != 1 {
		t.Fatal("inspection changed RON spell execution")
	}
}

func TestInventoryConflictKeepsFirstReprintPosition(t *testing.T) {
	files := fstest.MapFS{
		"a.ron": {Data: []byte(testBolt)},
		"b.ron": {Data: []byte(testBolt)},
		"c.ron": {Data: []byte(strings.Replace(testBolt, "Integer(3)", "Integer(4)", 1))},
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 2 || len(report.Issues) != 1 || !strings.Contains(report.Issues[0].Message, "a.ron:1:1") {
		t.Fatalf("first definition: %+v", report.Issues)
	}
}

func TestInventoryInvalidFieldsHaveStableDiagnostics(t *testing.T) {
	files := fstest.MapFS{"a.ron": {Data: []byte(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), UnknownZ: 1, UnknownA: 2)`)}}
	for range 100 {
		report, err := InspectFS(files, "*.ron")
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Issues) != 1 || !strings.Contains(report.Issues[0].Message, "unknown field UnknownA") {
			t.Fatalf("unstable field diagnostic: %+v", report.Issues)
		}
	}
}

func TestInventoryCombatPair(t *testing.T) {
	src := `Card(Name:"Nested Pair",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[TriggerA(WhenACreatureBlocksACreature(SinglePermanent(ThisPermanent),And([IsCardtype(Creature),IsNonCreatureType(Wall)])),ActionList([UnknownAction(Trigger_ThatOtherCreature)]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range report.Signatures {
		if entry.Name == "SinglePermanent" {
			found = true
			if entry.Role != "permanent_filter" {
				t.Fatalf("subject role=%s", entry.Role)
			}
		}
		if entry.Name == "IsNonCreatureType" && entry.Role != "permanent_filter" {
			t.Fatalf("nested filter role=%s", entry.Role)
		}
	}
	if !found || report.Compiled != 0 || len(report.Issues) != 1 {
		t.Fatal("inventory lost rejected descendants")
	}
}
