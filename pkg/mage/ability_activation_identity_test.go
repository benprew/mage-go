package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type activationIdentityRecorder struct{ events *[]GameEvent }

func (r activationIdentityRecorder) CheckTriggerCond(event *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
	*r.events = append(*r.events, *event)
	return false
}

func TestAbilityActivationEventSourceIdentity(t *testing.T) {
	for _, tc := range []struct {
		name    string
		card    Card
		planned bool
	}{
		{"ordinary", NewArtifact("Ordinary", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1))))), false},
		{"sacrificed", NewArtifact("Sacrificed", "", WithAbility(Activate(Costs(SacrificeSourceCost()), GainLifeFor(You, Fixed(1))))), false},
		{"mana", NewArtifact("Mana", "", WithAbility(NewManaAbility(Blue))), false},
		{"planned mana", NewArtifact("Planned mana", "", WithAbility(NewManaAbility(Blue))), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			var events []GameEvent
			watcher := NewEnchantment("Watcher", "", WithAbility(NewTriggered(EvtAbilityActivated, false, GainLifeFor(You, Fixed(1))).SetConditionData(activationIdentityRecorder{&events})))
			g.PutOnBattlefield(watcher, b.PlayerID())
			g.PutOnBattlefield(tc.card, a.PlayerID())
			old := g.FindPermanent(tc.card.ID()).incarnationID
			var err error
			if tc.planned {
				err = g.AutoTapForCost(a.PlayerID(), ManaCost{Blue: 1})
			} else {
				err = g.ActivateAbilityByIndex(a.PlayerID(), tc.card.ID(), 0, nil)
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != 1 || events[0].SourceID != tc.card.ID() || events[0].SourceIncarnation != old {
				t.Fatalf("activation events = %+v, want source %s incarnation %s", events, tc.card.ID(), old)
			}
			if tc.name == "sacrificed" {
				g.PutOnBattlefield(tc.card, a.PlayerID())
				if got := g.FindPermanent(tc.card.ID()).incarnationID; got == old {
					t.Fatal("returned source retained its old incarnation")
				}
				if events[0].SourceIncarnation != old {
					t.Fatal("event identity followed the returned card")
				}
			}
		})
	}
}
