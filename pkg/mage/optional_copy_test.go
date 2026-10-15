package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type optionalCopyPlayer struct {
	*BasePlayer
	accept bool
	offers int
}

func (p *optionalCopyPlayer) ChooseMayAbility(string) bool { p.offers++; return p.accept }

func TestOptionalCopyDecision(t *testing.T) {
	for _, accept := range []bool{false, true} {
		a := &optionalCopyPlayer{BasePlayer: NewBasePlayer("A"), accept: accept}
		b := &optionalCopyPlayer{BasePlayer: NewBasePlayer("B"), accept: true}
		g := NewGame(a, b)
		p := g.PutOnBattlefield(entryCopyPhysical(t), b.PlayerID())
		effect := MayCopyPermanent(You, This, This, true)
		card := NewArtifact("Optional", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		if a.offers != 0 || b.offers != 0 {
			t.Fatal("declaration made a choice")
		}
		visit := p.incarnationID
		if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		p = g.FindPermanent(p.ID())
		if a.offers != 1 || b.offers != 0 || (p.resolvingCopy != nil) != accept || p.incarnationID != visit {
			t.Fatal("wrong chooser, decision or visit")
		}
	}
}

func TestOptionalCopyDeclarations(t *testing.T) {
	for _, effect := range []Effect{
		MayCopyPermanent(nil, This, This, false),
		MayCopyPermanent(You, nil, This, false),
		MayCopyPermanent(You, This, nil, false),
		MayCopyPermanent(You, This, DeclarePermanentTarget("unbound", AnyPermanent), false),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid optional copy accepted")
		}
	}
}

func TestOptionalCopyWaiting(t *testing.T) {
	a := &optionalCopyPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	p := g.PutOnBattlefield(entryCopyPhysical(t), a.PlayerID())
	saved := g.CaptureDamageSource(p.ID())
	if err := ApplyEffect(g, AtNext(EachEndStep, MayCopyPermanent(You, This, This, false)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for _, transition := range []string{"present", "phased", "departed"} {
		switch transition {
		case "phased":
			g.PhaseOut(p.ID())
		case "departed":
			g.BouncePermanentToHand(g.FindPermanentIncludingPhased(p.ID()))
		}
		for range 3 {
			sources := g.WaitingDelayedPermanentSources()
			if len(sources) != 1 || !sources[0].SameObject(saved) || a.offers != 0 {
				t.Fatal("waiting query lost source or made a choice")
			}
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if a.offers != 0 {
		t.Fatal("missing recipient offered an impossible copy")
	}
}

func TestOptionalCopyExplicitPlayer(t *testing.T) {
	a := &optionalCopyPlayer{BasePlayer: NewBasePlayer("A"), accept: false}
	b := &optionalCopyPlayer{BasePlayer: NewBasePlayer("B"), accept: true}
	g := NewGame(a, b)
	p := g.PutOnBattlefield(entryCopyPhysical(t), b.PlayerID())
	if err := ApplyEffect(g, MayCopyPermanent(ControllerOf(This), This, This, false), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if b.offers != 1 || a.offers != 0 || g.FindPermanent(p.ID()).resolvingCopy == nil {
		t.Fatal("explicit player did not decide")
	}
	if !g.FindPermanent(p.ID()).HasType(core.TypeCreature) {
		t.Fatal("self-copy lost characteristics")
	}
}
