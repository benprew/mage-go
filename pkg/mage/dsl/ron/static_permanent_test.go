package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONStaticPermanentLords(t *testing.T) {
	for _, tc := range []struct {
		name, subtype, creature, land string
		keyword                       core.Attr
	}{
		{"Goblin King", "Goblin", "Mons's Goblin Raiders", "Mountain", core.Mountainwalk},
		{"Lord of Atlantis", "Merfolk", "Merfolk of the Pearl Trident", "Island", core.Islandwalk},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			factory := staticLordFactory(t, tc.name)
			card := factory()
			if err := mage.ValidateCardDefinition(tc.name, card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			own := g.PutOnBattlefield(repositoryCard(t, "LEB", tc.creature), a.PlayerID())
			other := g.PutOnBattlefield(repositoryCard(t, "LEB", tc.creature), b.PlayerID())
			nonmatching := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
			land := g.PutOnBattlefield(repositoryCard(t, "LEB", tc.land), b.PlayerID())
			check := func(game *mage.Game, p *mage.Permanent, active bool) {
				t.Helper()
				current := game.FindPermanent(p.ID())
				want := 1
				if active {
					want = 2
				}
				if current.CurrentPower(game) != want || current.CurrentToughness(game) != want || current.HasKeyword(tc.keyword) != active {
					t.Fatalf("recipient = %d/%d, landwalk %v; want %d/%d, %v", current.CurrentPower(game), current.CurrentToughness(game), current.HasKeyword(tc.keyword), want, want, active)
				}
			}
			check(g, own, true)
			check(g, other, true)
			check(g, nonmatching, false)
			if source.CurrentPower(g) != 2 || source.CurrentToughness(g) != 2 || source.HasKeyword(tc.keyword) {
				t.Fatal("source affected itself")
			}
			if !mage.HasLandwalkEvasion(own, b.PlayerID(), g) || mage.HasLandwalkEvasion(own, a.PlayerID(), g) {
				t.Fatal("landwalk used wrong defender")
			}
			phasedLand := g.PhaseOut(land.ID())
			g.ApplyContinuousEffects()
			if mage.HasLandwalkEvasion(g.FindPermanent(own.ID()), b.PlayerID(), g) {
				t.Fatal("phased land enabled landwalk")
			}
			g.PhaseIn(phasedLand)
			g.ApplyContinuousEffects()
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			check(g, own, true)
			check(g, other, true)
			clone := g.Clone()
			clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
			check(clone, own, false)
			check(g, own, true)
			phased := g.PhaseOut(source.ID())
			g.ApplyContinuousEffects()
			check(g, own, false)
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			check(g, own, true)
			phased = g.PhaseOut(own.ID())
			g.ApplyContinuousEffects()
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			check(g, own, true)
			remover := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
			removal.SetSourceID(remover.ID())
			g.AddContinuousEffect(removal)
			check(g, own, false)
			g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
			check(g, own, true)
			g.RemoveFromBattlefield(g.FindPermanent(own.ID()))
			own = g.PutOnBattlefield(own.Card, b.PlayerID())
			check(g, own, true)
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			check(g, own, false)
			source = g.PutOnBattlefield(card, a.PlayerID())
			check(g, own, true)
			if source.HasKeyword(tc.keyword) {
				t.Fatal("returned source received own grant")
			}
			second := g.PutOnBattlefield(factory(), b.PlayerID())
			for _, p := range []*mage.Permanent{source, second} {
				if p.CurrentPower(g) != 3 || p.CurrentToughness(g) != 3 || !p.HasKeyword(tc.keyword) {
					t.Fatal("same-factory sources did not affect each other")
				}
			}
			if p := g.FindPermanent(own.ID()); p.CurrentPower(g) != 3 || p.CurrentToughness(g) != 3 {
				t.Fatal("lord bonuses did not accumulate")
			}
			g.RemoveFromBattlefield(second)
			if p := g.FindPermanent(source.ID()); p.CurrentPower(g) != 2 || p.HasKeyword(tc.keyword) {
				t.Fatal("independent source departure failed")
			}
		})
	}
}

func staticLordFactory(t *testing.T, name string) mage.CardFactory {
	t.Helper()
	data, err := os.ReadFile("../../../../cards/mtgish/LEB.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEB.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range nodes {
		if node.Fields["Name"].String == name {
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			return factory
		}
	}
	t.Fatalf("missing card %s", name)
	return nil
}

