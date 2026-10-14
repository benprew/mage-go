package ron

import (
	"errors"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONRestrictedFixedManaWorkshop(t *testing.T) {
	for _, scenario := range []string{"manual", "artifact", "artifact-creature", "creature", "ability", "insufficient", "phased"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, _ := ronGame()
			land := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Mishra's Workshop"), a.PlayerID())
			if scenario == "manual" {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if g.StackSize() != 0 || a.ManaPool().Count(core.Colorless) != 3 || a.ManaPool().CanPay(core.ManaCost{Generic: 1}, nil) {
					t.Fatal("manual mana production incorrect")
				}
				return
			}
			if scenario == "phased" {
				g.PhaseOut(land.ID())
			}
			if scenario == "ability" {
				card := mage.NewArtifact("Activation", "", mage.WithAbility(mage.Activate(mage.Costs(mage.ManaCostOf("{1}")), mage.GainLifeFor(mage.You, mage.Fixed(1)))))
				p := g.PutOnBattlefield(card, a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || g.FindPermanent(land.ID()).Tapped || a.Life() != 20 {
					t.Fatal("restricted mana paid an artifact activation")
				}
				return
			}
			cost := "{3}"
			if scenario == "insufficient" {
				cost = "{4}"
			}
			spell := mage.NewArtifact("Spell", cost)
			if scenario == "creature" {
				spell = mage.NewCreature("Spell", cost, 1, 1)
			}
			if scenario == "artifact-creature" {
				spell.AddType(core.TypeCreature)
			}
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			clone := g.Clone()
			err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0)
			valid := scenario == "artifact" || scenario == "artifact-creature"
			if (err == nil) != valid {
				t.Fatalf("cast error = %v", err)
			}
			if clone.FindPermanentIncludingPhased(land.ID()).Tapped || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
				t.Fatal("original payment changed clone")
			}
			if !valid {
				if g.FindPermanentIncludingPhased(land.ID()).Tapped || a.ManaPool().TotalMana() != 0 || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("failed cast changed resources")
				}
			} else if !g.FindPermanent(land.ID()).Tapped || a.ManaPool().TotalMana() != 0 || g.StackSize() != 1 {
				t.Fatal("automatic payment incorrect")
			}
		})
	}
}

func TestRONRestrictedFixedManaInvalid(t *testing.T) {
	for _, body := range []string{
		`AddManaWithModifiers(AnyManaColor,CanOnlySpendOnSpells(IsCardtype(Artifact)))`,
		`AddManaWithModifiers(And([]),CanOnlySpendOnSpells(IsCardtype(Artifact)))`,
		`AddManaWithModifiers(And([ManaProduceC,And([])]),CanOnlySpendOnSpells(IsCardtype(Artifact)))`,
		`AddManaWithModifiers(And([ManaProduceC,AnyManaColor]),CanOnlySpendOnSpells(IsCardtype(Artifact)))`,
		`AddManaWithModifiers(ManaProduceC,CanOnlySpendOnSpells(IsCardtype(Land)))`,
		`AddManaWithModifiers(ManaProduceC,CanOnlySpendOnSpells(And([IsCardtype(Artifact),IsCardtype(Creature)])))`,
		`AddManaWithModifiers(ManaProduceC,CanOnlySpendOnSpells())`,
		`AddManaWithModifiers(ManaProduceC,CanOnlySpendOnSpells(IsCardtype(Artifact)),Integer(1))`,
	} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Land],Subtypes:[]),Rules:[Activated(TapPermanent(ThisPermanent),ActionList([` + body + `]))])`
		fs := fstest.MapFS{"invalid.ron": &fstest.MapFile{Data: []byte(source)}}
		registry := mage.NewRegistry()
		registry.Register("Retained", func() mage.Card { return mage.NewLand("Retained") })
		err := LoadInto(registry, fs, "*.ron")
		var rejected *LoadError
		if !errors.As(err, &rejected) || rejected.Records != 1 || len(rejected.Issues) != 1 || !registry.CardRegistered("Retained") {
			t.Fatalf("accepted %s", body)
		}
	}
	for _, rule := range []string{`AsPermanentEnters([AddManaWithModifiers(ManaProduceC,CanOnlySpendOnSpells(IsCardtype(Artifact)))])`, `PermanentLayerEffect(ThisPermanent,[AddManaWithModifiers(ManaProduceC,CanOnlySpendOnSpells(IsCardtype(Artifact)))])`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Land],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid declaration context")
		}
	}
}

func TestRONRestrictedFixedManaRegistry(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/ATQ.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Mishra's Workshop"`) {
			record = line
		}
	}
	files := fstest.MapFS{"workshop.ron": {Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	for _, player := range []*mage.BasePlayer{a, b} {
		card, err := registry.CreateCard("Mishra's Workshop")
		if err != nil {
			t.Fatal(err)
		}
		card.SetOwner(player.PlayerID())
		land := g.PutOnBattlefield(card, player.PlayerID())
		if err := g.ActivateAbilityByIndex(player.PlayerID(), land.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.DestroyPermanent(g.FindPermanent(land.ID()))
		pool := player.ManaPool()
		if pool.TotalMana() != 3 || !pool.CanPay(core.ManaCost{Generic: 3}, &core.SpellPaymentContext{IsArtifact: true}) || pool.CanPay(core.ManaCost{Generic: 3}, nil) {
			t.Fatal("source departure lost restricted pool")
		}
	}
	if a.ManaPool() == b.ManaPool() {
		t.Fatal("factory instances share pools")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory failed: %v", err)
	}
	for name, role := range map[string]string{"CanOnlySpendOnSpells": "mana_spending_restriction", "And": "mana_production", "IsCardtype": "spell_filter"} {
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
}

func TestRONRestrictedFixedManaSpellCopies(t *testing.T) {
	for _, restriction := range []string{"Artifact", "Creature"} {
		for _, production := range []string{`AddManaWithModifiers(And([ManaProduceU,And([ManaProduceG,ManaProduceC])]),CanOnlySpendOnSpells(IsCardtype(` + restriction + `)))`, `AddManaRepeatedWithModifiers(AnyManaColor,Integer(3),CanOnlySpendOnSpells(IsCardtype(` + restriction + `)))`} {
			source := `Card(Name:"Restricted spell",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + production + `]))])`
			registry, err := LoadFS(fstest.MapFS{"spell.ron": {Data: []byte(source)}}, "*.ron")
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			spell, err := registry.CreateCard("Restricted spell")
			if err != nil {
				t.Fatal(err)
			}
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			g.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
				t.Fatal("resolution changed clone")
			}
			clone.ResolveStack()
			context := &core.SpellPaymentContext{IsArtifact: restriction == "Artifact", IsCreature: restriction == "Creature"}
			for _, game := range []*mage.Game{g, clone} {
				for _, player := range []mage.Player{game.GetPlayer(a.PlayerID()), game.GetPlayer(b.PlayerID())} {
					pool := player.ManaPool()
					if pool.TotalMana() != 3 || !pool.CanPay(core.ManaCost{Generic: 3}, context) || pool.CanPay(core.ManaCost{Generic: 1}, nil) || pool.CanPay(core.ManaCost{Generic: 1}, &core.SpellPaymentContext{}) {
						t.Fatal("copy or clone lost controller or restriction")
					}
					if strings.HasPrefix(production, "AddManaWithModifiers") && (pool.Count(core.Blue) != 1 || pool.Count(core.Green) != 1 || pool.Count(core.Colorless) != 1) {
						t.Fatal("nested fixed production changed colors")
					}
				}
			}
		}
	}
}
