package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type optionalAttachmentPlayer struct {
	*attachmentMovePlayer
	accept bool
	offers int
	offer  func()
}

func (p *optionalAttachmentPlayer) ChooseMayAbility(string) bool {
	p.offers++
	if p.offer != nil {
		p.offer()
	}
	return p.accept
}

func TestOptionalAttachmentContracts(t *testing.T) {
	for _, state := range []string{"accept", "decline", "no-host", "protected", "shroud", "hexproof", "phase", "return", "during-return", "during-protection", "chooser-control"} {
		t.Run(state, func(t *testing.T) {
			a := &optionalAttachmentPlayer{attachmentMovePlayer: &attachmentMovePlayer{BasePlayer: NewBasePlayer("A")}, accept: state != "decline"}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			old := g.PutOnBattlefield(NewLand("Old"), a.PlayerID())
			next := g.PutOnBattlefield(NewLand("Next"), b.PlayerID())
			auraCard := NewAura("Aura", "{G}", WithCastTarget(TargetLand()))
			aura := g.PutOnBattlefield(auraCard, b.PlayerID())
			g.Attach(aura.ID(), old.ID())
			a.choose = func(candidates []*Permanent) *Permanent { return next }
			wantOffers, want := 1, next.ID()
			switch state {
			case "decline":
				want = old.ID()
			case "no-host":
				g.RemoveFromBattlefield(next)
				wantOffers, want = 0, old.ID()
			case "protected":
				if err := ApplyEffect(g, Until(core.EndOfTurn, GrantProtectionColors(This, core.Green)), next.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				wantOffers, want = 0, old.ID()
			case "shroud":
				next.GrantBaseAttr(core.Shroud)
			case "hexproof":
				next.GrantBaseAttr(core.Hexproof)
			case "phase":
				g.PhaseOut(aura.ID())
				wantOffers, want = 0, old.ID()
			case "return":
				a.offer = func() {
					g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
					g.PutOnBattlefield(auraCard, b.PlayerID())
				}
				want = old.ID()
			case "during-return":
				a.choose = func([]*Permanent) *Permanent {
					g.RemoveFromBattlefield(next)
					g.PutOnBattlefield(next.Card, b.PlayerID())
					return next
				}
				g.PutOnBattlefield(NewLand("Alternative"), b.PlayerID())
				want = old.ID()
			case "during-protection":
				a.offer = func() {
					if err := ApplyEffect(g, Until(core.EndOfTurn, GrantProtectionColors(This, core.Green)), next.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				want = old.ID()
			case "chooser-control":
				g.PutOnBattlefield(NewLand("Alternative"), b.PlayerID())
				a.offer = func() {
					g.AddControlEffect(ControlEffectSpec{TargetID: old.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
			}
			body := MayAttachPermanentToChoice(ControllerOfWithLKI(AttachedPermanent), This, MatchingPermanents(NewPermanentFilter("other hosts", func(p *Permanent, _ *Game) bool { return p.ID() != old.ID() })))
			if err := ApplyEffect(g, body, aura.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			present := g.FindPermanentIncludingPhased(aura.ID())
			if state == "return" {
				if present.AttachedTo == next.ID() {
					t.Fatal("returned Aura inherited action")
				}
			} else if present.AttachedTo != want {
				t.Fatalf("host %v, want %v", present.AttachedTo, want)
			}
			if a.offers != wantOffers {
				t.Fatalf("offers %d, want %d", a.offers, wantOffers)
			}
			if state == "chooser-control" && a.calls != 1 {
				t.Fatal("chooser changed after offer")
			}
		})
	}
}

func TestOptionalAttachmentDeclarations(t *testing.T) {
	for _, body := range []Effect{
		MayAttachPermanentToChoice(nil, This, MatchingPermanents(AnyPermanent)),
		MayAttachPermanentToChoice(You, nil, MatchingPermanents(AnyPermanent)),
		MayAttachPermanentToChoice(You, This, nil),
		MayAttachPermanentToChoice(You, From(BindPermanent("missing")), MatchingPermanents(AnyPermanent)),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestOptionalAttachmentWaiting(t *testing.T) {
	a := &optionalAttachmentPlayer{attachmentMovePlayer: &attachmentMovePlayer{BasePlayer: NewBasePlayer("A")}, accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	old := g.PutOnBattlefield(NewLand("Old"), b.PlayerID())
	next := g.PutOnBattlefield(NewLand("Next"), b.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Aura", "", WithCastTarget(TargetLand())), a.PlayerID())
	g.Attach(aura.ID(), old.ID())
	reads := 0
	hosts := MatchingPermanents(NewPermanentFilter("guard", func(p *Permanent, _ *Game) bool { reads++; return p.HasType(core.TypeLand) && p.ID() != old.ID() }))
	body := MayAttachPermanentToChoice(You, This, hosts)
	if err := ApplyEffect(g, AtNext(EachEndStep, body), aura.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.SetRandomResults([]int{7})
	if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || a.offers != 0 || g.RandIntn(10) != 7 || aura.AttachedTo != old.ID() {
		t.Fatal("waiting query ran optional choice")
	}
	clone := g.Clone()
	cloneChooser := &optionalAttachmentPlayer{attachmentMovePlayer: &attachmentMovePlayer{BasePlayer: clone.GetPlayer(a.PlayerID()).(*SearchPlayer).BasePlayer}, accept: true}
	clone.SetPlayerAt(0, cloneChooser)
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.ResolveStack()
	if clone.FindPermanent(aura.ID()).AttachedTo != next.ID() || g.FindPermanent(aura.ID()).AttachedTo != old.ID() || a.offers != 0 {
		t.Fatal("pending clone shared attachment choice")
	}
}

func TestOptionalAttachmentCurrentHost(t *testing.T) {
	a := &optionalAttachmentPlayer{attachmentMovePlayer: &attachmentMovePlayer{BasePlayer: NewBasePlayer("A")}, accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewLand("Host"), b.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Aura", "", WithCastTarget(TargetLand())), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtAttach, false, GainLifeFor(You, Fixed(1))))), b.PlayerID())
	before := len(host.Attachments)
	if err := ApplyEffect(g, MayAttachPermanentToChoice(You, This, MatchingPermanents(AnyPermanent)), aura.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if a.offers != 1 || a.calls != 0 || aura.AttachedTo != host.ID() || len(host.Attachments) != before || g.StackSize() != 0 {
		t.Fatal("current legal host must permit a no-op choice")
	}
}
