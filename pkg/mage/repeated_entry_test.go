package mage

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type repeatedEntryPlayer struct {
	*handEntryPlayer
	answers []bool
	order   *[]string
	onOffer func()
}

func (p *repeatedEntryPlayer) ChooseMayAbility(string) bool {
	p.offers++
	if p.order != nil {
		*p.order = append(*p.order, p.Name())
	}
	if p.onOffer != nil {
		p.onOffer()
	}
	if len(p.answers) == 0 {
		return false
	}
	answer := p.answers[0]
	p.answers = p.answers[1:]
	return answer
}
func repeatedEntryGame() (*Game, *repeatedEntryPlayer, *repeatedEntryPlayer) {
	a := &repeatedEntryPlayer{handEntryPlayer: &handEntryPlayer{BasePlayer: NewBasePlayer("A")}}
	b := &repeatedEntryPlayer{handEntryPlayer: &handEntryPlayer{BasePlayer: NewBasePlayer("B")}}
	return NewGame(a, b), a, b
}
func TestRepeatedEntryRounds(t *testing.T) {
	g, a, b := repeatedEntryGame()
	var order []string
	a.order, b.order = &order, &order
	a.answers, b.answers = []bool{false, true, false}, []bool{true, false, false}
	for _, p := range []*repeatedEntryPlayer{a, b} {
		for range 2 {
			card := NewLand("Land")
			card.SetOwner(p.PlayerID())
			p.AddToHand(card)
		}
	}
	g.SetActivePlayerIndex(1)
	result := BindNumber("round entries")
	body := EachPlayerMayPutFromHandOntoBattlefield(EachPlayer, You, IsPermanentCard, result)
	effect := RepeatWhilePositive(body, FromNumber(result))
	card := NewSorcery("Procedure", "", WithAbility(Spell(effect)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(order, []string{"A", "B", "A", "B", "A", "B"}) {
		t.Fatal(order)
	}
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 || len(g.AllBattlefield()) != 2 {
		t.Fatal("round choices or stop")
	}
	if len(g.resolution.numberBindings) != 0 {
		t.Fatal("round result escaped")
	}
}
func TestRepeatedEntryActualResults(t *testing.T) {
	for _, scenario := range []string{"decline", "empty", "returned", "removed", "prohibited", "no-host", "phased-host", "protected-host", "entry-replaced", "entered"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := repeatedEntryGame()
			card := NewLand("Land")
			if scenario == "no-host" || scenario == "phased-host" || scenario == "protected-host" {
				card = NewBoostAura("Aura", "{W}", 1, 1)
			}
			card.SetOwner(a.PlayerID())
			if scenario != "empty" {
				a.AddToHand(card)
			}
			a.answers = []bool{scenario != "decline", false}
			if scenario == "returned" {
				a.change = func() { a.RemoveFromHand(card.ID()); a.AddToHand(card) }
			}
			if scenario == "removed" {
				a.change = func() { a.RemoveFromHand(card.ID()); a.AddToGraveyard(card) }
			}
			if scenario == "prohibited" {
				g.PutOnBattlefield(NewEnchantment("Rule", "", WithStaticAbility(WormsOfTheEarthEffect())), b.PlayerID())
			}
			if scenario == "phased-host" {
				host := g.PutOnBattlefield(NewCreature("Host", "", 2, 2), b.PlayerID())
				g.PhaseOut(host.ID())
			}
			if scenario == "protected-host" {
				g.PutOnBattlefield(NewCreature("Host", "", 2, 2, WithAbility(ProtectionFromColor(core.White))), b.PlayerID())
			}
			if scenario == "entry-replaced" {
				card = NewCreature("Failed entry", "", 1, 1, WithAbility(&optionalEntryProbe{run: func(*Game, *Permanent) bool { return false }}))
				card.SetOwner(a.PlayerID())
				a.RemoveFromHand(a.Hand()[0].ID())
				a.AddToHand(card)
			}
			result := BindNumber("round")
			effect := EachPlayerMayPutFromHandOntoBattlefield(EachPlayer, You, IsPermanentCard, result)
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 0
			if scenario == "entered" {
				want = 1
			}
			if got := FromNumber(result).resolveValue(ctx); got != want {
				t.Fatalf("got %d want %d", got, want)
			}
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if FromNumber(result).resolveValue(ctx) != 0 {
				t.Fatal("old success remained")
			}
		})
	}
}

type repeatTestEffect struct{ apply func(*EffectContext) error }

