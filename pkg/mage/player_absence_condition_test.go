package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPlayerAbsenceConditionIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewEnchantment("Source", "")
	source := g.PutOnBattlefield(card, a.PlayerID())
	source.ChosenPlayer, source.ChosenColor = b.PlayerID(), core.Red
	ctx := StateContext{SourceID: source.ID(), SourceIncarnation: source.incarnationID, ControllerID: a.PlayerID()}
	fixed := PlayerControlsNone(ChosenPlayerOf(This), HasColorFilter(core.Red))
	chosen := PlayerControlsNoneOfChosenColor(ChosenPlayerOf(This), Not(IsToken))
	check := func(game *Game, want bool) {
		t.Helper()
		if chosen.checkState(game, ctx) != want {
			t.Fatalf("absence want %v", want)
		}
	}
	check(g, true)
	member := g.PutOnBattlefield(NewArtifact("Member", "{R}{U}"), b.PlayerID())
	check(g, false)
	if fixed.checkState(g, ctx) {
		t.Fatal("fixed absence ignored membership")
	}
	clone := g.Clone()
	clone.MutablePermanent(source.ID()).ChosenColor = core.Green
	check(clone, true)
	check(g, false)
	g.MutablePermanent(member.ID()).IsToken = true
	check(g, true)
	g.MutablePermanent(member.ID()).IsToken = false
	phased := g.PhaseOut(member.ID())
	check(g, true)
	g.PhaseIn(phased)
	check(g, false)
	source.ChosenPlayer = uuid.New()
	check(g, false)
	if fixed.checkState(g, ctx) {
		t.Fatal("unknown player satisfied absence")
	}
	source.ChosenPlayer = b.PlayerID()
	source.ChosenColor = core.AnyColor
	check(g, false)
	source.ChosenColor = core.Red
	phased = g.PhaseOut(source.ID())
	check(g, false)
	if fixed.checkState(g, ctx) {
		t.Fatal("phased source supplied choice")
	}
	g.PhaseIn(phased)
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	check(g, false)
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, a.PlayerID())
	returned.ChosenPlayer, returned.ChosenColor = b.PlayerID(), core.Green
	check(g, false)
	ctx.SourceIncarnation = returned.incarnationID
	check(g, true)
}

func TestPlayerAbsenceConditionDeclarations(t *testing.T) {
	for _, constructor := range []func(PlayerRef, PermanentFilter) StateCondition{PlayerControlsNone, PlayerControlsNoneOfChosenColor} {
		for _, player := range []PlayerRef{nil, ChosenPlayerOf(nil), ChosenPlayerOf(AttachedPermanent), FromPlayer(BindPlayer("unbound"))} {
			card := NewEnchantment("Invalid", "", WithAbility(WhenState(constructor(player, AnyPermanent), Destroy(This))))
			if err := ValidateCardDefinition(card.Name(), card); err == nil {
				t.Fatalf("invalid player accepted: %T", player)
			}
		}
		for _, player := range []PlayerRef{You, ActivePlayer, ChosenPlayerOf(This)} {
			card := NewEnchantment("Valid", "", WithAbility(WhenState(constructor(player, AnyPermanent), Destroy(This))))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
		}
	}
}
