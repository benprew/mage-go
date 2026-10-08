package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingPreventionSpellReferences(t *testing.T) {
	for _, state := range []string{"live", "color", "control", "depart", "recast", "copy", "consume", "cleanup", "continued", "phase", "exact", "forgotten"} {
		t.Run(state, func(t *testing.T) {
			g, recipient, spell := selectedSourceFixture(t, state == "exact")
			saved := g.CaptureSpellDamageSource(spell.ID)
			g.SetStep(core.PrecombatMain)
			before := g.Clone()
			switch state {
			case "color":
				g.ChangeObjectColor(spell.ID, core.Blue)
			case "control":
				spell.Controller = recipient.PlayerID()
			case "forgotten":
				g.CounterStackObject(spell.ID)
				clear(g.stack.spellSources)
			case "depart", "recast":
				g.ChangeObjectColor(spell.ID, core.Blue)
				g.CounterStackObject(spell.ID)
				if state == "recast" {
					owner := g.GetPlayer(spell.Controller)
					owner.RemoveFromGraveyard(spell.Card.ID())
					owner.AddToHand(spell.Card)
					owner.ManaPool().Add(core.Red, 1)
					if err := g.CastSpellByID(owner.PlayerID(), spell.Card.ID(), nil, 0); err != nil {
						t.Fatal(err)
					}
				}
			case "copy":
				if g.CopyStackObjectDirect(spell, spell.Controller, false) == nil {
					t.Fatal("copy")
				}
			case "consume":
				g.DealDamageToPlayerFromSource(recipient, 1, saved)
			case "cleanup":
				g.effects.ClearReplacementsEndOfTurn()
			case "continued", "phase":
				g.ResolveTopOfStack()
				if state == "phase" {
					g.PhaseOut(spell.Card.ID())
				}
			}
			for _, game := range []*Game{g, g.Clone()} {
				got := game.WaitingPreventionSpellSources()
				want := 1
				if state == "consume" || state == "cleanup" {
					want = 0
				}
				if len(got) != want {
					t.Fatalf("sources=%d want=%d", len(got), want)
				}
				if want == 0 {
					continue
				}
				if !got[0].SameObject(saved) || got[0].HasColor(core.Blue) != (state == "color" || state == "depart" || state == "recast") {
					t.Fatal("identity or current/departure data")
				}
				if state == "control" && got[0].Controller() != recipient.PlayerID() {
					t.Fatal("controller")
				}
				got[0].card.AddType(core.TypeArtifact)
				if game.WaitingPreventionSpellSources()[0].HasType(core.TypeArtifact) {
					t.Fatal("mutable query result")
				}
				if state == "continued" || state == "phase" {
					refs := game.WaitingPreventionPermanentSources()
					if len(refs) != 1 || refs[0].Incarnation() == uuid.Nil || refs[0].SameObject(saved) {
						t.Fatal("continued permanent identity")
					}
				}
			}
			if got := before.WaitingPreventionSpellSources(); len(got) != 1 || !got[0].SameObject(saved) || !got[0].HasColor(core.Red) {
				t.Fatal("clone changed")
			}
		})
	}
}

func TestWaitingPreventionSpellChoice(t *testing.T) {
	g, recipient, spell := selectedSourceFixture(t, false)
	saved := g.CaptureSpellDamageSource(spell.ID)
	shield := g.effects.replacements[0]
	g.effects.AddReplacement(shield.Clone())
	g.effects.cycleReplacements = append(g.effects.cycleReplacements, shield.Clone())
	g.effects.AddReplacement(&sourcePreventionReplacement{dmgSource: spell.SourceID})
	g.effects.AddReplacement(&sourcePreventionReplacement{exactSource: DamageSource{id: spell.SourceID, stackID: spell.ID, incarnation: uuid.New()}})
	if got := g.WaitingPreventionSpellSources(); len(got) != 1 || !got[0].SameObject(saved) {
		t.Fatal("deduplication or legacy reference")
	}
	chooser := &sourceChoicePlayer{BasePlayer: recipient}
	g.players[1] = chooser
	one, two := BindDamageSource("one"), BindDamageSource("two")
	choice := ChooseDamageSourceFromWaitingPreventionSpellReferences(You, DamageSourceHasColor(core.Red), one)
	ctx := &EffectContext{Game: g, Controller: recipient.PlayerID()}
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 || !FromDamageSource(one).resolveDamageSource(ctx).SameObject(saved) {
		t.Fatal("choice identity")
	}
	if err := ChooseDamageSourceFromWaitingPreventionSpellReferences(You, DamageSourceHasColor(core.Blue), two).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(two).resolveDamageSource(ctx).Valid() || !FromDamageSource(one).resolveDamageSource(ctx).Valid() {
		t.Fatal("independent outputs")
	}
	g.effects.ClearReplacementsEndOfTurn()
	g.effects.cycleReplacements = nil
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(one).resolveDamageSource(ctx).Valid() || chooser.calls != 1 {
		t.Fatal("empty choice reused binding")
	}
	for _, bad := range []ResolvingEffect{
		ChooseDamageSourceFromWaitingPreventionSpellReferences(nil, DamageSourceFilter{}, one),
		ChooseDamageSourceFromWaitingPreventionSpellReferences(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromWaitingPreventionSpellReferences(You, DamageSourceHasColor(core.Colorless), one),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), bad)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid definition accepted")
		}
	}
}

func TestWaitingPreventionSpellSeparateCopies(t *testing.T) {
	g, recipient, spell := selectedSourceFixture(t, false)
	original := g.CaptureSpellDamageSource(spell.ID)
	copied := g.CopyStackObjectDirect(spell, spell.Controller, false)
	if copied == nil {
		t.Fatal("copy")
	}
	copySource := g.CaptureSpellDamageSource(copied.ID)
	binding := BindDamageSource("copy")
	g.resolution.damageSourceBindings[binding] = copySource
	effect := PreventNextDamageFromSelectedSource(FromDamageSource(binding), You, DamageSourceHasColor(core.Red))
	if err := effect.Apply(&EffectContext{Game: g, Controller: recipient.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	for _, game := range []*Game{g, g.Clone()} {
		got := game.WaitingPreventionSpellSources()
		if len(got) != 2 || !got[0].SameObject(original) || !got[1].SameObject(copySource) {
			t.Fatal("copy identity or order")
		}
		game.DealSimultaneousDamage(PlayerDamageAssignment(game.GetPlayer(recipient.PlayerID()), 2, copySource), PlayerDamageAssignment(game.GetPlayer(recipient.PlayerID()), 3, copySource))
		if game.GetPlayer(recipient.PlayerID()).Life() != 20 {
			t.Fatal("simultaneous prevention")
		}
		got = game.WaitingPreventionSpellSources()
		if len(got) != 1 || !got[0].SameObject(original) {
			t.Fatal("copy consumed original shield")
		}
	}
}
