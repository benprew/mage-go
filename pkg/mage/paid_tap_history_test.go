package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPaidTapHistoryActivation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(TapNMatchingIncludingSourceCost(1, IsCreature, "Tap a creature")), GainLifeFor(You, Fixed(1))))), a.PlayerID())
	payer := g.PutOnBattlefield(NewCreature("Payer", "", 1, 3), a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	set := TappedForAbilitiesThisTurn(This)
	if len(set.resolvePermanents(ctx)) != 0 {
		t.Fatal("query changed history")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if got := set.resolvePermanents(ctx); len(got) != 1 || got[0].ID() != payer.ID() {
		t.Fatal("paid tap missing")
	}
	clone := g.Clone()
	g.trackers.Turn.ResetForNewTurn(a.PlayerID())
	if len(set.resolvePermanents(ctx)) != 0 {
		t.Fatal("history crossed turn")
	}
	cloneCtx := *ctx
	cloneCtx.Game = clone
	if len(set.resolvePermanents(&cloneCtx)) != 1 {
		t.Fatal("clone shared history")
	}
	clone.PhaseOut(payer.ID())
	if len(set.resolvePermanents(&cloneCtx)) != 0 {
		t.Fatal("phased payer selected")
	}
	clone.PhaseIn([]uuid.UUID{payer.ID()})
	if len(set.resolvePermanents(&cloneCtx)) != 1 {
		t.Fatal("phasing erased history")
	}
}

func TestPaidTapHistoryCostForms(t *testing.T) {
	for _, shape := range []string{"symbol", "matching", "matching-many", "attached", "exact", "mana", "planned-mana", "direct-mana"} {
		t.Run(shape, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first := g.PutOnBattlefield(NewCreature("First", "", 1, 3), a.PlayerID())
			g.MutablePermanent(first.ID()).RevokeBaseAttr(core.AttrSummonSick)
			g.PutOnBattlefield(NewCreature("Second", "", 1, 3), a.PlayerID())
			cost := Cost(Tap())
			want := 1
			switch shape {
			case "matching":
				cost = TapMatchingCost(IsCreature, "Tap a creature")
			case "matching-many":
				cost = TapNMatchingIncludingSourceCost(2, IsCreature, "Tap two creatures")
				want = 2
			case "attached":
				cost = TapAttachedPermanentCost()
			case "exact":
				cost = &exactPermanentTapCost{subject: capturedPermanent{first.ID(), first.incarnationID}}
			}
			card := NewArtifact("Source", "", WithAbility(Activate(Costs(cost), GainLifeFor(You, Fixed(1)))))
			if shape == "attached" {
				card = NewAura("Source", "", WithCastTarget(TargetCreature()), WithAbility(Activate(Costs(cost), GainLifeFor(You, Fixed(1)))))
			}
			if shape == "mana" || shape == "planned-mana" || shape == "direct-mana" {
				card = NewArtifact("Source", "", WithManaAbility(core.Blue))
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			if shape == "attached" {
				g.Attach(source.ID(), first.ID())
			}
			switch shape {
			case "planned-mana":
				if err := g.mana.AutoTapForCost(g, a.PlayerID(), core.ParseManaCost("{U}"), AutoTapHint{}, nil); err != nil {
					t.Fatal(err)
				}
			case "direct-mana":
				if err := g.mana.TapForMana(g, a.PlayerID(), source.ID(), core.Blue); err != nil {
					t.Fatal(err)
				}
			default:
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
			if got := len(TappedForAbilitiesThisTurn(This).resolvePermanents(ctx)); got != want {
				t.Fatalf("paid visits %d, want %d", got, want)
			}
		})
	}
}

func TestPaidTapHistoryRollbackAndBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(TapNMatchingIncludingSourceCost(1, IsCreature, "Tap a creature"), LifePayCost(21)), GainLifeFor(You, Fixed(1))))), a.PlayerID())
	payer := g.PutOnBattlefield(NewCreature("Payer", "", 1, 3), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("unpayable activation")
	}
	if a.Life() != 20 || g.FindPermanent(payer.ID()).Tapped || len(g.trackers.Turn.paidTapHistory) != 0 {
		t.Fatal("failed payment left state")
	}
	g.TapPermanent(g.FindPermanent(payer.ID()))
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	if len(TappedForAbilitiesThisTurn(This).resolvePermanents(ctx)) != 0 {
		t.Fatal("ordinary tap became a payment")
	}
	g.UntapPermanent(g.FindPermanent(payer.ID()))
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{TapNMatchingIncludingSourceCost(1, IsCreature, "Tap a creature")}})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(TappedForAbilitiesThisTurn(This).resolvePermanents(ctx)) != 0 {
		t.Fatal("resolution payment became activation history")
	}
	invalid := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), TapEach(TappedForAbilitiesThisTurn(nil)))))
	if err := ValidateCardDefinition("Invalid", invalid); err == nil {
		t.Fatal("nil source accepted")
	}
}

func TestPaidTapHistoryWaitingQuery(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Source", "")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	saved := g.CaptureDamageSource(source.ID())
	if err := ApplyEffect(g, AtNext(EachEndStep, TapEach(TappedForAbilitiesThisTurn(This))), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, b.PlayerID())
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		for range 3 {
			got := game.WaitingDelayedPermanentSources()
			if len(got) != 1 || !got[0].SameObject(saved) || len(game.trackers.Turn.paidTapHistory) != 0 {
				t.Fatal("query lost source visit or made history")
			}
		}
	}
}
