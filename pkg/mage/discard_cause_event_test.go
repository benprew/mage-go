package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func discardCauseCard() Card {
	cause := BindPlayer("discard cause controller")
	return NewSorcery("Self discard", "", WithAbility(WhenEvent(
		ThisCardDiscardedByOpponentEffect(cause), LoseLifeFor(FromPlayer(cause), Fixed(5)),
	)))
}

func TestDiscardCauseEventLifecycle(t *testing.T) {
	for _, mode := range []string{"effect", "own-effect", "cost", "cleanup", "no-source", "returned", "two"} {
		t.Run(mode, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := discardCauseCard()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			controller := b.PlayerID()
			if mode == "own-effect" {
				controller = a.PlayerID()
			}
			source := g.PutOnBattlefield(NewArtifact("cause", ""), controller)
			switch mode {

			case "cost":
				g.PlayerDiscardByCost(a, card.ID(), source.ID())
			case "cleanup":
				g.PlayerDiscard(a, card.ID())
			case "no-source":
				g.PlayerDiscardByEffect(a, card.ID(), uuid.Nil)
			}
			if mode == "effect" || mode == "own-effect" || mode == "returned" || mode == "two" {
				if mode == "two" {
					second := discardCauseCard()
					second.SetOwner(a.PlayerID())
					a.AddToHand(second)
					g.discardHandsTogether([]uuid.UUID{a.PlayerID()}, source.ID())
				} else {
					g.PlayerDiscardByEffect(a, card.ID(), source.ID())
				}
			}
			want := 20
			if mode == "effect" || mode == "returned" {
				want = 15
			}
			if mode == "two" {
				want = 10
			}
			if mode == "returned" {
				moved, _ := a.RemoveFromGraveyard(card.ID())
				a.AddToHand(moved)
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(source.Card, a.PlayerID())
			}
			clone := g.Clone()
			g.ResolveStack()
			if b.Life() != want || a.Life() != 20 {
				t.Fatalf("life A=%d B=%d want B=%d", a.Life(), b.Life(), want)
			}
			if clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("clone shared trigger results")
			}
			clone.ResolveStack()
			if clone.GetPlayer(b.PlayerID()).Life() != want {
				t.Fatal("clone lost pending cause")
			}
		})
	}
}

func TestDiscardCauseEventResolvingController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := discardCauseCard()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	source := g.PutOnBattlefield(NewArtifact("ability source", ""), a.PlayerID())
	g.stack.resolvingReferenceObjects = append(g.stack.resolvingReferenceObjects, &StackObject{ID: uuid.New(), SourceID: source.ID(), Controller: b.PlayerID(), IsAbility: true})
	g.PlayerDiscardByEffect(a, card.ID(), source.ID())
	g.stack.resolvingReferenceObjects = nil
	g.ResolveStack()
	if b.Life() != 15 || a.Life() != 20 {
		t.Fatal("current source controller replaced ability controller")
	}
}

