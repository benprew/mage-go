package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestResolutionLifePayment(t *testing.T) {
	for _, tc := range []struct {
		name         string
		life, amount int
		accept, paid bool
		offers       int
	}{
		{"pay", 20, 3, true, true, 1},
		{"decline", 20, 3, false, false, 1},
		{"short", 2, 3, true, false, 0},
		{"exact", 3, 3, true, true, 1},
		{"zero-pay", -2, 0, true, true, 1},
		{"zero-decline", 0, 0, false, false, 1},
		{"negative", 20, -3, true, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, ifPaid := range []bool{false, true} {
				a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: tc.accept}
				b := NewBasePlayer("B")
				g := NewGame(a, b)
				a.SetLife(tc.life)
				body := GainLifeFor(ActivePlayer, Fixed(7))
				effect := UnlessLifePaid(You, Fixed(tc.amount), body)
				if ifPaid {
					effect = IfLifePaid(You, Fixed(tc.amount), body)
				}
				if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want := tc.life
				if tc.paid {
					want -= max(0, tc.amount)
				}
				if tc.paid == ifPaid {
					want += 7
				}
				if a.Life() != want || a.offers != tc.offers || b.Life() != 20 {
					t.Fatalf("life=%d want=%d offers=%d want=%d", a.Life(), want, a.offers, tc.offers)
				}
			}
		})
	}
}

type lifeChoicePlayer struct {
	*BasePlayer
	choose func() bool
}

func (p *lifeChoicePlayer) ChooseMayAbility(string) bool { return p.choose() }

type countedLifeValue struct {
	player *BasePlayer
	reads  *int
}

func (countedLifeValue) Text() string                      { return "half life" }
func (v countedLifeValue) resolveValue(*EffectContext) int { *v.reads++; return v.player.Life() / 2 }

