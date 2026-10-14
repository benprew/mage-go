package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPluralSacrificeCostGameplay(t *testing.T) {
	record := repositoryRecord(t, "LEG", "Sword of the Ages", "LEG.lines.ron")
	nodes, err := Parse(strings.NewReader("ActionList([DrawACard])"), "producer-body.ron")
	if err != nil {
		t.Fatal(err)
	}
	rules := record.Fields["Rules"]
	rules.Args[1].Args[1] = nodes[0]
	record.Fields["Rules"] = rules
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, count := range []int{0, 1, 2} {
		a := &ronNumericChooser{BasePlayer: mage.NewBasePlayer("A"), answer: count}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		source := g.PutOnBattlefield(factory(), a.PlayerID())
		g.UntapPermanent(source)
		for range 2 {
			g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 2), a.PlayerID())
		}
		opponent := g.PutOnBattlefield(mage.NewCreature("Opponent", "", 2, 2), b.PlayerID())
		a.SetLibrary([]mage.Card{mage.NewLand("Drawn")})
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if len(a.Graveyard()) != 1+count || g.FindPermanent(opponent.ID()) == nil || len(a.Hand()) != 0 || a.calls != 1 {
			t.Fatal("payment or timing", count)
		}
		g.ResolveStack()
		if len(a.Hand()) != 1 {
			t.Fatal("body did not resolve")
		}
	}
}

func TestRONPluralSacrificeCostInvalid(t *testing.T) {
	for _, cost := range []string{
		`SacrificeAnyNumberOfPermanents()`,
		`SacrificeAnyNumberOfPermanents(IsCardtype(Creature))`,
		`SacrificeAnyNumberOfPermanents(And([]))`,
		`SacrificeAnyNumberOfPermanents(And([ControlledByAPlayer(Opponent),IsCardtype(Creature)]))`,
		`SacrificeAnyNumberOfPermanents(Or([ControlledByAPlayer(SinglePlayer(You)),IsCardtype(Creature)]))`,
		`And([SacrificeAnyNumberOfPermanents(And([ControlledByAPlayer(SinglePlayer(You))])),SacrificeAnyNumberOfPermanents(And([ControlledByAPlayer(SinglePlayer(You))]))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),ManaCost:[],Rules:[Activated(`+cost+`,ActionList([DrawACard]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", cost)
		}
	}
}

func TestRONPluralSacrificeCostSwordBoundary(t *testing.T) {
	_, err := Compile(repositoryRecord(t, "LEG", "Sword of the Ages", "LEG.lines.ron"))
	if err == nil || strings.Contains(err.Error(), "unsupported cost") {
		t.Fatal("payment did not lower or incomplete card accepted", err)
	}
}

func TestRONPluralSacrificeCostFactoryAndAtomicLoad(t *testing.T) {
	const record = `Card(Name:"Plural producer",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),ManaCost:[],Rules:[Activated(SacrificeAnyNumberOfPermanents(And([ControlledByAPlayer(SinglePlayer(You)),IsCardtype(Creature)])),ActionList([DrawACard]))])`
	files := fstest.MapFS{"good.ron": {Data: []byte(record)}}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Plural producer")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Plural producer")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID() == second.ID() || first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
		t.Fatal("shared factory identities")
	}
	for _, card := range []mage.Card{first, second} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "SacrificeAnyNumberOfPermanents" && signature.Role == "cost" && signature.CompiledOccurrences == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("missing compiled cost role")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.ReplaceAll(record, "Plural producer", "Bad producer"), "SinglePlayer(You)", "Opponent"))}
	if LoadInto(registry, files, "*.ron") == nil {
		t.Fatal("invalid collection accepted")
	}
	if _, err := registry.CreateCard("Plural producer"); err != nil {
		t.Fatal("failed load changed registry", err)
	}
}
