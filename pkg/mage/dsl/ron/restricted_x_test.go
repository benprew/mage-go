package ron

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONRestrictedXGameplay(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := repositoryRecord(t, set, "Drain Life", set+".ron")
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			for _, enough := range []bool{false, true} {
				g, a, b := ronGame()
				c := factory()
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
				a.ManaPool().Add(core.Black, 1)
				a.ManaPool().Add(core.Red, 3)
				if enough {
					a.ManaPool().Clear()
					a.ManaPool().Add(core.Black, 3)
					a.ManaPool().Add(core.Red, 1)
				}
				if got := g.MaxXValue(a.PlayerID(), c.ManaCost(), mage.SpellContextForCard(c)); (got == 2) != enough {
					t.Fatalf("X bound: %d", got)
				}
				err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{b.PlayerID()}, 2)
				if (err == nil) != enough {
					t.Fatalf("cast: %v", err)
				}
				if !enough {
					if a.ManaPool().TotalMana() != 4 || len(a.Hand()) != 1 || g.StackSize() != 0 {
						t.Fatal("failed cast changed resources")
					}
					continue
				}
				g.ResolveStack()
				if b.Life() != 18 || a.Life() != 22 {
					t.Fatal("damage or life gain")
				}
			}
		})
	}
}

func TestRONRestrictedXInvalid(t *testing.T) {
	for _, rule := range []string{
		`CastEffect(SpendOnlyColorManaOnX)`, `CastEffect(SpendOnlyColorManaOnX())`,
		`CastEffect(SpendOnlyColorManaOnX(Black,Red))`, `CastEffect(SpendOnlyColorManaOnX(Black()))`,
		`CastEffect(SpendOnlyColorManaOnX(Colorless))`, `CastEffect(SpendOnlyColorManaOnX(AnyColor))`,
		`CastEffect(SpendOnlyColorManaOnX(Black)),CastEffect(SpendOnlyColorManaOnX(Black))`,
		`SpellActions(ActionList([SpendOnlyColorManaOnX(Black)]))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),ManaCost:[ManaCostX],Rules:[` + rule + `,SpellActions(ActionList([GainLife(Integer(1))]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	for _, src := range []string{
		`Card(Name:"No X",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[CastEffect(SpendOnlyColorManaOnX(Black)),SpellActions(ActionList([GainLife(Integer(1))]))])`,
		`Card(Name:"Permanent",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),ManaCost:[ManaCostX],Rules:[CastEffect(SpendOnlyColorManaOnX(Black))])`,
		`Card(Name:"Modal",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[ManaCostX],Rules:[CastEffect(SpendOnlyColorManaOnX(Black)),SpellActions(Modal_ChooseOne([ActionList([GainLife(Integer(1))])]))])`,
	} {
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", src)
		}
	}
}

func TestRONRestrictedXColors(t *testing.T) {
	for _, color := range []string{"White", "Blue", "Black", "Red", "Green"} {
		src := `Card(Name:"Colored X",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[ManaCostX],Rules:[CastEffect(SpendOnlyColorManaOnX(` + color + `)),SpellActions(ActionList([GainLife(ValueX)]))])`
		nodes, err := Parse(strings.NewReader(src), "colors.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		g, a, _ := ronGame()
		c := factory()
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		mana := map[string]core.Color{"White": core.White, "Blue": core.Blue, "Black": core.Black, "Red": core.Red, "Green": core.Green}[color]
		a.ManaPool().Add(mana, 2)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 2); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if a.Life() != 22 {
			t.Fatal("color payment")
		}
	}
}

func TestRONRestrictedXStrictLoadAndInventory(t *testing.T) {
	valid := `Card(Name:"Restricted X",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),ManaCost:[ManaCostX],Rules:[CastEffect(SpendOnlyColorManaOnX(Black)),SpellActions(ActionList([GainLife(ValueX)]))])`
	invalid := strings.Replace(valid, "(Black)", "(Colorless)", 1)
	fs := fstest.MapFS{"a.ron": {Data: []byte(valid)}, "b.ron": {Data: []byte(invalid)}}
	registry := mage.NewRegistry()
	registry.Register("Retained", func() mage.Card { return mage.NewLand("Retained") })
	err := LoadInto(registry, fs, "*.ron")
	var rejected *LoadError
	if !errors.As(err, &rejected) || len(rejected.Issues) != 1 || !registry.CardRegistered("Retained") || registry.CardRegistered("Restricted X") {
		t.Fatalf("atomic load: %v", err)
	}
	delete(fs, "b.ron")
	if err := LoadInto(registry, fs, "*.ron"); err != nil {
		t.Fatal(err)
	}
	report, err := InspectFS(fs, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "Black" && signature.Role == "color" {
			found = true
		}
	}
	if !found {
		t.Fatal("X spending color role missing")
	}
}
