package ron

import (
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const dynamicManaColorsRule = `Activated(TapPermanent(ThisPermanent),ActionList([AddMana(AnyManaColorAPermanentCouldProduce(And([IsCardtype(Land),ControlledByAPlayer(Opponent)])))]))`

func TestRONDynamicManaColorsFellwarStone(t *testing.T) {
	for _, scenario := range []string{"empty", "blue", "green", "wrong-color", "two-colors", "phased-land", "phased-source", "removed-land-ability"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Fellwar Stone")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			stone := g.FindPermanent(card.ID())
			if stone == nil || !mage.IsActivatedManaAbility(stone.RuntimeAbilities[0]) {
				t.Fatal("complete card cast or mana classification")
			}
			if scenario == "empty" {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), stone.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if !stone.Tapped || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 {
					t.Fatal("empty activation")
				}
				return
			}
			land := g.PutOnBattlefield(mage.NewLand("Opponent mixed", mage.WithMultiManaAbility(mage.ManaProduction{Color: core.Blue, Amount: 1}, mage.ManaProduction{Color: core.Green, Amount: 1})), b.PlayerID())
			land.Tapped = true
			if scenario == "phased-land" {
				g.PhaseOut(land.ID())
			}
			if scenario == "phased-source" {
				g.PhaseOut(stone.ID())
			}
			if scenario == "removed-land-ability" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(land.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			}
			cost := "{U}"
			if scenario == "green" {
				cost = "{G}"
			}
			if scenario == "wrong-color" {
				cost = "{W}"
			}
			if scenario == "two-colors" {
				cost = "{U}{G}"
			}
			spell := mage.NewArtifact("Payment fixture", cost)
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			clone := g.Clone()
			err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0)
			valid := scenario == "blue" || scenario == "green"
			if (err == nil) != valid {
				t.Fatalf("cast error = %v", err)
			}
			if !valid && (g.FindPermanentIncludingPhased(stone.ID()).Tapped || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0) {
				t.Fatal("failed payment changed resources")
			}
			if valid && (!g.FindPermanent(stone.ID()).Tapped || g.StackSize() != 1 || a.ManaPool().TotalMana() != 0) {
				t.Fatal("automatic payment")
			}
			if clone.FindPermanentIncludingPhased(stone.ID()).Tapped || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
				t.Fatal("clone changed")
			}
		})
	}
}

func TestRONDynamicManaColorsInvalid(t *testing.T) {
	for _, rule := range []string{
		strings.Replace(dynamicManaColorsRule, "AnyManaColorAPermanentCouldProduce(And([IsCardtype(Land),ControlledByAPlayer(Opponent)]))", "AnyManaColorAPermanentCouldProduce", 1),
		strings.Replace(dynamicManaColorsRule, "AnyManaColorAPermanentCouldProduce(And([IsCardtype(Land),ControlledByAPlayer(Opponent)]))", "AnyManaColorAPermanentCouldProduce()", 1),
		strings.Replace(dynamicManaColorsRule, "AnyManaColorAPermanentCouldProduce(And([IsCardtype(Land),ControlledByAPlayer(Opponent)]))", "AnyManaColorAPermanentCouldProduce(And([]))", 1),
		strings.Replace(dynamicManaColorsRule, "ControlledByAPlayer(Opponent)", "ControlledByAPlayer(Opponent,You)", 1),
		strings.Replace(dynamicManaColorsRule, "ControlledByAPlayer(Opponent)", "IsCardtype(Creature)", 1),
		strings.Replace(dynamicManaColorsRule, "IsCardtype(Land)", "ControlledByAPlayer(Opponent)", 1),
		strings.Replace(dynamicManaColorsRule, "IsCardtype(Land)", "And([])", 1),
		strings.Replace(dynamicManaColorsRule, "IsCardtype(Land)", "IsCardtype(UnknownType)", 1),
		strings.Replace(dynamicManaColorsRule, "ThisPermanent", "HostPermanent", 1),
		strings.Replace(dynamicManaColorsRule, "TapPermanent(ThisPermanent)", "PayMana([])", 1),
		strings.Replace(dynamicManaColorsRule, "Opponent", "You", 1),
		strings.Replace(dynamicManaColorsRule, "Opponent", "Ref_TargetPlayer", 1),
		strings.Replace(dynamicManaColorsRule, "IsCardtype(Land)", "IsTapped", 1),
		strings.Replace(dynamicManaColorsRule, "IsCardtype(Land)", "IsCardtype(Land,Creature)", 1),
		strings.Replace(dynamicManaColorsRule, "AddMana(", "AddManaRepeated(", 1),
		`SpellActions(ActionList([AddMana(AnyManaColorAPermanentCouldProduce(And([IsCardtype(Land),ControlledByAPlayer(Opponent)])))]))`,
		strings.Replace(dynamicManaColorsRule, "Activated(", "ActivatedWithModifiers(", 1),
		strings.Replace(dynamicManaColorsRule, "ActionList([AddMana", "ActionList([DrawACard,AddMana", 1),
	} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rule + `])`
		registry := mage.NewRegistry()
		registry.Register("Retained", func() mage.Card { return mage.NewLand("Retained") })
		if err := LoadInto(registry, fstest.MapFS{"invalid.ron": &fstest.MapFile{Data: []byte(source)}}, "*.ron"); err == nil || !registry.CardRegistered("Retained") {
			t.Fatalf("invalid declaration published: %s", rule)
		}
	}
}

