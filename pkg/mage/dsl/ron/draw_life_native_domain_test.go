package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestRONDrawLifeNativeDomainPlans(t *testing.T) {
	for _, text := range []string{
		`ReplaceWouldDraw(PlayerWouldDrawDuringTheirDrawStep(You),[PlayerAction(WouldDrawACard_ThatPlayer,DrawACard)])`,
		`ReplaceWouldDraw(APlayerWouldDrawExceptFirstDrawStepDraw(AnyPlayer),[PlayerAction(WouldDrawACard_ThatPlayer,DrawACard)])`,
	} {
		nodes, err := Parse(strings.NewReader(text), "plan.ron")
		if err != nil {
			t.Fatal(err)
		}
		plan, err := compileScopedDrawReplacementPlan(nodes[0], scope{permanent: true})
		if err != nil {
			t.Fatal(err)
		}
		for range 2 {
			card := mage.NewEnchantment("Plan", "", plan.build())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
		}
		for _, bad := range []string{strings.Replace(text, "ReplaceWouldDraw", "Unknown", 1), strings.Replace(text, "You)", "Opponent)", 1), strings.Replace(text, "DrawACard)", "DrawACard())", 1)} {
			if bad == text {
				continue
			}
			ns, err := Parse(strings.NewReader(bad), "bad.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := compileScopedDrawReplacementPlan(ns[0], scope{permanent: true}); err == nil {
				t.Fatal("accepted", bad)
			}
		}
		for _, ctx := range []scope{{}, {permanent: true, resolving: true}, {permanent: true, enteringSource: true}, {permanent: true, drawReplacementBody: true}} {
			if _, err := compileScopedDrawReplacementPlan(nodes[0], ctx); err == nil {
				t.Fatal("accepted invalid static context")
			}
		}
	}
	nodes, _ := Parse(strings.NewReader(ronLifeGainDrawRule), "life.ron")
	plan, err := compileLifeGainDrawPlan(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	card := mage.NewEnchantment("Gain", "", plan.build())
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	future, _ := Parse(strings.NewReader(`CreateFutureReplaceWouldDraw(NextTimePlayerWouldDrawACardThisTurn(You),[DrawACard])`), "future.ron")
	p, err := compileFutureDrawReplacementPlan(future[0], scope{resolving: true, permanent: true})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		c := mage.NewArtifact("Future", "", mage.WithAbility(mage.Activate(mage.Costs(), p.build(instanceBindings{}))))
		if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
			t.Fatal(err)
		}
	}
	for _, ctx := range []scope{{}, {permanent: true}, {resolving: true, enteringSource: true}} {
		if _, err := compileFutureDrawReplacementPlan(future[0], ctx); err == nil {
			t.Fatal("accepted invalid future context")
		}
	}
}

func TestRONDrawLifeNativeDomainBindings(t *testing.T) {
	target := &playerTargetPlan{name: "Ref_TargetPlayer"}
	nodes, err := Parse(strings.NewReader(`CreateFutureReplaceWouldDraw(NextTimePlayerWouldDrawACardThisTurn(Ref_TargetPlayer),[PlayerAction(Ref_TargetPlayer,DrawACard)])`), "bound.ron")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := compileFutureDrawReplacementPlan(nodes[0], scope{resolving: true, playerTarget: target})
	if err != nil {
		t.Fatal(err)
	}
	var previous mage.Effect
	for range 2 {
		binding := mage.DeclarePlayerTarget("player")
		effect := plan.build(instanceBindings{players: map[*playerTargetPlan]*mage.PlayerTargetBinding{target: binding}})
		if effect == previous {
			t.Fatal("shared installation")
		}
		previous = effect
		valid := mage.NewArtifact("Valid", "", mage.WithAbility(mage.Activate(mage.Costs(), effect, mage.WithTargets(binding))))
		if err := mage.ValidateCardDefinition(valid.Name(), valid); err != nil {
			t.Fatal(err)
		}
		detached := mage.NewArtifact("Detached", "", mage.WithAbility(mage.Activate(mage.Costs(), effect, mage.WithTargets(mage.DeclarePlayerTarget("player")))))
		if err := mage.ValidateCardDefinition(detached.Name(), detached); err == nil {
			t.Fatal("lost binding identity")
		}
	}
	if _, err := compileFutureDrawReplacementPlan(nodes[0], scope{resolving: true}); err == nil {
		t.Fatal("accepted unbound player")
	}
}
