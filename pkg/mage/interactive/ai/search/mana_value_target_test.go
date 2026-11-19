package search

import (
	"fmt"
	"testing"
	"uuid"

	_ "github.com/benprew/mage-go/cards/antiquities"
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestManaValueTargetMoves(t *testing.T) {
	for _, generate := range []struct {
		name  string
		moves func(*mage.Game, mage.Player, *mage.Card) []Move
	}{
		{"search", func(g *mage.Game, p mage.Player, card *mage.Card) []Move { return expandSpellMoves(p, g, card) }},
		{"unified", expandSpell},
	} {
		t.Run(generate.name, func(t *testing.T) {
			g, a, b := makeGame()
			a.ManaPool().Add(core.Red, 13)
			for _, cost := range []string{"{0}", "{3}", "{9}", "{14}"} {
				g.PutOnBattlefield(mage.NewArtifact(cost, cost), b.PlayerID())
			}
			spell := mage.NewSorcery("Composed removal", "{X}{R}", mage.NewTargetedSpell(mage.TargetPermanent(mage.IsCardType(core.TypeArtifact), mage.ManaValueIs(mage.EqualTo(mage.XValue()))), mage.DestroyTarget()))
			g.SetXValue(7)
			moves := generate.moves(g, a, spell)
			seen := map[int]bool{}
			for _, move := range moves {
				if len(move.Targets) != 1 {
					t.Fatalf("missing target: %+v", move)
				}
				target := g.FindPermanent(move.Targets[0])
				if target.ManaValue() != move.XValue || move.XValue > 12 {
					t.Fatalf("illegal pair: %+v", move)
				}
				seen[move.XValue] = true
			}
			for _, x := range []int{0, 3, 9} {
				if !seen[x] {
					t.Errorf("missing X=%d", x)
				}
			}
			if g.XValue() != 7 {
				t.Fatal("move generation changed resolution X")
			}
		})
	}
}

func TestManaValueTargetZeroWithOnlyBaseMana(t *testing.T) {
	g, a, b := makeGame()
	a.ManaPool().Add(core.Red, 1)
	g.PutOnBattlefield(mage.NewArtifact("Zero", "{0}"), b.PlayerID())
	spell := mage.NewSorcery("Removal", "{X}{R}", mage.NewTargetedSpell(mage.TargetArtifactWithManaValueX(), mage.DestroyTarget()))
	if moves := expandSpellMoves(a, g, spell); len(moves) != 1 || moves[0].XValue != 0 {
		t.Fatalf("missing X=0: %+v", moves)
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
			moves := expandSpellMoves(a, g, spell)
			if len(moves) != 1 || moves[0].XValue != x || len(moves[0].Targets) != 1 || moves[0].Targets[0] != victim.ID() {
				t.Fatalf("wrong Detonate moves: %+v", moves)
			}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), moves[0].Targets, x); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(victim.ID()) != nil || b.Life() != 20-x {
				t.Fatal("Detonate did not resolve correctly")
			}
		})
	}
}

func TestSpellBlastMoves(t *testing.T) {
	for _, generate := range []struct {
		name  string
		moves func(*mage.Game, mage.Player, *mage.Card) []Move
	}{
		{"search", func(g *mage.Game, p mage.Player, card *mage.Card) []Move { return expandSpellMoves(p, g, card) }},
		{"unified", expandSpell},
	} {
		t.Run(generate.name, func(t *testing.T) {
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
					moves := generate.moves(g, a, spell)
					if len(moves) != 1 || moves[0].XValue != x || len(moves[0].Targets) != 1 || moves[0].Targets[0] != target.ID() {
						t.Fatalf("wrong Spell Blast moves: %+v", moves)
					}
					if err := g.CastSpellByID(a.PlayerID(), spell.ID(), moves[0].Targets, moves[0].XValue); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
					if a.Life() != 20 {
						t.Fatal("Spell Blast did not counter the threat")
					}
				})
			}
		})
	}
}

func TestXCounterMovesUseAffordableTargets(t *testing.T) {
	g, a, b := makeGame()
	g.SetStep(core.PrecombatMain)
	g.SetChannelActive(a.PlayerID())
	a.SetLife(5)
	a.ManaPool().Add(core.Blue, 1)
	b.ManaPool().Add(core.Red, 6)
	discounted := mage.NewInstant("Discounted target", "{3}")
	for _, target := range []*mage.Card{mage.NewInstant("Full price target", "{3}"), discounted} {
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

	for _, tc := range []struct {
		name  string
		moves []Move
	}{
		{"search", expandSpellMoves(a, g, spell)},
		{"unified", expandSpell(g, a, spell)},
	} {
		if len(tc.moves) != 1 || tc.moves[0].XValue != 3 ||
			len(tc.moves[0].Targets) != 1 || tc.moves[0].Targets[0] != discounted.ID() {
			t.Errorf("%s moves = %+v, want the discounted X=3 cast", tc.name, tc.moves)
		}
	}
}
