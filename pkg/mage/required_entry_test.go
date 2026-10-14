package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRequiredEntryOperationsOrderAndFailure(t *testing.T) {
	for _, reject := range []bool{false, true} {
		a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: false}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		calls := 0
		first := &optionalEntryProbe{run: func(g *Game, p *Permanent) bool {
			calls++
			if p.ChosenPlayer != b.PlayerID() || p.Tapped {
				t.Fatal("wrong entry order")
			}
			for _, live := range g.AllBattlefield() {
				if live.ID() == p.ID() {
					t.Fatal("child ran after battlefield registration")
				}
			}
			return !reject
		}}
		last := &optionalEntryProbe{run: func(_ *Game, p *Permanent) bool {
			calls++
			if !p.Tapped {
				t.Fatal("later child ran before tapped entry")
			}
			return true
		}}
		children := []AsEntersBattlefieldAbility{ChooseOpponentAsEnters(), first, EnterTappedAsEnters(), last}
		ability := RequiredEntryOperations(children...)
		children[0] = nil
		card := NewArtifact("Required", "", WithAbility(ability))
		card.SetOwner(b.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		if a.offers != 0 {
			t.Fatal("required entry offered an optional choice")
		}
		if reject {
			if p != nil || calls != 1 || len(b.Graveyard()) != 1 {
				t.Fatal("rejected child did not stop entry")
			}
		} else if p == nil || !p.Tapped || calls != 2 || g.StackSize() != 0 {
			t.Fatal("required entry did not finish")
		}
	}
}

func TestRequiredEntryOperationsDeclarations(t *testing.T) {
	var typedNil *optionalEntryProbe
	for _, ability := range []AsEntersBattlefieldAbility{RequiredEntryOperations(), RequiredEntryOperations(nil), RequiredEntryOperations(typedNil), RequiredEntryOperations(RequiredEntryOperations())} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(ability))); err == nil {
			t.Fatal("invalid required entry accepted")
		}
	}
	if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithAbility(RequiredEntryOperations(ChooseColorAsEnters())))); err == nil {
		t.Fatal("nonpermanent accepted")
	}
	if err := ValidateCardDefinition("Valid", NewArtifact("Valid", "", WithAbility(RequiredEntryOperations(ChooseColorAsEnters(), ChooseOpponentAsEnters())))); err != nil {
		t.Fatal(err)
	}
}

func TestRequiredEntryOperationsCopyReplacementAndBatch(t *testing.T) {
	a := &colorEntryPlayer{BasePlayer: NewBasePlayer("A"), answer: core.Black}
	b := &colorEntryPlayer{BasePlayer: NewBasePlayer("B"), answer: core.Red}
	g := NewGame(a, b)
	donorPlan, err := NewCardDefinitionPlan(func() Card {
		return NewCreature("Joint donor", "", 2, 2, WithAbility(RequiredEntryOperations(ChooseColorAsEnters(), ChooseOpponentAsEnters())))
	})
	if err != nil {
		t.Fatal(err)
	}
	donorCard, err := donorPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	donor := g.PutOnBattlefield(donorCard, b.PlayerID())
	copyPlan, err := NewCardDefinitionPlan(func() Card { return NewCreature("Copy", "", 1, 1, WithAbility(MayCopyAsEnters(IsCreature))) })
	if err != nil {
		t.Fatal(err)
	}
	copyCard, err := copyPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	copied := g.PutOnBattlefield(copyCard, a.PlayerID())
	if copied == nil || copied.Name() != "Joint donor" || copied.ChosenColor != core.Black || copied.ChosenPlayer != b.PlayerID() || donor.ChosenColor != core.Red || donor.ChosenPlayer != a.PlayerID() || a.calls != 1 || b.calls != 1 {
		t.Fatal("copy did not acquire fresh joint entry choices")
	}
	observed := 0
	watcher := NewArtifact("Watcher", "", WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(evt *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		if evt.ToZone != core.ZoneBattlefield {
			return false
		}
		p := reader.FindPermanent(evt.SourceID)
		if p == nil || p.Name() != "Batch joint" {
			return false
		}
		observed++
		count := 0
		for _, member := range reader.AllBattlefield() {
			if member.Name() == "Batch joint" {
				count++
				if member.ChosenColor != core.Green || member.ChosenPlayer != b.PlayerID() {
					t.Fatal("batch event saw incomplete choices")
				}
			}
		}
		if count != 2 || a.calls != 3 {
			t.Fatal("batch event ran before all entries")
		}
		return false
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	a.answer = core.Green
	factory := func() Card {
		return NewArtifact("Batch joint", "", WithAbility(RequiredEntryOperations(ChooseColorAsEnters(), ChooseOpponentAsEnters())))
	}
	entries := g.putTokensOnBattlefieldTogether([]Card{factory(), factory()}, a.PlayerID())
	if len(entries) != 2 || observed != 2 {
		t.Fatal("wrong batch event count")
	}
}

type requiredEntryColorPlayer struct {
	*BasePlayer
	game  *Game
	calls int
	check func(*Permanent)
}

func (p *requiredEntryColorPlayer) ChooseManaColor(string) core.Color {
	p.calls++
	p.check(p.game.zones.enteringPermanent)
	return core.Blue
}

func TestRequiredEntryOperationsChoiceOrder(t *testing.T) {
	a := &requiredEntryColorPlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	a.game = g
	a.check = func(p *Permanent) {
		if p == nil || p.ChosenPlayer != uuid.Nil {
			t.Fatal("opponent choice preceded color choice")
		}
	}
	card := NewArtifact("Joint", "", WithAbility(RequiredEntryOperations(ChooseColorAsEnters(), ChooseOpponentAsEnters())))
	card.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	if p.ChosenColor != core.Blue || p.ChosenPlayer != b.PlayerID() || a.calls != 1 {
		t.Fatal("joint entry did not finish choices")
	}
	clone := g.Clone()
	clone.MutablePermanent(p.ID()).ChosenColor = core.Red
	clone.MutablePermanent(p.ID()).ChosenPlayer = a.PlayerID()
	if p.ChosenColor != core.Blue || p.ChosenPlayer != b.PlayerID() {
		t.Fatal("clone shares joint choice storage")
	}
	g.PhaseIn(g.PhaseOut(p.ID()))
	if a.calls != 1 {
		t.Fatal("phase cycle repeated choices")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	b.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if returned.incarnationID == p.incarnationID || a.calls != 2 {
		t.Fatal("return did not start a fresh entry")
	}
}