func TestRONStaticPermanentCastTiming(t *testing.T) {
	for _, tc := range []struct {
		name, creature string
		color          core.Color
		mana           int
	}{
		{"Goblin King", "Mons's Goblin Raiders", core.Red, 3},
		{"Lord of Atlantis", "Merfolk of the Pearl Trident", core.Blue, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			recipient := g.PutOnBattlefield(repositoryCard(t, "LEB", tc.creature), b.PlayerID())
			card := staticLordFactory(t, tc.name)()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("cast without mana succeeded")
			}
			if g.Stack().Size() != 0 || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("failed cost changed game")
			}
			a.ManaPool().Add(tc.color, tc.mana)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || recipient.CurrentPower(g) != 1 {
				t.Fatal("incorrect mana cost or premature bonus")
			}
			clone := g.Clone()
			clone.ResolveTopOfStack()
			if clone.FindPermanent(recipient.ID()).CurrentPower(clone) != 2 || g.FindPermanent(recipient.ID()).CurrentPower(g) != 1 || g.Stack().Size() != 1 {
				t.Fatal("cloned resolution changed original")
			}
			g.ResolveTopOfStack()
			if g.FindPermanent(recipient.ID()).CurrentPower(g) != 2 {
				t.Fatal("resolution did not install static layer")
			}
		})
	}
}

