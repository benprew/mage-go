package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingHandSources(t *testing.T) {
	for _, kind := range []string{"exact", "selected", "redirect"} {
		t.Run(kind, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Hand source", "{R}")
			a.AddToHand(card)

			saved := g.CaptureHandDamageSourceFor(a.PlayerID(), a.handVisit(card.ID()))
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
					return game.WaitingRedirectionHandSourcesFor(a.PlayerID())
				}
				return game.WaitingPreventionHandSourcesFor(a.PlayerID())
			}
			if kind == "redirect" {
				g.PhaseOut(subject.ID())
				if len(query(g)) != 1 {
					t.Fatal("phasing lost saved hand visit")
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
			if kind == "redirect" && len(g.WaitingPreventionHandSourcesFor(a.PlayerID())) != 0 {
				t.Fatal("redirection entered prevention category")
			}
			if kind != "redirect" && len(g.WaitingRedirectionHandSourcesFor(a.PlayerID())) != 0 {
				t.Fatal("prevention entered redirection category")
			}
			a.RemoveFromHand(card.ID())
			a.AddToHand(card)
			returned := g.CaptureHandDamageSourceFor(a.PlayerID(), a.handVisit(card.ID()))
			g.zones.ClearLKI()
			g.resolution.Clear()
			for _, game := range []*Game{g, g.Clone()} {
				refs = query(game)
				if len(refs) != 1 || !refs[0].SameObject(saved) || refs[0].SameObject(returned) {
					t.Fatal("return replaced saved visit")
				}
				choice := BindDamageSource("waiting")
				body := Sequence(ChooseDamageSourceFromWaitingPrecastHandReferences(You, DamageSourceFilter{}, choice), DealDamageFrom(FromDamageSource(choice), You, Fixed(2)))
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

func TestWaitingHandSourcesDeclarations(t *testing.T) {
	binding := BindDamageSource("waiting")
	choice := ChooseDamageSourceFromWaitingPrecastHandReferences(You, DamageSourceFilter{}, binding)
	consumer := DealDamageFrom(FromDamageSource(binding), You, Fixed(1))
	for _, tc := range []struct {
		effect ResolvingEffect
		valid  bool
	}{
		{Sequence(choice, consumer), true},
		{Sequence(consumer, choice), false},
		{ChooseDamageSourceFromWaitingPrecastHandReferences(nil, DamageSourceFilter{}, binding), false},
		{ChooseDamageSourceFromWaitingPrecastHandReferences(You, DamageSourceFilter{}, nil), false},
		{ChooseDamageSourceFromWaitingPrecastHandReferences(You, DamageSourceHasColor(core.Colorless), binding), false},
		{ChooseDamageSourceFromWaitingPrecastHandReferences(You, DamageSourceFilter{}, BindDamageSource("")), false},
	} {
		err := ValidateCardDefinition("Fixture", NewArtifact("Fixture", "", WithAbility(Activate(Costs(), tc.effect))))
		if (err == nil) != tc.valid {
			t.Fatal("wrong declaration result", err)
		}
	}
}

func TestWaitingHandSourcesVisibilityAndBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	chooser := &sourceChoicePlayer{BasePlayer: b}
	g := NewGame(a, chooser)
	card := NewArtifact("Hidden hand", "{R}")
	a.AddToHand(card)
	saved := g.CaptureHandDamageSourceFor(a.PlayerID(), a.handVisit(card.ID()))
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
		for _, sources := range [][]DamageSource{game.WaitingPreventionHandSourcesFor(b.PlayerID()), game.WaitingRedirectionHandSourcesFor(b.PlayerID())} {
			if len(sources) != 1 || !sources[0].SameObject(saved) || sources[0].Name() != "" || sources[0].HasColor(core.Red) || sources[0].HasType(core.TypeArtifact) {
				t.Fatal("hidden or malformed reference exposed facts", sources)
			}
		}
	}
	check(g)
	clone := g.Clone()
	a.RemoveFromHand(card.ID())
	a.AddToHand(card)
	check(g)
	check(clone)
	binding := BindDamageSource("waiting")
	ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
	if err := ChooseDamageSourceFromWaitingPrecastHandReferences(You, DamageSourceFilter{}, binding).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(chooser.seen) != 1 || !FromDamageSource(binding).resolveDamageSource(ctx).SameObject(saved) {
		t.Fatal("combined choice lost deduplicated visit")
	}
	if err := ChooseDamageSourceFromWaitingPrecastHandReferences(You, DamageSourceHasColor(core.Red), binding).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(binding).resolveDamageSource(ctx).Valid() {
		t.Fatal("hidden source passed color filter")
	}
	g.GetEffects().Apply(g)
	g.GetEffects().ClearReplacementsEndOfTurn()
	if len(g.WaitingPreventionHandSourcesFor(b.PlayerID())) != 0 || len(g.WaitingRedirectionHandSourcesFor(b.PlayerID())) != 0 || len(clone.WaitingPreventionHandSourcesFor(b.PlayerID())) != 1 {
		t.Fatal("cleanup crossed clone boundary")
	}
}

func TestWaitingHandSourcesRechecksAndBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Hand source", "{R}", 1, 1)
	g.PutOnBattlefield(NewEnchantment("Public hands", "", WithAbility(Static(RevealAllHands()))), a.PlayerID())
	a.AddToHand(card)
	saved := g.CaptureHandDamageSourceFor(a.PlayerID(), a.handVisit(card.ID()))
	seed, choice := BindDamageSource("seed"), BindDamageSource("choice")
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{seed: saved}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := PreventNextDamageFromSelectedSource(FromDamageSource(seed), You, DamageSourceFilter{}).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	body := Sequence(ChooseDamageSourceFromWaitingPrecastHandReferences(You, DamageSourceHasColor(core.Red), choice), PreventNextDamageFromSelectedSource(FromDamageSource(choice), You, DamageSourceHasColor(core.Red)))
	if err := ApplyEffect(g, body, uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	card.SetColorOverride([]core.Color{core.Blue})
	if g.WaitingPreventionHandSourcesFor(a.PlayerID())[0].HasColor(core.Red) {
		t.Fatal("query used stale live facts")
	}
	g.DealDamageToPlayerFromSource(b, 2, saved)
	if b.Life() != 18 {
		t.Fatal("nonmatching damage consumed shield")
	}
	card.SetColorOverride([]core.Color{core.Red})
	a.RemoveFromHand(card.ID())
	card.SetColorOverride([]core.Color{core.Blue})
	a.AddToHand(card)
	returned := g.CaptureHandDamageSourceFor(a.PlayerID(), a.handVisit(card.ID()))
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

func TestWaitingHandSourcesInspection(t *testing.T) {
	for _, publicDeparture := range []bool{false, true} {
		t.Run(map[bool]string{false: "private departure", true: "public departure"}[publicDeparture], func(t *testing.T) {
			g, a, b, card, visit := inspectableHandFixture()
			saved := g.CaptureHandDamageSource(visit)
			seed := BindDamageSource("seed")
			g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{seed: saved}
			ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
			if err := PreventNextDamageFromSelectedSource(FromDamageSource(seed), You, DamageSourceFilter{}).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			g.GetStack().Pop()
			unreferenced := NewArtifact("Unreferenced", "{R}")
			a.AddToHand(unreferenced)
			check := func(game *Game, viewer uuid.UUID, visible bool) {
				t.Helper()
				refs := game.WaitingPreventionHandSourcesFor(viewer)
				if len(refs) != 1 || !refs[0].SameObject(saved) || refs[0].HasColor(core.Red) != visible || (refs[0].Name() != "") != visible {
					t.Fatal("wrong exact visit or viewer information", refs)
				}
				if visible {
					refs[0].card.AddType(core.TypeLand)
					if game.WaitingPreventionHandSourcesFor(viewer)[0].HasType(core.TypeLand) {
						t.Fatal("query changed private facts")
					}
				}
			}
			check(g, a.PlayerID(), true)
			for _, viewer := range []uuid.UUID{b.PlayerID(), uuid.Nil, uuid.New()} {
				check(g, viewer, false)
			}
			binding := BindDamageSource("private choice")
			player := BindPlayer("hand player")
			g.resolution.playerBindings = map[*PlayerBinding]uuid.UUID{player: a.PlayerID()}
			if err := ChooseDamageSourceFromWaitingPrecastHandReferences(FromPlayer(player), DamageSourceHasColor(core.Red), binding).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if a.calls != 1 || !a.seen[0].HasColor(core.Red) || !FromDamageSource(binding).resolveDamageSource(ctx).SameObject(saved) {
				t.Fatal("choice used controller information instead of choosing player")
			}
			a.seen[0].card.AddType(core.TypeLand)
			if g.resolution.damageSourceBindings[binding].HasType(core.TypeLand) {
				t.Fatal("choice changed saved binding")
			}
			if err := ChooseDamageSourceFromWaitingPrecastHandReferences(You, DamageSourceHasColor(core.Red), binding).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if FromDamageSource(binding).resolveDamageSource(ctx).Valid() {
				t.Fatal("opponent chose hidden color or kept stale result")
			}
			if err := ChooseDamageSourceFromWaitingPrecastHandReferences(You, DamageSourceFilter{}, binding).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			ctx.Controller = uuid.New()
			if err := ChooseDamageSourceFromWaitingPrecastHandReferences(You, DamageSourceFilter{}, binding).Apply(ctx); err != nil || FromDamageSource(binding).resolveDamageSource(ctx).Valid() {
				t.Fatal("absent choosing player retained result", err)
			}
			public := g.PutOnBattlefield(NewEnchantment("Public hands", "", WithAbility(Static(RevealAllHands()))), a.PlayerID())
			check(g, b.PlayerID(), true)
			if !publicDeparture {
				g.PhaseOut(public.ID())
				g.ApplyContinuousEffects()
				check(g, b.PlayerID(), false)
				check(g, a.PlayerID(), true)
			}
			a.RemoveFromHand(card.ID())
			card.SetColorOverride([]core.Color{core.Blue})
			a.AddToHand(card)
			g.PhaseIn([]uuid.UUID{public.ID()})
			g.ApplyContinuousEffects()
			g.zones.ClearLKI()
			g.resolution.Clear()
			clone := g.Clone()
			for _, game := range []*Game{g, clone} {
				check(game, a.PlayerID(), true)
				check(game, b.PlayerID(), publicDeparture)
				check(game, uuid.Nil, publicDeparture)
			}
			g.DealDamageToPlayerFromSource(b, 2, saved)
			if b.Life() != 20 || len(g.WaitingPreventionHandSourcesFor(a.PlayerID())) != 0 || len(clone.WaitingPreventionHandSourcesFor(a.PlayerID())) != 1 {
				t.Fatal("hidden mechanical facts or clone lifetime failed")
			}
		})
	}
}

func TestWaitingHandSourcesRevealLifetime(t *testing.T) {
	for _, departDuringReveal := range []bool{false, true} {
		t.Run(map[bool]string{false: "live after reveal", true: "departure during reveal"}[departDuringReveal], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Revealed source", "{R}")
			a.AddToHand(card)
			keeper := NewArtifact("Reveal keeper", "", WithAbility(Activate(Costs(RevealFromHandCost(CardFilter{}, "reveal")), GainLifeFor(You, Fixed(1)))))
			g.PutOnBattlefield(keeper, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			saved := g.StackHandSources()[0]
			seed := BindDamageSource("revealed seed")
			body := Sequence(ChooseDamageSourceFromStackHandReferences(You, DamageSourceHasColor(core.Red), seed), PreventNextDamageFromSelectedSource(FromDamageSource(seed), You, DamageSourceHasColor(core.Red)))
			if err := ApplyEffect(g, body, uuid.Nil, b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if !g.WaitingPreventionHandSourcesFor(b.PlayerID())[0].HasColor(core.Red) {
				t.Fatal("active cost reveal lost public information")
			}
			if departDuringReveal {
				a.RemoveFromHand(card.ID())
			}
			g.GetStack().Pop()
			refs := g.WaitingPreventionHandSourcesFor(b.PlayerID())
			if len(refs) != 1 || !refs[0].SameObject(saved) || refs[0].HasColor(core.Red) != departDuringReveal {
				t.Fatal("waiting reference extended reveal or lost departure visibility")
			}
			if !departDuringReveal {
				a.RemoveFromHand(card.ID())
			}
			permanent := g.PutOnBattlefield(card, a.PlayerID())
			g.DealDamageToPlayerFromSource(b, 1, g.CaptureDamageSource(permanent.ID()))
			if b.Life() != 19 {
				t.Fatal("hand source continued to a permanent")
			}
			g.DealDamageToPlayerFromSource(b, 2, saved)
			if b.Life() != 19 || len(g.WaitingPreventionHandSourcesFor(b.PlayerID())) != 0 {
				t.Fatal("waiting shield lost hidden mechanical facts or consumption")
			}
		})
	}
}
