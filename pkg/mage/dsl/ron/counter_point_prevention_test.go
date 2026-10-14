package ron

import (
	"maps"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const counterPointRule = `PreventDamage(EachDamageWouldBeDealtToRecipient(Permanent(ThisPermanent)),[If(PermanentPassesFilter(ThisPermanent,HasACounterOfType(ChargeCounter)),[RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter,ThisPermanent)]),PreventThatDamage])])`

func counterPointFactory(t *testing.T) mage.CardFactory {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Counter fixture",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:100),Rules:[`+counterPointRule+`])`), "counter.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONCounterPointPrevention(t *testing.T) {
	for _, change := range []string{"live", "empty", "unpreventable", "removed", "phase", "cycle", "control", "return"} {
		t.Run(change, func(t *testing.T) {
			factory := counterPointFactory(t)
			g, a, b := ronGame()
			card := factory()
			p := g.PutOnBattlefield(card, a.PlayerID())
			p.AddCounter(core.Charge, 3)
			switch change {
			case "empty":
				p.RemoveCounter(core.Charge, 3)
			case "unpreventable":
				p.GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
			case "removed":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
			case "phase":
				g.PhaseOut(p.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "return":
				g.RemoveFromBattlefield(p)
				p = g.PutOnBattlefield(card, a.PlayerID())
			}
			clone := g.Clone()
			if change == "phase" {
				if len(g.WaitingPreventionPermanentSources()) != 0 {
					t.Fatal("phased rule is active")
				}
				return
			}
			g.DealDamageToPermanent(g.FindPermanent(p.ID()), 2, uuid.New())
			wantCounters, wantDamage := 1, 0
			if change == "empty" || change == "return" {
				wantCounters, wantDamage = 0, 2
			}
			if change == "removed" {
				wantCounters, wantDamage = 3, 2
			}
			if change == "unpreventable" {
				wantDamage = 2
			}
			if got := g.FindPermanent(p.ID()); got.Counters[core.Charge] != wantCounters || got.Damage != wantDamage {
				t.Fatalf("counters=%d damage=%d", got.Counters[core.Charge], got.Damage)
			}
			if clone.FindPermanent(p.ID()).Damage != 0 {
				t.Fatal("clone shared damage")
			}
		})
	}
}

func TestRONCounterPointPreventionOrdering(t *testing.T) {
	for mode := range 2 {
		a := mage.NewBasePlayer("A")
		b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B"), mode: mode}
		g := mage.NewGame(a, b)
		p := g.PutOnBattlefield(counterPointFactory(t)(), b.PlayerID())
		p.AddCounter(core.Charge, 2)
		g.SetCreatureDamageRedirect(p.ID(), b.PlayerID())
		g.DealDamageToPermanent(p, 2, uuid.New())
		wantLife, wantCounters := 18, 2
		if mode == 1 {
			wantLife, wantCounters = 20, 0
		}
		if b.Life() != wantLife || g.FindPermanent(p.ID()).Counters[core.Charge] != wantCounters || b.prompts != 1 {
			t.Fatal("affected controller order or recipient recheck")
		}
	}
}

func TestRONCounterPointPreventionBatchAndCombat(t *testing.T) {
	g, a, b := ronGame()
	factory := counterPointFactory(t)
	left, right := g.PutOnBattlefield(factory(), a.PlayerID()), g.PutOnBattlefield(factory(), b.PlayerID())
	left.AddCounter(core.Charge, 3)
	right.AddCounter(core.Charge, 10)
	source := g.PutOnBattlefield(mage.NewCreature("Source", "", 2, 20), b.PlayerID())
	saved := g.CaptureDamageSource(source.ID())
	g.RemoveFromBattlefield(source)
	g.PutOnBattlefield(source.Card, a.PlayerID())
	clone := g.Clone()
	g.DealSimultaneousDamage(mage.PermanentDamageAssignment(g.FindPermanent(left.ID()), 2, saved), mage.PermanentDamageAssignment(g.FindPermanent(left.ID()), 2, saved), mage.PermanentDamageAssignment(g.FindPermanent(right.ID()), 2, saved), mage.PlayerDamageAssignment(a, 2, saved))
	if g.FindPermanent(left.ID()).Damage != 1 || g.FindPermanent(left.ID()).Counters[core.Charge] != 0 || g.FindPermanent(right.ID()).Counters[core.Charge] != 8 || a.Life() != 18 {
		t.Fatal("simultaneous depletion, independent subjects or source LKI")
	}
	if clone.FindPermanent(left.ID()).Counters[core.Charge] != 3 || clone.FindPermanent(right.ID()).Counters[core.Charge] != 10 {
		t.Fatal("clone shared counters")
	}
	g, a, b = ronGame()
	hydra := g.PutOnBattlefield(repositoryCard(t, "LEA", "Rock Hydra"), a.PlayerID())
	hydra.AddCounter(core.P1P1, 5)
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 20), b.PlayerID())
	g.GetCombat().AddAttacker(attacker.ID(), a.PlayerID())
	g.GetCombat().AddBlocker(hydra.ID(), attacker.ID())
	g.ExecuteCombatDamage()
	if g.FindPermanent(hydra.ID()).Counters[core.P1P1] != 3 || g.FindPermanent(hydra.ID()).Damage != 0 || g.FindPermanent(attacker.ID()).Damage != 5 {
		t.Fatal("combat must assign power before counter removal")
	}
}

func TestRONCounterPointPreventionHydraShieldLifetime(t *testing.T) {
	for _, state := range []string{"normal", "copy", "control", "phase-before", "phase-after", "return-before", "return-after", "expiry"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEA", "Rock Hydra")
			p := g.PutOnBattlefield(card, a.PlayerID())
			p.AddCounter(core.P1P1, 10)
			a.ManaPool().Add(core.Red, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, nil); err != nil {
				t.Fatal(err)
			}
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			if state == "phase-before" {
				g.PhaseOut(p.ID())
			}
			if state == "return-before" {
				g.RemoveFromBattlefield(p)
				p = g.PutOnBattlefield(card, a.PlayerID())
				p.AddCounter(core.P1P1, 10)
			}
			g.ResolveStack()
			if state == "phase-before" || state == "phase-after" {
				if state == "phase-after" {
					g.PhaseOut(p.ID())
				}
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			if state == "return-after" {
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				p = g.PutOnBattlefield(card, a.PlayerID())
				p.AddCounter(core.P1P1, 10)
			}
			if state == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if state == "expiry" {
				g.RunStepWithPriority(core.Cleanup)
			}
			g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.DealDamageToPermanent(game.FindPermanent(p.ID()), 3, uuid.New())
				want := 2
				if state == "copy" {
					want = 1
				}
				if state == "phase-before" || state == "return-before" || state == "return-after" || state == "expiry" {
					want = 3
				}
				if game.FindPermanent(p.ID()).Damage != want || game.FindPermanent(p.ID()).Counters[core.P1P1] != 10 {
					t.Fatal("shield visit, timing, duration or copied controller")
				}
			}
		})
	}
}