func staticPermanentFixture(t *testing.T, types, subtype, rules string) mage.CardFactory {
	t.Helper()
	src := fmt.Sprintf(`Card(Name: "Static fixture", Typeline: (Supertypes: [], Cardtypes: [%s], Subtypes: [%s]), Rules: [%s])`, types, subtype, rules)
	nodes, err := Parse(strings.NewReader(src), "static-permanent.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Static fixture", factory()); err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONStaticPermanentNoncreaturesAndInstances(t *testing.T) {
	for _, controlled := range []bool{false, true} {
		t.Run(fmt.Sprint(controlled), func(t *testing.T) {
			filter := `And([Other(ThisPermanent), IsCreatureType(Goblin)])`
			if controlled {
				filter = `And([Other(ThisPermanent), And([IsCreatureType(Goblin), ControlledByAPlayer(SinglePlayer(You))])])`
			}
			factory := staticPermanentFixture(t, "Artifact", "Goblin", fmt.Sprintf(`EachPermanentLayerEffect(%s, [AdjustPT(1, 2), AddAbility([Flying, Landwalk(IsLandType(Mountain))])])`, filter))
			g, a, b := ronGame()
			first, second := g.PutOnBattlefield(factory(), a.PlayerID()), g.PutOnBattlefield(factory(), b.PlayerID())
			recipientFactory := staticPermanentFixture(t, "Artifact", "Goblin", "")
			own := g.PutOnBattlefield(recipientFactory(), a.PlayerID())
			other := g.PutOnBattlefield(recipientFactory(), b.PlayerID())
			for _, p := range []*mage.Permanent{own, other} {
				if p.HasType(core.TypeCreature) || !p.HasKeyword(core.Flying) || !p.HasKeyword(core.Mountainwalk) {
					t.Fatal("noncreature did not receive keywords")
				}
			}
			if first.HasKeyword(core.Flying) == controlled || second.HasKeyword(core.Flying) == controlled {
				t.Fatal("per-instance source exclusion or control failed")
			}
			animation := mage.TemporaryAnimate(own.ID(), 3, 4)
			animation.SetSourceID(first.ID())
			g.AddContinuousEffect(animation)
			wantP, wantT := 5, 8
			if controlled {
				wantP, wantT = 4, 6
			}
			if p := g.FindPermanent(own.ID()); p.CurrentPower(g) != wantP || p.CurrentToughness(g) != wantT {
				t.Fatal("noncreature modification missing after animation")
			}
			clone := g.Clone()
			clone.RemoveFromBattlefield(clone.FindPermanent(first.ID()))
			if controlled && clone.FindPermanent(own.ID()).HasKeyword(core.Flying) {
				t.Fatal("departed controlled source still grants")
			}
			if !g.FindPermanent(own.ID()).HasKeyword(core.Flying) {
				t.Fatal("clone changed original")
			}
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: second.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			if p := g.FindPermanent(own.ID()); p.CurrentPower(g) != 5 || p.CurrentToughness(g) != 8 {
				t.Fatal("changed controller did not update membership")
			}
			if controlled && g.FindPermanent(other.ID()).HasKeyword(core.Flying) {
				t.Fatal("old controller retained grant")
			}
			phased := g.PhaseOut(second.ID())
			g.ApplyContinuousEffects()
			if p := g.FindPermanent(own.ID()); p.CurrentPower(g) != 4 || p.CurrentToughness(g) != 6 {
				t.Fatal("phased source still boosts")
			}
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			if p := g.FindPermanent(own.ID()); p.CurrentPower(g) != 5 || p.CurrentToughness(g) != 8 {
				t.Fatal("source phase-in did not restore boost")
			}
			late := g.PutOnBattlefield(recipientFactory(), a.PlayerID())
			if !late.HasKeyword(core.Flying) {
				t.Fatal("late entry missing keyword")
			}
			g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
			if g.FindPermanent(second.ID()).HasKeyword(core.Flying) {
				t.Fatal("remaining source retained another instance's grant")
			}
			g.RemoveFromBattlefield(g.FindPermanent(second.ID()))
			if g.FindPermanent(own.ID()).HasKeyword(core.Flying) || g.FindPermanent(late.ID()).HasKeyword(core.Mountainwalk) {
				t.Fatal("departed grants remained")
			}
		})
	}
}

func TestRONStaticPermanentInvalid(t *testing.T) {
	for _, filter := range []string{
		`Other(HostPermanent)`, `Other(Ref_TargetPermanent)`, `ControlledByAPlayer(SinglePlayer(Opponent))`,
		`Or([Other(ThisPermanent), AnyPermanent])`, `Or([ControlledByAPlayer(SinglePlayer(You)), AnyPermanent])`,
		`And([])`, `Or([])`, `Or([And([]), AnyPermanent])`, `IsCreatureType(Goblin())`, `IsCreatureType(Goblin, Merfolk)`,
	} {
		t.Run(filter, func(t *testing.T) {
			src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [EachPermanentLayerEffect(%s, [AdjustPT(1, 1)])])`, filter)
			nodes, err := Parse(strings.NewReader(src), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid static filter accepted")
			}
		})
	}
}

func TestRONStaticPermanentCurrentMembership(t *testing.T) {
	factory := staticPermanentFixture(t, "Enchantment", "", `EachPermanentLayerEffect(Or([IsCreatureType(Goblin), And([IsCreatureType(Merfolk), IsCardtype(Creature)])]), [AdjustPT(2, 3), AddAbility([Flying])])`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	creature := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
	if creature.HasKeyword(core.Flying) {
		t.Fatal("nonmatching creature received grant")
	}
	change := mage.Until(core.EndOfTurn, mage.SetSubtypeFamily(mage.This, mage.SubtypeCreature, "Goblin"))
	if err := mage.ApplyEffect(g, change, creature.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if p := g.FindPermanent(creature.ID()); !p.HasKeyword(core.Flying) || p.CurrentPower(g) != 3 || p.CurrentToughness(g) != 4 {
		t.Fatal("layer 4 change did not update static membership")
	}
	clone := g.Clone()
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if clone.FindPermanent(creature.ID()).HasKeyword(core.Flying) || !g.FindPermanent(creature.ID()).HasKeyword(core.Flying) {
		t.Fatal("subtype expiry did not stay in clone")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	if p := g.FindPermanent(creature.ID()); p.HasKeyword(core.Flying) || p.CurrentPower(g) != 1 {
		t.Fatal("source departure retained modification")
	}
}

func TestRONStaticPermanentRootExclusionAndCreatureBoundary(t *testing.T) {
	factory := staticPermanentFixture(t, "Artifact", "", `EachPermanentLayerEffect(Other(ThisPermanent), [AdjustPT(1, 1), AddAbility([Flying])])`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
	if source.HasKeyword(core.Flying) || !land.HasKeyword(core.Flying) {
		t.Fatal("root source exclusion failed for noncreatures")
	}
	creatureOnly := staticPermanentFixture(t, "Enchantment", "", `EachPermanentLayerEffect(And([IsCreatureType(Goblin), IsCardtype(Creature)]), [AddAbility([Trample])])`)
	g.PutOnBattlefield(creatureOnly(), a.PlayerID())
	noncreature := g.PutOnBattlefield(staticPermanentFixture(t, "Artifact", "Goblin", "")(), a.PlayerID())
	if noncreature.HasKeyword(core.Trample) {
		t.Fatal("explicit creature restriction lost")
	}
	animation := mage.TemporaryAnimate(noncreature.ID(), 2, 2)
	animation.SetSourceID(source.ID())
	g.AddContinuousEffect(animation)
	if p := g.FindPermanent(noncreature.ID()); !p.HasKeyword(core.Trample) || p.CurrentPower(g) != 3 {
		t.Fatal("animation did not update explicit creature membership")
	}
}

func TestRONStaticPermanentInventory(t *testing.T) {
	src := `Card(Name: "Inventory", Typeline: (Supertypes: [], Cardtypes: [Enchantment], Subtypes: []), Rules: [EachPermanentLayerEffect(And([Other(ThisPermanent), IsCreatureType(Goblin)]), [AdjustPT(1, 1), AddAbility([Flying])])])`
	report, err := InspectFS(fstest.MapFS{"static.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 || len(report.Issues) != 0 {
		t.Fatal("new shape not included in compilation inventory")
	}
	seen := false
	for _, entry := range report.Signatures {
		if entry.Name == "Goblin" {
			seen = true
			if entry.Role != "creature_subtype" {
				t.Fatalf("subtype role = %s", entry.Role)
			}
		}
		if entry.Support != "unverified" {
			t.Fatal("inventory asserts gameplay support")
		}
	}
	if !seen {
		t.Fatal("nested subtype absent")
	}
}
