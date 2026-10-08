package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTokenBandsSingularGrant(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := StateConditionFunc("token quality", []StateDependency{StateBattlefield}, func(StateReader, StateContext) bool { return true })
	token := DefineToken("Member", TokenPT(1, 20), TokenTypes(core.TypeCreature), TokenAbilities(Static(While(condition, GrantBandsWithOther(This, Named("Member")), GrantBandsWithOther(This, IsCreature)))))
	effect := CreateTokenCopies(You, Fixed(2), token)
	if err := ValidateCardDefinition("Maker", NewInstant("Maker", "", Spell(effect))); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	members := g.FilterBattlefield(IsToken)
	stranger := g.PutOnBattlefield(NewCreature("Stranger", "", 1, 20), a.PlayerID())
	g.ApplyContinuousEffects()
	for _, p := range members {
		if len(g.bandsQualities(p)) != 2 {
			t.Fatalf("missing independent qualities: %d for %s", len(g.bandsQualities(p)), p.ID())
		}
	}
	if len(g.bandsQualities(stranger)) != 0 {
		t.Fatal("singular grant reached another creature")
	}
	if members[0].RuntimeAbilities[0].AbilityID() == members[1].RuntimeAbilities[0].AbilityID() {
		t.Fatal("shared ability identity")
	}
	clone := g.Clone()
	if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveBandsWithOther(This)), members[0].ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.bandsQualities(g.FindPermanent(members[0].ID()))) != 0 || len(g.bandsQualities(g.FindPermanent(members[1].ID()))) != 2 || len(clone.bandsQualities(clone.FindPermanent(members[0].ID()))) != 2 {
		t.Fatal("removal or clone isolation")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.PhaseOut(members[0].ID())
	g.ApplyContinuousEffects()
	if len(g.bandsQualities(g.FindPermanentIncludingPhased(members[0].ID()))) != 0 {
		t.Fatal("phased quality active")
	}
	g.PhaseIn([]uuid.UUID{members[0].ID()})
	g.ApplyContinuousEffects()
	if len(g.bandsQualities(g.FindPermanent(members[0].ID()))) != 2 {
		t.Fatal("phase cycle lost abilities")
	}
	for _, p := range g.FilterBattlefield(IsToken) {
		p.RevokeBaseAttr(core.AttrSummonSick)
	}
	stranger.RevokeBaseAttr(core.AttrSummonSick)
	ids := []uuid.UUID{members[0].ID(), stranger.ID()}
	g.ExecuteAttackers(a.PlayerID(), ids)
	if err := g.FormAttackingBand(ids); err != nil {
		t.Fatal("second quality did not permit formation", err)
	}
	if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Static(While(condition, GrantBandsWithOther(nil, IsCreature)))))); err == nil {
		t.Fatal("nil subject accepted")
	}
}

func TestTokenBandsCardCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := StateConditionFunc("token quality", []StateDependency{StateBattlefield}, func(StateReader, StateContext) bool { return true })
	token := DefineToken("Member", TokenPT(1, 20), TokenTypes(core.TypeCreature), TokenAbilities(Static(While(condition, GrantBandsWithOther(This, Named("Member"))))))
	if err := ApplyEffect(g, CreateTokenCopies(You, Fixed(1), token), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	original := g.FilterBattlefield(IsToken)[0]
	copied := g.PutOnBattlefield(original.Card.Copy(), a.PlayerID())
	g.ApplyContinuousEffects()
	if len(g.bandsQualities(original)) != 1 || len(g.bandsQualities(copied)) != 1 || original.RuntimeAbilities[0].AbilityID() == copied.RuntimeAbilities[0].AbilityID() || original.RuntimeAbilities[0].Source() != original.ID() {
		t.Fatal("token card copy shares ability source state")
	}
	g.BouncePermanentToHand(copied)
	g.ApplyContinuousEffects()
	if len(g.bandsQualities(original)) != 1 {
		t.Fatal("copy departure removed original quality")
	}
}
