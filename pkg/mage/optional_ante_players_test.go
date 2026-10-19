package mage

import (
	"math"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestOptionalAntePlayersBatch(t *testing.T) {
	for _, active := range []int{0, 1} {
		a := &antePaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
		b := &antePaymentPlayer{BasePlayer: NewBasePlayer("B"), accept: true}
		g, err := NewGameWithAnte(a, b, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		g.SetActivePlayerIndex(active)
		a.SetLibrary([]Card{ownedTestCard(a.PlayerID(), "A top")})
		b.SetLibrary([]Card{ownedTestCard(b.PlayerID(), "B top")})
		var order []uuid.UUID
		for _, p := range []*antePaymentPlayer{a, b} {
			p.onOffer = func() {
				order = append(order, p.PlayerID())
				if len(a.Ante()) != 0 || len(b.Ante()) != 0 {
					t.Fatal("movement preceded choices")
				}
			}
		}
		result := BindPlayerSet("paid")
		ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
		events := 0
		g.PutOnBattlefield(NewArtifact("observer", "", WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
			if e.ToZone == core.ZoneAnte {
				events++
				if len(a.Ante()) != 1 || len(b.Ante()) != 1 || len(FromPlayerSet(result).resolvePlayers(ctx)) != 2 {
					t.Error("partial payment batch")
				}
			}
			return false
		}))), a.PlayerID())
		if err := MayAnteTopEach(EachPlayer, result).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		want := []uuid.UUID{a.PlayerID(), b.PlayerID()}
		if active == 1 {
			slices.Reverse(want)
		}
		if !slices.Equal(order, want) || events != 2 {
			t.Fatal("APNAP or batch events", order, events)
		}
		a.SetLife(5)
		b.SetLife(30)
		if err := SetLifeTotalEach(FromPlayerSet(result), Fixed(20)).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 20 || b.Life() != 20 {
			t.Fatal("life setting")
		}
		if err := MayAnteTopEach(EachPlayer, result).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if len(FromPlayerSet(result).resolvePlayers(ctx)) != 0 {
			t.Fatal("stale paid result")
		}
	}
}

func TestOptionalAntePlayersLegality(t *testing.T) {
	for _, state := range []string{"decline", "empty", "foreign", "token", "disabled", "settled", "stale"} {
		t.Run(state, func(t *testing.T) {
			a := &antePaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: state != "decline"}
			b := NewBasePlayer("B")
			g, _ := NewGameWithAnte(a, b, nil, nil)
			top := ownedTestCard(a.PlayerID(), "top")
			a.SetLibrary([]Card{top})
			switch state {
			case "empty":
				a.SetLibrary(nil)
			case "foreign":
				top.SetOwner(b.PlayerID())
			case "token":
				token := NewToken("token", 1, 1, []core.CardType{core.TypeCreature}, nil)
				token.SetOwner(a.PlayerID())
				a.SetLibrary([]Card{token})
			case "disabled":
				g.anteEnabled = false
			case "settled":
				g.anteSettled = true
			case "stale":
				a.onOffer = func() { a.SetLibrary([]Card{ownedTestCard(a.PlayerID(), "new top")}) }
			}
			saved := BindPlayerSet("paid")
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			if err := MayAnteTopEach(EachPlayer, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if len(a.Ante()) != 0 || len(FromPlayerSet(saved).resolvePlayers(ctx)) != 0 {
				t.Fatal("invalid payment succeeded")
			}
			if state != "stale" && state != "decline" && a.offers != 0 {
				t.Fatal("ineligible payment offered")
			}
		})
	}
}

func TestOptionalAntePlayersLifeRules(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.SetLife(5)
	b.SetLife(30)
	g.AddReplacementEffect(&lifeDrawRewrite{replace: func(gain *LifeGainAction, _ *Game) Action {
		return NewLifeGainAction(gain.ActionSource(), gain.PlayerID(), gain.Amount()*2)
	}})
	events := 0
	g.PutOnBattlefield(NewArtifact("observer", "", WithAbility(NewTriggered(core.EvtLifeGained, false).SetCondition(func(_ *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		events++
		if a.Life() != 35 || b.Life() != 20 {
			t.Error("partial life batch")
		}
		return false
	}))), a.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := SetLifeTotalEach(EachPlayer, Fixed(20)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 35 || b.Life() != 20 || events != 1 {
		t.Fatal("set total bypassed replacement")
	}
	g.effects.Rules.AddCantGainLife(b.PlayerID())
	g.effects.Rules.AddCantLoseLife(a.PlayerID())
	if err := SetLifeTotalEach(EachPlayer, Fixed(25)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 35 || b.Life() != 20 {
		t.Fatal("life restrictions bypassed")
	}
	clone := g.Clone()
	if err := SetLifeTotalEach(EachPlayer, Fixed(-2)).Apply(&EffectContext{Game: clone}); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != -2 {
		t.Fatal("clone isolation")
	}
}

func TestOptionalAntePlayersDefinitions(t *testing.T) {
	for _, e := range []ResolvingEffect{MayAnteTopEach(nil, BindPlayerSet("paid")), MayAnteTopEach(EachPlayer, nil), MayAnteTopEach(EachPlayer, &PlayerSetBinding{}), SetLifeTotalEach(FromPlayerSet(&PlayerSetBinding{}), Fixed(20)), SetLifeTotalEach(nil, Fixed(20)), SetLifeTotalEach(EachPlayer, nil), SetLifeTotalEach(FromPlayerSet(nil), Fixed(20)), SetLifeTotalEach(FromPlayerSet(BindPlayerSet("unbound")), Fixed(20))} {
		c := NewInstant("invalid", "", WithAbility(Spell(e)))
		if err := ValidateCardDefinition("invalid", c); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestOptionalAntePlayersSavedCloneAndOverflow(t *testing.T) {
	a := &antePaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g, _ := NewGameWithAnte(a, b, nil, nil)
	a.SetLibrary([]Card{ownedTestCard(a.PlayerID(), "top")})
	paid := BindPlayerSet("paid")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := MayAnteTopEach(EachPlayer, paid).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := MayAnteTopEach(EachPlayer, paid).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(FromPlayerSet(paid).resolvePlayers(ctx)) != 0 || len(FromPlayerSet(paid).resolvePlayers(&EffectContext{Game: clone})) != 1 {
		t.Fatal("clone shared paid membership")
	}
	a.SetLife(math.MinInt)
	b.SetLife(5)
	if err := SetLifeTotalEach(EachPlayer, Fixed(math.MaxInt)).Apply(ctx); err == nil {
		t.Fatal("overflow accepted")
	}
	if a.Life() != math.MinInt || b.Life() != 5 {
		t.Fatal("overflow made partial changes")
	}
}
