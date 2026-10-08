package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingCommandSources(t *testing.T) {
	for _, kind := range []string{"exact", "selected", "redirect"} {
		t.Run(kind, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Command", "{R}")
			visit, err := g.AddCommandObject(card, false)
			if err != nil {
				t.Fatal(err)
			}
			saved := g.CommandDamageSources()[0]
			binding := BindDamageSource("seed")
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			if err := ChooseDamageSourceFromCommandObjects(You, DamageSourceFilter{}, binding).Apply(ctx); err != nil {
				t.Fatal(err)
			}
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
					return game.WaitingRedirectionCommandSources()
				}
				return game.WaitingPreventionCommandSources()
			}
			if kind == "redirect" {
				g.PhaseOut(subject.ID())
				if len(query(g)) != 1 {
					t.Fatal("phasing lost saved command visit")
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
			if kind == "redirect" && len(g.WaitingPreventionCommandSources()) != 0 {
				t.Fatal("redirection entered prevention category")
			}
			if kind != "redirect" && len(g.WaitingRedirectionCommandSources()) != 0 {
				t.Fatal("prevention entered redirection category")
			}
			g.RemoveCommandObject(visit)
			if _, err := g.AddCommandObject(card, false); err != nil {
				t.Fatal(err)
			}
			returned := g.CommandDamageSources()[0]
			g.zones.ClearLKI()
			g.resolution.Clear()
			for _, game := range []*Game{g, g.Clone()} {
				refs = query(game)
				if len(refs) != 1 || !refs[0].SameObject(saved) || refs[0].SameObject(returned) {
					t.Fatal("return replaced saved visit")
				}
				choice := BindDamageSource("waiting")
				body := Sequence(ChooseDamageSourceFromWaitingCommandReferences(You, DamageSourceHasColor(core.Red), choice), DealDamageFrom(FromDamageSource(choice), You, Fixed(2)))
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

func TestWaitingCommandSourcesVisibilityAndCleanup(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	visit, err := g.AddCommandObject(NewArtifact("Command", "{R}"), false)
	if err != nil {
		t.Fatal(err)
	}
	binding := BindDamageSource("source")
	if err := ApplyEffect(g, Sequence(ChooseDamageSourceFromCommandObjects(You, DamageSourceFilter{}, binding), PreventNextDamageFromSelectedSource(FromDamageSource(binding), You, DamageSourceFilter{})), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.SetCommandObjectFaceDown(visit, true)
	g.RemoveCommandObject(visit)
	refs := g.WaitingPreventionCommandSources()
	if len(refs) != 1 || refs[0].Name() != "" || refs[0].HasColor(core.Red) || refs[0].HasType(core.TypeArtifact) {
		t.Fatal("hidden departure exposed characteristics")
	}
	if refs := clone.WaitingPreventionCommandSources(); len(refs) != 1 || !refs[0].HasColor(core.Red) {
		t.Fatal("clone visibility changed")
	}
	g.GetEffects().ClearReplacementsEndOfTurn()
	if len(g.WaitingPreventionCommandSources()) != 0 || len(clone.WaitingPreventionCommandSources()) != 1 {
		t.Fatal("cleanup leaked between clones")
	}
}

func TestWaitingCommandSourcesDeclarations(t *testing.T) {
	binding := BindDamageSource("waiting")
	choice := ChooseDamageSourceFromWaitingCommandReferences(You, DamageSourceFilter{}, binding)
	consumer := DealDamageFrom(FromDamageSource(binding), You, Fixed(1))
	for _, tc := range []struct {
		effect ResolvingEffect
		valid  bool
	}{
		{Sequence(choice, consumer), true},
		{Sequence(consumer, choice), false},
		{ChooseDamageSourceFromWaitingCommandReferences(nil, DamageSourceFilter{}, binding), false},
		{ChooseDamageSourceFromWaitingCommandReferences(You, DamageSourceFilter{}, nil), false},
		{ChooseDamageSourceFromWaitingCommandReferences(You, DamageSourceFilter{}, BindDamageSource("")), false},
	} {
		err := ValidateCardDefinition("Fixture", NewArtifact("Fixture", "", WithAbility(Activate(Costs(), tc.effect))))
		if (err == nil) != tc.valid {
			t.Fatal("wrong declaration result", err)
		}
	}
}

func TestWaitingCommandSourcesExcludeOtherIdentities(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	chooser := &sourceChoicePlayer{BasePlayer: a}
	g := NewGame(chooser, b)
	if _, err := g.AddCommandObject(NewArtifact("Command", "{R}"), false); err != nil {
		t.Fatal(err)
	}
	saved := g.CommandDamageSources()[0]
	permanent := g.PutOnBattlefield(NewArtifact("Permanent", "{R}"), a.PlayerID())
	malformed := saved
	malformed.incarnation = uuid.New()
	for _, source := range []DamageSource{g.CaptureDamageSource(permanent.ID()), {id: uuid.New(), stackID: uuid.New()}, malformed, {}, saved} {
		g.effects.AddReplacement(&sourcePreventionReplacement{exactSource: source})
	}
	g.effects.cycleReplacements = append(g.effects.cycleReplacements, &sourcePreventionReplacement{exactSource: saved})
	g.effects.AddReplacement(&selectedSourceRedirectReplacement{source: saved})
	refs := g.WaitingPreventionCommandSources()
	if len(refs) != 1 || !refs[0].SameObject(saved) {
		t.Fatal("other identities entered command category")
	}
	binding := BindDamageSource("choice")
	if err := ChooseDamageSourceFromWaitingCommandReferences(You, DamageSourceFilter{}, binding).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(chooser.seen) != 1 || !g.resolution.damageSourceBindings[binding].SameObject(saved) {
		t.Fatal("combined choice lost source")
	}
	if err := ChooseDamageSourceFromWaitingCommandReferences(You, DamageSourceHasColor(core.Blue), binding).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[binding].Valid() {
		t.Fatal("nonmatching choice retained result")
	}
	if err := ChooseDamageSourceFromWaitingCommandReferences(You, DamageSourceFilter{}, binding).Apply(&EffectContext{Game: g, Controller: uuid.New()}); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[binding].Valid() {
		t.Fatal("absent player retained result")
	}
}
