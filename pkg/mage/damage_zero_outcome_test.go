package mage

import (
	"fmt"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type damageOutcomeReplacement struct {
	replacementBase
	amount   int
	onlyZero bool
	calls    int
}

func (r *damageOutcomeReplacement) Matches(a Action, _ GameReader) bool {
	switch a := a.(type) {
	case *DamageToPlayerAction:
		return !r.onlyZero || a.Amount() <= 0
	case *DamageToCreatureAction:
		return !r.onlyZero || a.Amount() <= 0
	}
	return false
}
func (*damageOutcomeReplacement) IsActive(GameReader) bool   { return true }
func (r *damageOutcomeReplacement) Clone() ReplacementEffect { c := *r; return &c }
func (r *damageOutcomeReplacement) Replace(a Action, _ *Game) Action {
	r.calls++
	switch a := a.(type) {
	case *DamageToPlayerAction:
		return a.WithAmount(r.amount)
	case *DamageToCreatureAction:
		return a.WithAmount(r.amount)
	}
	return a
}

func TestDamageNonpositiveOutcome(t *testing.T) {
	for _, recipient := range []string{"player", "creature"} {
		for _, path := range []string{"executor", "replacement", "replacement-recheck"} {
			for _, amount := range []int{0, -2} {
				t.Run(fmt.Sprintf("%s/%s/%d", recipient, path, amount), func(t *testing.T) {
					a, b := NewBasePlayer("A"), NewBasePlayer("B")
					g := NewGame(a, b)
					source := g.PutOnBattlefield(NewCreature("Source", "", 2, 3, WithKeyword(Lifelink), WithKeyword(Deathtouch),
						WithAbility(WhenEvent(DealsDamage(This, BindNumber("damage")), DrawFor(You, Fixed(1))))), a.PlayerID())
					victim := g.PutOnBattlefield(NewCreature("Victim", "", 0, 5), b.PlayerID())
					victim.Damage = 1
					source.FaceDown = true
					callbacks := 0
					g.damage.SetOnDamageDealt(func(string, string, int, bool) { callbacks++ })
					g.damage.SetDamageReflection(b.PlayerID(), source.ID(), source.ID())
					g.damage.SetResolvingCombatDamage(true)
					increase := &damageOutcomeReplacement{amount: 3, onlyZero: true}
					if path != "executor" {
						g.effects.AddReplacement(&damageOutcomeReplacement{amount: amount})
						if path == "replacement-recheck" {
							g.effects.AddReplacement(increase)
						}
					}
					if recipient == "player" {
						if path == "executor" {
							g.damage.ExecuteDamageToPlayer(g, NewDamageToPlayerAction(source.ID(), b.PlayerID(), amount, true))
						} else {
							g.DealDamageToPlayer(b, 2, source.ID())
						}
					} else {
						if path == "executor" {
							g.damage.ExecuteDamageToCreature(g, NewDamageToCreatureAction(source.ID(), victim.ID(), amount, true))
						} else {
							g.DealDamageToPermanent(victim, 2, source.ID())
						}
					}
					g.damage.FlushCombatDamageAggregator(g)
					if a.Life() != 20 || b.Life() != 20 || g.FindPermanent(victim.ID()).Damage != 1 {
						t.Errorf("nonpositive damage changed life or marked damage: %d, %d, %d", a.Life(), b.Life(), g.FindPermanent(victim.ID()).Damage)
					}
					if callbacks != 0 || len(g.triggers.Pending()) != 0 {
						t.Error("nonpositive damage emitted a callback or trigger")
					}
					if g.HasDealtDamageToPlayer(source.ID(), b.PlayerID()) || g.HasDealtDamageToPermanent(source.ID(), victim.ID()) || len(g.GetDamageSources(victim.ID())) != 0 {
						t.Error("nonpositive damage entered damage history")
					}
					if g.trackers.Turn.DamageTaken(b.PlayerID()) != 0 || g.PlayerLifeGainedThisTurn(a.PlayerID()) != 0 {
						t.Error("nonpositive damage changed turn totals")
					}
					if !g.FindPermanent(source.ID()).FaceDown {
						t.Error("nonpositive damage turned the source face up")
					}
					if _, ok := g.damage.GetDamageReflection(b.PlayerID()); !ok {
						t.Error("nonpositive damage consumed reflection")
					}
					if increase.calls != 0 {
						t.Error("later replacement increased an absent damage event")
					}
				})
			}
		}
	}
}
