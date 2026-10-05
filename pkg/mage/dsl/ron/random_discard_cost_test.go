package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONRandomDiscardCosts(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		mana      int
		tap       bool
	}{{"ATQ", "Coral Helm", 3, false}, {"FEM", "Draconian Cylix", 2, true}} {
		for _, mode := range []string{"pay", "mana", "hand", "illegal", "phased", "departed", "copy", "source-departed", "source-phased"} {
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				g, a, b := ronGame()
				source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
				if err := mage.ValidateCardDefinition(source.Card.Name(), source.Card); err != nil {
					t.Fatal(err)
				}
				target := g.PutOnBattlefield(mage.NewCreature("Target", "", 2, 3), b.PlayerID())
				first, second := mage.NewLand("First"), mage.NewLand("Second")
				a.AddToHand(first)
				a.AddToHand(second)
				if mode == "hand" {
					a.SetHand(nil)
				}
				mana := tc.mana
				if mode == "mana" {
					mana--
				}
				a.ManaPool().Add(core.Colorless, mana)
				g.SetRandomResults([]int{1, 7})
				id := target.ID()
				if mode == "illegal" {
					id = b.PlayerID()
				}
				err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{id})
				if mode == "hand" || mode == "mana" || mode == "illegal" {
					if err == nil {
						t.Fatal("unpayable activation accepted")
					}
					wantHand := 2
					if mode == "hand" {
						wantHand = 0
					}
					if len(a.Hand()) != wantHand || g.StackSize() != 0 || source.Tapped || a.ManaPool().Count(core.Colorless) != mana || len(a.Graveyard()) != 0 || g.RandIntn(10) != 1 {
						t.Fatal("failed activation changed state")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if len(a.Hand()) != 1 || a.Hand()[0].ID() != first.ID() || len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != second.ID() || source.Tapped != tc.tap || a.ManaPool().Count(core.Colorless) != 0 || g.RandIntn(10) != 7 {
					t.Fatal("wrong random activation payment")
				}
				if mode == "phased" {
					g.PhaseOut(target.ID())
				}
				if mode == "departed" {
					g.RemoveFromBattlefield(target)
					target = g.PutOnBattlefield(target.Card, b.PlayerID())
				}
				if mode == "source-departed" {
					g.RemoveFromBattlefield(source)
				}
				if mode == "source-phased" {
					g.PhaseOut(source.ID())
				}
				if mode == "copy" {
					if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				eligible := mode != "phased" && mode != "departed"
				if tc.tap {
					if g.HasRegenerationShield(target.ID()) != eligible {
						t.Fatal("wrong regeneration subject")
					}
				} else {
					want := 2
					if eligible {
						want += 2
						if mode == "copy" {
							want += 2
						}
					}
					if g.FindPermanentIncludingPhased(target.ID()).CurrentPower(g) != want {
						t.Fatal("wrong temporary boost")
					}
				}
				if len(a.Graveyard()) != 1 || len(a.Hand()) != 1 {
					t.Fatal("resolution paid cost again")
				}
				if tc.tap {
					if clone.HasRegenerationShield(target.ID()) {
						t.Fatal("live resolution changed pending clone")
					}
				} else if clone.FindPermanentIncludingPhased(target.ID()).CurrentPower(clone) != 2 {
					t.Fatal("live resolution changed pending clone")
				}
				clone.ResolveStack()
				if tc.tap {
					if clone.HasRegenerationShield(target.ID()) != eligible {
						t.Fatal("clone lost regeneration target")
					}
				} else if clone.FindPermanentIncludingPhased(target.ID()).CurrentPower(clone) != g.FindPermanentIncludingPhased(target.ID()).CurrentPower(g) {
					t.Fatal("clone lost temporary boost target")
				}
				if tc.tap && eligible {
					g.DestroyPermanent(g.FindPermanent(target.ID()))
					if mode == "copy" {
						if !g.HasRegenerationShield(target.ID()) {
							t.Fatal("copy did not create its own shield")
						}
						g.DestroyPermanent(g.FindPermanent(target.ID()))
					}
					if g.FindPermanent(target.ID()) == nil || g.HasRegenerationShield(target.ID()) {
						t.Fatal("shield did not replace destruction")
					}
				}
				if !tc.tap && eligible {
					g.RunStepWithPriority(core.Cleanup)
					if g.FindPermanentIncludingPhased(target.ID()).CurrentPower(g) != 2 {
						t.Fatal("boost did not expire")
					}
				}
			})
		}
	}
}

