package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestChosenStaticConditionIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewEnchantment("Choice", "")
	source := g.PutOnBattlefield(card, a.PlayerID())
	source.ChosenPlayer, source.ChosenColor = b.PlayerID(), core.Red
	ctx := StateContext{SourceID: source.ID(), SourceIncarnation: source.incarnationID, ControllerID: a.PlayerID()}
	condition := PlayerControlsChosenColor(ChosenPlayerOf(This), Not(IsToken))
	fixed := PlayerControls(ChosenPlayerOf(This), HasColorFilter(core.Red))
	member := g.PutOnBattlefield(NewCreature("Member", "{R}{U}", 1, 4), b.PlayerID())
	check := func(game *Game, want bool) {
		t.Helper()
		if condition.checkState(game, ctx) != want {
			t.Fatalf("membership want %v", want)
		}
	}
	check(g, true)
	if !fixed.checkState(g, ctx) {
		t.Fatal("fixed filter lost chosen player")
	}
	clone := g.Clone()
	clone.MutablePermanent(source.ID()).ChosenColor = core.Green
	check(clone, false)
	check(g, true)
	source.ChosenColor = core.AnyColor
	check(g, false)
	source.ChosenColor = core.Red
	source.ChosenPlayer = uuid.New()
	check(g, false)
	source.ChosenPlayer = b.PlayerID()
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	check(g, true)
	phased := g.PhaseOut(member.ID())
	check(g, false)
	g.PhaseIn(phased)
	check(g, true)
	g.MutablePermanent(member.ID()).IsToken = true
	check(g, false)
	g.MutablePermanent(member.ID()).IsToken = false
	g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: member.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	check(g, false)
	if !PlayerControlsChosenColor(You, Not(IsToken)).checkState(g, ctx) {
		t.Fatal("current effect controller ignored")
	}
	phasedSource := g.PhaseOut(source.ID())
	if fixed.checkState(g, ctx) {
		t.Fatal("phased source supplied choice")
	}
	check(g, false)
	g.PhaseIn(phasedSource)
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	check(g, false)
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, a.PlayerID())
	returned.ChosenPlayer, returned.ChosenColor = b.PlayerID(), core.Red
	check(g, false)
	ctx.SourceIncarnation = returned.incarnationID
	check(g, true)
}

func TestChosenStaticConditionDeclarations(t *testing.T) {
	for _, player := range []PlayerRef{nil, ChosenPlayerOf(nil), ChosenPlayerOf(AttachedPermanent), ChosenPlayerOf(From(BindPermanent("saved"))), FromPlayer(BindPlayer("unbound"))} {
		card := NewEnchantment("Invalid", "", WithAbility(Static(While(PlayerControlsChosenColor(player, AnyPermanent), ModifyEachPT(MatchingPermanents(IsCreature), Fixed(2), Fixed(1))))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("invalid player accepted: %T", player)
		}
	}
	for _, player := range []PlayerRef{You, ActivePlayer, ChosenPlayerOf(This)} {
		card := NewEnchantment("Valid", "", WithAbility(Static(While(PlayerControlsChosenColor(player, AnyPermanent), ModifyEachPT(MatchingPermanents(IsCreature), Fixed(2), Fixed(1))))))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}
