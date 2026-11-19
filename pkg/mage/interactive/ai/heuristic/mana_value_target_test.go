package heuristic

import (
	"fmt"
	"testing"
	"uuid"

	_ "github.com/benprew/mage-go/cards/antiquities"
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
)

func TestManaValueTargetAnnouncement(t *testing.T) {
	for _, x := range []int{0, 3, 9} {
		t.Run(fmt.Sprint(x), func(t *testing.T) {
			g, a, b := makeGame()
			g.SetStep(core.PrecombatMain)
			addLands(g, a, "Mountain", x+1)
			victim := g.PutOnBattlefield(mage.NewCreature("Victim", fmt.Sprintf("{%d}", x), 0, 2, mage.WithCardType(core.TypeArtifact)), b.PlayerID())
			g.PutOnBattlefield(mage.NewArtifact("Too expensive", "{20}"), b.PlayerID())
			spell := mage.NewSorcery("Composed removal", "{X}{R}", mage.NewTargetedSpell(mage.TargetPermanent(mage.IsCardType(core.TypeArtifact), mage.ManaValueIs(mage.EqualTo(mage.XValue()))), mage.DestroyTarget()))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			s := &Strategy{Personality: ai.AggroPersonality}
			action := s.PriorityAction(a, g, 0, true)
			if action.Type != interactive.ActionCastSpell || action.XValue != x || len(action.Targets) != 1 || action.Targets[0] != victim.ID() {
				t.Fatalf("wrong announcement: %+v", action)
			}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), action.Targets, action.XValue); err != nil {
				t.Fatalf("AI returned illegal cast: %v", err)
			}
		})
	}
}

func TestDetonateAnnouncement(t *testing.T) {
	for _, x := range []int{0, 3} {
		t.Run(fmt.Sprint(x), func(t *testing.T) {
			g, a, b := makeGame()
			g.SetStep(core.PrecombatMain)
			addLands(g, a, "Mountain", x+1)
			name := "Ornithopter"
			if x == 3 {
				name = "Jalum Tome"
			}
			card, err := mage.CreateCard(name)
			if err != nil {
				t.Fatal(err)
			}
			victim := g.PutOnBattlefield(card, b.PlayerID())
			spell, err := mage.CreateCard("Detonate")
			if err != nil {
				t.Fatal(err)
			}
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			s := &Strategy{Personality: ai.AggroPersonality}
			action := s.PriorityAction(a, g, 0, true)
			if action.Type != interactive.ActionCastSpell || action.XValue != x || len(action.Targets) != 1 || action.Targets[0] != victim.ID() {
				t.Fatalf("wrong Detonate announcement: %+v", action)
			}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), action.Targets, action.XValue); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(victim.ID()) != nil || b.Life() != 20-x {
				t.Fatal("Detonate did not resolve correctly")
			}
		})
	}
}

func TestSpellBlastAnnouncement(t *testing.T) {
	for _, x := range []int{0, 3, 9} {
		t.Run(fmt.Sprint(x), func(t *testing.T) {
			g, a, b := makeGame()
			g.SetStep(core.PrecombatMain)
			addLands(g, a, "Island", x+1)
			b.ManaPool().Add(core.Red, x)
			target := mage.NewInstant("Threat", fmt.Sprintf("{%d}", x), mage.NewTargetedSpell(mage.TargetPlayer(), mage.DealDamage(mage.Fixed(4))))
			b.AddToHand(target)
			if err := g.CastSpellByID(b.PlayerID(), target.ID(), []uuid.UUID{a.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
			spell, err := mage.CreateCard("Spell Blast")
			if err != nil {
				t.Fatal(err)
			}
			a.AddToHand(spell)
			s := &Strategy{Personality: ai.AggroPersonality}
			action := s.PriorityAction(a, g, 0, false)
			if action.Type != interactive.ActionCastSpell || action.CardID != spell.ID() || action.XValue != x || len(action.Targets) != 1 || action.Targets[0] != target.ID() {
				t.Fatalf("wrong Spell Blast announcement: %+v", action)
			}
			if err := g.CastSpellByID(a.PlayerID(), action.CardID, action.Targets, action.XValue); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("Spell Blast did not counter the threat")
			}
		})
	}
}

func TestXCounterResponseUsesAffordableTarget(t *testing.T) {
	g, a, b := makeGame()
	g.SetStep(core.PrecombatMain)
	g.SetChannelActive(a.PlayerID())
	a.SetLife(5)
	a.ManaPool().Add(core.Blue, 1)
	b.ManaPool().Add(core.Red, 6)
	discounted := mage.NewInstant("Discounted target", "{3}", mage.NewSpell(mage.DrawCards(mage.Fixed(1))))
	for _, target := range []*mage.Card{
		mage.NewInstant("Full price target", "{3}", mage.NewSpell(mage.DrawCards(mage.Fixed(5)))),
		discounted,
	} {
		b.AddToHand(target)
		if err := g.CastSpellByID(b.PlayerID(), target.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
	}
	spell := mage.NewInstant("Counter", "{2}{X}{U}",
		mage.NewTargetedSpell(mage.TargetSpellOnStack(mage.ManaValueIs(mage.EqualTo(mage.XValue()))), mage.CounterSpell()),
		mage.WithTargetConditionalCostReduction(2, func(_ *mage.Game, _ uuid.UUID, _ *mage.Card, targets []uuid.UUID) bool {
			return len(targets) == 1 && targets[0] == discounted.ID()
		}))
	a.AddToHand(spell)

	s := &Strategy{Personality: ai.AggroPersonality}
	action := s.evaluateResponse(a, g)
	if action == nil || action.CardID != spell.ID() || action.XValue != 3 ||
		len(action.Targets) != 1 || action.Targets[0] != discounted.ID() {
		t.Fatalf("expected the discounted X=3 cast, got %+v", action)
	}
	if err := g.CastSpellByID(a.PlayerID(), action.CardID, action.Targets, action.XValue); err != nil {
		t.Fatal(err)
	}
}
