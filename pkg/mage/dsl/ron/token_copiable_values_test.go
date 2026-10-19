package ron

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTokenCopiableValuesGameplay(t *testing.T) {
	for _, name := range []string{"The Hive", "Boris Devilboon", "Master of the Hunt", "Serpent Generator"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			set := "LEG"
			if name == "The Hive" {
				set = "LEB"
			}
			source := g.PutOnBattlefield(repositoryCard(t, set, name), a.PlayerID())
			g.RunStepWithPriority(core.Untap)
			g.SetStep(core.PrecombatMain)
			a.ManaPool().Add(core.Colorless, 6)
			a.ManaPool().Add(core.Green, 2)
			a.ManaPool().Add(core.Black, 1)
			a.ManaPool().Add(core.Red, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			pending := g.Clone()
			g.ResolveStack()
			token := g.FilterBattlefield(mage.IsToken)[0]
			values, err := g.CaptureCopiableValues(token)
			if err != nil {
				t.Fatal(err)
			}
			if values.HasManaCost() {
				t.Fatal("token acquired a mana cost")
			}
			g.MutablePermanent(token.ID()).AddCounter(core.P1P1, 2)
			physical := repositoryCard(t, "LEB", "Air Elemental")
			physical.SetOwner(a.PlayerID())
			copyCard, err := mage.BuildEntryCopy(physical, values)
			if err != nil {
				t.Fatal(err)
			}
			copied := g.PutOnBattlefield(copyCard, b.PlayerID())
			if copied.IsToken || copied.Name() != token.Name() || copied.CurrentPower(g) != 1 || copied.Counters[core.P1P1] != 0 || copied.Card.ManaCost().CMC() != 0 || copied.HasKeyword(core.Flying) != (name == "The Hive") {
				t.Fatal("copied token values or physical token status")
			}
			if name == "Serpent Generator" {
				g.DealDamageToPlayer(a, 1, copied.ID())
				g.PutTriggersOnStack()
				g.ResolveStack()
				if a.PoisonCounters() != 1 || b.PoisonCounters() != 0 {
					t.Fatal("copied token trigger binding")
				}
			}
			if len(token.RuntimeAbilities) > 0 && token.RuntimeAbilities[0].AbilityID() == copied.RuntimeAbilities[0].AbilityID() {
				t.Fatal("copied rules share identity")
			}
			g.PhaseOut(token.ID())
			if _, err := g.CaptureCopiableValues(token); err == nil {
				t.Fatal("phased token capture")
			}
			g.PhaseIn([]uuid.UUID{token.ID()})
			g.DoSacrifice(g.FindPermanent(token.ID()))
			if _, err := g.CaptureCopiableValues(token); err == nil {
				t.Fatal("departed token capture")
			}

			g.BouncePermanentToHand(copied)
			if len(a.Hand()) != 1 || a.Hand()[0].Name() != "Air Elemental" || g.LKI(copied.ID()).Name != token.Name() {
				t.Fatal("physical departure or copy LKI")
			}
			fresh, err := values.Build()
			if err != nil || fresh.IsToken() || fresh.ID() == token.ID() || fresh.Owner() == a.PlayerID() {
				t.Fatal("snapshot retained token identity", err)
			}
			if len(pending.FilterBattlefield(mage.IsToken)) != 0 {
				t.Fatal("live game changed pending clone")
			}
			pending.ResolveStack()
			cloneToken := pending.FilterBattlefield(mage.IsToken)[0]
			if _, err := pending.CaptureCopiableValues(cloneToken); err != nil {
				t.Fatal("clone lost provenance", err)
			}
		})
	}
}

func TestRONTokenCopiableValuesDeclarations(t *testing.T) {
	paths, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	supported := 0
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), path)
		if err != nil {
			t.Fatal(err)
		}
		for _, card := range nodes {
			name := card.Fields["Name"].String
			if seen[name] {
				continue
			}
			var tokens []Node
			var visit func(Node)
			visit = func(n Node) {
				if n.Name == "TokenWithPT" || n.Name == "NamedTokenWithPT" {
					tokens = append(tokens, n)
				}
				for _, child := range n.Args {
					visit(child)
				}
				for _, child := range n.Fields {
					visit(child)
				}
			}
			visit(card)
			if len(tokens) == 0 {
				continue
			}
			seen[name] = true
			for _, node := range tokens {
				factory, err := compileCreatureToken(node)
				if err != nil {
					t.Fatalf("%s: %v", name, err)
				}
				g, a, b := ronGame()
				if err := mage.ApplyEffect(g, mage.CreateTokenCopies(mage.You, mage.ValueAtResolution(mage.Fixed(2)), factory()), a.PlayerID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				created := g.FilterBattlefield(mage.IsToken)
				if len(created) != 2 {
					t.Fatal("token batch size")
				}
				values, err := g.CaptureCopiableValues(created[0])
				if err != nil {
					t.Fatalf("%s: %v", name, err)
				}
				prototype, err := values.Build()
				if err != nil {
					t.Fatal(err)
				}
				p := g.PutOnBattlefield(prototype, b.PlayerID())
				if p.IsToken || values.HasManaCost() || p.Name() != created[0].Name() || p.CurrentPower(g) != created[0].Card.Power() || p.CurrentToughness(g) != created[0].Card.Toughness() || !slices.Equal(p.Colors(), created[0].Colors()) || !slices.Equal(p.Card.Types(), created[0].Card.Types()) || !slices.Equal(p.Card.SubTypes(), created[0].Card.SubTypes()) || !maps.Equal(p.Card.AttrSeeds(), created[0].Card.AttrSeeds()) {
					t.Fatalf("%s: token copy characteristics", name)
				}
				for i, ability := range p.RuntimeAbilities {
					if ability.AbilityID() == created[0].RuntimeAbilities[i].AbilityID() || ability.AbilityID() == created[1].RuntimeAbilities[i].AbilityID() || ability.Source() != p.ID() {
						t.Fatalf("%s: token ability isolation", name)
					}
				}
				supported++
			}
		}
	}
	if len(seen) != 17 || supported != 17 {
		t.Fatalf("token names=%d supported declarations=%d", len(seen), supported)
	}
}
