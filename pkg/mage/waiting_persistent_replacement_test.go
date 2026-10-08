package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingPersistentReplacementSourcesLifecycle(t *testing.T) {
	for _, kind := range []string{"life", "draw", "counter"} {
		for _, state := range []string{"live", "phase", "cycle", "depart", "return", "control", "remove-ability"} {
			t.Run(kind+"/"+state, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewEnchantment("Rule", "{R}")
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				register := func() {
					switch kind {
					case "life":
						g.SetLichActive(a.PlayerID(), p.ID())
					case "draw":
						g.AddIslandSanctuaryReplacement(a.PlayerID(), p.ID())
					case "counter":
						g.AddCounterDoubler(p.ID(), core.P1P1, PermanentFilter{})
					}
				}
				register()
				saved := g.CaptureDamageSource(p.ID())
				old := g.effects.replacements[len(g.effects.replacements)-1].Clone()
				clone := g.Clone()
				want := 1
				switch state {
				case "phase":
					g.PhaseOut(p.ID())
					want = 0
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "depart", "return":
					g.RemoveFromBattlefield(p)
					g.zones.ClearLKI()
					want = 0
					if state == "return" {
						p = g.PutOnBattlefield(card, b.PlayerID())
					}
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "remove-ability":
					g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.EndOfTurn))
				}
				g.effects.replacements = []ReplacementEffect{old}
				if old.IsActive(g) != (want == 1) {
					t.Fatal("execution lifecycle differs from registration")
				}
				refs := g.WaitingPersistentReplacementPermanentSources()
				if len(refs) != want {
					t.Fatalf("sources=%d want=%d", len(refs), want)
				}
				if want == 1 {
					if !refs[0].SameObject(saved) {
						t.Fatal("wrong incarnation")
					}
					if state == "control" && refs[0].Controller() != b.PlayerID() {
						t.Fatal("stale controller")
					}
					refs[0].card.SetOwner(b.PlayerID())
					if g.WaitingPersistentReplacementPermanentSources()[0].card.Owner() != a.PlayerID() {
						t.Fatal("query changed card")
					}
				}
				if refs := clone.WaitingPersistentReplacementPermanentSources(); len(refs) != 1 || !refs[0].SameObject(saved) {
					t.Fatal("clone changed")
				}
				if state == "return" {
					register()
					refs = g.WaitingPersistentReplacementPermanentSources()
					if len(refs) != 1 || refs[0].SameObject(saved) || old.IsActive(g) {
						t.Fatal("returned incarnation used stale registration")
					}
				}
			})
		}
	}
}

func TestWaitingPersistentReplacementSourcesExecution(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewEnchantment("Rule", "{R}"), a.PlayerID())
	target := g.PutOnBattlefield(NewCreature("Target", "", 1, 1), b.PlayerID())
	a.SetLibrary([]Card{NewArtifact("First", ""), NewArtifact("Second", "")})
	g.SetLichActive(a.PlayerID(), p.ID())
	g.AddIslandSanctuaryReplacement(a.PlayerID(), p.ID())
	g.AddCounterDoubler(p.ID(), core.P1P1, PermanentFilter{})
	if len(g.WaitingPersistentReplacementPermanentSources()) != 1 || len(a.Hand()) != 0 {
		t.Fatal("query ran actions or failed to deduplicate")
	}
	g.PlayerGainLife(a, 1)
	if a.Life() != 20 || len(a.Hand()) != 1 {
		t.Fatal("life replacement did not draw")
	}
	if g.effects.ApplyReplacements(NewDrawCardAction(uuid.Nil, a.PlayerID(), true), g) != nil {
		t.Fatal("draw replacement did not apply")
	}
	g.AddCountersWithReplacement(g.FindPermanent(target.ID()), core.P1P1, 1, p.ID(), false)
	if g.FindPermanent(target.ID()).Counters[core.P1P1] != 2 {
		t.Fatal("counter replacement did not apply")
	}
	g.PhaseOut(p.ID())
	g.PlayerGainLife(a, 1)
	if a.Life() != 21 || len(a.Hand()) != 1 {
		t.Fatal("phased life rule applied")
	}
	if g.effects.ApplyReplacements(NewDrawCardAction(uuid.Nil, a.PlayerID(), true), g) == nil {
		t.Fatal("phased draw rule applied")
	}
	g.AddCountersWithReplacement(g.FindPermanent(target.ID()), core.P1P1, 1, p.ID(), false)
	if g.FindPermanent(target.ID()).Counters[core.P1P1] != 3 {
		t.Fatal("phased counter rule applied")
	}
}

