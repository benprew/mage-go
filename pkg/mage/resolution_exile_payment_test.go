package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestResolutionExilePayment(t *testing.T) {
	for _, state := range []string{"paid", "declined", "zero", "empty", "control", "phase", "phase-cycle", "orphan", "source-return", "redirect", "cancel"} {
		t.Run(state, func(t *testing.T) {
			a := &resolutionCounterPayer{BasePlayer: NewBasePlayer("A"), accept: state != "declined", choice: 2}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			link := NewCreatedTokenLink()
			count := BindNumber("exiled")
			set := TokensCreatedWith(link, This)
			payment := PayChosenExileOrElse(You, set, count, GainLifeFor(You, AddValues(Fixed(1), FromNumber(count))), GainLifeFor(You, Fixed(7)))
			card := NewArtifact("Maker", "", WithAbility(Activate(Costs(), payment)))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: source.ID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
			create := CreateTokenCopiesLinked(You, Fixed(2), DefineToken("Member", TokenPT(1, 1), TokenTypes(core.TypeCreature)), link)
			if err := create.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			members := set.resolvePermanents(ctx)
			other := g.PutOnBattlefield(NewArtifact("Other maker", ""), a.PlayerID())
			otherCtx := *ctx
			otherCtx.SourceID = other.ID()
			otherCtx.sourceIncarnation = other.incarnationID
			if err := create.Apply(&otherCtx); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "zero":
				a.choice = 0
			case "empty":
				g.ExilePermanents(members)
			case "control":
				for _, p := range members {
					g.AddControlEffect(ControlEffectSpec{SourceID: other.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
			case "phase":
				g.PhaseOut(members[0].ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(members[0].ID()))
			case "orphan", "source-return":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if state == "source-return" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "redirect":
				g.AddReplacementEffect(&exileCompetingReplacement{})
			case "cancel":
				g.AddReplacementEffect(&exileCompetingReplacement{cancel: true})
			}
			clone := g.Clone()
			g.ResolveStack()
			wantLife, wantTokens := 23, 2
			switch state {
			case "declined":
				wantLife, wantTokens = 27, 4
			case "zero", "cancel":
				wantLife, wantTokens = 21, 4
			case "empty", "redirect":
				wantLife, wantTokens = 21, 2
			case "phase":
				wantLife, wantTokens = 22, 2
			}
			if a.Life() != wantLife || len(g.FilterBattlefield(IsToken)) != wantTokens {
				t.Fatalf("life %d tokens %d; want %d %d", a.Life(), len(g.FilterBattlefield(IsToken)), wantLife, wantTokens)
			}
			if len(g.resolution.numberBindings) != 0 {
				t.Fatal("payment result escaped its branch")
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone shared payment")
			}
			if len(TokensCreatedWith(link, This).resolvePermanents(&EffectContext{Game: g, Controller: a.PlayerID(), SourceID: other.ID()})) != 2 {
				t.Fatal("other producer paid")
			}
		})
	}
}

func TestResolutionExilePaymentSelectionLock(t *testing.T) {
	for _, returned := range []bool{false, true} {
		a := &resolutionCounterPayer{BasePlayer: NewBasePlayer("A"), accept: true, choice: 2}
		g := NewGame(a, NewBasePlayer("B"))
		card := NewArtifact("First", "")
		card.SetOwner(a.PlayerID())
		first := g.PutOnBattlefield(card, a.PlayerID())
		second := g.PutOnBattlefield(NewArtifact("Second", ""), a.PlayerID())
		a.onNumber = func() {
			if returned {
				g.BouncePermanentToHand(first)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			} else {
				g.PhaseOut(first.ID())
			}
		}
		result := BindNumber("exiled")
		e := PayChosenExileOrElse(You, MatchingPermanents(IsArtifact), result, GainLifeFor(You, Fixed(1)), GainLifeFor(You, Fixed(7)))
		if err := e.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 27 || g.FindPermanent(second.ID()) == nil {
			t.Fatal("invalid selection made a partial payment")
		}
	}
}

func TestResolutionExilePaymentInvalid(t *testing.T) {
	result := BindNumber("exiled")
	for _, e := range []ResolvingEffect{
		PayChosenExileOrElse(nil, MatchingPermanents(IsArtifact), result, GainLifeFor(You, FromNumber(result)), nil),
		PayChosenExileOrElse(You, MatchingPermanents(IsArtifact), BindNumber(""), GainLifeFor(You, Fixed(1)), nil),
		PayChosenExileOrElse(You, nil, result, GainLifeFor(You, Fixed(1)), nil),
		PayChosenExileOrElse(You, MatchingPermanents(IsArtifact), nil, GainLifeFor(You, Fixed(1)), nil),
		PayChosenExileOrElse(You, MatchingPermanents(IsArtifact), result, nil, nil),
		PayChosenExileOrElse(You, MatchingPermanents(IsArtifact), result, GainLifeFor(You, Fixed(1)), GainLifeFor(You, FromNumber(result))),
		Sequence(PayChosenExileOrElse(You, MatchingPermanents(IsArtifact), result, GainLifeFor(You, Fixed(1)), nil), GainLifeFor(You, FromNumber(result))),
	} {
		if ValidateCardDefinition("Bad", NewSorcery("Bad", "", Spell(e))) == nil {
			t.Fatal("invalid payment accepted")
		}
	}
}

func TestResolutionExilePaymentBatchAndReplacementOrder(t *testing.T) {
	a := &resolutionCounterPayer{BasePlayer: NewBasePlayer("A"), accept: true, choice: 2}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 1), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 1), a.PlayerID())
	first.IsToken = true
	calls := 0
	watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(evt *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if evt.FromZone == core.ZoneBattlefield && evt.ToZone == core.ZoneExile {
			calls++
			if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil {
				t.Error("partial payment visible to events")
			}
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), a.PlayerID())
	var order []uuid.UUID
	g.AddReplacementEffect(&exileOrderProbe{order: &order})
	result := BindNumber("exiled")
	body := PayChosenExileOrElse(You, UnionPermanents(MatchingPermanents(IsCreature), MatchingPermanents(IsCreature)), result, GainLifeFor(You, FromNumber(result)), nil)
	if err := body.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || a.Life() != 22 || len(order) != 2 || order[0] != second.ID() || order[1] != first.ID() {
		t.Fatal("batch count, duplicate visit or replacement order")
	}
}

