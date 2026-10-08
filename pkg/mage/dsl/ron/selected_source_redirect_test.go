package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func selectedSourceRedirectRONFixture(t *testing.T) mage.CardFactory {
	t.Helper()
	record := staticBaseRecord(t, "LEA", "Jade Monolith")
	activated := record.Fields["Rules"].Args[0]
	targeted := activated.Args[1]
	actions := targeted.Args[1].Args[0].Args
	choice, err := compileChosenDamageSourceChoice(actions[0])
	if err != nil {
		t.Fatal(err)
	}
	targeted.Args[1].Args[0].Args = actions[1:]
	plan, err := compileActions(targeted, scope{chosenDamageSource: choice, permanent: true})
	if err != nil {
		t.Fatal(err)
	}
	plan.effects = append([]effectFactory{func(bindings instanceBindings) mage.Effect {
		return mage.ChooseDamageSourceFrom(mage.You, []mage.DamageSourceRef{mage.DamageSourceOf(mage.This)}, bindings.chosenDamageSources[choice])
	}}, plan.effects...)
	return func() mage.Card {
		binding := mage.BindDamageSource("chosen")
		body := plan.buildWithBindings(instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{choice: binding}})
		body.effects = []mage.Effect{mage.Sequence(body.effects...)}
		return mage.NewArtifact("RON action fixture", "", mage.WithAbility(Activated([]mage.Cost{PayMana(core.ParseManaCost("{1}"))}, body)))
	}
}

func TestRONSelectedSourceRedirectAction(t *testing.T) {
	for _, state := range []string{"live", "illegal", "source-leaves", "return", "phase", "cleanup", "control"} {
		t.Run(state, func(t *testing.T) {
			factory := selectedSourceRedirectRONFixture(t)
			g, a, b := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			target := g.PutOnBattlefield(mage.NewCreature("Target", "", 1, 20), b.PlayerID())
			if err := mage.ValidateCardDefinition("RON action fixture", source.Card); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil || g.StackSize() != 0 {
				t.Fatal("failed payment changed stack")
			}
			a.ManaPool().Add(core.White, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if len(g.WaitingRedirectionPermanentSources()) != 0 {
				t.Fatal("installed before resolution")
			}
			if state == "illegal" {
				g.PhaseOut(target.ID())
			}
			if state == "source-leaves" {
				g.RemoveFromBattlefield(source)
			}
			g.ResolveStack()
			saved := g.CaptureDamageSource(source.ID())
			if state == "illegal" {
				g.PhaseIn([]uuid.UUID{target.ID()})
			}
			if state == "return" {
				g.RemoveFromBattlefield(target)
				g.PutOnBattlefield(target.Card, b.PlayerID())
			}
			if state == "phase" {
				g.PhaseOut(target.ID())
				g.PhaseIn([]uuid.UUID{target.ID()})
			}
			if state == "cleanup" {
				g.GetEffects().ClearReplacementsEndOfTurn()
			}
			if state == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			for _, game := range []*mage.Game{g, g.Clone()} {
				game.DealDamageToPermanentFromSource(game.FindPermanent(target.ID()), 2, saved)
				want := 18
				if state == "illegal" || state == "source-leaves" || state == "return" || state == "cleanup" {
					want = 20
				}
				if game.GetPlayer(a.PlayerID()).Life() != want {
					t.Fatalf("life=%d want=%d", game.GetPlayer(a.PlayerID()).Life(), want)
				}
				game.DealDamageToPermanentFromSource(game.FindPermanent(target.ID()), 1, saved)
				wantDamage := 1
				if want == 20 {
					wantDamage = 3
				}
				if game.FindPermanent(target.ID()).Damage != wantDamage {
					t.Fatal("wrong one-shot recipient")
				}
			}
			for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
				if _, err := Compile(staticBaseRecord(t, set, "Jade Monolith")); err != nil {
					t.Fatal("complete source-choice record rejected", err)
				}
			}
		})
	}
}

