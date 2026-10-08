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

func TestRONConditionalAnimationRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			card := repositoryCard(t, set, "Animate Artifact")
			g, a, b := ronGame()
			host := g.PutOnBattlefield(mage.NewArtifact("Host", "{3}"), b.PlayerID())
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
				t.Fatal("unpaid Aura cast")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed payment changed state")
			}
			a.ManaPool().Add(core.Blue, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			pending := g.Clone()
			pending.ResolveStack()
			checkConditionalPT(t, pending, host.ID(), 3, 3)
			g.ResolveStack()
			checkConditionalPT(t, g, host.ID(), 3, 3)
			if !g.FindPermanent(host.ID()).HasType(core.TypeArtifact) || !g.FindPermanent(host.ID()).HasType(core.TypeCreature) {
				t.Fatal("host types")
			}
			clone := g.Clone()
			other := g.PutOnBattlefield(mage.NewArtifact("Other", "{5}"), a.PlayerID())
			g.Attach(card.ID(), other.ID())
			if g.FindPermanent(host.ID()).HasType(core.TypeCreature) {
				t.Fatal("old host keeps animation")
			}
			checkConditionalPT(t, g, other.ID(), 5, 5)
			checkConditionalPT(t, clone, host.ID(), 3, 3)
			phased := g.PhaseOut(card.ID())
			g.ApplyContinuousEffects()
			if g.FindPermanent(other.ID()).HasType(core.TypeCreature) {
				t.Fatal("phased source animates")
			}
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, other.ID(), 5, 5)
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), card.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, other.ID(), 5, 5)
			g.BouncePermanentToHand(g.FindPermanent(other.ID()))
			a.RemoveFromHand(other.Card.ID())
			returned := g.PutOnBattlefield(other.Card, a.PlayerID())
			if returned.HasType(core.TypeCreature) {
				t.Fatal("old attachment follows returned host")
			}
		})
	}
}

func TestRONConditionalAnimationInvalid(t *testing.T) {
	for _, body := range []string{
		`If(PermanentPassesFilter(ThisPermanent,IsUntapped),[PermanentLayerEffect(ThisPermanent,[AddCardtype(Creature),SetPowerAndToughnessBoth(ValueX)])])`,
		`If(PermanentPassesFilter(ThisPermanent,IsUntapped),[PermanentLayerEffect(ThisPermanent,[AddCardtype(Creature),SetPowerAndToughnessBoth(Integer(2)),SetPowerAndToughnessBoth(Integer(3))])])`,
		`If(PermanentPassesFilter(ThisPermanent,IsUntapped),[PermanentLayerEffect(ThisPermanent,[AddCardtype(Creature()),SetPowerAndToughnessBoth(Integer(2))])])`,
		`If(PermanentPassesFilter(ThisPermanent,IsUntapped),[PermanentLayerEffect(ThisPermanent,[AddCardtype(Creature),SetPowerAndToughnessBoth(Integer(2)),LosesAllAbilities])])`,
	} {
		assertConditionalRejected(t, body)
	}
}

func TestRONConditionalAnimationLegalityAndExistingCreature(t *testing.T) {
	for _, scenario := range []string{"nonartifact", "departed", "phased", "creature"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			hostCard := mage.NewCreature("Creature", "{6}", 2, 7, mage.WithCardType(core.TypeArtifact))
			if scenario == "nonartifact" {
				hostCard = mage.NewCreature("Creature", "{6}", 2, 7)
			}
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			card := repositoryCard(t, "LEB", "Animate Artifact")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 4)
			err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0)
			if scenario == "nonartifact" {
				if err == nil || a.ManaPool().Count(core.Blue) != 4 || g.StackSize() != 0 {
					t.Fatal("illegal target paid or cast")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "departed" {
				g.BouncePermanentToHand(host)
			}
			if scenario == "phased" {
				g.PhaseOut(host.ID())
			}
			g.ResolveStack()
			if scenario != "creature" {
				if g.FindPermanent(card.ID()) != nil {
					t.Fatal("Aura resolved with an illegal target")
				}
				return
			}
			checkConditionalPT(t, g, host.ID(), 2, 7)
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(1), mage.Fixed(2))), host.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, host.ID(), 3, 9)
		})
	}
}

func TestRONConditionalAnimationCopyAndLayers(t *testing.T) {
	g, a, b := ronGame()
	host := g.PutOnBattlefield(mage.NewArtifact("Host", "{4}"), b.PlayerID())
	card := repositoryCard(t, "LEB", "Animate Artifact")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	g.ResolveStack()
	checkConditionalPT(t, g, host.ID(), 4, 4)
	clone := g.Clone()
	g.BouncePermanentToHand(g.FindPermanent(card.ID()))
	checkConditionalPT(t, g, host.ID(), 4, 4)
	if g.FindPermanent(copied.SourceID) == nil {
		t.Fatal("copy did not become an Aura")
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(2), mage.Fixed(3)), mage.SwitchPowerToughness(mage.This)), host.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(host.ID()).AddCounter(core.P1P1, 1)
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, host.ID(), 8, 7)
	checkConditionalPT(t, clone, host.ID(), 4, 4)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, host.ID(), 5, 5)
	g.RemoveFromBattlefield(g.FindPermanent(copied.SourceID))
	g.ApplyContinuousEffects()
	if g.FindPermanent(host.ID()).HasType(core.TypeCreature) {
		t.Fatal("animation survives all source departures")
	}
}

func TestRONConditionalAnimationStrictLoad(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.HasPrefix(line, `Card(Name: "Animate Artifact",`) {
				files[set+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	if len(files) != 6 {
		t.Fatal("missing reprint")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("reprints were counted separately")
	}
	one, err := registry.CreateCard("Animate Artifact")
	if err != nil {
		t.Fatal(err)
	}
	two, err := registry.CreateCard("Animate Artifact")
	if err != nil {
		t.Fatal(err)
	}
	if one.ID() == two.ID() || one.Abilities()[0].AbilityID() == two.Abilities()[0].AbilityID() {
		t.Fatal("factory instances share identity")
	}
	files["unsupported.ron"] = &fstest.MapFile{Data: []byte(testBolt + "\nNotACard")}
	if partial, err := LoadFS(files, "*.ron"); err == nil || partial != nil {
		t.Fatal("invalid collection published a partial registry")
	}
}
