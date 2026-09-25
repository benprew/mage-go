package mage

import (
	"fmt"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTemporaryPTMetadata(t *testing.T) {
	for _, delta := range [][2]int{{0, 0}, {2, 1}, {-1, 0}, {0, -2}, {-4, -4}, {2, -1}, {-1, 2}} {
		for _, mass := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/%d/mass=%t", delta[0], delta[1], mass), func(t *testing.T) {
				modification := ModifyPT(This, Fixed(delta[0]), Fixed(delta[1]))
				if mass {
					modification = ModifyEachPT(MatchingPermanents(IsCreature), Fixed(delta[0]), Fixed(delta[1]))
				}
				want := OutcomeBenefit
				if delta[0] < 0 || delta[1] < 0 {
					want = OutcomeDetriment
				}
				for _, body := range []ResolvingEffect{Until(EndOfTurn, modification), UntilSourceLeaves(This, modification)} {
					props := InspectDefinition(body).Properties
					if props.Outcome != want || props.Mass != mass || props.PowerBoost != delta[0] || props.ToughnessBoost != delta[1] {
						t.Fatalf("P/T metadata = %+v, want outcome %v and delta %v", props, want, delta)
					}
				}
			})
		}
	}
}