func TestResolutionLifePaymentFixedBeforeChoice(t *testing.T) {
	for _, short := range []bool{false, true} {
		a := &lifeChoicePlayer{BasePlayer: NewBasePlayer("A")}
		g := NewGame(a, NewBasePlayer("B"))
		reads := 0
		amount := countedLifeValue{a.BasePlayer, &reads}
		a.choose = func() bool {
			if short {
				a.SetLife(1)
			} else {
				a.SetLife(30)
			}
			return true
		}
		if err := ApplyEffect(g, UnlessLifePaid(You, amount, GainLifeFor(You, Fixed(2))), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := 20
		if short {
			want = 3
		}
		if reads != 1 || a.Life() != want {
			t.Fatalf("reads=%d life=%d want=%d", reads, a.Life(), want)
		}
	}
}

func TestResolutionLifePaymentDeclarations(t *testing.T) {
	result := BindCard("reveal")
	body := GainLifeFor(You, Fixed(1))
	for _, effect := range []ResolvingEffect{
		UnlessLifePaid(nil, Fixed(1), body),
		UnlessLifePaid(You, nil, body),
		IfLifePaid(You, Fixed(1), nil),
		UnlessLifePaid(You, RevealedCardManaValue(nil), body),
		UnlessLifePaid(You, RevealedCardManaValue(result), body),
		Sequence(UnlessLifePaid(You, RevealedCardManaValue(result), body), RevealRandomHandCardInto(You, result)),
	} {
		c := NewInstant("Invalid", "", Spell(effect))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	c := NewInstant("Valid", "", Spell(Sequence(RevealRandomHandCardInto(You, result), IfLifePaid(You, RevealedCardManaValue(result), body))))
	if err := ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
}

func TestRevealedCardManaValue(t *testing.T) {
	for _, producer := range []string{"random-hand", "saved-hand", "library"} {
		t.Run(producer, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Held", "{X}{X}{3}{U}")
			card.SetOwner(a.PlayerID())
			result := BindCard("reveal")
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			value := RevealedCardManaValue(result)
			if value.resolveValue(ctx) != 0 {
				t.Fatal("missing reveal has a value")
			}
			var effect ResolvingEffect
			if producer == "library" {
				a.SetLibrary([]Card{card})
				effect = RevealLibraryTopInto(You, result)
			} else {
				a.AddToHand(card)
				effect = RevealRandomHandCardInto(You, result)
				if producer == "saved-hand" {
					input := BindCard("input")
					g.resolution.cardBindings = map[*CardBinding]capturedCard{input: {id: card.ID(), visit: g.handVisit(card.ID()), zone: core.ZoneHand}}
					effect = RevealHandCardInto(FromCard(input), result)
				}
			}
			g.SetXValue(9)
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if value.resolveValue(ctx) != 4 {
				t.Fatal("reveal included enclosing X")
			}
			clone := g.Clone()
			card.manaCost = core.ParseManaCost("{9}")
			if producer == "library" {
				a.SetLibrary(nil)
				a.SetLibrary([]Card{card})
			} else {
				a.RemoveFromHand(card.ID())
				a.AddToHand(card)
			}
			if value.resolveValue(ctx) != 4 {
				t.Fatal("saved value followed new characteristics or visit")
			}
			if _, ok := FromCard(result).resolveCard(ctx); ok {
				t.Fatal("saved facts permitted a returned-card action")
			}
			if producer == "library" {
				a.SetLibrary(nil)
			} else {
				a.RemoveFromHand(card.ID())
			}
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if value.resolveValue(ctx) != 0 || value.resolveValue(&EffectContext{Game: clone, Controller: a.PlayerID()}) != 4 {
				t.Fatal("clearing leaked across clone frames")
			}
		})
	}
}

func TestResolutionLifePaymentWaitingAndFrames(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phased", "cycle"} {
		t.Run(state, func(t *testing.T) {
			a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			result := BindCard("reveal")
			delay := AtNext(EachEndStep, IfLifePaid(ControllerOfWithLKI(This), RevealedCardManaValue(result), GainLifeFor(You, Fixed(1))))
			card := NewArtifact("Source", "", WithAbility(Activate(Costs(), Sequence(RevealRandomHandCardInto(You, result), delay))))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			held := NewArtifact("Held", "{3}")
			held.SetOwner(a.PlayerID())
			a.AddToHand(held)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			a.RemoveFromHand(held.ID())
			a.AddToHand(held)
			switch state {
			case "departed", "returned":
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			for range 2 {
				refs := g.WaitingDelayedPermanentSources()
				if len(refs) != 1 || !refs[0].SameObject(saved) || a.offers != 0 || a.Life() != 20 || len(g.PublicReveals()) != 1 {
					t.Fatal("waiting query executed payment or lost exact source")
				}
			}
			clone := g.Clone()
			ca := &countedMayPlayer{BasePlayer: clone.GetPlayer(a.PlayerID()).(*SearchPlayer).BasePlayer, accept: false}
			clone.SetPlayerAt(0, ca)
			for _, game := range []*Game{g, clone} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
			}
			if a.Life() != 18 || a.offers != 1 || ca.Life() != 20 || ca.offers != 1 || b.Life() != 20 {
				t.Fatal("delayed facts, original controller, or cloned choices changed")
			}
		})
	}
	for _, location := range []string{"amount", "player", "body"} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		p := g.PutOnBattlefield(NewCreature("Source", "", 0, 1), a.PlayerID())
		player, amount, body := PlayerRef(You), ResolutionValue(Fixed(0)), GainLifeFor(You, Fixed(0))
		switch location {
		case "amount":
			amount = PowerOfWithLKI(This)
		case "player":
			player = ControllerOfWithLKI(This)
		case "body":
			body = TapSubject(This)
		}
		if err := ApplyEffect(g, AtNext(EachEndStep, UnlessLifePaid(player, amount, body)), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(g.WaitingDelayedPermanentSources()) != 1 || p.Tapped || a.Life() != 20 {
			t.Fatal("waiting declaration lost operand or executed body")
		}
	}
}

func TestResolutionLifePaymentAbsentAndIndependent(t *testing.T) {
	a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := BindCard("first"), BindCard("second")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	c := NewArtifact("First", "{2}")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := RevealRandomHandCardInto(You, first).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	a.RemoveFromHand(c.ID())
	d := NewArtifact("Second", "{7}")
	d.SetOwner(a.PlayerID())
	a.AddToHand(d)
	if err := RevealRandomHandCardInto(You, second).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := IfLifePaid(You, RevealedCardManaValue(first), GainLifeFor(You, Fixed(1))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := UnlessLifePaid(You, RevealedCardManaValue(second), GainLifeFor(You, Fixed(100))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 12 || a.offers != 2 {
		t.Fatal("independent payments shared amount or result")
	}
	if err := ApplyEffect(g, UnlessLifePaid(You, Fixed(2), GainLifeFor(ActivePlayer, Fixed(100))), uuid.Nil, uuid.New(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 12 || a.offers != 2 {
		t.Fatal("absent payer executed a branch")
	}
	if err := ApplyEffect(g, GainLifeFor(You, RevealedCardManaValue(first)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 12 || RevealedCardManaValue(first).resolveValue(ctx) != 2 {
		t.Fatal("nested frame read or cleared outer reveal facts")
	}
}
