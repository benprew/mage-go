package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPermanentFilterGoblinCaves(t *testing.T) {
	for _, scenario := range []string{"normal", "unpaid", "nonland", "departed", "phased"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			hostCard := repositoryCard(t, "LEB", "Mountain")
			if scenario == "nonland" {
				hostCard = mage.NewCreature("Host", "", 2, 2)
			}
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			goblin := g.PutOnBattlefield(mage.NewCreature("Goblin", "", 2, 2, mage.WithSubTypes("Goblin")), a.PlayerID())
			opposing := g.PutOnBattlefield(mage.NewCreature("Other Goblin", "", 2, 2, mage.WithSubTypes("Goblin")), b.PlayerID())
			plain := g.PutOnBattlefield(mage.NewCreature("Plain", "", 2, 2), a.PlayerID())
			card := repositoryCard(t, "DRK", "Goblin Caves")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if scenario != "unpaid" {
				a.ManaPool().Add(core.Red, 3)
			}
			err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0)
			if scenario == "unpaid" || scenario == "nonland" {
				if err == nil || len(a.Hand()) != 1 || !g.Stack().IsEmpty() {
					t.Fatal("invalid cast changed state")
				}
				if scenario == "nonland" && a.ManaPool().TotalMana() != 3 {
					t.Fatal("invalid target paid mana")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			checkConditionalPT(t, g, goblin.ID(), 2, 2)
			if scenario == "departed" {
				g.BouncePermanentToHand(host)
			}
			if scenario == "phased" {
				g.PhaseOut(host.ID())
			}
			g.ResolveStack()
			if scenario != "normal" {
				if g.FindPermanent(card.ID()) != nil {
					t.Fatal("illegal Aura resolved")
				}
				return
			}
			if a.ManaPool().TotalMana() != 0 {
				t.Fatal("cast cost not paid")
			}
			check := func(game *mage.Game, toughness int) {
				t.Helper()
				checkConditionalPT(t, game, goblin.ID(), 2, toughness)
				checkConditionalPT(t, game, opposing.ID(), 2, toughness)
				checkConditionalPT(t, game, plain.ID(), 2, 2)
			}
			check(g, 4)
			clone := g.Clone()
			nonbasic := g.PutOnBattlefield(mage.NewLand("Nonbasic Mountain", mage.WithSubTypes("Mountain")), a.PlayerID())
			g.Attach(card.ID(), nonbasic.ID())
			check(g, 2)
			check(clone, 4)
			forest := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			g.Attach(card.ID(), forest.ID())
			check(g, 2)
			g.Attach(card.ID(), host.ID())
			check(g, 4)
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetSubtypeFamily(mage.This, mage.SubtypeLand, "Forest")), host.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			check(g, 2)
			g.RunStepWithPriority(core.Cleanup)
			check(g, 4)
			later := g.PutOnBattlefield(mage.NewCreature("Later Goblin", "", 1, 1, mage.WithSubTypes("Goblin")), b.PlayerID())
			checkConditionalPT(t, g, later.ID(), 1, 3)
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetSubtypeFamily(mage.This, mage.SubtypeCreature, "Elf")), goblin.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			checkConditionalPT(t, g, goblin.ID(), 2, 2)
			checkConditionalPT(t, g, opposing.ID(), 2, 4)
			g.RunStepWithPriority(core.Cleanup)
			check(g, 4)
			phased := g.PhaseOut(card.ID())
			g.ApplyContinuousEffects()
			check(g, 2)
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			check(g, 4)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			check(g, 4)
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), card.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			check(g, 2)
			g.RunStepWithPriority(core.Cleanup)
			check(g, 4)
			g.BouncePermanentToHand(g.FindPermanent(host.ID()))
			check(g, 2)
			b.RemoveFromHand(hostCard.ID())
			g.PutOnBattlefield(hostCard, b.PlayerID())
			check(g, 2)
			check(clone, 4)
		})
	}
}

func TestRONPermanentFilterInvalid(t *testing.T) {
	for _, condition := range []string{
		`PermanentPassesFilter(HostPermanent, IsCardtype(Land))`,
		`PermanentPassesFilter(Ref_TargetPermanent, IsCardtype(Land))`,
		`PermanentPassesFilter(Trigger_ThatPermanent, IsCardtype(Land))`,
		`PermanentPassesFilter(ThisPermanent(), IsCardtype(Land))`,
		`PermanentPassesFilter(ThisPermanent, And([]))`,
		`PermanentPassesFilter(ThisPermanent, IsSupertype(Basic()))`,
		`PermanentPassesFilter(ThisPermanent, PowerIs(EqualTo(Integer(2))))`,
		`PermanentPassesFilter(ThisPermanent, HasAbility(Flying))`,
		`PermanentPassesFilter(ThisPermanent, IsCardtype(Land), IsUntapped)`,
	} {
		assertConditionalRejected(t, `If(`+condition+`, [PermanentLayerEffect(ThisPermanent, [AdjustPT(1, 1)])])`)
	}
}

func TestRONPermanentFilterSourceInstances(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name: "Rule", Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: []), CardPT: (Power: 2, Toughness: 3), Rules: [If(PermanentPassesFilter(ThisPermanent, IsTapped), [PermanentLayerEffect(ThisPermanent, [AdjustPT(1, 2)])])])`), "source-condition.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	first, second := factory(), factory()
	if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
		t.Fatal(err)
	}
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(first, a.PlayerID())
	q := g.PutOnBattlefield(second, b.PlayerID())
	baseP, baseT := first.Power(), first.Toughness()
	checkConditionalPT(t, g, p.ID(), baseP, baseT)
	g.TapPermanent(p)
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, p.ID(), baseP+1, baseT+2)
	checkConditionalPT(t, g, q.ID(), baseP, baseT)
	clone := g.Clone()
	g.UntapPermanent(g.FindPermanent(p.ID()))
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, p.ID(), baseP, baseT)
	checkConditionalPT(t, clone, p.ID(), baseP+1, baseT+2)
	g.TapPermanent(q)
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, q.ID(), baseP+1, baseT+2)
}
