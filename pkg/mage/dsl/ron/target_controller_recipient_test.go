package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestRONNativeTargetControllerRecipientLowering(t *testing.T) {
	record := repositoryRecord(t, "ATQ", "Detonate", "controller.ron")
	if _, err := Compile(record); err != nil {
		t.Fatal(err)
	}
	action := contextualQueryNode(t, record, "SpellDealsDamage")
	target := &permanentTargetPlan{name: "Ref_TargetPermanent"}
	build, err := compileDamagePacketRecipient(action.Args[2], scope{resolving: true, permanentTarget: target})
	if err != nil {
		t.Fatal(err)
	}
	first, second := mage.DeclarePermanentTarget("artifact", mage.IsArtifact), mage.DeclarePermanentTarget("artifact", mage.IsArtifact)
	for _, binding := range []*mage.PermanentTargetBinding{first, second} {
		ref := build(instanceBindings{permanents: map[*permanentTargetPlan]*mage.PermanentTargetBinding{target: binding}})
		if _, known := mage.InspectDamageRecipient(ref); !known {
			t.Fatal("full-record controller recipient lost its native declaration")
		}
	}
	for _, syntax := range []string{
		`Player(ControllerOfPermanent)`, `Player(ControllerOfPermanent())`,
		`Player(ControllerOfPermanent(Ref_TargetPermanent, Ref_TargetPermanent))`,
		`Player(ControllerOfPermanent(You))`, `Player(ControllerOfPermanent(Ref_TargetPermanent_2))`,
	} {
		if _, err := compileDamagePacketRecipient(nameScopeNode(t, syntax), scope{resolving: true, permanentTarget: target}); err == nil {
			t.Fatalf("invalid recipient accepted: %s", syntax)
		}
	}
	if _, err := compileDamagePacketRecipient(action.Args[2], scope{resolving: true}); err == nil {
		t.Fatal("unbound target accepted")
	}
}