func TestDiscardCauseEventDefinition(t *testing.T) {
	if err := ValidateCardDefinition("self", discardCauseCard()); err != nil {
		t.Fatal(err)
	}
	invalid := NewSorcery("invalid", "", WithAbility(WhenEvent(ThisCardDiscardedByOpponentEffect(nil), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition("invalid", invalid); err == nil {
		t.Fatal("nil cause binding accepted")
	}
	binding := BindPlayer("unbound")
	invalid = NewSorcery("invalid", "", WithAbility(Spell(LoseLifeFor(FromPlayer(binding), Fixed(5)))))
	if err := ValidateCardDefinition("invalid", invalid); err == nil {
		t.Fatal("cause binding escaped event")
	}
}

func TestDiscardCauseEventSnapshots(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := discardCauseCard()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	visit := g.handVisit(card.ID())
	source := g.PutOnBattlefield(NewArtifact("cause", ""), b.PlayerID())
	g.PlayerDiscardByEffect(a, card.ID(), source.ID())
	pending := g.triggers.Pending()
	if len(pending) != 1 {
		t.Fatalf("pending=%d", len(pending))
	}
	event := pending[0].event
	if event.DiscardedHandVisit != visit || event.DiscardCauseController != b.PlayerID() || event.DiscardCauseID != source.ID() || !event.DiscardByEffect || event.Type != core.EvtDiscard {
		t.Fatal("discard origin facts missing")
	}
}

func TestDiscardCauseEventReplacements(t *testing.T) {
	for _, mode := range []string{"library", "exile", "cancel", "returned", "new-proposal", "controller-change"} {
		t.Run(mode, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := discardCauseCard()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			source := g.PutOnBattlefield(NewArtifact("cause", ""), b.PlayerID())
			switch mode {
			case "library":
				g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
			case "exile":
				g.effects.AddReplacement(&factsDiscardToExile{})
			case "cancel":
				g.effects.AddReplacement(&preventRandomDiscardReplacement{card: card.ID()})
			case "returned":
				g.effects.AddReplacement(&playerDiscardReturningReplacement{card: card.ID()})
			case "new-proposal", "controller-change":
				g.effects.AddReplacement(&discardCauseProposalReplacement{change: mode == "controller-change", controller: a.PlayerID()})
			}
			_, moved := g.PlayerDiscardByEffect(a, card.ID(), source.ID())
			want := 15
			if mode == "library" || mode == "cancel" || mode == "returned" {
				want = 20
			}
			if moved != (mode != "cancel" && mode != "returned") {
				t.Fatal("discard completion lost exact hand visit")
			}
			if mode == "exile" && (g.FindExiledCard(card.ID()) == nil || len(a.Graveyard()) != 0) {
				t.Fatal("discard replacement destination lost")
			}
			if mode == "library" && (len(a.Library()) != 1 || len(a.Graveyard()) != 0) {
				t.Fatal("hidden destination lost")
			}
			g.ResolveStack()
			if b.Life() != want || a.Life() != 20 {
				t.Fatalf("life A=%d B=%d want B=%d", a.Life(), b.Life(), want)
			}
		})
	}
}

type discardCauseProposalReplacement struct {
	replacementBase
	change     bool
	controller uuid.UUID
}

func (*discardCauseProposalReplacement) Matches(a Action, _ GameReader) bool {
	_, ok := a.(*DiscardAction)
	return ok
}
func (r *discardCauseProposalReplacement) Replace(a Action, g *Game) Action {
	d := a.(*DiscardAction)
	if r.change {
		g.AddControlEffect(ControlEffectSpec{SourceID: r.controller, TargetID: d.ActionSource(), ControllerID: r.controller, Duration: core.Indefinite})
	}
	return NewDiscardAction(d.ActionSource(), d.PlayerID(), d.CardID())
}
func (*discardCauseProposalReplacement) IsActive(GameReader) bool   { return true }
func (r *discardCauseProposalReplacement) Clone() ReplacementEffect { c := *r; return &c }

func TestDiscardCauseEventActivationAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("discard player")
	cause := NewArtifact("sacrificed cause", "", WithAbility(Activate(Costs(SacrificeSourceCost()), Discard(target, Fixed(1)), WithTargets(target))))
	source := g.PutOnBattlefield(cause, b.PlayerID())
	card := discardCauseCard()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(source.ID()) != nil {
		t.Fatal("source sacrifice failed")
	}
	g.PutOnBattlefield(cause, a.PlayerID())
	g.PhaseOut(cause.ID())
	g.ResolveTopOfStack()
	if g.StackSize() != 1 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("trigger did not wait for resolution")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if b.Life() != 10 || a.Life() != 20 {
		t.Fatal("copy lost saved discard cause controller")
	}
}

func TestDiscardCauseEventCostRollback(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("cost source", "", WithAbility(Activate(Costs(Tap(), DiscardCost(1), ManaCostOf("{1}")), GainLifeFor(You, Fixed(1))))), a.PlayerID())
	card := discardCauseCard()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("unpaid activation accepted")
	}
	if source.Tapped || len(a.Hand()) != 1 || len(g.triggers.Pending()) != 0 {
		t.Fatal("failed payment leaked state")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("discard cost triggered a spell-effect event")
	}
}

func TestDiscardCauseEventSimultaneousAndAPNAP(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := discardCauseCard(), discardCauseCard()
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	a.AddToHand(first)
	b.AddToHand(second)
	sourceA := g.PutOnBattlefield(NewArtifact("A cause", ""), a.PlayerID())
	sourceB := g.PutOnBattlefield(NewArtifact("B cause", ""), b.PlayerID())
	events := 0
	observer := NewTriggered(core.EvtDiscard, false).SetCondition(func(_ *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		events++
		if len(a.Hand()) != 0 || len(b.Hand()) != 0 || a.Life() != 20 || b.Life() != 20 {
			t.Error("event saw partial batch or early trigger resolution")
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer)), a.PlayerID())
	if got := g.discardActionsTogether([]*DiscardAction{NewDiscardAction(sourceB.ID(), a.PlayerID(), first.ID()), NewDiscardAction(sourceA.ID(), b.PlayerID(), second.ID())}); got != 2 || events != 2 {
		t.Fatal("simultaneous discard count")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 || g.StackPeek().Controller != b.PlayerID() {
		t.Fatal("discard triggers lost APNAP order")
	}
	g.ResolveTopOfStack()
	if a.Life() != 15 || b.Life() != 20 {
		t.Fatal("nonactive trigger lost its own cause")
	}
	g.ResolveStack()
	if a.Life() != 15 || b.Life() != 15 {
		t.Fatal("active trigger lost its own cause")
	}
}
