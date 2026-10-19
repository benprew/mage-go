package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestRONNativeBoundedCombatLowering(t *testing.T) {
	record := hostUntapRecord(t, "LEG", "Glyph of Doom")
	if _, err := Compile(record); err != nil {
		t.Fatal(err)
	}
	declaration := contextualQueryNode(t, record, "CreateFutureTrigger")
	target := &permanentTargetPlan{name: "Ref_TargetPermanent"}
	build, err := compileDelayedStep(declaration, scope{resolving: true, permanentTarget: target})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		bindings := instanceBindings{permanents: map[*permanentTargetPlan]*mage.PermanentTargetBinding{target: mage.DeclarePermanentTarget("wall", mage.IsCreature)}}
		effect := build(bindings)
		got, ok := mage.InspectNextCombatThisTurnInstruction(effect)
		if !ok {
			t.Fatal("bounded delay is opaque")
		}
		body, ok := mage.InspectDirectDestruction(got.Body)
		if !ok || len(body) != 1 {
			t.Fatal("lost complete destruction body")
		}
		if _, ok := mage.InspectContextualPermanentCollection(body[0].Subjects); !ok {
			t.Fatal("lost native collection")
		}
	}
}

func TestRONNativeBoundedCombatInvalid(t *testing.T) {
	for _, action := range []string{
		`CreateFutureTrigger(AtNextEndOfCombatThisTurn(),ActionList([DestroyEachPermanent(IsCardtype(Creature))]))`,
		`CreateFutureTrigger(AtNextEndOfCombatThisTurn,ActionList([]))`,
		`CreateFutureTrigger(AtNextEndOfCombatThisTurn,ActionList([DestroyEachPermanent(WasBlockedByDefenderThisTurn(Ref_TargetPermanent_2))]))`,
		`CreateFutureTrigger(AtNextEndOfCombatThisTurn,ActionList([DestroyEachPermanent(WasBlockedByDefenderThisTurn(You))]))`,
		`CreateFutureTrigger(AtNextEndOfCombatThisTurn,ActionList([DestroyEachPermanent(And([IsCardtype(Creature),Unknown]))]))`,
		`CreateFutureTrigger(AtNextEndOfCombatThisTurn,ActionList([DestroyEachPermanent(IsCardtype(Creature)),Unknown]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([`+action+`])))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid bounded delay accepted", action)
		}
	}
}