func (*repeatTestEffect) Text() string                     { return "test round" }
func (*repeatTestEffect) Properties() EffectProperties     { return EffectProperties{} }
func (e *repeatTestEffect) Apply(ctx *EffectContext) error { return e.apply(ctx) }
func TestRepeatedEntryScopeAndError(t *testing.T) {
	g, a, _ := repeatedEntryGame()
	outer := BindNumber("outer")
	inner := BindNumber("inner")
	g.resolution.numberBindings = map[*NumberBinding]int{outer: 7}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	rounds := 0
	sentinel := errors.New("round error")
	body := &repeatTestEffect{apply: func(ctx *EffectContext) error {
		rounds++
		if FromNumber(outer).resolveValue(ctx) != 7 || FromNumber(inner).resolveValue(ctx) != 0 {
			t.Fatal("round frame was not fresh")
		}
		nested := RepeatWhilePositive(SnapshotNumber(Fixed(0), inner), FromNumber(inner))
		if err := nested.Apply(ctx); err != nil {
			return err
		}
		ctx.Game.resolution.numberBindings[outer] = 99
		ctx.Game.resolution.numberBindings[inner] = 1
		if rounds == 2 {
			return sentinel
		}
		return nil
	}}
	if err := RepeatWhilePositive(body, FromNumber(inner)).Apply(ctx); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	if rounds != 2 || FromNumber(outer).resolveValue(ctx) != 7 || FromNumber(inner).resolveValue(ctx) != 0 {
		t.Fatal("outer frame or error restoration")
	}
}
func TestRepeatedEntryInvalidDefinitions(t *testing.T) {
	result := BindNumber("result")
	for _, effect := range []ResolvingEffect{
		RepeatWhilePositive(nil, Fixed(0)), RepeatWhilePositive(DrawFor(You, Fixed(1)), nil),
		RepeatWhilePositive(DrawFor(You, Fixed(1)), FromNumber(result)),
		EachPlayerMayPutFromHandOntoBattlefield(nil, You, IsPermanentCard, result),
		EachPlayerMayPutFromHandOntoBattlefield(EachPlayer, nil, IsPermanentCard, result),
		EachPlayerMayPutFromHandOntoBattlefield(EachPlayer, You, IsPermanentCard, nil),
		Sequence(RepeatWhilePositive(SnapshotNumber(Fixed(0), result), FromNumber(result)), GainLifeFor(You, FromNumber(result))),
	} {
		card := NewSorcery("Bad", "", WithAbility(Spell(effect)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatalf("accepted %T", effect)
		}
	}
}
func TestRepeatedEntryAbsentStart(t *testing.T) {
	g, a, _ := repeatedEntryGame()
	start := BindPlayer("absent")
	g.resolution.playerBindings = map[*PlayerBinding]uuid.UUID{start: uuid.New()}
	result := BindNumber("round")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := EachPlayerMayPutFromHandOntoBattlefield(EachPlayer, FromPlayer(start), IsPermanentCard, result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromNumber(result).resolveValue(ctx) != 0 || a.offers != 0 {
		t.Fatal("absent start acted")
	}
}

func TestRepeatedEntryAllBindingFrames(t *testing.T) {
	g, a, b := repeatedEntryGame()
	player := BindPlayer("player")
	permanent := BindPermanent("permanent")
	result := BindNumber("round")
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := SnapshotPlayer(You, player).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{permanent: {source.ID(), source.incarnationID}}
	rounds := 0
	body := &repeatTestEffect{apply: func(ctx *EffectContext) error {
		rounds++
		if id, ok := FromPlayer(player).resolvePlayer(ctx); !ok || id != a.PlayerID() {
			t.Fatal("player leaked between rounds")
		}
		if current, ok := From(permanent).resolvePermanent(ctx); !ok || current == nil || current.ID() != source.ID() {
			t.Fatal("permanent leaked between rounds")
		}
		clone := ctx.Game.Clone()
		clone.resolution.playerBindings[player] = b.PlayerID()
		if id, _ := FromPlayer(player).resolvePlayer(ctx); id != a.PlayerID() {
			t.Fatal("clone shared round frame")
		}
		ctx.Game.resolution.playerBindings[player] = b.PlayerID()
		delete(ctx.Game.resolution.permanentBindings, permanent)
		again := 1
		if rounds == 3 {
			again = 0
		}
		return SnapshotNumber(Fixed(again), result).Apply(ctx)
	}}
	if err := RepeatWhilePositive(body, FromNumber(result)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	current, ok := From(permanent).resolvePermanent(ctx)
	if id, _ := FromPlayer(player).resolvePlayer(ctx); id != a.PlayerID() || !ok || current == nil || rounds != 3 {
		t.Fatal("outer binding restoration")
	}
}

func TestRepeatedEntryWaitingSource(t *testing.T) {
	for _, change := range []string{"live", "departed", "returned", "phased"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := repeatedEntryGame()
			result := BindNumber("entries")
			round := EachPlayerMayPutFromHandOntoBattlefield(EachPlayer, ControllerOfWithLKI(This), IsPermanentCard, result)
			sourceCard := NewArtifact("Source", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, RepeatWhilePositive(round, FromNumber(result))))))
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			original := g.CaptureDamageSource(source.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if change == "departed" || change == "returned" {
				g.BouncePermanentToHand(source)
				if change == "returned" {
					a.RemoveFromHand(source.ID())
					g.PutOnBattlefield(sourceCard, b.PlayerID())
				}
			}
			if change == "phased" {
				g.PhaseOut(source.ID())
			}
			sources := g.WaitingDelayedPermanentSources()
			if len(sources) != 1 || !sources[0].SameObject(original) || a.offers != 0 || a.choices != 0 {
				t.Fatal("waiting query executed or followed return")
			}
		})
	}
}
