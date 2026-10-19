package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDeathReturnHandRoutes(t *testing.T) {
	for _, route := range []string{"destroy", "sacrifice", "graveyard", "zero", "batch", "token", "phase", "removed", "indestructible", "bounce"} {
		t.Run(route, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetTurn(1)
			card := NewCreature("Return subject", "{R}", 3, 2, WithAbility(ReturnToHandInsteadOfDyingWithPlayLock()))
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			switch route {
			case "token":
				p.IsToken = true
			case "phase":
				g.PhaseOut(p.ID())
			case "removed":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), p.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "indestructible":
				p.GrantBaseAttr(core.Indestructible)
			}
			switch route {
			case "sacrifice":
				g.DoSacrifice(p)
			case "graveyard":
				g.PutPermanentIntoGraveyard(p)
			case "zero":
				g.AddCountersWithReplacement(p, core.M1M1, 2, uuid.Nil, false)
				g.CheckStateBasedActions()
			case "batch":
				other := g.PutOnBattlefield(NewCreature("Other", "", 1, 1), b.PlayerID())
				g.DestroyPermanents([]*Permanent{p, other})
				if len(b.Graveyard()) != 1 || g.CreatureDeaths() != 1 {
					t.Fatal("simultaneous destruction lost the other death")
				}
			case "bounce":
				g.BouncePermanentToHand(p)
			default:
				g.DestroyPermanent(p)
			}
			wantHand := route != "removed" && route != "token" && route != "phase" && route != "indestructible"
			wantLock := wantHand && route != "bounce"
			if (len(a.Hand()) == 1) != wantHand || len(b.Hand()) != 0 {
				t.Fatal("wrong destination or owner")
			}
			if g.IsHandCardRevealed(a.PlayerID(), card.ID()) != wantLock || g.IsSpellCastProhibited(a.PlayerID(), card) != wantLock {
				t.Fatal("hand reveal or play lock differs from the replaced move")
			}
			if g.IsHandRevealed(a.PlayerID()) {
				t.Fatal("one returned card revealed the full hand")
			}
			if route != "batch" && route != "removed" && g.CreatureDeaths() != 0 {
				t.Fatal("replacement counted as a death")
			}
		})
	}
}

type deathReturnOrderPlayer struct {
	*BasePlayer
	choose string
	calls  int
}

func (p *deathReturnOrderPlayer) ChooseMode(options []string, _ string) int {
	p.calls++
	for i, option := range options {
		if strings.Contains(option, p.choose) {
			return i
		}
	}
	return 0
}

func TestDeathReturnHandReplacementOrder(t *testing.T) {
	for _, choice := range []string{"return to hand", "regeneration", "exile"} {
		t.Run(choice, func(t *testing.T) {
			a := NewBasePlayer("Owner")
			b := &deathReturnOrderPlayer{BasePlayer: NewBasePlayer("Controller"), choose: choice}
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 3, 2, WithAbility(ReturnToHandInsteadOfDyingWithPlayLock()))
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			if choice == "exile" {
				if err := ApplyEffect(g, ExileIfWouldDieThisTurn(This), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			} else {
				g.AddRegenerationShield(p.ID())
			}
			g.DestroyPermanent(p)
			if b.calls != 1 || (len(a.Hand()) == 1) != (choice == "return to hand") || (g.FindPermanent(p.ID()) != nil) != (choice == "regeneration") || (g.FindExiledCard(card.ID()) != nil) != (choice == "exile") {
				t.Fatal("controller choice or replacement recheck failed")
			}
			if g.IsHandCardRevealed(a.PlayerID(), card.ID()) != (choice == "return to hand") {
				t.Fatal("uncompleted return installed a reveal")
			}
		})
	}
}

