package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestFutureDrawReplacementOnceAndFrames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(3)
	seedFutureDrawLibrary(a)
	seedFutureDrawLibrary(b)
	bound := BindPlayer("saved player")
	number := BindNumber("saved amount")
	g.resolution.playerBindings = map[*PlayerBinding]uuid.UUID{bound: a.PlayerID()}
	g.resolution.numberBindings = map[*NumberBinding]int{number: 3}
	g.resolution.SetX(4)
	body := Sequence(GainLifeFor(FromPlayer(bound), FromNumber(number)), DrawFor(You, Fixed(1)))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := ReplaceNextDrawThisTurn(FromPlayer(bound), body).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.resolution.playerBindings[bound] = b.PlayerID()
	g.resolution.numberBindings[number] = 9
	g.resolution.SetX(12)
	g.PlayerDrawCard(b)
	if len(b.Hand()) != 1 || a.Life() != 20 {
		t.Fatal("wrong player consumed replacement")
	}
	clone := g.Clone()
	g.PlayerDrawCard(a)
	if a.Life() != 23 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("replacement did not finish immediately or recursed")
	}
	if g.resolution.playerBindings[bound] != b.PlayerID() || g.resolution.numberBindings[number] != 9 || g.resolution.X() != 12 {
		t.Fatal("replacement changed outer frame")
	}
	clone.PlayerDrawCard(clone.GetPlayer(a.PlayerID()))
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || a.Life() != 23 {
		t.Fatal("clone shared replacement use or bindings")
	}
	g.PlayerDrawCard(a)
	if a.Life() != 23 || len(a.Hand()) != 2 {
		t.Fatal("replacement used twice")
	}
}

func seedFutureDrawLibrary(p *BasePlayer) {
	for range 5 {
		p.AddToLibrary(NewLand("Draw"))
	}
}

func TestFutureDrawReplacementLifecycle(t *testing.T) {
	for _, state := range []string{"control", "phase", "depart", "return", "remove", "expire"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetTurn(3)
			seedFutureDrawLibrary(a)
			card := NewCreature("Source", "", 3, 4)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			body := Sequence(GainLifeFor(You, PowerOfWithLKI(This)), TapSubject(This))
			if err := ApplyEffect(g, ReplaceNextDrawThisTurn(You, body), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseOut(p.ID())
			case "depart", "return":
				g.DestroyPermanent(p)
				if state == "return" {
					p = g.PutOnBattlefield(card, b.PlayerID())
					g.AddContinuousEffect(TemporaryBoost(p.ID(), 5, 0))
				}
			case "remove":
				g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.Indefinite))
			case "expire":
				g.RunStepWithPriority(core.Cleanup)
			}
			g.PlayerDrawCard(a)
			if state == "expire" {
				if a.Life() != 20 || len(a.Hand()) != 1 {
					t.Fatal("replacement survived cleanup")
				}
				return
			}
			if a.Life() != 23 || b.Life() != 20 || len(a.Hand()) != 0 {
				t.Fatal("saved controller or source information lost", a.Life())
			}
			if (state == "return" || state == "phase") && p.Tapped {
				t.Fatal("body acted on returned or phased source")
			}
		})
	}
}

