package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func singleTargetCounter(t *testing.T, filter string) mage.Card {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Single Counter",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[ManaCostU],Rules:[SpellActions(Targeted([TargetSpell(`+filter+`)],ActionList([CounterSpell(Ref_TargetSpell)])))])`), "single.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	return factory()
}

func TestRONSingleTargetGameplay(t *testing.T) {
	for _, scenario := range []string{"player", "permanent", "phased", "returned", "copy", "changed"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			subject := g.PutOnBattlefield(mage.NewCreature("Subject", "", 1, 8), a.PlayerID())
			bolt := repositoryCard(t, "LEB", "Lightning Bolt")
			bolt.SetOwner(a.PlayerID())
			a.AddToHand(bolt)
			a.ManaPool().Add(core.Red, 1)
			id := subject.ID()
			if scenario == "player" {
				id = b.PlayerID()
			}
			if err := g.CastSpellByID(a.PlayerID(), bolt.ID(), []uuid.UUID{id}, 0); err != nil {
				t.Fatal(err)
			}
			victim := g.StackPeek()
			if scenario == "copy" {
				victim = g.CopyStackObjectDirect(victim, a.PlayerID(), false)
				if victim == nil {
					t.Fatal("copy")
				}
			}
			if scenario == "phased" {
				g.PhaseOut(subject.ID())
			}
			if scenario == "returned" {
				g.RemoveFromBattlefield(subject)
				g.PutOnBattlefield(subject.Card, a.PlayerID())
			}
			counter := singleTargetCounter(t, `And([TargetsOnlyASingleTarget,IsCardtype(Instant)])`)
			counter.SetOwner(a.PlayerID())
			a.AddToHand(counter)
			if err := g.CastSpellByID(a.PlayerID(), counter.ID(), []uuid.UUID{victim.ID}, 0); err == nil {
				t.Fatal("unpaid counter")
			}
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), counter.ID(), []uuid.UUID{victim.ID}, 0); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			if scenario == "changed" {
				victim.Targets = append(victim.Targets, b.PlayerID())
			}
			g.ResolveTopOfStack()
			clone.ResolveTopOfStack()
			if (g.GetStack().FindByID(victim.ID) == nil) != (scenario != "changed") {
				t.Fatal("resolution did not recheck target count")
			}
			if clone.GetStack().FindByID(victim.ID) != nil {
				t.Fatal("clone counter failed")
			}
		})
	}
}

func TestRONSingleTargetRejectAndDeclarations(t *testing.T) {
	g, a, _ := ronGame()
	spell := repositoryCard(t, "LEB", "Llanowar Elves")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Green, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	victim := g.StackPeek()
	counter := singleTargetCounter(t, "TargetsOnlyASingleTarget")
	counter.SetOwner(a.PlayerID())
	a.AddToHand(counter)
	a.ManaPool().Add(core.Blue, 1)
	for _, targets := range [][]uuid.UUID{nil, {a.PlayerID(), a.PlayerID()}, {a.PlayerID(), uuid.New()}} {
		victim.Targets = targets
		if err := g.CastSpellByID(a.PlayerID(), counter.ID(), []uuid.UUID{victim.ID}, 0); err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 1 {
			t.Fatal("illegal target spent resources")
		}
	}
	for _, filter := range []string{"TargetsOnlyASingleTarget()", "TargetsOnlyASingleTarget(AnySpell)", "CanTargetOnly(IsCreatureType(Wall))"} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetSpell(`+filter+`)],ActionList([CounterSpell(Ref_TargetSpell)])))])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", filter)
		}
	}

	for _, rule := range []string{
		`PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfSpells(TargetsOnlyASingleTarget)])`,
		`SpellActions(ActionList([TargetsOnlyASingleTarget]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad context",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted wrong context", rule)
		}
	}
	mirror := staticBaseRecord(t, "DRK", "Reflecting Mirror")
	if _, err := Compile(mirror); err == nil {
		t.Fatal("incomplete Mirror accepted")
	}
	data := `Card(Name:"Counter",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetSpell(TargetsOnlyASingleTarget)],ActionList([CounterSpell(Ref_TargetSpell)])))])`
	report, err := InspectFS(fstest.MapFS{"single.ron": {Data: []byte(data)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal(err, report.Issues)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "TargetsOnlyASingleTarget" && signature.Role == "spell_filter" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing spell filter inventory role")
	}
	invalid := strings.Replace(data, "TargetsOnlyASingleTarget", "TargetsOnlyASingleTarget()", 1)
	if registry, err := LoadFS(fstest.MapFS{"good.ron": {Data: []byte(data)}, "bad.ron": {Data: []byte(invalid)}}, "*.ron"); err == nil || registry != nil {
		t.Fatal("invalid collection was published")
	}
}

func TestRONSingleTargetCastEvents(t *testing.T) {
	for _, name := range []string{"Lightning Bolt", "Llanowar Elves"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			nodes, err := Parse(strings.NewReader(`Card(Name:"Observer",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(WhenAPlayerCastsASpell(AnyPlayer,TargetsOnlyASingleTarget),ActionList([GainLife(Integer(1))]))])`), "event.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(factory(), a.PlayerID())
			spell := repositoryCard(t, "LEB", name)
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.Red, 1)
			a.ManaPool().Add(core.Green, 1)
			var targets []uuid.UUID
			if name == "Lightning Bolt" {
				targets = []uuid.UUID{b.PlayerID()}
			}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), targets, 0); err != nil {
				t.Fatal(err)
			}
			victim := g.StackPeek()
			g.PutTriggersOnStack()
			want := 1
			if name == "Lightning Bolt" {
				want = 2
			}
			if g.StackSize() != want {
				t.Fatalf("stack = %d, want %d", g.StackSize(), want)
			}
			if want == 2 {
				victim.Targets = nil
				g.ResolveTopOfStack()
				if a.Life() != 21 {
					t.Fatal("pending trigger repeated its detection check")
				}
			}
		})
	}
}

func TestRONSingleTargetActivationAndComposition(t *testing.T) {
	for _, filter := range []string{"TargetsOnlyASingleTarget", "Or([TargetsOnlyASingleTarget,IsCardtype(Sorcery)])", "ExceptFor(ExceptFor(TargetsOnlyASingleTarget))"} {
		g, a, b := ronGame()
		bolt := repositoryCard(t, "LEB", "Lightning Bolt")
		bolt.SetOwner(a.PlayerID())
		a.AddToHand(bolt)
		a.ManaPool().Add(core.Red, 1)
		if err := g.CastSpellByID(a.PlayerID(), bolt.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		victim := g.StackPeek()
		factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetSpell(`+filter+`)],ActionList([CounterSpell(Ref_TargetSpell)])))`)
		first, second := factory(), factory()
		if first.Abilities()[0] == second.Abilities()[0] {
			t.Fatal("shared ability")
		}
		source := g.PutOnBattlefield(first, a.PlayerID())
		a.ManaPool().Add(core.Colorless, 1)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID}); err != nil {
			t.Fatal(err)
		}
		g.RemoveFromBattlefield(source)
		g.ResolveTopOfStack()
		if g.GetStack().FindByID(victim.ID) != nil {
			t.Fatal("departed source lost the saved spell binding")
		}
	}
}
