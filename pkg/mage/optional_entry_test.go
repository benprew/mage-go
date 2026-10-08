package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type optionalEntryPlayer struct {
	*BasePlayer
	accept bool
	offers int
}

func (p *optionalEntryPlayer) ChooseMayAbility(string) bool { p.offers++; return p.accept }

type optionalEntryProbe struct {
	BaseAbility
	run func(*Game, *Permanent) bool
}

func (p *optionalEntryProbe) OnEnter(g *Game, perm *Permanent) bool { return p.run(g, perm) }

func TestOptionalEntryOperations(t *testing.T) {
	for _, accept := range []bool{false, true} {
		a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: accept}
		b := &optionalEntryPlayer{BasePlayer: NewBasePlayer("B"), accept: !accept}
		g := NewGame(a, b)
		calls := 0
		probe := &optionalEntryProbe{BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic}, run: func(g *Game, p *Permanent) bool {
			calls++
			if !p.Tapped || p.ChosenPlayer != b.PlayerID() {
				t.Fatal("entry operations ran out of order")
			}
			for _, live := range g.AllBattlefield() {
				if live.ID() == p.ID() {
					t.Fatal("optional body ran after entry")
				}
			}
			return true
		}}
		children := []AsEntersBattlefieldAbility{EnterTappedAsEnters(), ChooseOpponentAsEnters(), probe}
		ability := MayEntryOperations(children...)
		children[0] = nil
		card := NewArtifact("Optional entry", "", WithAbility(ability))
		card.SetOwner(b.PlayerID())
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		p := g.PutOnBattlefield(card, a.PlayerID())
		if p == nil || p.Tapped != accept || calls != optionalEntryBool(accept) || a.offers != 1 || b.offers != 0 || g.StackSize() != 0 {
			t.Fatal("optional entry choice, decline or controller")
		}
		g.PhaseIn(g.PhaseOut(p.ID()))
		if a.offers != 1 {
			t.Fatal("phasing repeated entry choice")
		}
		clone := g.Clone()
		clone.MutablePermanent(p.ID()).ChosenPlayer = a.PlayerID()
		if g.FindPermanent(p.ID()).ChosenPlayer == a.PlayerID() {
			t.Fatal("clone shared entry state")
		}
		g.BouncePermanentToHand(g.FindPermanent(p.ID()))
		b.RemoveFromHand(card.ID())
		a.accept = !accept
		returned := g.PutOnBattlefield(card, a.PlayerID())
		if returned == nil || returned.incarnationID == p.incarnationID || returned.Tapped != !accept || a.offers != 2 {
			t.Fatal("returned permanent did not make a fresh choice")
		}
	}
}

func TestOptionalEntryFailure(t *testing.T) {
	a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	calls := 0
	fail := &optionalEntryProbe{run: func(*Game, *Permanent) bool { return false }}
	later := &optionalEntryProbe{run: func(*Game, *Permanent) bool { calls++; return true }}
	card := NewArtifact("Failed entry", "", WithAbility(MayEntryOperations(fail, later)))
	card.SetOwner(b.PlayerID())
	if g.PutOnBattlefield(card, a.PlayerID()) != nil || calls != 0 || len(b.Graveyard()) != 1 {
		t.Fatal("child rejection must stop entry and the rest of the body")
	}
}

func TestOptionalEntryDeclarations(t *testing.T) {
	var typedNil *optionalEntryProbe
	for _, ability := range []AsEntersBattlefieldAbility{MayEntryOperations(), MayEntryOperations(nil), MayEntryOperations(typedNil), MayEntryOperations(MayEntryOperations())} {
		card := NewArtifact("Invalid", "", WithAbility(ability))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid optional entry accepted")
		}
	}
	if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithAbility(MayEntryOperations(EnterTappedAsEnters())))); err == nil {
		t.Fatal("optional entry on a nonpermanent accepted")
	}
}

func TestOptionalEntryKeepsRequiredTappedStatus(t *testing.T) {
	a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: false}
	g := NewGame(a, NewBasePlayer("B"))
	card := NewArtifact("Required tapped entry", "", WithKeyword(core.EntersTapped), WithAbility(MayEntryOperations(EnterTappedAsEnters())))
	p := g.PutOnBattlefield(card, a.PlayerID())
	if !p.Tapped || a.offers != 1 {
		t.Fatal("decline undid required tapped entry")
	}
}

func optionalEntryBool(b bool) int {
	if b {
		return 1
	}
	return 0
}
