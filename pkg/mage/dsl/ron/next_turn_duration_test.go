package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNextTurnDuration(t *testing.T) {
	for _, player := range []string{"You", "Ref_TargetPlayer", "ControllerOfPermanent(ThisPermanent)"} {
		t.Run(player, func(t *testing.T) {
			body := `ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([Flying]),AdjustPT(1,2)],UntilPlayersNextTurn(` + player + `))])`
			if player == "Ref_TargetPlayer" {
				body = `Targeted([TargetPlayer(AnyPlayer)],` + body + `)`
			}
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),`+body+`)`)
			g, a, b := ronGame()
			g.RunStepWithPriority(core.Untap)
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			var targets []uuid.UUID
			if player == "Ref_TargetPlayer" {
				targets = []uuid.UUID{b.PlayerID()}
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err == nil {
				t.Fatal("unpaid activation succeeded")
			}
			if !g.GetStack().IsEmpty() {
				t.Fatal("failed cost installed action")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err != nil {
				t.Fatal(err)
			}
			if source.HasKeyword(core.Flying) {
				t.Fatal("announcement installed effect")
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			if !g.FindPermanent(source.ID()).HasKeyword(core.Flying) {
				t.Fatal("RON effect missing")
			}
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			g.RunStepWithPriority(core.Upkeep)
			if !g.FindPermanent(source.ID()).HasKeyword(core.Flying) {
				t.Fatal("upkeep expired turn duration")
			}
			g.AdvanceToNextTurn()
			g.RunStepWithPriority(core.PrecombatMain)
			if g.FindPermanent(source.ID()).HasKeyword(core.Flying) != (player != "Ref_TargetPlayer") {
				t.Fatal("wrong saved player")
			}
			g.AdvanceToNextTurn()
			g.RunStepWithPriority(core.Untap)
			if g.FindPermanent(source.ID()).HasKeyword(core.Flying) {
				t.Fatal("duration survived next turn")
			}
			if !clone.FindPermanent(source.ID()).HasKeyword(core.Flying) {
				t.Fatal("clone shared expiration")
			}
		})
	}
}

func TestRONNextTurnDurationInvalid(t *testing.T) {
	for _, duration := range []string{`UntilPlayersNextTurn`, `UntilPlayersNextTurn()`, `UntilPlayersNextTurn(You,You)`, `UntilPlayersNextTurn(You())`, `UntilPlayersNextTurn(ThisPermanent)`, `UntilPlayersNextTurn(Ref_TargetPlayer)`, `UntilPlayersNextTurn(Trigger_ThatPlayer)`, `UntilPlayersNextTurn(TheChosenPlayer)`, `UntilPlayersNextTurn(OwnerOfTheCardReturnedToHandThisWay)`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([Flying])],` + duration + `)]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", duration)
		}
	}
}

func TestRONNextTurnDurationInventory(t *testing.T) {
	src := `Card(Name:"Lifetime",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AdjustPT(1,1)],UntilPlayersNextTurn(You))]))])`
	report, err := InspectFS(fstest.MapFS{"duration.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "You" && strings.Contains(signature.Context, "UntilPlayersNextTurn") {
			found = true
			if signature.Role != "player_reference" {
				t.Fatalf("operand role %s", signature.Role)
			}
		}
	}
	if !found {
		t.Fatal("missing lifetime player inventory")
	}
}

func TestRONNextTurnDurationInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([Flying])],UntilPlayersNextTurn(Ref_TargetPlayer))])))`)
	g, a, b := ronGame()
	g.RunStepWithPriority(core.Untap)
	left := g.PutOnBattlefield(factory(), a.PlayerID())
	right := g.PutOnBattlefield(factory(), a.PlayerID())
	for _, pair := range []struct{ source, player uuid.UUID }{{left.ID(), a.PlayerID()}, {right.ID(), b.PlayerID()}} {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), pair.source, 0, []uuid.UUID{pair.player}); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(left.ID()).HasKeyword(core.Flying) || g.FindPermanent(right.ID()).HasKeyword(core.Flying) {
		t.Fatal("factories shared player bindings")
	}
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(left.ID()).HasKeyword(core.Flying) {
		t.Fatal("independent duration survived")
	}
}