func TestWaitingPersistentReplacementSourcesChoices(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewEnchantment("Rule", "{R}"), a.PlayerID())
	g.AddIslandSanctuaryReplacement(a.PlayerID(), p.ID())
	g.AddIslandSanctuaryReplacement(a.PlayerID(), p.ID())
	g.AddCounterDoubler(uuid.New(), core.P1P1, PermanentFilter{})
	card := NewArtifact("Not entered", "")
	g.AddIslandSanctuaryReplacement(a.PlayerID(), card.ID())
	g.PutOnBattlefield(card, a.PlayerID())
	if len(g.WaitingPersistentReplacementPermanentSources()) != 1 {
		t.Fatal("duplicate or nonlive registration")
	}
	first, second := BindDamageSource("first"), BindDamageSource("second")
	ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
	for _, binding := range []*DamageSourceBinding{first, second} {
		choice := ChooseDamageSourceFromWaitingPersistentReplacementReferences(You, DamageSourceHasColor(core.Red), binding)
		if err := ValidateCardDefinition("Choice", NewArtifact("Choice", "", WithAbility(Activate(Costs(), choice)))); err != nil {
			t.Fatal(err)
		}
		if err := choice.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if !g.resolution.damageSourceBindings[binding].SameObject(g.CaptureDamageSource(p.ID())) {
			t.Fatal("choice identity")
		}
	}
	if err := ChooseDamageSourceFromWaitingPersistentReplacementReferences(You, DamageSourceHasColor(core.Blue), first).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[first].Valid() || !g.resolution.damageSourceBindings[second].Valid() {
		t.Fatal("binding isolation")
	}
	ctx.Controller = uuid.New()
	if err := ChooseDamageSourceFromWaitingPersistentReplacementReferences(You, DamageSourceFilter{}, second).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[second].Valid() {
		t.Fatal("absent player retained choice")
	}
	for _, effect := range []ResolvingEffect{
		ChooseDamageSourceFromWaitingPersistentReplacementReferences(nil, DamageSourceFilter{}, first),
		ChooseDamageSourceFromWaitingPersistentReplacementReferences(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromWaitingPersistentReplacementReferences(You, DamageSourceFilter{}, BindDamageSource("")),
		Sequence(DealDamageFrom(FromDamageSource(first), You, Fixed(1)), ChooseDamageSourceFromWaitingPersistentReplacementReferences(You, DamageSourceFilter{}, first)),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestWaitingPersistentReplacementSourcesOrderAndClones(t *testing.T) {
	a := &choiceRecordingPlayer{BasePlayer: NewBasePlayer("A"), chosenMode: 1}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	doubler := g.PutOnBattlefield(NewArtifact("Doubler", ""), b.PlayerID())
	extra := g.PutOnBattlefield(NewArtifact("Extra", ""), b.PlayerID())
	target := g.PutOnBattlefield(NewCreature("Target", "", 1, 1), a.PlayerID())
	g.AddCounterDoubler(doubler.ID(), core.P1P1, PermanentFilter{})
	g.AddETBAdditionalCounters(extra.ID(), core.P1P1, 2, PermanentFilter{}, false)
	clone := g.Clone()
	action := NewAddCountersAction(doubler.ID(), target.ID(), core.P1P1, 1, true)
	result := g.effects.ApplyReplacements(action, g).(*AddCountersAction)
	if result.Amount() != 6 {
		t.Fatal("affected player order did not apply addition before doubling", result.Amount())
	}
	result = clone.effects.ApplyReplacements(action, clone).(*AddCountersAction)
	if result.Amount() != 4 {
		t.Fatal("clone did not keep its independent default choice", result.Amount())
	}
	if len(g.WaitingPersistentReplacementPermanentSources()) != 1 || len(clone.WaitingPersistentReplacementPermanentSources()) != 1 {
		t.Fatal("query changed persistent replacement state")
	}
	clone.PhaseOut(doubler.ID())
	result = clone.effects.ApplyReplacements(action, clone).(*AddCountersAction)
	if result.Amount() != 3 || len(clone.WaitingPersistentReplacementPermanentSources()) != 0 || len(g.WaitingPersistentReplacementPermanentSources()) != 1 {
		t.Fatal("phasing changed original replacement")
	}
}
