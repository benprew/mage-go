package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type branchMarkerEffect struct {
	called *bool
}

func (e branchMarkerEffect) Text() string { return "mark branch" }
func (e branchMarkerEffect) Properties() EffectProperties {
	return EffectProperties{}
}
func (e branchMarkerEffect) Apply(*EffectContext) error {
	*e.called = true
	return nil
}

func TestIfPlayerPaysBranchesAndVarPlayer(t *testing.T) {
	t.Run("paid", func(t *testing.T) {
		payer := &acceptingMayPlayer{NewBasePlayer("payer")}
		opponent := NewBasePlayer("opponent")
		g := NewGame(payer, opponent)
		paid, unpaid := false, false
		ctx := &EffectContext{Game: g, Controller: opponent.PlayerID(), Vars: map[string]any{"payer": payer.PlayerID()}}
		effect := IfPlayerPays(
			VarPlayer("payer"),
			LifePayCost(2),
			"pay 2 life",
			branchMarkerEffect{called: &paid},
			branchMarkerEffect{called: &unpaid},
		)
		if err := effect.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if !paid || unpaid || payer.Life() != 18 {
			t.Fatalf("paid=%v unpaid=%v life=%d", paid, unpaid, payer.Life())
		}
	})

	t.Run("declined", func(t *testing.T) {
		payer := NewBasePlayer("payer")
		opponent := NewBasePlayer("opponent")
		g := NewGame(payer, opponent)
		paid, unpaid := false, false
		ctx := &EffectContext{Game: g, Vars: map[string]any{"payer": payer.PlayerID()}}
		effect := IfPlayerPays(VarPlayer("payer"), LifePayCost(2), "pay", branchMarkerEffect{&paid}, branchMarkerEffect{&unpaid})
		if err := effect.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if paid || !unpaid || payer.Life() != 20 {
			t.Fatalf("paid=%v unpaid=%v life=%d", paid, unpaid, payer.Life())
		}
	})

	t.Run("unpayable", func(t *testing.T) {
		payer := &acceptingMayPlayer{NewBasePlayer("payer")}
		payer.SetLife(2)
		opponent := NewBasePlayer("opponent")
		g := NewGame(payer, opponent)
		paid, unpaid := false, false
		ctx := &EffectContext{Game: g, Vars: map[string]any{"payer": payer.PlayerID()}}
		effect := IfPlayerPays(VarPlayer("payer"), LifePayCost(2), "pay", branchMarkerEffect{&paid}, branchMarkerEffect{&unpaid})
		if err := effect.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if paid || !unpaid || payer.Life() != 2 {
			t.Fatalf("paid=%v unpaid=%v life=%d", paid, unpaid, payer.Life())
		}
	})
}

func TestIfPlayerPaysRequiresExactlyOnePayer(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	effect := IfPlayerPays(SelectEachPlayer(), LifePayCost(1), "pay", nil, nil)
	if err := effect.Apply(&EffectContext{Game: g, Vars: map[string]any{}}); err == nil {
		t.Fatal("multiple payers were accepted")
	}
	effect = IfPlayerPays(VarPlayer("missing"), LifePayCost(1), "pay", nil, nil)
	if err := effect.Apply(&EffectContext{Game: g, Vars: map[string]any{}}); err == nil {
		t.Fatal("missing payer was accepted")
	}
}

func TestPipelineOwnershipEffects(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g, err := NewGameWithAnte(a, b, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	card := ownedTestCard(a.PlayerID(), "exiled")
	g.ExileCard(card, uuid.Nil)
	ctx := &EffectContext{Game: g, Controller: b.PlayerID(), Vars: map[string]any{"card": card.ID(), "newOwner": b.PlayerID()}}
	if err := ChangeOwnerGathered("card", VarPlayer("newOwner")).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := g.FindCardAnywhere(card.ID()); got == nil || got.Owner() != b.PlayerID() || g.CardZone(card.ID()) != ZoneExile {
		t.Fatal("ChangeOwnerGathered did not preserve exile")
	}
	if err := MoveExiledGatheredToGraveyard("card").Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(b.Graveyard()) != 1 || b.Graveyard()[0].ID() != card.ID() || g.CardZone(card.ID()) != ZoneGraveyard {
		t.Fatal("MoveExiledGatheredToGraveyard used the wrong destination")
	}
}

func TestSnapshotPermanentStoresOwner(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := ownedTestCard(a.PlayerID(), "controlled by B")
	perm := g.PutOnBattlefield(card, b.PlayerID())
	ctx := &EffectContext{Game: g, Targets: []uuid.UUID{perm.ID()}, Vars: map[string]any{}}
	if err := SnapshotPermanent(SelectTarget, "target").Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := ctx.TryGetUUID("target.owner"); got != a.PlayerID() {
		t.Fatalf("target.owner = %s, want %s", got, a.PlayerID())
	}
}
