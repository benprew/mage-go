package mage

import (
	"testing"

	"github.com/google/uuid"
)

type damageDuringLifeGain struct {
	replacementBase
	playerID uuid.UUID
	fired    bool
}

func (r *damageDuringLifeGain) Matches(a Action, _ GameReader) bool {
	_, ok := a.(*LifeGainAction)
	return ok && !r.fired
}

func (*damageDuringLifeGain) IsActive(GameReader) bool   { return true }
func (r *damageDuringLifeGain) Clone() ReplacementEffect { c := *r; return &c }
func (r *damageDuringLifeGain) Replace(a Action, g *Game) Action {
	r.fired = true
	g.DealDamageToPlayer(g.GetPlayer(r.playerID), 1, r.sourceID)
	return a
}

func TestReverseDamageShieldConsumedBeforeLifeGainReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Damage source", "", 2, 2), b.PlayerID())
	g.AddReverseDamageShield(a.PlayerID(), source.ID())
	g.effects.AddReplacement(&damageDuringLifeGain{
		replacementBase: replacementBase{sourceID: source.ID()}, playerID: a.PlayerID(),
	})
	g.DealDamageToPlayer(a, 2, source.ID())
	if a.Life() != 21 || g.PlayerLifeGainedThisTurn(a.PlayerID()) != 2 {
		t.Error("later damage reused the shield during life-gain replacement")
	}
	if !g.HasDealtDamageToPlayer(source.ID(), a.PlayerID()) {
		t.Error("later damage was not dealt")
	}
}

func TestReplacementLifeGainOutcomes(t *testing.T) {
	for _, path := range []string{"reverse-damage", "fasting"} {
		for _, scenario := range []struct {
			name                  string
			double, replace, lich bool
			amount, want          int
		}{
			{name: "normal", want: 2},
			{name: "double", double: true, want: 4},
			{name: "replacement", replace: true, amount: 7, want: 7},
			{name: "zero", replace: true},
			{name: "negative", replace: true, amount: -1},
			{name: "lich", lich: true},
		} {
			t.Run(path+"/"+scenario.name, func(t *testing.T) {
				a := &acceptingMayPlayer{NewBasePlayer("A")}
				b := NewBasePlayer("B")
				g := NewGame(a, b)
				watcher := NewEnchantment("Watcher", "", WithAbility(WheneverYouGainLifeTrigger(
					LoseLifeFor(You, ValueAtResolution(EventAmountValue())), false)))
				if path == "fasting" {
					WithAbility(Static(SkipDrawStepForLife(2)))(watcher)
				}
				g.PutOnBattlefield(watcher, a.PlayerID())
				source := g.PutOnBattlefield(NewCreature("Damage source", "", 2, 2), b.PlayerID())
				if path == "reverse-damage" {
					g.AddReverseDamageShield(a.PlayerID(), source.ID())
				}
				if scenario.double {
					g.AddLifeGainModifier(watcher.ID(), DoubleLifeGainForController(watcher.ID(), a.PlayerID()))
				}
				if scenario.replace {
					g.effects.AddReplacement(&lifeOutcomeReplacement{amount: scenario.amount})
				}
				if scenario.lich {
					g.SetLichActive(a.PlayerID(), watcher.ID())
				}
				for range 4 {
					a.AddToLibrary(NewLand("Card"))
				}
				clone := g.Clone()
				if path == "fasting" {
					g.effects.ApplyReplacements(&beginDrawStepAction{playerID: a.PlayerID()}, g)
					g.applyPendingStepLifeGains()
				} else {
					g.DealDamageToPlayer(a, 2, source.ID())
					if g.HasDealtDamageToPlayer(source.ID(), a.PlayerID()) {
						t.Error("prevented damage entered damage history")
					}
				}
				if got := a.Life(); got != 20+scenario.want {
					t.Errorf("life = %d, want %d", got, 20+scenario.want)
				}
				if got := g.PlayerLifeGainedThisTurn(a.PlayerID()); got != scenario.want {
					t.Errorf("gain history = %d, want %d", got, scenario.want)
				}
				wantDraws := 0
				if scenario.lich {
					wantDraws = 2
				}
				if got := len(a.Hand()); got != wantDraws {
					t.Errorf("draws = %d, want %d", got, wantDraws)
				}
				g.PutTriggersOnStack()
				wantTriggers := 0
				if scenario.want > 0 {
					wantTriggers = 1
				}
				if got := g.StackSize(); got != wantTriggers {
					t.Errorf("gain triggers = %d, want %d", got, wantTriggers)
				}
				g.ResolveStack()
				if a.Life() != 20 {
					t.Error("trigger did not receive the actual gain")
				}
				if path == "reverse-damage" {
					g.DealDamageToPlayer(a, 1, source.ID())
					wantLife := 19
					if scenario.lich {
						wantLife = 20
					}
					if a.Life() != wantLife || !g.HasDealtDamageToPlayer(source.ID(), a.PlayerID()) {
						t.Error("shield did not expire after one damage event")
					}
					clone.DealDamageToPlayer(clone.GetPlayer(a.PlayerID()), 2, source.ID())
					if clone.GetPlayer(a.PlayerID()).Life() != 20+scenario.want {
						t.Error("original consumed the cloned shield")
					}
				} else {
					clone.PutTriggersOnStack()
					if clone.GetPlayer(a.PlayerID()).Life() != 20 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 0 ||
						clone.PlayerLifeGainedThisTurn(a.PlayerID()) != 0 || clone.StackSize() != 0 {
						t.Error("replacement changed the clone")
					}
				}
			})
		}
	}
}

func TestFastingLifeGainRequiresAcceptedDrawStep(t *testing.T) {
	for _, scenario := range []string{"declined", "effect-draw", "other-player"} {
		t.Run(scenario, func(t *testing.T) {
			var a Player = &acceptingMayPlayer{NewBasePlayer("A")}
			if scenario == "declined" {
				a = NewBasePlayer("A")
			}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			g.PutOnBattlefield(NewEnchantment("Draw replacement", "", WithAbility(Static(SkipDrawStepForLife(2)))), a.PlayerID())
			playerID := a.PlayerID()
			if scenario == "other-player" {
				playerID = b.PlayerID()
			}
			var action Action = &beginDrawStepAction{playerID: playerID}
			if scenario == "effect-draw" {
				action = NewDrawCardAction(uuid.Nil, playerID, false)
			}
			if got := g.effects.ApplyReplacements(action, g); got != action {
				t.Error("replacement suppressed an ineligible or declined draw")
			}
			if a.Life() != 20 || g.PlayerLifeGainedThisTurn(a.PlayerID()) != 0 {
				t.Error("ineligible or declined draw gained life")
			}
		})
	}
}
