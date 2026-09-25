package dsl_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestCombatBoostTypedDefinitions(t *testing.T) {
	for _, tc := range []struct {
		name             string
		power, toughness int
	}{{"Army of Allah", 2, 0}, {"Piety", 0, 3}, {"Morale", 1, 1}} {
		t.Run(tc.name, func(t *testing.T) {
			card, err := mage.CreateCard(tc.name)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(tc.name, card); err != nil {
				t.Fatal(err)
			}
			ability := mage.UnwrapAbility(card.Abilities()[0]).(*mage.SpellAbility)
			if len(ability.Targets()) != 0 {
				t.Fatal("combat collection must not target")
			}
			metadata := mage.InspectDefinition(ability.Effects()[0])
			if metadata.Role != "until" {
				t.Fatalf("effect role = %q, want until", metadata.Role)
			}
			props := metadata.Properties
			if !props.Mass || props.Outcome != mage.OutcomeBenefit || props.PowerBoost != tc.power || props.ToughnessBoost != tc.toughness {
				t.Fatalf("combat boost metadata = %+v", props)
			}
		})
	}
}

func TestCombatBoostResolutionMembership(t *testing.T) {
	for _, tc := range []struct {
		name             string
		power, toughness int
		blocking         bool
	}{{"Army of Allah", 2, 0, false}, {"Piety", 0, 3, true}, {"Morale", 1, 1, false}} {
		for _, opponentCasts := range []bool{false, true} {
			label := tc.name + "/combatant casts"
			if opponentCasts {
				label = tc.name + "/opponent casts"
			}
			t.Run(label, func(t *testing.T) {
				a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				caster := a.PlayerID()
				if opponentCasts {
					caster = b.PlayerID()
				}
				card, err := mage.CreateCard(tc.name)
				if err != nil {
					t.Fatal(err)
				}
				card.SetOwner(caster)
				makeCreature := func(name string) *mage.Permanent {
					c := mage.NewCreature(name, "", 2, 2)
					c.SetOwner(a.PlayerID())
					return g.PutOnBattlefield(c, a.PlayerID())
				}
				joinCombat := func(p *mage.Permanent) {
					if tc.blocking {
						attacker := uuid.New()
						g.GetCombat().AddAttacker(attacker, a.PlayerID())
						g.GetCombat().AddBlocker(p.ID(), attacker)
					} else {
						g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
					}
				}
				member := makeCreature("Member")
				removed := makeCreature("Removed before resolution")
				later := makeCreature("Joins after resolution")
				joinCombat(member)
				joinCombat(removed)
				ability := mage.UnwrapAbility(card.Abilities()[0]).(*mage.SpellAbility)
				g.PushStack(&mage.StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: caster, Effects: ability.Effects()})
				g.RemoveFromCombat(removed.ID())
				g.ResolveStack()
				joinCombat(later)
				g.RemoveFromCombat(member.ID())
				check := func(game *mage.Game, id uuid.UUID, power, toughness int) {
					t.Helper()
					game.ApplyContinuousEffects()
					p := game.FindPermanent(id)
					if p.CurrentPower(game) != power || p.CurrentToughness(game) != toughness {
						t.Fatalf("%s P/T = %d/%d, want %d/%d", p.Card.Name(), p.CurrentPower(game), p.CurrentToughness(game), power, toughness)
					}
				}
				check(g, member.ID(), 2+tc.power, 2+tc.toughness)
				check(g, removed.ID(), 2, 2)
				check(g, later.ID(), 2, 2)
				clone := g.Clone()
				clone.GetEffects().RemoveEndOfTurn()
				check(clone, member.ID(), 2, 2)
				check(g, member.ID(), 2+tc.power, 2+tc.toughness)
				g.RemoveFromBattlefield(g.FindPermanent(member.ID()))
				g.PutOnBattlefield(member.Card, a.PlayerID())
				check(g, member.ID(), 2, 2)
			})
		}
	}
}
