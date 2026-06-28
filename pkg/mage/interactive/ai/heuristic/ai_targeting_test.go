package heuristic

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"

	_ "github.com/benprew/mage-go/cards" // register all card sets
)

func createCard(t *testing.T, name string, owner uuid.UUID) mage.Card {
	t.Helper()
	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard %q: %v", name, err)
	}
	card.SetOwner(owner)
	return card
}

// Stream of Life is a positive effect, so the AI should gain the life itself
// rather than handing it to the opponent.
func TestAutoSelectTargets_StreamOfLife_TargetsSelf(t *testing.T) {
	g, pa, _ := makeGame()
	card := createCard(t, "Stream of Life", pa.PlayerID())
	pa.AddToHand(card)

	s := New(ai.MidrangeWeighted)
	targets := s.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != pa.PlayerID() {
		t.Fatalf("Stream of Life should target self, got %v", targets)
	}
}

// Control Magic should steal the opponent's most valuable creature, never one
// of our own.
func TestAutoSelectTargets_ControlMagic_OpponentBest(t *testing.T) {
	g, pa, pb := makeGame()
	own := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	small := makePerm("Goblin", "{R}", 1, 1, pb.PlayerID())
	big := makePerm("Dragon", "{4}{R}{R}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(own, small, big)

	card := createCard(t, "Control Magic", pa.PlayerID())
	pa.AddToHand(card)

	s := New(ai.MidrangeWeighted)
	targets := s.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != big.ID() {
		t.Fatalf("Control Magic should steal opponent's best creature (Dragon), got %v", targets)
	}
}

// Swords to Plowshares is removal: it must hit the opponent's best creature.
func TestAutoSelectTargets_SwordsToPlowshares_OpponentBest(t *testing.T) {
	g, pa, pb := makeGame()
	own := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	small := makePerm("Goblin", "{R}", 1, 1, pb.PlayerID())
	big := makePerm("Dragon", "{4}{R}{R}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(own, small, big)

	card := createCard(t, "Swords to Plowshares", pa.PlayerID())
	pa.AddToHand(card)

	s := New(ai.MidrangeWeighted)
	targets := s.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != big.ID() {
		t.Fatalf("Swords to Plowshares should hit opponent's best creature (Dragon), got %v", targets)
	}
}

// Weakness (-2/-1) is a debuff and should be aimed at the opponent's creature,
// not our own.
func TestAutoSelectTargets_Weakness_OpponentCreature(t *testing.T) {
	g, pa, pb := makeGame()
	own := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	opp := makePerm("Ogre", "{2}{R}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(own, opp)

	card := createCard(t, "Weakness", pa.PlayerID())
	pa.AddToHand(card)

	s := New(ai.MidrangeWeighted)
	targets := s.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != opp.ID() {
		t.Fatalf("Weakness should target opponent creature, got %v", targets)
	}
}

// Immolation (+2/-2) only helps us when it kills, so it must target an opponent
// creature with toughness <= 2, not a larger one it would merely buff.
func TestAutoSelectTargets_Immolation_KillsLowToughness(t *testing.T) {
	g, pa, pb := makeGame()
	killable := makePerm("Lion", "{1}{W}", 3, 2, pb.PlayerID())
	tooBig := makePerm("Dragon", "{4}{R}{R}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(killable, tooBig)

	card := createCard(t, "Immolation", pa.PlayerID())
	pa.AddToHand(card)

	s := New(ai.MidrangeWeighted)
	targets := s.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != killable.ID() {
		t.Fatalf("Immolation should target the killable (toughness 2) creature, got %v", targets)
	}
}

// Immolation should not be cast (no valid target) when no opponent creature has
// toughness it can reduce to zero — buffing a survivor only helps the opponent.
func TestAutoSelectTargets_Immolation_NoTargetWhenNoneKillable(t *testing.T) {
	g, pa, pb := makeGame()
	own := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	tooBig := makePerm("Dragon", "{4}{R}{R}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(own, tooBig)

	card := createCard(t, "Immolation", pa.PlayerID())
	pa.AddToHand(card)

	s := New(ai.MidrangeWeighted)
	targets := s.autoSelectTargets(pa, g, card)
	if len(targets) != 0 {
		t.Fatalf("Immolation should have no target when nothing is killable, got %v", targets)
	}
}

// Giant Growth and other pump tricks should be held through the declare-
// attackers window and cast after blockers are declared.
func TestPriorityAction_HoldsGiantGrowthUntilBlockers(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetActivePlayerIndex(0)
	_ = pb

	attacker := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(attacker)
	addLands(g, pa, "Forest", 1)

	card := createCard(t, "Giant Growth", pa.PlayerID())
	pa.AddToHand(card)

	s := New(ai.AggroWeighted)

	g.SetStep(core.DeclareAttackers)
	act := s.PriorityAction(pa, g, 0, false)
	if act.Type == interactive.ActionCastSpell && act.CardName == "Giant Growth" {
		t.Fatalf("Giant Growth should be held during declare-attackers, but it was cast")
	}

	g.SetStep(core.DeclareBlockers)
	act = s.PriorityAction(pa, g, 0, false)
	if act.Type != interactive.ActionCastSpell || act.CardName != "Giant Growth" {
		t.Fatalf("Giant Growth should be cast after blockers are declared, got %+v", act)
	}
	if len(act.Targets) != 1 || act.Targets[0] != attacker.ID() {
		t.Fatalf("Giant Growth should target our own creature, got %v", act.Targets)
	}
}