func TestFutureDrawReplacementOrdering(t *testing.T) {
	for _, choice := range []int{0, 1} {
		a := &drawReplacementChooser{BasePlayer: NewBasePlayer("A"), choice: choice}
		g := NewGame(a, NewBasePlayer("B"))
		g.SetTurn(3)
		seedFutureDrawLibrary(a.BasePlayer)
		for _, amount := range []int{1, 10} {
			if err := ApplyEffect(g, ReplaceNextDrawThisTurn(You, Sequence(GainLifeFor(You, Fixed(amount)), DrawFor(You, Fixed(1)))), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
		}
		g.PlayerDrawCard(a)
		if a.Life() != 31 || len(a.Hand()) != 1 || a.offers != 1 {
			t.Fatal("independent replacements or ordering", a.Life(), len(a.Hand()), a.offers)
		}
		g.PlayerDrawCard(a)
		if a.Life() != 31 || len(a.Hand()) != 2 {
			t.Fatal("consumed rule repeated")
		}
	}
}

func TestFutureDrawReplacementCompetition(t *testing.T) {
	for _, choice := range []int{0, 1} {
		a := &drawReplacementChooser{BasePlayer: NewBasePlayer("A"), choice: choice}
		g := NewGame(a, NewBasePlayer("B"))
		g.SetTurn(3)
		g.SetStep(core.Draw)
		seedFutureDrawLibrary(a.BasePlayer)
		player := BindPlayer("drawing player")
		g.PutOnBattlefield(NewEnchantment("Static rule", "", WithAbility(Static(ReplaceDraw(ControllerDrawStep, player, Sequence(GainLifeFor(FromPlayer(player), Fixed(10)), DrawFor(FromPlayer(player), Fixed(1))))))), a.PlayerID())
		if err := ApplyEffect(g, ReplaceNextDrawThisTurn(You, GainLifeFor(You, Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.PlayerDrawCard(a)
		want := 21
		if choice == 1 {
			want = 31
		}
		if a.Life() != want || len(a.Hand()) != 0 || a.offers != 1 {
			t.Fatal("affected player order or static ancestry", a.Life(), want)
		}
		g.PlayerDrawCard(a)
		if a.Life() != want+10 || len(a.Hand()) != 1 {
			t.Fatal("remaining static rule lost or future rule repeated")
		}
	}
}

func TestFutureDrawReplacementSelectedRecheck(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(3)
	seedFutureDrawLibrary(a)
	if err := ApplyEffect(g, ReplaceNextDrawThisTurn(You, GainLifeFor(You, Fixed(2))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	action := g.drawAction(a.PlayerID(), false)
	selected := g.effects.replacementCandidates(action, g, nil)[0]
	other := g.drawAction(b.PlayerID(), false)
	if got := selected.Replace(other, g); got != other || !selected.IsActive(g) {
		t.Fatal("wrong event consumed selected rule")
	}
	g.PlayerDrawCard(a)
	if got := selected.Replace(action, g); got != action || a.Life() != 22 {
		t.Fatal("selected consumed rule applied twice")
	}
}

func TestFutureDrawReplacementUnchosenAndSequence(t *testing.T) {
	for _, choice := range []int{0, 1} {
		a := &drawReplacementChooser{BasePlayer: NewBasePlayer("A"), choice: choice}
		g := NewGame(a, NewBasePlayer("B"))
		g.SetTurn(3)
		seedFutureDrawLibrary(a.BasePlayer)
		for _, amount := range []int{1, 10} {
			if err := ApplyEffect(g, ReplaceNextDrawThisTurn(You, GainLifeFor(You, Fixed(amount))), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
		}
		g.PlayerDrawCard(a)
		want := 21
		if choice == 1 {
			want = 30
		}
		if a.Life() != want || len(a.Hand()) != 0 {
			t.Fatal("wrong one-shot chosen")
		}
		if err := ApplyEffect(g, DrawFor(You, Fixed(2)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 31 || len(a.Hand()) != 1 {
			t.Fatal("unchosen rule lost or draw sequence interrupted")
		}
	}
}

func TestFutureDrawReplacementEmptyAndResult(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	g.SetTurn(3)
	if err := ApplyEffect(g, ReplaceNextDrawThisTurn(You, GainLifeFor(You, Fixed(2))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PlayerDrawCard(a)
	if a.Life() != 22 || a.DrewFromEmpty() {
		t.Fatal("empty library bypassed replacement")
	}
	seedFutureDrawLibrary(a)
	result := BindCard("drawn card")
	if err := ApplyEffect(g, ReplaceNextDrawThisTurn(You, DrawFor(You, Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := DrawFor(You, Fixed(1), DrawCardInto(result)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 || len(g.resolution.cardBindings) != 0 {
		t.Fatal("replacement draw supplied original card result")
	}
}

func TestFutureDrawReplacementInvalid(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		ReplaceNextDrawThisTurn(nil, GainLifeFor(You, Fixed(1))),
		ReplaceNextDrawThisTurn(You, nil),
		ReplaceNextDrawThisTurn(FromPlayer(BindPlayer("absent")), GainLifeFor(You, Fixed(1))),
		ReplaceNextDrawThisTurn(You, GainLifeFor(FromPlayer(BindPlayer("absent")), Fixed(1))),
	} {
		card := NewInstant("Invalid", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}