func TestRONCounterPointPreventionInventory(t *testing.T) {
	files := fstest.MapFS{"counter.ron": {Data: []byte(`Card(Name:"Rule",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + counterPointRule + `])`)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	for name, role := range map[string]string{"EachDamageWouldBeDealtToRecipient": "prevention_event", "Permanent": "prevention_recipient", "If": "prevention_operation", "PermanentPassesFilter": "condition", "RemoveCounters": "prevention_operation", "ACounterOfTypeFromPermanent": "counter_removal_instruction"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatal("missing role", name, role)
		}
	}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[PreventDamage(EachDamageWouldBeDealtToRecipient(Player(You)),[PreventThatDamage])])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection changed the registry")
	}
}

func TestRONCounterPointPreventionRasputinSourceCorrection(t *testing.T) {
	n := repositoryRecord(t, "LEG", "Rasputin Dreamweaver", "rasputin.ron")
	if _, err := Compile(n); err == nil || !strings.Contains(err.Error(), "unsupported self untap operations") {
		t.Fatal("remaining counter-cap contract", err)
	}
	prerequisite := n
	prerequisite.Fields = maps.Clone(n.Fields)
	rules := n.Fields["Rules"]
	rules.Args = rules.Args[:3:3]
	prerequisite.Fields["Rules"] = rules
	factory, err := Compile(prerequisite)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	if p.Counters[core.Dream] != 7 {
		t.Fatal("entry counters")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
		t.Fatal("counter mana payment")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.DealDamageToPermanent(g.FindPermanent(p.ID()), 2, uuid.New())
	g.DealDamageToPlayer(b, 1, p.ID())
	if g.FindPermanent(p.ID()).Damage != 1 || g.FindPermanent(p.ID()).Counters[core.Dream] != 5 || b.Life() != 19 {
		t.Fatal("recipient shield or separate payments")
	}
}

func TestRONCounterPointPreventionHydraZeroToughness(t *testing.T) {
	g, a, _ := ronGame()
	card := repositoryCard(t, "LEA", "Rock Hydra")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	p.AddCounter(core.P1P1, 3)
	g.DealDamageToPermanent(p, 3, uuid.New())
	if got := g.FindPermanent(p.ID()); got.Counters[core.P1P1] != 0 || got.Damage != 0 {
		t.Fatal("all damage must be prevented before state actions")
	}
	g.AddRegenerationShield(p.ID())
	g.CheckStateBasedActions()
	if g.FindPermanent(p.ID()) != nil || len(a.Graveyard()) != 1 {
		t.Fatal("zero toughness must move the Hydra without regeneration")
	}
}

func TestRONCounterPointPreventionInvalid(t *testing.T) {
	for _, rule := range []string{
		strings.ReplaceAll(counterPointRule, "ThisPermanent", "HostPermanent"),
		strings.ReplaceAll(counterPointRule, "ThisPermanent", "Ref_TargetPermanent"),
		strings.ReplaceAll(counterPointRule, "ThisPermanent", "ThisPermanent()"),
		strings.ReplaceAll(counterPointRule, "ChargeCounter", "UnknownCounter"),
		strings.Replace(counterPointRule, "ACounterOfTypeFromPermanent(ChargeCounter", "ACounterOfTypeFromPermanent(DreamCounter", 1),
		strings.Replace(counterPointRule, "HasACounterOfType(ChargeCounter)", "HasNoCountersOfType(ChargeCounter)", 1),
		strings.Replace(counterPointRule, "PreventThatDamage", "PreventThatDamage()", 1),
		strings.Replace(counterPointRule, ",PreventThatDamage", "", 1),
		strings.Replace(counterPointRule, "EachDamageWouldBeDealtToRecipient(Permanent(ThisPermanent))", "EachDamageWouldBeDealtToRecipient(Player(You))", 1),
		`Activated(PayMana([]),ActionList([` + counterPointRule + `]))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:10),Rules:[` + rule + `])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONCounterPointPreventionHydra(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, set, "Rock Hydra")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 5); err == nil {
				t.Fatal("unpaid cast")
			}
			a.ManaPool().Add(core.Red, 2)
			a.ManaPool().Add(core.Colorless, 5)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 5); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p == nil || p.Counters[core.P1P1] != 5 || p.CurrentPower(g) != 5 {
				t.Fatal("entry X")
			}
			g.DealDamageToPermanent(p, 2, uuid.New())
			if p = g.FindPermanent(card.ID()); p.Counters[core.P1P1] != 3 || p.Damage != 0 {
				t.Fatal("per-point removal")
			}
			a.ManaPool().Add(core.Red, 4)
			g.SetStep(core.PrecombatMain)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 3, nil); err == nil || a.ManaPool().TotalMana() != 4 {
				t.Fatal("upkeep restriction or rollback")
			}
			g.SetStep(core.Upkeep)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 3, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(p.ID()).Counters[core.P1P1] != 4 {
				t.Fatal("upkeep counter")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
			g.DealDamageToPermanent(g.FindPermanent(p.ID()), 1, uuid.New())
			if g.FindPermanent(p.ID()).Damage != 0 || g.FindPermanent(p.ID()).Counters[core.P1P1] != 4 {
				t.Fatal("resolved finite shield must protect its recipient after ability removal")
			}
			g.DealDamageToPlayer(b, 1, p.ID())
			if b.Life() != 19 {
				t.Fatal("recipient shield prevented source damage")
			}
		})
	}
}