type ronDynamicManaPlayer struct {
	*mage.BasePlayer
	choice string
	offers [][]string
}

func (p *ronDynamicManaPlayer) ChooseString(options []string, _ string) string {
	p.offers = append(p.offers, slices.Clone(options))
	return p.choice
}

func TestRONDynamicManaColorsChoiceAndRegistry(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Fellwar Stone"`) {
			record = line
		}
	}
	files := fstest.MapFS{"stone.ron": {Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, choice := range []string{"Green", "Blue", "White", ""} {
		t.Run(choice, func(t *testing.T) {
			a := &ronDynamicManaPlayer{BasePlayer: mage.NewBasePlayer("A"), choice: choice}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			first, err := registry.CreateCard("Fellwar Stone")
			if err != nil {
				t.Fatal(err)
			}
			second, err := registry.CreateCard("Fellwar Stone")
			if err != nil {
				t.Fatal(err)
			}
			if first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
				t.Fatal("factory shared ability identity")
			}
			stone := g.PutOnBattlefield(first, a.PlayerID())
			g.PutOnBattlefield(mage.NewLand("Mixed", mage.WithMultiManaAbility(mage.ManaProduction{Color: core.Blue, Amount: 3}, mage.ManaProduction{Color: core.Green, Amount: 2})), b.PlayerID())
			for range 3 {
				g.CanAfford(a.PlayerID(), core.ManaCost{Green: 1}, nil)
			}
			if len(a.offers) != 0 {
				t.Fatal("query made choices")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), stone.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			want := core.Blue
			if choice == "Green" {
				want = core.Green
			}
			if a.ManaPool().TotalMana() != 1 || a.ManaPool().Count(want) != 1 || len(a.offers) != 1 || !slices.Equal(a.offers[0], []string{"Blue", "Green"}) || g.StackSize() != 0 {
				t.Fatalf("choice result: offers=%v pool=%v", a.offers, a.ManaPool())
			}
		})
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory: %v", err)
	}
	for name, role := range map[string]string{"AnyManaColorAPermanentCouldProduce": "mana_production", "And": "permanent_filter", "ControlledByAPlayer": "permanent_filter", "Opponent": "player_filter"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing semantic role: %s/%s", name, role)
		}
	}
}

func TestRONDynamicManaColorsOpponentRecords(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
	}{
		{"ARN", "City of Brass", core.Green},
		{"FEM", "Dwarven Ruins", core.Red}, {"FEM", "Bottomless Vault", core.Black},
		{"FEM", "Sand Silos", core.Blue}, {"FEM", "Dwarven Hold", core.Red}, {"FEM", "Hollow Trees", core.Green}, {"FEM", "Icatian Store", core.White}, {"ATQ", "Mishra's Workshop", core.Colorless},
		{"ATQ", "Urza's Tower", core.Colorless},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			stone := g.PutOnBattlefield(repositoryCard(t, "DRK", "Fellwar Stone"), a.PlayerID())
			land := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), b.PlayerID())
			land.Tapped = true
			if err := mage.ApplyEffect(g, mage.ReplaceTappedManaTypeUntilLeaves(mage.This, tc.color, core.White), land.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if !g.CanAfford(a.PlayerID(), core.ManaCost{White: 1}, nil) {
				t.Fatal("replacement-aware complete-record query")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), stone.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().Count(core.White) != 1 || a.ManaPool().TotalMana() != 1 || b.ManaPool().TotalMana() != 0 || !land.Tapped || g.StackSize() != 0 {
				t.Fatal("query paid a land cost, copied a restriction, or ran a rider")
			}
		})
	}
}

func TestRONDynamicManaColorsCopyArtifact(t *testing.T) {
	g, a, b := ronGame()
	chooser := &ronEntryCopyPlayer{Player: a, accept: true}
	g.SetPlayerAt(0, chooser)
	stone := g.PutOnBattlefield(repositoryCard(t, "DRK", "Fellwar Stone"), b.PlayerID())
	g.PutOnBattlefield(mage.NewLand("Own blue", mage.WithManaAbility(core.Blue)), a.PlayerID())
	g.PutOnBattlefield(mage.NewLand("Opponent green", mage.WithManaAbility(core.Green)), b.PlayerID())
	card := repositoryCard(t, "LEB", "Copy Artifact")
	copied := g.PutOnBattlefield(card, a.PlayerID())
	if copied.Name() != "Fellwar Stone" || !copied.HasType(core.TypeEnchantment) || chooser.choices != 1 {
		t.Fatal("copy entry")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), copied.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), stone.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(core.Green) != 1 || b.ManaPool().Count(core.Blue) != 1 || g.StackSize() != 0 {
		t.Fatal("copy retained donor controller or state")
	}
}
