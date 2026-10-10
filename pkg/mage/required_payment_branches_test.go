package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestRequiredPaymentBranchesDiscard(t *testing.T) {
	for _, mode := range []string{"paid", "empty", "replaced", "prevented", "rollback", "clone"} {
		t.Run(mode, func(t *testing.T) {
			a := &requiredPaymentPlayer{BasePlayer: NewBasePlayer("A")}
			b := &requiredPaymentPlayer{BasePlayer: NewBasePlayer("B")}
			g := NewGame(a, b)
			card := NewLand("Payment")
			card.SetOwner(b.PlayerID())
			if mode != "empty" {
				b.AddToHand(card)
			}
			costs := Costs(DiscardCost(1))
			if mode == "rollback" {
				costs = Costs(DiscardCost(1), DiscardCost(1))
			}
			if mode == "replaced" {
				g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
			}
			if mode == "prevented" {
				g.effects.AddReplacement(&preventCostDiscardReplacement{})
			}
			effect := PayRequiredOrElse(ControllerOf(This), costs, GainLifeFor(You, Fixed(3)), GainLifeFor(You, Fixed(7)))
			source := NewArtifact("Source", "", WithAbility(Activate(Costs(), effect)))
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(source, b.PlayerID())
			run := g
			if mode == "clone" {
				run = g.Clone()
			}
			if err := ApplyEffect(run, effect, p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			paid := mode == "paid" || mode == "replaced" || mode == "clone"
			want := 27
			if paid {
				want = 23
			}
			if run.GetPlayer(a.PlayerID()).Life() != want || run.GetPlayer(b.PlayerID()).Life() != 20 || a.offers != 0 || b.offers != 0 {
				t.Fatal("wrong branch, payer, or optional offer")
			}
			if paid && len(run.GetPlayer(b.PlayerID()).Hand()) != 0 {
				t.Fatal("payment did not discard")
			}
			if !paid && mode != "empty" && (len(b.Hand()) != 1 || len(b.Graveyard()) != 0 || g.PlayerDiscardCountThisTurn(b.PlayerID()) != 0) {
				t.Fatal("failed payment changed the hand or history")
			}
			if mode == "replaced" && (len(b.Library()) != 1 || len(b.Graveyard()) != 0) {
				t.Fatal("replacement destination")
			}
			if mode == "clone" && (a.Life() != 20 || len(b.Hand()) != 1 || len(b.Graveyard()) != 0) {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestRequiredPaymentBranchesNested(t *testing.T) {
	a := &requiredPaymentPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	a.AddToHand(NewLand("Only"))
	inner := PayRequiredOrElse(You, Costs(DiscardCost(1)), GainLifeFor(You, Fixed(100)), GainLifeFor(You, Fixed(2)))
	effect := PayRequiredOrElse(You, Costs(DiscardCost(1)), Sequence(inner, GainLifeFor(You, Fixed(3))), GainLifeFor(You, Fixed(100)))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 25 || a.offers != 0 || len(a.Graveyard()) != 1 {
		t.Fatal("nested result escaped")
	}
}

func TestRequiredPaymentBranchesInvalid(t *testing.T) {
	body := GainLifeFor(You, Fixed(1))
	for _, effect := range []ResolvingEffect{
		PayRequiredOrElse(nil, Costs(DiscardCost(1)), body, body),
		PayRequiredOrElse(You, Costs(), body, body),
		PayRequiredOrElse(You, Costs(DiscardCost(-1)), body, body),
		PayRequiredOrElse(You, Costs(DiscardCost(1)), nil, body),
		PayRequiredOrElse(You, Costs(DiscardCost(1)), body, nil),
		PayRequiredOrElse(You, Costs(ManaCostOf("{1}")), body, body),
		PayRequiredOrElse(You, Costs(DiscardCostWithCards(1, BindCards("paid"))), body, body),
	} {
		card := NewInstant("Invalid", "", Spell(effect))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid declaration")
		}
	}
}

func TestRequiredPaymentBranchesReplacementOrder(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	a.AddToHand(NewLand("Only"))
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	g.effects.AddReplacement(&discardToHandReplacement{})
	effect := PayRequiredOrElse(You, Costs(DiscardCost(1)), GainLifeFor(You, Fixed(100)), GainLifeFor(You, Fixed(2)))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || len(a.Hand()) != 1 || len(a.Library()) != 0 || len(a.Graveyard()) != 0 {
		t.Fatal("replacement recheck or rollback")
	}

}

func TestRequiredPaymentBranchesEffectDiscard(t *testing.T) {
	for _, mode := range []string{"paid", "empty", "prevented", "library", "clone"} {
		t.Run(mode, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			if mode != "empty" {
				a.AddToHand(NewLand("Only"))
			}
			if mode == "prevented" {
				g.effects.AddReplacement(&discardToHandReplacement{})
				g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
			}
			if mode == "library" {
				g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
			}
			effect := DiscardOneOrElse(You, GainLifeFor(You, Fixed(3)), GainLifeFor(You, Fixed(7)))
			c := NewInstant("Action", "", Spell(effect))
			if err := ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			run := g
			if mode == "clone" {
				run = g.Clone()
			}
			if err := ApplyEffect(run, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			want := 23
			if mode == "empty" || mode == "prevented" {
				want = 27
			}
			if run.GetPlayer(a.PlayerID()).Life() != want {
				t.Fatal("actual discard result")
			}
			if mode == "library" && (len(a.Library()) != 1 || len(a.Graveyard()) != 0) {
				t.Fatal("library destination must count as discarded")
			}
			if mode == "clone" && (a.Life() != 20 || len(a.Hand()) != 1) {
				t.Fatal("clone state")
			}
		})
	}
}

func TestRequiredPaymentBranchesEffectDiscardInvalid(t *testing.T) {
	body := GainLifeFor(You, Fixed(1))
	for _, effect := range []ResolvingEffect{DiscardOneOrElse(nil, body, body), DiscardOneOrElse(You, nil, nil)} {
		card := NewInstant("Invalid", "", Spell(effect))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid discard branch")
		}
	}
	saved := BindNumber("one branch")
	card := NewInstant("Invalid flow", "", Spell(DiscardOneOrElse(You, SnapshotNumber(Fixed(3), saved), GainLifeFor(You, FromNumber(saved)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("branch result escaped to sibling")
	}
}

type requiredDiscardVisitPlayer struct {
	*BasePlayer
	mode string
}

func (p *requiredDiscardVisitPlayer) ChooseCardsFromHand(_ int, _ string, _ GameReader) []Card {
	c := p.Hand()[0]
	switch p.mode {
	case "foreign":
		return []Card{NewLand("Foreign")}
	case "nil":
		return []Card{nil}
	case "multiple":
		return []Card{c, c}
	case "returned":
		p.RemoveFromHand(c.ID())
		p.AddToHand(c)
	}
	return []Card{c}
}

func TestRequiredPaymentBranchesDiscardVisit(t *testing.T) {
	for _, mode := range []string{"foreign", "nil", "multiple", "returned"} {
		t.Run(mode, func(t *testing.T) {
			a := &requiredDiscardVisitPlayer{BasePlayer: NewBasePlayer("A"), mode: mode}
			g := NewGame(a, NewBasePlayer("B"))
			a.AddToHand(NewLand("Only"))
			effect := DiscardOneOrElse(You, GainLifeFor(You, Fixed(100)), GainLifeFor(You, Fixed(7)))
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 27 || len(a.Hand()) != 1 || len(a.Graveyard()) != 0 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 {
				t.Fatal("invalid selection discarded a different hand visit")
			}
		})
	}
}

func TestRequiredPaymentBranchesWaitingReads(t *testing.T) {
	a := &requiredPaymentPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	g.SetTurn(3)
	source := g.PutOnBattlefield(NewCreature("Source", "", 3, 4), a.PlayerID())
	saved := g.CaptureDamageSource(source.ID())
	body := DiscardOneOrElse(You, GainLifeFor(You, PowerOfWithLKI(This)), TapSubject(This))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(source)
	for range 2 {
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(saved) || a.Life() != 20 || len(a.Hand()) != 1 || a.offers != 0 {
			t.Fatal("waiting query changed state or lost source visit")
		}
	}
}