func TestDeathReturnHandPayment(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		card := NewCreature("Payment", "", 3, 2, WithAbility(ReturnToHandInsteadOfDyingWithPlayLock()))
		p := g.PutOnBattlefield(card, a.PlayerID())
		costs := Costs(SacrificeCreatureCost())
		if rollback {
			costs = Costs(SacrificeCreatureCost(), SacrificeCreatureCost())
		}
		if err := ApplyEffect(g, IfRequiredPaid(You, costs, GainLifeFor(You, Fixed(2))), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if (g.FindPermanent(p.ID()) != nil) != rollback || (len(a.Hand()) == 1) == rollback || g.IsHandCardRevealed(a.PlayerID(), card.ID()) == rollback || (a.Life() == 22) == rollback || g.CreatureDeaths() != 0 {
			t.Fatal("replacement broke paid sacrifice or rollback")
		}
	}
}

func TestDeathReturnHandInformationAndReentry(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Subject", "{R}", 3, 2, WithAbility(ReturnToHandInsteadOfDyingWithPlayLock()))
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.PhaseIn(g.PhaseOut(p.ID()))
	g.DestroyPermanent(g.FindPermanent(p.ID()))
	visit := g.handVisit(card.ID())
	if g.CaptureHandDamageSourceFor(b.PlayerID(), visit).Name() != card.Name() {
		t.Fatal("public visit concealed from opponent")
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		player := game.GetPlayer(a.PlayerID())
		returned, _ := player.RemoveFromHand(card.ID())
		if game.handVisitInformationFor(b.PlayerID(), card.ID(), visit, false).Name() != card.Name() {
			t.Fatal("public departure facts lost, including in clone")
		}
		p = game.PutOnBattlefield(returned, a.PlayerID())
		if p == nil {
			t.Fatal("hand lock prohibited noncast entry")
		}
		game.DestroyPermanent(p)
		if !game.IsSpellCastProhibited(a.PlayerID(), card) || game.handVisit(card.ID()) == visit {
			t.Fatal("returned battlefield visit lost replacement or fresh hand lock")
		}
	}
}

func TestDeathReturnHandLandAndExtraTurn(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(1)
	g.SetStep(core.PrecombatMain)
	card := NewLand("Land", WithAbility(ReturnToHandInsteadOfDyingWithPlayLock()))
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.DestroyPermanent(p)
	if err := g.PlayLand(a.PlayerID(), card.ID()); err == nil {
		t.Fatal("play lock permitted land play")
	}
	g.GrantExtraTurn(a.PlayerID())
	g.AdvanceToNextTurn()
	g.SkipNextOccurrenceOfStep(core.Untap)
	g.RunStepWithPriority(core.PrecombatMain)
	if g.IsHandCardRevealed(a.PlayerID(), card.ID()) {
		t.Fatal("own extra turn with skipped untap retained lock")
	}
	if err := g.PlayLand(a.PlayerID(), card.ID()); err != nil {
		t.Fatal(err)
	}
}

type deathReturnDestinationReplacement struct {
	replacementBase
	destination core.Zone
}

func (*deathReturnDestinationReplacement) IsActive(GameReader) bool { return true }
func (*deathReturnDestinationReplacement) Matches(action Action, _ GameReader) bool {
	_, destination := deathMoveSubject(action)
	return destination == core.ZoneHand
}
func (r *deathReturnDestinationReplacement) Replace(action Action, _ *Game) Action {
	if sacrifice, ok := action.(*SacrificePermanentAction); ok {
		cloned := *sacrifice
		cloned.destination = r.destination
		return &cloned
	}
	move := *action.(*permanentGraveyardAction)
	move.destination = r.destination
	return &move
}
func (r *deathReturnDestinationReplacement) Clone() ReplacementEffect { cloned := *r; return &cloned }