func TestRONRandomDiscardCostInvalid(t *testing.T) {
	for _, cost := range []string{`DiscardACardAtRandom()`, `DiscardACardAtRandom(Integer(1))`, `DiscardNumberCardsAtRandom(Integer(1))`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(`+cost+`,ActionList([GainLife(Integer(1))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", cost)
		}
	}
}

func TestRONRandomDiscardCostNativeLoadAndInstances(t *testing.T) {
	files := fstest.MapFS{}
	for _, tc := range []struct{ set, name string }{{"ATQ", "Coral Helm"}, {"FEM", "Draconian Cylix"}} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "`+tc.name+`"`) {
				files[tc.set+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	old := mage.DefaultRegistry
	for _, name := range []string{"Coral Helm", "Draconian Cylix"} {
		g, a, b := ronGame()
		sources := make([]*mage.Permanent, 0, 2)
		for range 2 {
			card, err := registry.CreateCard(name)
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Colorless, 3)
			if err = g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			sources = append(sources, g.FindPermanent(card.ID()))
		}
		if len(sources) != 2 {
			t.Fatal("native casting did not resolve both instances")
		}
		one := g.PutOnBattlefield(mage.NewCreature("One", "", 2, 3), a.PlayerID())
		two := g.PutOnBattlefield(mage.NewCreature("Two", "", 2, 3), b.PlayerID())
		for _, label := range []string{"First", "Second", "Third"} {
			a.AddToHand(mage.NewLand(label))
		}
		g.SetRandomResults([]int{2, 0, 7})
		a.ManaPool().Add(core.Colorless, 6)
		for i, target := range []*mage.Permanent{one, two} {
			if err = g.ActivateAbilityByIndex(a.PlayerID(), sources[i].ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
		}
		g.ResolveTopOfStack()
		if name == "Coral Helm" {
			if one.CurrentPower(g) != 2 || two.CurrentPower(g) != 4 {
				t.Fatal("instance target state was shared")
			}
		} else {
			if g.HasRegenerationShield(one.ID()) || !g.HasRegenerationShield(two.ID()) {
				t.Fatal("instance regeneration state was shared")
			}
		}
		g.ResolveStack()
		if len(a.Hand()) != 1 || a.Hand()[0].Name() != "Second" || len(a.Graveyard()) != 2 || g.RandIntn(10) != 7 {
			t.Fatal("instance payments were shared")
		}
	}
	if mage.DefaultRegistry != old {
		t.Fatal("fixture load changed production registry")
	}
}

func TestRONRandomDiscardCostModifierAndNestedContext(t *testing.T) {
	g, a, _ := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(And([DiscardACardAtRandom,And([PayMana([ManaCostGeneric(1)]),TapPermanent(ThisPermanent)])]),ActionList([GainLife(Integer(2))]),ActivateOnlyOnceEachTurn)`)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	a.AddToHand(mage.NewLand("First"))
	a.AddToHand(mage.NewLand("Second"))
	a.ManaPool().Add(core.Colorless, 2)
	g.SetRandomResults([]int{1, 7})
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.UntapPermanent(g.FindPermanent(source.ID()))
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("random cost bypassed activation modifier")
	}
	g.ResolveStack()
	if a.Life() != 22 || len(a.Hand()) != 1 || a.Hand()[0].Name() != "First" || a.ManaPool().Count(core.Colorless) != 1 || g.RandIntn(10) != 7 {
		t.Fatal("nested cost or failed repeat spent wrong resources")
	}
}
