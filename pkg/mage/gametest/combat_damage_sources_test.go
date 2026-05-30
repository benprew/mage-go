package gametest

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// CombatDamageSourcesThisStep exposes per-source attribution for each
// EvtCombatDamageDealt fire. Trigger condition closures use it to filter on
// source attributes (e.g., "non-Human creatures you control" — Keeper of
// Fables). Two attackers connect; only the non-Human contributes, so the
// trigger draws exactly one card.
func TestCombatDamageSourcesThisStep_NonHumanFilter(t *testing.T) {
	const cardName = "CDS Keeper-like"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			cond := func(evt *core.GameEvent, g mage.GameReader, sourceID, controllerID uuid.UUID) bool {
				if evt.PlayerID != controllerID {
					return false
				}
				bySrc := g.CombatDamageSourcesThisStep(evt.PlayerID, evt.TargetID)
				for srcID := range bySrc {
					perm := g.FindPermanent(srcID)
					if perm == nil {
						continue
					}
					if !perm.HasSubType("Human") {
						return true
					}
				}
				return false
			}
			return mage.NewCreature(cardName, "{3}{G}{G}", 4, 5,
				mage.WithSubTypes("Cat"),
				mage.WithAbility(mage.NewTriggered(core.EvtCombatDamageDealt, false,
					mage.DrawCards(mage.Fixed(1)),
				).SetCondition(cond)),
			)
		})
	}

	mage.Register("CDS Human Soldier", func() mage.Card {
		return mage.NewCreature("CDS Human Soldier", "{1}{W}", 2, 2,
			mage.WithSubTypes("Human", "Soldier"))
	})
	mage.Register("CDS Wolf", func() mage.Card {
		return mage.NewCreature("CDS Wolf", "{1}{G}", 2, 2,
			mage.WithSubTypes("Wolf"))
	})

	// Two attackers, only one is non-Human; trigger fires once and draws one card.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "CDS Human Soldier")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "CDS Wolf")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Mountain", 5)
	tg.Attack(3, PlayerA, "CDS Human Soldier", "CDS Wolf")
	tg.StopAt(3, core.EndStep)
	tg.Execute()
	tg.AssertHandCount(PlayerA, "Mountain", 1)

	// Now: only Human attacker connects; trigger should not fire.
	tg2 := NewTestGame(t)
	tg2.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg2.AddCard(core.ZoneBattlefield, PlayerA, "CDS Human Soldier")
	tg2.AddCard(core.ZoneLibrary, PlayerA, "Mountain", 5)
	tg2.Attack(3, PlayerA, "CDS Human Soldier")
	tg2.StopAt(3, core.EndStep)
	tg2.Execute()
	tg2.AssertHandCount(PlayerA, "Mountain", 0)
}
