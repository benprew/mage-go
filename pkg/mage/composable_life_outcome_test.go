package mage

import (
	"testing"

	"github.com/google/uuid"
)

type lifeOutcomeReplacement struct {
	replacementBase
	amount  int
	consume bool
}

func (*lifeOutcomeReplacement) Matches(a Action, _ GameReader) bool {
	_, ok := a.(*LifeGainAction)
	return ok
}
func (*lifeOutcomeReplacement) IsActive(GameReader) bool   { return true }
func (r *lifeOutcomeReplacement) Clone() ReplacementEffect { cloned := *r; return &cloned }
func (r *lifeOutcomeReplacement) Replace(a Action, _ *Game) Action {
	if r.consume {
		return nil
	}
	gain := a.(*LifeGainAction)
	return NewLifeGainAction(gain.ActionSource(), gain.PlayerID(), r.amount)
}

func TestLifeGainOutcomeEvents(t *testing.T) {
	for _, adapter := range []string{"typed", "legacy-target", "variable"} {
		for _, scenario := range []struct {
			name                                   string
			requested, modifier, replacement, want int
			replace, consume, lich                 bool
		}{
			{name: "normal", requested: 3, modifier: 1, want: 3},
			{name: "double", requested: 3, modifier: 2, want: 6},
			{name: "suppressed-modifier", requested: 3, modifier: 0},
			{name: "negative-modifier", requested: 3, modifier: -1},
			{name: "zero", requested: 0, modifier: 1},
			{name: "negative", requested: -2, modifier: 1},
			{name: "replacement-amount", requested: 3, modifier: 2, replace: true, replacement: 7, want: 7},
			{name: "replacement-zero", requested: 3, modifier: 1, replace: true},
			{name: "replacement-negative", requested: 3, modifier: 1, replace: true, replacement: -1},
			{name: "consumed", requested: 3, modifier: 1, replace: true, consume: true},
			{name: "lich", requested: 3, modifier: 1, lich: true},
		} {
			t.Run(adapter+"/"+scenario.name, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				watcher := NewEnchantment("Watcher", "", WithAbility(WheneverYouGainLifeTrigger(LoseLifeFor(You, ValueAtResolution(EventAmountValue())), false)))
				g.PutOnBattlefield(watcher, b.PlayerID())
				if scenario.replace {
					g.effects.AddReplacement(&lifeOutcomeReplacement{amount: scenario.replacement, consume: scenario.consume})
				}
				if scenario.lich {
					g.SetLichActive(b.PlayerID(), watcher.ID())
					for range 3 {
						b.AddToLibrary(NewLand("Card"))
					}
				}
				g.AddLifeGainModifier(watcher.ID(), func(_ *Game, _ uuid.UUID, amount int) int { return amount * scenario.modifier })
				clone := g.Clone()
				ctx := &EffectContext{Game: clone, SourceID: watcher.ID(), Vars: make(map[string]any), Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}}
				var effect Effect
				switch adapter {
				case "typed":
					effect = GainLifeFor(ControllerOf(This), Fixed(scenario.requested))
				case "legacy-target":
					effect = GainLifeTarget(Fixed(scenario.requested))
				case "variable":
					ctx.SetInt("amount", scenario.requested)
					ctx.SetUUID("player", b.PlayerID())
					effect = GainLifeFromVar("player", "amount")
				}
				if err := effect.Apply(ctx); err != nil {
					t.Fatal(err)
				}
				if got := clone.GetPlayer(b.PlayerID()).Life(); got != 20+scenario.want {
					t.Errorf("life = %d, want %d", got, 20+scenario.want)
				}
				if got := clone.PlayerLifeGainedThisTurn(b.PlayerID()); got != scenario.want {
					t.Errorf("event amount = %d, want %d", got, scenario.want)
				}
				clone.PutTriggersOnStack()
				wantTriggers := 0
				if scenario.want > 0 {
					wantTriggers = 1
				}
				if got := clone.stack.Size(); got != wantTriggers {
					t.Errorf("triggers = %d, want %d", got, wantTriggers)
				}
				if wantTriggers == 1 && clone.stack.Size() == 1 {
					clone.ResolveStackObject(clone.stack.Pop())
					if clone.GetPlayer(b.PlayerID()).Life() != 20 {
						t.Error("trigger did not capture the actual gain amount")
					}
				}
				if scenario.lich && len(clone.GetPlayer(b.PlayerID()).Hand()) != 3 {
					t.Error("replacement did not draw three cards")
				}
				if a.Life() != 20 || b.Life() != 20 || g.PlayerLifeGainedThisTurn(b.PlayerID()) != 0 {
					t.Error("clone changed original life or tracker")
				}
			})
		}
	}
}
