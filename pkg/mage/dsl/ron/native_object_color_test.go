package ron

import (
	"slices"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNativeObjectColor(t *testing.T) {
	for _, tc := range ronLaces {
		for _, category := range []string{"permanent", "spell"} {
			for _, state := range []string{"normal", "illegal", "phase", "cycle", "return", "clone", "copy"} {
				t.Run(tc.name+"/"+category+"/"+state, func(t *testing.T) {
					g, a, b := ronModalGame(0)
					victim := mage.NewCreature("Subject", "{W}{U}", 2, 3)
					id := victim.ID()
					if category == "spell" {
						id = castRONSpell(t, g, b, victim).ID
					} else {
						g.PutOnBattlefield(victim, b.PlayerID())
					}
					obj := castLace(t, g, a, tc.name, id)
					instruction, ok := mage.InspectObjectColorInstruction(obj.Effects[0])
					if !ok || instruction.Color != tc.color {
						t.Fatal("lowering lost native color operands")
					}
					if _, ok := mage.InspectObjectTarget(instruction.Subject); !ok {
						t.Fatal("lowering lost native union target")
					}
					illegal := false
					switch state {
					case "illegal":
						if category == "spell" {
							g.CounterStackObject(id)
						} else {
							g.FindPermanent(id).GrantBaseAttr(core.Shroud)
						}
						illegal = true
					case "phase":
						if category == "permanent" {
							g.PhaseOut(id)
							illegal = true
						}
					case "cycle":
						if category == "permanent" {
							g.PhaseIn(g.PhaseOut(id))
						}
					case "return":
						if category == "permanent" {
							g.RemoveFromBattlefield(g.FindPermanent(id))
							g.PutOnBattlefield(victim, b.PlayerID())
							illegal = true
						} else {
							g.CounterStackObject(id)
							illegal = true
						}
					case "clone":
						g = g.Clone()
						obj = g.StackPeek()
					case "copy":
						obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
					}
					initial := slices.Clone(g.EffectiveColors(id))
					for range 2 {
						if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
							t.Fatal("complete color tree is unknown")
						}
					}
					if !slices.Equal(g.EffectiveColors(id), initial) {
						t.Fatal("query changed colors")
					}
					g.ResolveTopOfStack()
					want := []core.Color{tc.color}
					if illegal {
						want = initial
					}
					if !slices.Equal(g.EffectiveColors(id), want) {
						t.Fatal("resolution lost identity or native color")
					}
				})
			}
		}
	}
}
