package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedLibraryChoiceOperands(t *testing.T) {
	for name, body := range map[string]ResolvingEffect{
		"scry":    ScryAtResolution(PowerOfWithLKI(This)),
		"surveil": SurveilAtResolution(ManaValueOfWithLKI(This)),
		"zero":    ScryAtResolution(SubtractValues(PowerOfWithLKI(This), PowerOfWithLKI(This))),
	} {
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Source", "{U}", 2, 4)
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			check := func() {
				t.Helper()
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) || len(g.LibraryLooks(a.PlayerID())) != 0 {
					t.Fatal("lost reference or ran inspection", got)
				}
			}
			check()
			phased := g.PhaseOut(p.ID())
			check()
			g.PhaseIn(phased)
			check()
			g.BouncePermanentToHand(p)
			g.zones.ClearLKI()
			check()
			a.RemoveFromHand(c.ID())
			g.PutOnBattlefield(c, b.PlayerID())
			check()
			clone := g.Clone()
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("shared wait lifetime")
			}
		})
	}
}

func TestWaitingDelayedLibraryChoicePurity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 2, 4), a.PlayerID())
	for range 4 {
		a.AddToLibrary(NewLand("Land"))
	}
	reads := 0
	body := ScryAtResolution(AddValues(PowerOfWithLKI(This), ValueAtResolution(waitingTokenCount{&reads})))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || len(g.LibraryLooks(a.PlayerID())) != 0 {
			t.Fatal("query ran count")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if reads != 1 || len(g.LibraryLooks(a.PlayerID())) != 0 || len(a.Library()) != 4 || len(g.LibraryLooks(b.PlayerID())) != 0 {
		t.Fatal("inspection timing or privacy", reads)
	}
	for _, body := range []ResolvingEffect{ScryAtResolution(nil), SurveilAtResolution(nil), ScryAtResolution(PowerOf(From(BindPermanent("missing"))))} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), body)))) == nil {
			t.Fatal("invalid count accepted")
		}
	}
	for _, body := range []ResolvingEffect{ScryAtResolution(Fixed(0)), SurveilAtResolution(PowerOfWithLKI(AttachedPermanent)), ScryAtResolution(ValueAtResolution(waitingTokenCount{&reads})), AtNext(EachEndStep, SurveilAtResolution(PowerOfWithLKI(This)))} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("inferred implicit or future reference")
	}
}

type waitingLibraryChoicePlayer struct {
	*BasePlayer
	calls, count int
}

func (p *waitingLibraryChoicePlayer) ChooseScryPlacement(cards []Card, _ string, _ GameReader) ([]uuid.UUID, []uuid.UUID) {
	p.calls++
	p.count = len(cards)
	top := make([]uuid.UUID, 0, len(cards)-1)
	for _, card := range cards[1:] {
		top = append(top, card.ID())
	}
	return []uuid.UUID{cards[0].ID()}, top
}

func (p *waitingLibraryChoicePlayer) ChooseSurveilPlacement(cards []Card, reason string, game GameReader) ([]uuid.UUID, []uuid.UUID) {
	return p.ChooseScryPlacement(cards, reason, game)
}

func TestWaitingDelayedLibraryChoiceResolution(t *testing.T) {
	for name, build := range map[string]func(ResolutionValue) ResolvingEffect{"scry": ScryAtResolution, "surveil": SurveilAtResolution} {
		for _, state := range []string{"live", "control", "departed", "returned", "phased", "cycle"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				a := &waitingLibraryChoicePlayer{BasePlayer: NewBasePlayer("A")}
				b := &waitingLibraryChoicePlayer{BasePlayer: NewBasePlayer("B")}
				g := NewGame(a, b)
				card := NewCreature("Source", "", 2, 4)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				first := NewLand("First")
				first.SetOwner(a.PlayerID())
				a.SetLibrary([]Card{first, NewLand("Second"), NewLand("Third")})
				reads := 0
				body := build(AddValues(PowerOfWithLKI(This), ValueAtResolution(waitingTokenCount{&reads})))
				if err := ApplyEffect(g, AtNextModes(EachEndStep, ModeOf("choice", body), ModeOf("other", body)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				switch state {
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "departed", "returned":
					g.BouncePermanentToHand(p)
					g.zones.ClearLKI()
					if state == "returned" {
						a.RemoveFromHand(card.ID())
						q := g.PutOnBattlefield(card, b.PlayerID())
						q.AddCounter(core.M1M1, 2)
					}
				case "phased":
					g.PhaseOut(p.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				}
				for range 3 {
					if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || a.calls != 0 || b.calls != 0 {
						t.Fatal("query ran amount or choice")
					}
				}
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				if len(g.WaitingDelayedPermanentSources()) != 0 || reads != 0 || a.calls != 0 {
					t.Fatal("detection ran body")
				}
				g.PutTriggersOnStack()
				g.ResolveStack()
				if reads != 1 || a.calls != 1 || a.count != 3 || b.calls != 0 {
					t.Fatalf("wrong count or timing: reads=%d calls=%d count=%d", reads, a.calls, a.count)
				}
				if name == "scry" && (len(a.Library()) != 3 || a.Library()[2].ID() != first.ID() || len(a.Graveyard()) != 0) {
					t.Fatal("scry placement failed")
				}
				if name == "surveil" && (len(a.Library()) != 2 || len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != first.ID()) {
					t.Fatal("surveil placement failed")
				}
				if len(g.PublicReveals()) != 0 {
					t.Fatal("choice revealed private cards")
				}
			})
		}
	}
}
