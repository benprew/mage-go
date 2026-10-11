package ron

import (
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func rebuildRONDefinition(t *testing.T, card mage.Card) mage.Card {
	t.Helper()
	fresh, err := mage.RebuildCardDefinition(card)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.ID() == card.ID() || fresh.Owner() != uuid.Nil {
		t.Fatal("reconstruction kept instance identity or owner")
	}
	if err := mage.ValidateCardDefinition(fresh.Name(), fresh); err != nil {
		t.Fatal(err)
	}
	return fresh
}

func TestRONDefinitionPlanGameplay(t *testing.T) {
	g, a, b := ronGame()
	original := repositoryCard(t, "FEM", "Icatian Javelineers")
	original.SetOwner(a.PlayerID())
	first := g.PutOnBattlefield(original, a.PlayerID())
	first.RevokeBaseAttr(core.AttrSummonSick)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 1, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	fresh := rebuildRONDefinition(t, original.Copy())
	fresh.SetOwner(b.PlayerID())
	second := g.PutOnBattlefield(fresh, b.PlayerID())
	second.RevokeBaseAttr(core.AttrSummonSick)
	if second.Counters[core.Javelin] != 1 || first.Counters[core.Javelin] != 0 {
		t.Fatal("entry counters or independent payment failed")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), second.ID(), 1, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 19 || !first.Tapped || !second.Tapped {
		t.Fatal("independent sources or targets failed")
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("resolution changed pending clone")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 19 || clone.GetPlayer(b.PlayerID()).Life() != 19 {
		t.Fatal("pending clone lost reconstructed declarations")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 1, []uuid.UUID{b.PlayerID()}); err == nil {
		t.Fatal("spent instance regained its payment")
	}
}

func TestRONDefinitionPlanTargetsAndLKI(t *testing.T) {
	g, a, b := ronGame()
	original := repositoryCard(t, "LEB", "Swords to Plowshares")
	original.SetOwner(a.PlayerID())
	fresh := rebuildRONDefinition(t, original)
	fresh.SetOwner(b.PlayerID())
	first := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEB", "Hill Giant"), b.PlayerID())
	a.AddToHand(original)
	b.AddToHand(fresh)
	if err := g.CastSpellByID(a.PlayerID(), original.ID(), []uuid.UUID{second.ID()}, 0); err == nil {
		t.Fatal("unpaid reconstructed spell cast")
	}
	if g.StackSize() != 0 || len(a.Hand()) != 1 {
		t.Fatal("failed payment changed game state")
	}
	a.ManaPool().Add(core.White, 1)
	b.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), original.ID(), []uuid.UUID{second.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByID(b.PlayerID(), fresh.ID(), []uuid.UUID{first.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 24 || b.Life() != 23 || g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil {
		t.Fatal("reconstructed target declaration or departure information failed")
	}
}

func TestRONDefinitionPlanDelayedBindings(t *testing.T) {
	g, a, b := ronGame()
	original := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([ChooseAPlayer(Opponent),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerAction(TheChosenPlayer,GainLife(Integer(3)))]))]))`)()
	fresh := rebuildRONDefinition(t, original)
	for i, card := range []mage.Card{original, fresh} {
		player := a
		if i == 1 {
			player = b
		}
		card.SetOwner(player.PlayerID())
		player.AddToHand(card)
		if err := g.CastSpellByID(player.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 23 || game.GetPlayer(b.PlayerID()).Life() != 23 {
			t.Fatal("reconstructed binding or saved clone frame failed")
		}
	}
}

func TestRONDefinitionPlanAuraAndMetadata(t *testing.T) {
	g, a, b := ronGame()
	original := repositoryCard(t, "LEB", "Holy Strength")
	fresh := rebuildRONDefinition(t, original)
	first := g.PutOnBattlefield(repositoryCard(t, "LEB", "Wall of Wood"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEB", "Hill Giant"), b.PlayerID())
	for i, card := range []mage.Card{original, fresh} {
		target := first
		if i == 1 {
			target = second
		}
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.White, 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	g.ApplyContinuousEffects()
	if first.CurrentPower(g) != 1 || first.CurrentToughness(g) != 5 || second.CurrentPower(g) != 4 || second.CurrentToughness(g) != 5 {
		t.Fatal("reconstructed aura shares attachment or layer state")
	}
	for _, name := range []string{"Birds of Paradise", "Wall of Wood"} {
		card := repositoryCard(t, "LEB", name)
		fresh := rebuildRONDefinition(t, card)
		if fresh.Name() != card.Name() || fresh.Power() != card.Power() || fresh.Toughness() != card.Toughness() || fresh.ManaCost().String() != card.ManaCost().String() {
			t.Fatal("printed metadata changed")
		}
		if fresh.(*mage.BaseCard).AttrSeeds()[core.Flying] != card.(*mage.BaseCard).AttrSeeds()[core.Flying] || fresh.(*mage.BaseCard).AttrSeeds()[core.Defender] != card.(*mage.BaseCard).AttrSeeds()[core.Defender] {
			t.Fatal("printed keyword seeds changed")
		}
	}
}

func TestRONDefinitionPlanAcceptedCollection(t *testing.T) {
	paths, err := os.ReadDir("../../../../cards/mtgish")
	if err != nil {
		t.Fatal(err)
	}
	accepted, rejected := 0, 0
	for _, path := range paths {
		if !strings.HasSuffix(path.Name(), ".ron") {
			continue
		}
		data, err := os.ReadFile("../../../../cards/mtgish/" + path.Name())
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), path.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			build, err := Compile(node)
			if err != nil {
				rejected++
				continue
			}
			accepted++
			card := build()
			fresh := rebuildRONDefinition(t, card)
			if fresh.Name() != card.Name() {
				t.Fatal("printed name changed")
			}
		}
	}
	if accepted != 2176 || rejected != 269 {
		t.Fatalf("accepted=%d rejected=%d", accepted, rejected)
	}
}

func TestRONDefinitionPlanCopyConsumerBoundaries(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEB.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEB.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, node := range nodes {
		switch node.Fields["Name"].String {
		case "Clone", "Copy Artifact", "Vesuvan Doppelganger":
			found++
			_, err := Compile(node)
			if node.Fields["Name"].String != "Vesuvan Doppelganger" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "AddAbility") {
				t.Fatalf("copy consumer: %v", err)
			}
		}
	}
	if found != 3 {
		t.Fatal("copy consumer records missing")
	}
}
