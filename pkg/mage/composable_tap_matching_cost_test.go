package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type matchingTapChoicePlayer struct {
	*BasePlayer
	choices int
}

func (p *matchingTapChoicePlayer) ChoosePermanent(candidates []*Permanent, _ string, _ GameReader) *Permanent {
	p.choices++
	return candidates[len(candidates)-1]
}

func TestTapNOtherMatchingCostPaysDistinctWhiteCreatures(t *testing.T) {
	a := &matchingTapChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "{W}", 2, 6, WithKeyword(core.Haste))
	g.PutOnBattlefield(source, a.PlayerID())
	chosen := make([]Card, 3)
	for i := range chosen {
		chosen[i] = NewCreature("White", "{W}", 1, 1)
		g.PutOnBattlefield(chosen[i], a.PlayerID())
	}
	blue := NewCreature("Blue", "{U}", 1, 1)
	g.PutOnBattlefield(blue, a.PlayerID())
	opponent := NewCreature("Opponent", "{W}", 1, 1)
	g.PutOnBattlefield(opponent, b.PlayerID())
	cost := TapNOtherMatchingCost(3, And(IsCreature, HasColorFilter(core.White)), "Tap three white creatures")
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{Tap(), cost}})
	if err != nil {
		t.Fatal(err)
	}
	if a.choices != 3 {
		t.Fatalf("choices = %d, want 3", a.choices)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(source.ID()).Tapped {
		t.Fatal("source tap cost was not paid")
	}
	for _, card := range chosen {
		if !g.FindPermanent(card.ID()).Tapped {
			t.Fatalf("%s was not tapped", card.Name())
		}
	}
	if g.FindPermanent(blue.ID()).Tapped || g.FindPermanent(opponent.ID()).Tapped || a.choices != 3 {
		t.Fatal("payment used an ineligible object or chose again at commit")
	}
}

func TestTapNOtherMatchingCostRejectsChangedChoiceWithoutPartialPayment(t *testing.T) {
	for _, change := range []string{"tap", "phase", "return", "color"} {
		t.Run(change, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			source := NewArtifact("Source", "")
			g.PutOnBattlefield(source, a.PlayerID())
			cards := make([]Card, 3)
			for i := range cards {
				cards[i] = NewCreature("White", "{W}", 1, 1)
				g.PutOnBattlefield(cards[i], a.PlayerID())
			}
			cost := TapNOtherMatchingCost(3, And(IsCreature, HasColorFilter(core.White)), "Tap three white creatures")
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{cost}})
			if err != nil {
				t.Fatal(err)
			}
			changed := cards[0]
			switch change {
			case "tap":
				g.TapPermanent(g.FindPermanent(changed.ID()))
			case "phase":
				g.PhaseOut(changed.ID())
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(changed.ID()))
				g.PutOnBattlefield(changed, a.PlayerID())
			case "color":
				colors := []core.Color{core.Blue}
				g.MutablePermanent(changed.ID()).ColorOverride = &colors
			}
			clone := g.Clone()
			if err := tx.Commit(); err == nil {
				t.Fatal("stale payment succeeded")
			}
			for _, card := range cards[1:] {
				if g.FindPermanent(card.ID()).Tapped || clone.FindPermanent(card.ID()).Tapped {
					t.Fatal("failed payment tapped another creature or changed clone")
				}
			}
		})
	}
}

func TestTapNOtherMatchingCostRegistration(t *testing.T) {
	for _, tc := range []struct {
		count      int
		text, want string
	}{
		{3, "Tap three white creatures", ""},
		{0, "Tap", "positive"},
		{-1, "Tap", "positive"},
		{3, "", "text"},
	} {
		card := NewArtifact("Synthetic", "", WithAbility(Activate(
			Costs(TapNOtherMatchingCost(tc.count, IsCreature, tc.text)),
			GainLifeFor(You, Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("count %d, text %q: validation = %v, want %q", tc.count, tc.text, err, tc.want)
		}
	}
}

func TestTapNOtherMatchingCostEventsSeeCompleteTap(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	source := NewArtifact("Source", "")
	g.PutOnBattlefield(source, a.PlayerID())
	cards := make([]Card, 3)
	for i := range cards {
		cards[i] = NewCreature("White", "{W}", 1, 1)
		g.PutOnBattlefield(cards[i], a.PlayerID())
	}
	events, incomplete := 0, false
	watcher := NewEnchantment("Watcher", "", WithAbility(NewTriggered(core.EvtTapped, false,
		GainLifeFor(You, Fixed(1))).SetCondition(func(_ *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		events++
		for _, card := range cards {
			if !reader.FindPermanent(card.ID()).Tapped {
				incomplete = true
			}
		}
		return false
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{
		TapNOtherMatchingCost(3, And(IsCreature, HasColorFilter(core.White)), "Tap three white creatures"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	events, incomplete = 0, false
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if events != 3 || incomplete {
		t.Fatalf("events = %d, incomplete = %t", events, incomplete)
	}
}

func TestTapNOtherMatchingCostInsufficientAndRollback(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	source := NewArtifact("Source", "")
	g.PutOnBattlefield(source, a.PlayerID())
	cards := make([]Card, 3)
	for i := range cards {
		cards[i] = NewCreature("White", "{W}", 1, 1)
		g.PutOnBattlefield(cards[i], a.PlayerID())
	}
	cost := TapNOtherMatchingCost(3, And(IsCreature, HasColorFilter(core.White)), "Tap three white creatures")
	g.TapPermanent(g.FindPermanent(cards[0].ID()))
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{LifePayCost(2), cost}}); err == nil {
		t.Fatal("payment with only two eligible creatures succeeded")
	}
	if a.Life() != 20 || g.FindPermanent(cards[1].ID()).Tapped || g.FindPermanent(cards[2].ID()).Tapped {
		t.Fatal("failed preparation changed live payment state")
	}
	g.MutablePermanent(cards[0].ID()).Tapped = false
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{LifePayCost(2), cost}})
	if err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(cards[0].ID())
	if err := tx.Commit(); err == nil || a.Life() != 20 {
		t.Fatalf("changed choice paid life: %v", err)
	}
}