func TestRONSelectedSourceRedirectInvalid(t *testing.T) {
	for _, text := range []string{
		`CreateFutureReplaceWouldDealDamage()`,
		`CreateFutureReplaceWouldDealDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource),[DealToPlayerInstead(You)])`,
		`CreateFutureReplaceWouldDealDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Permanent()),[DealToPlayerInstead(You)])`,
		`CreateFutureReplaceWouldDealDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Permanent(ThisPermanent)),[DealToPlayerInstead()])`,
		`CreateFutureReplaceWouldDealDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Permanent(ThisPermanent)),[DealToCreatureOrPlaneswalkerInstead(ThisPermanent)])`,
		`CreateFutureReplaceWouldDealDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(1),Permanent(ThisPermanent)),[DealToPlayerInstead(You)])`,
		`CreateFutureReplaceWouldDealDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Permanent(ThisPermanent)),[DealToPlayerInstead(You)])`,
		`CreateFutureReplaceWouldDealDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(ThisPermanent,Permanent(ThisPermanent)),[DealToPlayerInstead(You)])`,
		`CreateFutureReplaceWouldDealDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Player(You)),[DealToPlayerInstead(You)])`,
		`CreateFutureReplaceWouldDealDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Permanent(Ref_TargetPermanent)),[DealToPlayerInstead(You)])`,
		`CreateFutureReplaceWouldDealDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Permanent(ThisPermanent)),[])`,
		`CreateFutureReplaceWouldDealDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Permanent(ThisPermanent)),[DealToPlayerInstead(Trigger_ThatPlayer)])`,
	} {
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		ctx := scope{chosenDamageSource: &chosenDamageSourcePlan{}, permanent: true}
		if strings.Contains(text, "Permanent(ThisPermanent)),[DealToPlayerInstead(You)]") && strings.Contains(text, "TheChosenDamageSource") {
			ctx.chosenDamageSource = nil
		}
		if _, err := compileEffect(nodes[0], ctx); err == nil {
			t.Fatal("invalid declaration accepted", text)
		}
	}
}

func TestRONSelectedSourceRedirectInstances(t *testing.T) {
	g, a, b := ronGame()
	factory := selectedSourceRedirectRONFixture(t)
	first := g.PutOnBattlefield(factory(), a.PlayerID())
	second := g.PutOnBattlefield(factory(), b.PlayerID())
	targetA := g.PutOnBattlefield(mage.NewCreature("Target A", "", 1, 20), a.PlayerID())
	targetB := g.PutOnBattlefield(mage.NewCreature("Target B", "", 1, 20), b.PlayerID())
	a.ManaPool().Add(core.White, 1)
	b.ManaPool().Add(core.White, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, []uuid.UUID{targetA.ID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), second.ID(), 0, []uuid.UUID{targetB.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	baseline := g.Clone()
	for _, game := range []*mage.Game{g, g.Clone()} {
		game.DealDamageToPermanent(game.FindPermanent(targetA.ID()), 2, second.ID())
		game.DealDamageToPermanent(game.FindPermanent(targetB.ID()), 3, first.ID())
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatal("factory bindings crossed")
		}
		game.DealDamageToPermanent(game.FindPermanent(targetA.ID()), 2, first.ID())
		game.DealDamageToPermanent(game.FindPermanent(targetB.ID()), 3, second.ID())
		if game.GetPlayer(a.PlayerID()).Life() != 18 || game.GetPlayer(b.PlayerID()).Life() != 17 || game.FindPermanent(targetA.ID()).Damage != 2 || game.FindPermanent(targetB.ID()).Damage != 3 {
			t.Fatal("instances did not keep independent subjects and players")
		}
	}
	if len(baseline.WaitingRedirectionPermanentSources()) != 4 || baseline.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("clone mutated original rules")
	}
}
