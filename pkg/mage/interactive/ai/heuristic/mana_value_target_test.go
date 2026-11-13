package heuristic

import (
	"fmt"
	"testing"

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
