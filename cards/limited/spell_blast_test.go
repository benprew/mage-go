package limited

import (
	"slices"
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
)

func TestSpellBlast(t *testing.T) {
	for _, tc := range []struct {
		name string
		cost string
		x    int
		want int
	}{
		{name: "zero", cost: "{0}"},
		{name: "fixed", cost: "{3}", want: 3},
		{name: "X", cost: "{X}{R}", x: 4, want: 5},
		{name: "two X symbols", cost: "{X}{X}", x: 3, want: 6},
		{name: "hybrid", cost: "{2}{W/U}", want: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, announcement := range []struct {
				name string
				x    int
			}{
				{name: "lower", x: tc.want - 1},
				{name: "exact", x: tc.want},
				{name: "higher", x: tc.want + 1},
			} {
				if announcement.x < 0 {
					continue
				}
				t.Run(announcement.name, func(t *testing.T) {
					a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
					g := mage.NewGame(a, b)
					g.SetStep(core.PrecombatMain)
					target := mage.NewInstant("Target", tc.cost, mage.NewSpell(mage.GainLife(1)))
					a.AddToHand(target)
					for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green, core.Colorless} {
						a.ManaPool().Add(color, 20)
					}
					if err := g.CastSpellByID(a.PlayerID(), target.ID(), nil, tc.x); err != nil {
						t.Fatal(err)
					}
					spell, err := mage.CreateCard("Spell Blast")
					if err != nil {
						t.Fatal(err)
					}
					b.AddToHand(spell)
					b.ManaPool().Add(core.Blue, 20)
					g.SetXValue(9)

					err = g.CastSpellByID(b.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, announcement.x)
					if announcement.x != tc.want {
						if err == nil {
							t.Fatal("accepted an X that differs from the target mana value")
						}
						if len(b.Hand()) != 1 || b.ManaPool().TotalMana() != 20 || len(g.StackObjects()) != 1 || g.XValue() != 9 {
							t.Fatal("rejected cast changed game state")
						}
						g.ResolveStack()
						if a.Life() != 21 {
							t.Fatal("rejected cast stopped the target from resolving")
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					if got := b.ManaPool().TotalMana(); got != 19-tc.want {
						t.Fatalf("mana after cast = %d, want %d", got, 19-tc.want)
					}
					g.SetXValue(0)
					g.ResolveStack()
					if a.Life() != 20 || len(a.Graveyard()) != 1 || len(b.Graveyard()) != 1 {
						t.Fatal("Spell Blast did not counter its target")
					}
				})
			}
		})
	}
}

func TestSpellBlastTargetOptions(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	a.ManaPool().Add(core.Red, 30)
	targets := make([]*mage.Card, 0, 4)
	for _, cost := range []string{"{0}", "{3}", "{X}{R}", "{20}"} {
		target := mage.NewInstant("Target", cost)
		a.AddToHand(target)
		if err := g.CastSpellByID(a.PlayerID(), target.ID(), nil, 4); err != nil {
			t.Fatal(err)
		}
		targets = append(targets, target)
	}
	spell, err := mage.CreateCard("Spell Blast")
	if err != nil {
		t.Fatal(err)
	}
	b.AddToHand(spell)
	b.ManaPool().Add(core.Blue, 6)
	g.PutOnBattlefield(mage.NewArtifact("Unrelated", "{2}"), a.PlayerID())
	g.SetXValue(8)

	for _, query := range []func(*mage.Game, uuid.UUID) []interactive.ActionOption{
		interactive.GetAvailableActions,
		interactive.GetAvailableActionsWithoutLabels,
	} {
		found := false
		for _, action := range query(g, b.PlayerID()) {
			if action.CardID != spell.ID() {
				continue
			}
			found = true
			if !action.NeedsX || !action.NeedsTarget || !mage.TargetsUseX(spell.CastTargets()) {
				t.Fatal("Spell Blast must use X to select its target")
			}
			if !slices.Equal(action.ValidTargets, []uuid.UUID{targets[0].ID(), targets[1].ID(), targets[2].ID()}) {
				t.Fatalf("targets = %v, want affordable stack spells", action.ValidTargets)
			}
			if len(action.XTargetOptions) != 3 {
				t.Fatalf("X target options = %+v, want three choices", action.XTargetOptions)
			}
			for i, x := range []int{0, 3, 5} {
				option := action.XTargetOptions[i]
				if option.XValue != x || len(option.Candidates) != 1 || !slices.Equal(option.Candidates[0], []uuid.UUID{targets[i].ID()}) {
					t.Fatalf("wrong X target option: %+v", option)
				}
			}
		}
		if !found {
			t.Fatal("Spell Blast is missing from available actions")
		}
	}
	if g.XValue() != 8 || b.ManaPool().TotalMana() != 6 || len(g.StackObjects()) != 4 {
		t.Fatal("action query changed game state")
	}
}
