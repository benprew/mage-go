package ron

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const testBolt = `Card(Name: "RON Bolt", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), ManaCost: [ManaCostR], Rules: [SpellActions(Targeted([AnyTarget], ActionList([SpellDealsDamage(ThisSpell, Integer(3), Ref_AnyTarget)])))])`
const testBird = `Card(Name: "RON Bird", Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: [Bird]), ManaCost: [ManaCostG], Rules: [Flying, Activated(TapPermanent(ThisPermanent), ActionList([AddMana(AnyManaColor)]))], CardPT: (Power: 0, Toughness: 1))`

func TestCompilePermanentTargetsHaveSeparateDeclarations(t *testing.T) {
	src := `Card(Name: "RON Regenerate", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([RegeneratePermanent(Ref_TargetPermanent)])))])`
	nodes, err := Parse(strings.NewReader(src), "regenerate.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	a := factory().Abilities()[0].(*mage.ActionDefinition)
	b := factory().Abilities()[0].(*mage.ActionDefinition)
	if a.Targets()[0] == b.Targets()[0] {
		t.Fatal("card instances share a permanent target declaration")
	}
}

func TestLoadFSBuildsIndependentCards(t *testing.T) {
	files := fstest.MapFS{"a.ron": {Data: []byte(testBolt + "\n" + testBird)}, "b.ron": {Data: []byte(testBolt)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("reprints must share one factory")
	}
	a, err := registry.CreateCard("RON Bolt")
	if err != nil {
		t.Fatal(err)
	}
	b, err := registry.CreateCard("RON Bolt")
	if err != nil {
		t.Fatal(err)
	}
	aa, bb := a.Abilities()[0].(*mage.ActionDefinition), b.Abilities()[0].(*mage.ActionDefinition)
	if a.ID() == b.ID() || aa.AbilityID() == bb.AbilityID() || aa.Targets()[0] == bb.Targets()[0] {
		t.Fatal("card instances share mutable state")
	}
	bird, err := registry.CreateCard("RON Bird")
	if err != nil {
		t.Fatal(err)
	}
	if !bird.HasSubType("Bird") || bird.AttrSeeds()[core.Flying] != 1 || bird.Power() != 0 || bird.Toughness() != 1 || bird.ManaCost().Green != 1 {
		t.Fatal("metadata was not loaded")
	}
	productions := mage.ManaProductionsForAbility(bird.Abilities()[0])
	if len(productions) != 1 || productions[0].Color != core.AnyColor {
		t.Fatalf("mana: %+v", productions)
	}
}

func TestLoadFSRejectsWholeCollection(t *testing.T) {
	files := fstest.MapFS{"a.ron": {Data: []byte(testBolt)}, "b.ron": {Data: []byte(`Card(Name: "Unsupported", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [NotAnEngineRule])`)}}
	old := mage.DefaultRegistry
	registry, err := LoadFS(files, "*.ron")
	var loadErr *LoadError
	if registry != nil || !errors.As(err, &loadErr) || len(loadErr.Issues) != 1 || mage.DefaultRegistry != old {
		t.Fatalf("partial registration: %v, %v", registry, err)
	}
	if !strings.Contains(err.Error(), "Unsupported") || !strings.Contains(err.Error(), "NotAnEngineRule") || !strings.Contains(err.Error(), "b.ron:1:") {
		t.Fatalf("missing diagnostic: %v", err)
	}
}

func TestLoadIntoChangesRegistryOnlyAfterSuccessfulCompilation(t *testing.T) {
	registry := mage.NewRegistry()
	registry.Register("Old", func() mage.Card { return mage.NewLand("Old") })
	bad := fstest.MapFS{"bad.ron": {Data: []byte(testBolt + "\nNotACard")}}
	if err := LoadInto(registry, bad, "*.ron"); err == nil || !registry.CardRegistered("Old") || registry.CardRegistered("RON Bolt") {
		t.Fatal("failed load changed the registry")
	}
	good := fstest.MapFS{"good.ron": {Data: []byte(testBolt)}}
	if err := LoadInto(registry, good, "*.ron"); err != nil {
		t.Fatal(err)
	}
	if registry.CardRegistered("Old") || !registry.CardRegistered("RON Bolt") {
		t.Fatal("successful load did not replace the registry")
	}
}

func TestCompileUsesRulesSubtypeNames(t *testing.T) {
	src := `Card(Name: "RON Typed Land", Typeline: (Supertypes: [], Cardtypes: [Land], Subtypes: [Urzas, PowerPlant]))`
	nodes, err := Parse(strings.NewReader(src), "subtypes.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	card := factory()
	if !card.HasSubType("Urza's") || !card.HasSubType("Power-Plant") {
		t.Fatalf("RON enum names leaked into subtypes: %v", card.SubTypes())
	}
}

func TestCompileRejectsInvalidDeclarations(t *testing.T) {
	for _, src := range []string{
		`Card(Name: "A", Typeline: (Supertypes: [], Cardtypes: [UnknownType], Subtypes: []))`,
		`Card(Name: "A", Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: []))`,
		`Card(Name: "A", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [SpellActions(ActionList([DestroyPermanent(Ref_TargetPermanent)]))])`,
		`Card(Name: "A", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [SpellActions(Targeted([TargetPlayer(AnyPlayer)], ActionList([DestroyPermanent(Ref_TargetPermanent)])))])`,
		`Card(Name: "A", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [SpellActions(Targeted([AnyTarget, AnyTarget], ActionList([])))])`,
		`Card(Name: "A", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), ExtraField: 1)`,
		`Card(Name: "A", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), ManaCost: [ManaCostGeneric(-1)])`,
		`Card(Name: "A", Typeline: (Supertypes: [], Cardtypes: [Sorcery], Subtypes: []), Rules: [])`,
	} {
		t.Run(src, func(t *testing.T) {
			nodes, err := Parse(strings.NewReader(src), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid declaration compiled")
			}
		})
	}
}

func TestLoadFSRejectsConflictingReprintsAndEmptyFiles(t *testing.T) {
	for _, files := range []fstest.MapFS{
		{}, {"a.ron": {Data: []byte("// empty")}},
		{"a.ron": {Data: []byte(testBolt)}, "b.ron": {Data: []byte(strings.Replace(testBolt, "Integer(3)", "Integer(4)", 1))}},
	} {
		if registry, err := LoadFS(files, "*.ron"); registry != nil || err == nil {
			t.Fatalf("expected failure: %v, %v", registry, err)
		}
	}
}
