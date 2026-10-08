package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingExileSources(t *testing.T) {
	for _, kind := range []string{"exact", "selected", "redirect"} {
		t.Run(kind, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Exile source", "{R}")
			g.ExileCard(card, uuid.Nil)

			saved := g.CaptureExiledDamageSource(g.FindExiledCard(card.ID()).Visit)
			binding := BindDamageSource("seed")
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: saved}
			subject := g.PutOnBattlefield(NewCreature("Subject", "", 1, 10), a.PlayerID())
			ctx.SourceID = subject.ID()
			var install ResolvingEffect
			switch kind {
			case "exact":
				install = PreventNextDamageFromExactSource(FromDamageSource(binding), You, DamageSourceHasColor(core.Red))
			case "selected":
				install = PreventNextDamageFromSelectedSource(FromDamageSource(binding), You, DamageSourceHasColor(core.Red))
			case "redirect":
				install = RedirectNextDamageFromSelectedSource(FromDamageSource(binding), This, You, DamageSourceHasColor(core.Red))
			}
			for range 2 {
				if err := install.Apply(ctx); err != nil {
					t.Fatal(err)
				}
			}
			query := func(game *Game) []DamageSource {
				if kind == "redirect" {
					return game.WaitingRedirectionExileSources()
				}
				return game.WaitingPreventionExileSources()
			}
			if kind == "redirect" {
				g.PhaseOut(subject.ID())
				if len(query(g)) != 1 {
					t.Fatal("phasing lost saved exile visit")
				}
				g.PhaseIn([]uuid.UUID{subject.ID()})
			}
			refs := query(g)
			if len(refs) != 1 || !refs[0].SameObject(saved) {
				t.Fatal("exact reference not deduplicated", refs)
			}
			refs[0].colors[0] = core.Blue
			if !query(g)[0].HasColor(core.Red) {
				t.Fatal("query changed saved information")
			}
			if kind == "redirect" && len(g.WaitingPreventionExileSources()) != 0 {
				t.Fatal("redirection entered prevention category")
			}
			if kind != "redirect" && len(g.WaitingRedirectionExileSources()) != 0 {
				t.Fatal("prevention entered redirection category")
			}
			g.RemoveFromExile(card.ID())
			g.ExileCard(card, uuid.Nil)
			returned := g.CaptureExiledDamageSource(g.FindExiledCard(card.ID()).Visit)
			g.zones.ClearLKI()
			g.resolution.Clear()
			for _, game := range []*Game{g, g.Clone()} {
				refs = query(game)
				if len(refs) != 1 || !refs[0].SameObject(saved) || refs[0].SameObject(returned) {
					t.Fatal("return replaced saved visit")
				}
				choice := BindDamageSource("waiting")
				body := Sequence(ChooseDamageSourceFromWaitingPrecastExileReferences(You, DamageSourceHasColor(core.Red), choice), DealDamageFrom(FromDamageSource(choice), You, Fixed(2)))
				if err := ApplyEffect(game, body, uuid.Nil, b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if game.GetPlayer(b.PlayerID()).Life() != 18 {
					t.Fatal("waiting choice lost departed source")
				}
				if kind == "redirect" {
					for range 2 {
						game.DealDamageToPermanentFromSource(game.FindPermanent(subject.ID()), 1, saved)
					}
				} else {
					for range 2 {
						game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 1, saved)
					}
				}
				if len(query(game)) != 0 {
					t.Fatal("consumed references remained")
				}
				before := game.GetPlayer(b.PlayerID()).Life()
				if err := ApplyEffect(game, body, uuid.Nil, b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if game.GetPlayer(b.PlayerID()).Life() != before {
					t.Fatal("empty choice retained result")
				}
			}
		})
	}
}

func TestWaitingExileSourcesDeclarations(t *testing.T) {
	binding := BindDamageSource("waiting")
	choice := ChooseDamageSourceFromWaitingPrecastExileReferences(You, DamageSourceFilter{}, binding)
	consumer := DealDamageFrom(FromDamageSource(binding), You, Fixed(1))
	for _, tc := range []struct {
		effect ResolvingEffect
		valid  bool
	}{
		{Sequence(choice, consumer), true},
		{Sequence(consumer, choice), false},
		{ChooseDamageSourceFromWaitingPrecastExileReferences(nil, DamageSourceFilter{}, binding), false},
		{ChooseDamageSourceFromWaitingPrecastExileReferences(You, DamageSourceFilter{}, nil), false},
		{ChooseDamageSourceFromWaitingPrecastExileReferences(You, DamageSourceHasColor(core.Colorless), binding), false},
		{ChooseDamageSourceFromWaitingPrecastExileReferences(You, DamageSourceFilter{}, BindDamageSource("")), false},
	} {
		err := ValidateCardDefinition("Fixture", NewArtifact("Fixture", "", WithAbility(Activate(Costs(), tc.effect))))
		if (err == nil) != tc.valid {
			t.Fatal("wrong declaration result", err)
		}
	}
}

func TestWaitingExileSourcesVisibilityAndBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	chooser := &sourceChoicePlayer{BasePlayer: a}
	g := NewGame(chooser, b)
	card := NewArtifact("Hidden exile", "{R}")
	g.ExileCardFaceDown(card, uuid.Nil, a.PlayerID())
	saved := g.CaptureExiledDamageSource(g.FindExiledCard(card.ID()).Visit)
	malformed := saved
	malformed.stackID = uuid.New()
	permanent := g.PutOnBattlefield(NewArtifact("Permanent", "{R}"), a.PlayerID())
	for _, source := range []DamageSource{malformed, g.CaptureDamageSource(permanent.ID()), {}, saved} {
		g.effects.AddReplacement(&sourcePreventionReplacement{exactSource: source, replacementBase: replacementBase{duration: core.EndOfTurn}})
	}
	g.effects.cycleReplacements = append(g.effects.cycleReplacements, &sourcePreventionReplacement{exactSource: saved})
	g.effects.AddReplacement(&selectedSourceRedirectReplacement{source: saved, replacementBase: replacementBase{duration: core.EndOfTurn}})
	check := func(game *Game) {
		t.Helper()
		for _, sources := range [][]DamageSource{game.WaitingPreventionExileSources(), game.WaitingRedirectionExileSources()} {
			if len(sources) != 1 || !sources[0].SameObject(saved) || sources[0].Name() != "" || sources[0].HasColor(core.Red) || sources[0].HasType(core.TypeArtifact) {
				t.Fatal("hidden or malformed reference exposed facts", sources)
			}
		}
	}
	check(g)
	clone := g.Clone()
	g.RemoveFromExile(card.ID())
	g.ExileCard(card, uuid.Nil)
	check(g)
	check(clone)
	binding := BindDamageSource("waiting")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := ChooseDamageSourceFromWaitingPrecastExileReferences(You, DamageSourceFilter{}, binding).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(chooser.seen) != 1 || !FromDamageSource(binding).resolveDamageSource(ctx).SameObject(saved) {
		t.Fatal("combined choice lost deduplicated visit")
	}
	if err := ChooseDamageSourceFromWaitingPrecastExileReferences(You, DamageSourceHasColor(core.Red), binding).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(binding).resolveDamageSource(ctx).Valid() {
		t.Fatal("hidden source passed color filter")
	}
	g.GetEffects().Apply(g)
	g.GetEffects().ClearReplacementsEndOfTurn()
	if len(g.WaitingPreventionExileSources()) != 0 || len(g.WaitingRedirectionExileSources()) != 0 || len(clone.WaitingPreventionExileSources()) != 1 {
		t.Fatal("cleanup crossed clone boundary")
	}
}

func TestWaitingExileSourcesRechecksAndBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Exile source", "{R}", 1, 1)
	g.ExileCard(card, uuid.Nil)
	saved := g.CaptureExiledDamageSource(g.FindExiledCard(card.ID()).Visit)
	seed, choice := BindDamageSource("seed"), BindDamageSource("choice")
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{seed: saved}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := PreventNextDamageFromSelectedSource(FromDamageSource(seed), You, DamageSourceFilter{}).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	body := Sequence(ChooseDamageSourceFromWaitingPrecastExileReferences(You, DamageSourceHasColor(core.Red), choice), PreventNextDamageFromSelectedSource(FromDamageSource(choice), You, DamageSourceHasColor(core.Red)))
	if err := ApplyEffect(g, body, uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	card.SetColorOverride([]core.Color{core.Blue})
	if g.WaitingPreventionExileSources()[0].HasColor(core.Red) {
		t.Fatal("query used stale live facts")
	}
	g.DealDamageToPlayerFromSource(b, 2, saved)
	if b.Life() != 18 {
		t.Fatal("nonmatching damage consumed shield")
	}
	card.SetColorOverride([]core.Color{core.Red})
	g.RemoveFromExile(card.ID())
	card.SetColorOverride([]core.Color{core.Blue})
	g.ExileCard(card, uuid.Nil)
	returned := g.CaptureExiledDamageSource(g.FindExiledCard(card.ID()).Visit)
	clone := g.Clone()
	outcomes := g.dealSimultaneousDamageOutcomes(PlayerDamageAssignment(b, 2, saved), PlayerDamageAssignment(b, 3, saved), PlayerDamageAssignment(b, 1, returned))
	if b.Life() != 17 || len(outcomes) != 1 || !outcomes[0].source.SameObject(returned) {
		t.Fatal("batch used returned facts or consumed between packets")
	}
	clone.DealDamageToPlayerFromSource(clone.GetPlayer(b.PlayerID()), 2, saved)
	if clone.GetPlayer(b.PlayerID()).Life() != 18 {
		t.Fatal("consumption crossed clone")
	}
	g.DealDamageToPlayerFromSource(b, 1, saved)
	if b.Life() != 16 {
		t.Fatal("shield did not consume")
	}
}