func TestResolutionExilePaymentCompetingReplacements(t *testing.T) {
	for _, index := range []int{0, 1} {
		a := &resolutionCounterPayer{BasePlayer: NewBasePlayer("A"), accept: true, choice: 1}
		b := &deathExileOrderPlayer{BasePlayer: NewBasePlayer("B"), index: index}
		g := NewGame(a, b)
		p := g.PutOnBattlefield(NewCreature("Subject", "", 1, 1), b.PlayerID())
		g.AddReplacementEffect(&exileCompetingReplacement{cancel: true})
		g.AddReplacementEffect(&exileCompetingReplacement{})
		result := BindNumber("exiled")
		body := PayChosenExileOrElse(You, MatchingPermanents(IsCreature), result, GainLifeFor(You, AddValues(Fixed(1), FromNumber(result))), GainLifeFor(You, Fixed(7)))
		if err := body.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if b.choices != 1 || a.Life() != 21 || (g.FindPermanent(p.ID()) != nil) != (index == 0) {
			t.Fatal("replacement choice or paid outcome")
		}
	}
}

func TestResolutionExilePaymentDelayedSource(t *testing.T) {
	a := &resolutionCounterPayer{BasePlayer: NewBasePlayer("A"), accept: true, choice: 2}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Maker", ""), a.PlayerID())
	link := NewCreatedTokenLink()
	result := BindNumber("exiled")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: source.ID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	if err := CreateTokenCopiesLinked(You, Fixed(2), DefineToken("Member", TokenPT(1, 1), TokenTypes(core.TypeCreature)), link).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	payment := PayChosenExileOrElse(You, TokensCreatedWith(link, This), result, GainLifeFor(You, FromNumber(result)), nil)
	if !effectReadsSourcePermanent(payment) {
		t.Fatal("waiting payment lost source read")
	}
	if err := AtNext(EachEndStep, payment).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.mayCalls != 0 || len(g.FilterBattlefield(IsToken)) != 2 {
		t.Fatal("waiting payment executed early")
	}
	g.ExilePermanent(g.FindPermanent(source.ID()))
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 22 || len(g.FilterBattlefield(IsToken)) != 0 {
		t.Fatal("delayed source link lost")
	}
}