func TestDeathReturnHandChangedDestination(t *testing.T) {
	for _, destination := range []core.Zone{core.ZoneGraveyard, core.ZoneExile} {
		for _, sacrifice := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 3, 2, WithAbility(ReturnToHandInsteadOfDyingWithPlayLock()))
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.AddReplacementEffect(&deathReturnDestinationReplacement{destination: destination})
			if sacrifice {
				g.DoSacrifice(p)
			} else {
				g.DestroyPermanent(p)
			}
			if len(a.Hand()) != 0 || g.IsSpellCastProhibited(a.PlayerID(), card) || g.IsHandCardRevealed(a.PlayerID(), card.ID()) || (len(a.Graveyard()) == 1) != (destination == core.ZoneGraveyard) || (g.FindExiledCard(card.ID()) != nil) != (destination == core.ZoneExile) {
				t.Fatal("replacement recheck repeated return or installed an uncompleted hand result")
			}
		}
	}
}

func TestDeathReturnHandSimultaneousAndSkippedTurn(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(1)
	for range 8 {
		a.AddToLibrary(NewLand("A"))
		b.AddToLibrary(NewLand("B"))
	}
	first := g.PutOnBattlefield(NewCreature("First", "", 3, 2, WithAbility(ReturnToHandInsteadOfDyingWithPlayLock())), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 3, 2, WithAbility(ReturnToHandInsteadOfDyingWithPlayLock())), a.PlayerID())
	probe := &deathExileBatchProbe{first: first.ID(), second: second.ID(), t: t}
	g.AddReplacementEffect(probe)
	g.DestroyPermanents([]*Permanent{first, second})
	if probe.calls != 2 || len(a.Hand()) != 2 || !g.IsHandCardRevealed(a.PlayerID(), first.Card.ID()) || !g.IsHandCardRevealed(a.PlayerID(), second.Card.ID()) || g.CreatureDeaths() != 0 {
		t.Fatal("batch choices or independent hand results")
	}
	g.SkipNextTurnFor(a.PlayerID())
	g.AdvanceToNextTurn()
	g.Run(0, core.Untap, 4)
	if !g.IsSpellCastProhibited(a.PlayerID(), first.Card) {
		t.Fatal("skipped turn expired the lock")
	}
	g.Run(0, core.Untap, 5)
	if g.IsHandCardRevealed(a.PlayerID(), first.Card.ID()) || g.IsHandCardRevealed(a.PlayerID(), second.Card.ID()) {
		t.Fatal("actual next turn retained independent locks")
	}
}

func TestDeathReturnHandVisitAndExpiry(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(1)
	g.SetStep(core.PrecombatMain)
	card := NewCreature("Subject", "{R}", 3, 2, WithAbility(ReturnToHandInsteadOfDyingWithPlayLock()))
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.DestroyPermanent(p)
	other := card.Copy()
	a.AddToHand(other)
	if g.IsSpellCastProhibited(a.PlayerID(), other) || g.IsHandCardRevealed(a.PlayerID(), other.ID()) {
		t.Fatal("lock matched another physical card")
	}
	clone := g.Clone()
	g.SetTurn(2)
	g.SetActivePlayerIndex(1)
	g.BeginActiveTurn()
	if !g.IsSpellCastProhibited(a.PlayerID(), card) {
		t.Fatal("other player's turn ended the lock")
	}
	g.SetTurn(3)
	g.SetActivePlayerIndex(0)
	g.BeginActiveTurn()
	if g.IsSpellCastProhibited(a.PlayerID(), card) || g.IsHandCardRevealed(a.PlayerID(), card.ID()) {
		t.Fatal("owner's next turn retained the lock")
	}
	if !clone.IsSpellCastProhibited(a.PlayerID(), card) {
		t.Fatal("expiry changed the clone")
	}
	cp := clone.GetPlayer(a.PlayerID())
	departed, ok := cp.RemoveFromHand(card.ID())
	if !ok {
		t.Fatal("missing returned card")
	}
	cp.AddToHand(departed)
	if clone.IsSpellCastProhibited(a.PlayerID(), departed) || clone.IsHandCardRevealed(a.PlayerID(), departed.ID()) {
		t.Fatal("new hand visit inherited the lock")
	}
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
