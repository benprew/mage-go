package dsl_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestLibraryChoiceCompletionEvents(t *testing.T) {
	for _, surveil := range []bool{false, true} {
		for _, adapter := range []string{"primitive", "legacy", "typed"} {
			for _, size := range []int{0, 1, 3} {
				for _, amount := range []int{-1, 0, 2} {
					t.Run(fmt.Sprintf("surveil=%t/%s/size=%d/amount=%d", surveil, adapter, size, amount), func(t *testing.T) {
						a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
						g := mage.NewGame(a, b)
						for i := range size {
							b.AddToLibrary(mage.NewLand(fmt.Sprintf("Card %d", i)))
						}
						original := append([]mage.Card(nil), b.Library()...)
						eventType := dsl.EvtScry
						if surveil {
							eventType = dsl.EvtSurveil
						}
						var events []core.GameEvent
						trigger := mage.NewTriggered(eventType, false, dsl.Sequence(dsl.GainLife(dsl.You, dsl.Fixed(1)), dsl.GainLife(dsl.You, dsl.ValueAtResolution(dsl.EventAmountValue()))))
						trigger.SetCondition(func(evt *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
							events = append(events, *evt)
							if amount > 0 && size > 0 {
								if surveil {
									if len(b.Graveyard()) != 1 || b.Graveyard()[0].ID() != original[0].ID() {
										t.Error("event preceded graveyard placement")
									}
								} else if b.Library()[size-1].ID() != original[0].ID() {
									t.Error("event preceded bottom placement")
								}
							}
							return evt.PlayerID == b.PlayerID()
						})
						g.PutOnBattlefield(mage.NewEnchantment("Watcher", "", mage.WithAbility(trigger)), a.PlayerID())
						if size > 0 && amount > 0 {
							if surveil {
								b.AddSurveilDecision([]string{"Card 0"}, nil)
							} else {
								b.AddScryDecision([]string{"Card 0"}, nil)
							}
						}
						var effect mage.Effect
						switch adapter {
						case "primitive":
							var got int
							if surveil {
								got = g.PerformSurveil(b, amount)
							} else {
								got = g.PerformScry(b, amount)
							}
							if want := max(0, min(size, amount)); got != want {
								t.Errorf("examined = %d, want %d", got, want)
							}
						case "legacy":
							effect = mage.Scry(mage.Fixed(amount))
							if surveil {
								effect = mage.Surveil(mage.Fixed(amount))
							}
						case "typed":
							effect = dsl.Scry(dsl.Fixed(amount))
							if surveil {
								effect = dsl.Surveil(dsl.Fixed(amount))
							}
						}
						if effect != nil {
							if err := mage.ApplyEffect(g, effect, uuid.Nil, b.PlayerID(), nil); err != nil {
								t.Fatal(err)
							}
						}
						wantEvents := 0
						if amount > 0 {
							wantEvents = 1
						}
						if len(events) != wantEvents {
							t.Fatalf("events = %d, want %d", len(events), wantEvents)
						}
						if len(events) == 1 && (events[0].Amount != min(size, amount) || events[0].PlayerID != b.PlayerID()) {
							t.Fatalf("event = %+v", events[0])
						}
						if a.Life() != 20 {
							t.Fatal("trigger resolved during the action")
						}
						g.PutTriggersOnStack()
						if g.StackSize() != wantEvents {
							t.Fatalf("stack = %d, want %d", g.StackSize(), wantEvents)
						}
						if wantEvents > 0 {
							g.ResolveTopOfStack()
						}
						if a.Life() != 20+wantEvents+max(0, min(size, amount)) {
							t.Fatalf("watcher life = %d", a.Life())
						}
					})
				}
			}
		}
	}
}
