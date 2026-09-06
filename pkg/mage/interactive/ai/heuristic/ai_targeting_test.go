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

// Swords to Plowshares exiles, so it bypasses indestructibility: it must still
// target the opponent's biggest threat even when that threat is indestructible,
// unlike destroy-based removal which would avoid it.
func TestAutoSelectTargets_SwordsToPlowshares_TargetsIndestructible(t *testing.T) {
	g, pa, pb := makeGame()
	small := makePerm("Goblin", "{R}", 1, 1, pb.PlayerID())
	big := makePerm("Avatar", "{4}{R}{R}", 5, 5, pb.PlayerID(), mage.WithKeyword(core.Indestructible))
	g.AddToBattlefield(small, big)

	card := createCard(t, "Swords to Plowshares", pa.PlayerID())
	pa.AddToHand(card)

	s := New(ai.MidrangeWeighted)
	targets := s.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != big.ID() {
		t.Fatalf("Swords to Plowshares should exile the opponent's indestructible threat (Avatar), got %v", targets)
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

// A counterspell must recognize an opponent's spell on the stack as a target
// and be cast in response, picking the opposing spell (never our own).
func TestPriorityAction_CountersOpponentSpell(t *testing.T) {
	g, pa, pb := makeGame()
	addLands(g, pa, "Island", 2)

	counter := createCard(t, "Counterspell", pa.PlayerID())
	pa.AddToHand(counter)

	// Opponent's creature spell is on the stack.
	dragon := createCard(t, "Grizzly Bears", pb.PlayerID())
	g.PushStack(&mage.StackObject{
		ID:         uuid.New(),
		Card:       dragon,
		Controller: pb.PlayerID(),
		SourceID:   dragon.ID(),
	})

	s := New(ai.ControlWeighted)
	action := s.PriorityAction(pa, g, 0, false)
	if action.Type != interactive.ActionCastSpell || action.CardName != "Counterspell" {
		t.Fatalf("expected Counterspell to be cast in response, got %+v", action)
	}
	if len(action.Targets) != 1 || action.Targets[0] != dragon.ID() {
		t.Fatalf("expected Counterspell to target the opposing spell %v, got %v", dragon.ID(), action.Targets)
	}
}

// A counterspell selects the highest-value opposing spell when several are on
// the stack, and never targets our own spell.
func TestAutoSelectTargets_Counterspell_PicksBestOpposingSpell(t *testing.T) {
	g, pa, pb := makeGame()
	addLands(g, pa, "Island", 2)

	counter := createCard(t, "Counterspell", pa.PlayerID())
	pa.AddToHand(counter)

	mine := createCard(t, "Grizzly Bears", pa.PlayerID())
	cheap := createCard(t, "Grizzly Bears", pb.PlayerID())
	bomb := createCard(t, "Shivan Dragon", pb.PlayerID())
	for _, sp := range []struct {
		card mage.Card
		ctrl uuid.UUID
	}{{mine, pa.PlayerID()}, {cheap, pb.PlayerID()}, {bomb, pb.PlayerID()}} {
		g.PushStack(&mage.StackObject{
			ID:         uuid.New(),
			Card:       sp.card,
			Controller: sp.ctrl,
			SourceID:   sp.card.ID(),
		})
	}

	s := New(ai.ControlWeighted)
	targets := s.autoSelectTargets(pa, g, counter)
	if len(targets) != 1 || targets[0] != bomb.ID() {
		t.Fatalf("Counterspell should target the opponent's highest-value spell (Shivan Dragon), got %v", targets)
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

// Combat tricks (pump spells) should never be cast proactively in PostcombatMain
// where their temporary combat buff expires without effect.
func TestPriorityAction_DoesNotCastCombatTrickInPostcombatMain(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetActivePlayerIndex(0)

	creature := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(creature)
	addLands(g, pa, "Forest", 1)

	card := createCard(t, "Giant Growth", pa.PlayerID())
	pa.AddToHand(card)

	s := New(ai.AggroWeighted)

	g.SetStep(core.PostcombatMain)
	act := s.PriorityAction(pa, g, 0, true)
	if act.Type == interactive.ActionCastSpell && act.CardName == "Giant Growth" {
		t.Fatalf("Giant Growth should NOT be cast in PostcombatMain, but it was cast")
	}
	if act.Type != interactive.ActionPass {
		t.Fatalf("expected ActionPass in PostcombatMain, got %+v", act)
	}
}

// Combat tricks (pump spells) should never be cast in non-combat steps like EndStep.
func TestPriorityAction_DoesNotCastCombatTrickInEndStep(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetActivePlayerIndex(0)

	creature := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(creature)
	addLands(g, pa, "Forest", 1)

	card := createCard(t, "Giant Growth", pa.PlayerID())
	pa.AddToHand(card)

	s := New(ai.AggroWeighted)

	g.SetStep(core.EndStep)
	act := s.PriorityAction(pa, g, 0, false)
	if act.Type == interactive.ActionCastSpell && act.CardName == "Giant Growth" {
		t.Fatalf("Giant Growth should NOT be cast in EndStep, but it was cast")
	}
	if act.Type != interactive.ActionPass {
		t.Fatalf("expected ActionPass in EndStep, got %+v", act)
	}
}

// When an opponent threat is on the stack, Giant Growth should be cast to save the creature even in EndStep.
func TestPriorityAction_CastsGiantGrowthAgainstRemovalInEndStep(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetActivePlayerIndex(1)

	creature := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(creature)
	addLands(g, pa, "Forest", 1)

	card := createCard(t, "Giant Growth", pa.PlayerID())
	pa.AddToHand(card)

	bolt := createCard(t, "Lightning Bolt", pb.PlayerID())
	g.PushStack(&mage.StackObject{
		ID:         uuid.New(),
		Card:       bolt,
		Controller: pb.PlayerID(),
		SourceID:   bolt.ID(),
		Targets:    []uuid.UUID{creature.ID()},
	})

	s := New(ai.AggroWeighted)

	g.SetStep(core.EndStep)
	act := s.PriorityAction(pa, g, 0, false)
	if act.Type != interactive.ActionCastSpell || act.CardName != "Giant Growth" {
		t.Fatalf("expected Giant Growth to be cast in response to Lightning Bolt, got %+v", act)
	}
	if len(act.Targets) != 1 || act.Targets[0] != creature.ID() {
		t.Fatalf("Giant Growth should target the threatened creature, got %v", act.Targets)
	}
}
