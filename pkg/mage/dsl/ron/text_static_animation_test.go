package ron

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func animationCheck(t *testing.T, g *mage.Game, p *mage.Permanent, animated bool, colors ...core.Color) {
	t.Helper()
	current := g.FindPermanent(p.ID())
	if current == nil || current.HasType(core.TypeCreature) != animated || !current.HasType(core.TypeLand) {
		t.Fatal("wrong animation membership")
	}
	if animated {
		staticPTCheck(t, g, p, 1, 1)
	}
	if colors != nil && (len(current.Colors()) != len(colors) || current.Colors()[0] != colors[0]) {
		t.Fatal("wrong animation colors", current.Colors())
	}
}

func TestRONTextStaticAnimationRecords(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			name := node.Fields["Name"].String
			if name != "Kormus Bell" && name != "Living Lands" && name != "Living Plane" {
				continue
			}
			count++
			t.Run(filepath.Base(file)+"/"+name, func(t *testing.T) {
				factory, err := Compile(node)
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				oldLand := "Forest"
				from := core.ForestLandType
				if name == "Kormus Bell" {
					oldLand = "Swamp"
					from = core.SwampLandType
				}
				old := g.PutOnBattlefield(repositoryCard(t, "LEA", oldLand), a.PlayerID())
				island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
				card := factory()
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
					t.Fatal("unpaid casting succeeded")
				}
				if len(a.Hand()) != 1 || g.Stack().Size() != 0 || old.HasType(core.TypeCreature) {
					t.Fatal("failed casting changed state")
				}
				a.ManaPool().Add(core.Green, 4)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				if old.HasType(core.TypeCreature) {
					t.Fatal("animation began before entry")
				}
				g.ResolveTopOfStack()
				source := g.FindPermanent(card.ID())
				id := textStaticID(t, source)
				change, _ := mage.ReplaceBasicLandWord(from, core.IslandLandType)
				if err := g.RevisePrintedStaticAnimation(source, id, change); err != nil {
					t.Fatal(err)
				}
				if textStaticID(t, g.FindPermanent(source.ID())) != id {
					t.Fatal("declaration identity changed")
				}
				animationCheck(t, g, island, true)
				animationCheck(t, g, old, name == "Living Plane")
				if name == "Kormus Bell" {
					color, _ := mage.ReplaceColorWord(core.Black, core.Red)
					if err := g.RevisePrintedStaticAnimation(g.FindPermanent(source.ID()), id, color); err != nil {
						t.Fatal(err)
					}
					animationCheck(t, g, island, true, core.Red)
					if len(source.Card.ManaCost().Colors()) != 0 || source.Card.Name() != name {
						t.Fatal("text changed printed metadata")
					}
				}
				clone := g.Clone()
				next, _ := mage.ReplaceBasicLandWord(core.IslandLandType, from)
				if err := clone.RevisePrintedStaticAnimation(clone.FindPermanent(source.ID()), id, next); err != nil {
					t.Fatal(err)
				}
				animationCheck(t, clone, old, true)
				animationCheck(t, g, island, true)
				animationCheck(t, clone, island, name == "Living Plane")
				late := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), a.PlayerID())
				animationCheck(t, g, late, true)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), late.ID(), 0, nil); err == nil {
					t.Fatal("animated land ignored summoning sickness")
				}
				phased := g.PhaseOut(source.ID())
				g.ApplyContinuousEffects()
				animationCheck(t, g, island, false)
				if err := g.RevisePrintedStaticAnimation(source, id, next); err == nil {
					t.Fatal("phased source accepted")
				}
				g.PhaseIn(phased)
				g.ApplyContinuousEffects()
				animationCheck(t, g, island, true)
				phased = g.PhaseOut(island.ID())
				g.ApplyContinuousEffects()
				g.PhaseIn(phased)
				g.ApplyContinuousEffects()
				animationCheck(t, g, island, true)
				remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
				removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
				removal.SetSourceID(remover.ID())
				g.AddContinuousEffect(removal)
				animationCheck(t, g, island, true)
				if err := g.RevisePrintedStaticAnimation(g.FindPermanent(source.ID()), id, next); err == nil {
					t.Fatal("removed declaration accepted")
				}
				g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
				animationCheck(t, g, island, true)
				if err := g.RevisePrintedStaticAnimation(g.FindPermanent(source.ID()), id, next); err != nil {
					t.Fatal(err)
				}
				animationCheck(t, g, old, true)
				g.BouncePermanentToHand(g.FindPermanent(old.ID()))
				if lki := g.LKI(old.ID()); lki == nil || lki.Power != 1 || !lki.Snapshot.HasType(core.TypeCreature) {
					t.Fatal("departure lost revised animation")
				}
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				animationCheck(t, g, island, false)
				returned := g.PutOnBattlefield(card, a.PlayerID())
				if err := g.RevisePrintedStaticAnimation(source, id, change); err == nil {
					t.Fatal("returned source inherited old visit")
				}
				old = g.PutOnBattlefield(repositoryCard(t, "LEA", oldLand), a.PlayerID())
				animationCheck(t, g, old, true)
				if name == "Kormus Bell" {
					animationCheck(t, g, old, true, core.Black)
				}
				if textStaticID(t, returned) != id {
					t.Fatal("return lost printed identity")
				}
			})
		}
	}
	if count != 13 {
		t.Fatal("record coverage", count)
	}
}

func TestRONTextStaticAnimationBindings(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `EachPermanentLayerEffect(And([IsLandType(Forest),ControlledByAPlayer(SinglePlayer(You)),Other(ThisPermanent)]),[SetPT(PT(1,1)),AddCardtype(Creature)]),EachPermanentLayerEffect(IsLandType(Forest),[SetPT(PT(2,3)),AddCardtype(Creature)])`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	own := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), a.PlayerID())
	other := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	first, second := source.RuntimeAbilities[0].AbilityID(), source.RuntimeAbilities[1].AbilityID()
	change, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	if err := g.RevisePrintedStaticAnimation(source, first, change); err != nil {
		t.Fatal(err)
	}
	animationCheck(t, g, own, true)
	animationCheck(t, g, other, false)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	animationCheck(t, g, own, false)
	animationCheck(t, g, other, true)
	if err := g.RevisePrintedStaticAnimation(g.FindPermanent(source.ID()), second, change); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, own, 2, 3)
	staticPTCheck(t, g, other, 2, 3)
}

func TestRONTextStaticAnimationBoundaries(t *testing.T) {
	change, _ := mage.ReplaceColorWord(core.Black, core.Red)
	for _, tc := range []struct{ set, name string }{{"LEA", "Bad Moon"}, {"LEA", "Goblin King"}, {"LEA", "Holy Strength"}} {
		g, a, _ := ronGame()
		source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
		if err := g.RevisePrintedStaticAnimation(source, textStaticID(t, source), change); err == nil {
			t.Fatal("unmarked context accepted", tc.name)
		}
	}
	for _, tc := range []struct{ set, name string }{{"ATQ", "Titania's Song"}, {"LEA", "Magical Hack"}, {"LEA", "Sleight of Mind"}} {
		if _, err := Compile(staticBaseRecord(t, tc.set, tc.name)); err == nil {
			t.Fatal("dependent context accepted", tc.name)
		}
	}
	TestRONStaticAnimationInvalid(t)
}
