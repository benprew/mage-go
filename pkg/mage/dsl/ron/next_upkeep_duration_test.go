package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNextUpkeepErhnam(t *testing.T) {
	for _, change := range []string{"none", "source-left", "source-control", "target-control", "target-return", "target-phase", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ARN", "Erhnam Djinn")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("unpaid casting changed state")
			}
			a.ManaPool().Add(core.Green, 1)
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p.CurrentPower(g) != 4 || p.CurrentToughness(g) != 5 {
				t.Fatal("wrong printed values")
			}
			victimCard := mage.NewCreature("Victim", "", 2, 5)
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || victim.HasKeyword(core.Forestwalk) {
				t.Fatal("wrong trigger or early grant")
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			switch change {
			case "source-left":
				g.RemoveFromBattlefield(p)
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "target-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: victim.ID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "target-return":
				g.RemoveFromBattlefield(victim)
				g.PutOnBattlefield(victimCard, b.PlayerID())
			case "target-phase":
				g.PhaseOut(victim.ID())
			}
			clone := g.Clone()
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{victim.ID()})
			want := change != "target-control" && change != "target-return" && change != "target-phase"
			if g.FindPermanent(victimCard.ID()).HasKeyword(core.Forestwalk) != want {
				t.Fatal("resolution legality or source context")
			}
			g.SetActivePlayerIndex(1)
			g.RunStepWithPriority(core.Upkeep)
			if g.FindPermanent(victimCard.ID()).HasKeyword(core.Forestwalk) != want {
				t.Fatal("expired at opponent upkeep")
			}
			if p = g.FindPermanent(card.ID()); p != nil {
				g.RemoveFromBattlefield(p)
			}
			g.SetActivePlayerIndex(0)
			g.RunStepWithPriority(core.Upkeep)
			if g.FindPermanent(victimCard.ID()).HasKeyword(core.Forestwalk) {
				t.Fatal("survived next upkeep")
			}
			clone.ResolveStack()
			if clone.FindPermanent(victimCard.ID()) != nil && clone.FindPermanent(victimCard.ID()).HasKeyword(core.Forestwalk) != want {
				t.Fatal("clone lost pending grant")
			}
		})
	}
}

func TestRONNextUpkeepInvalid(t *testing.T) {
	for _, duration := range []string{"UntilNextUpkeep", "UntilNextUpkeep()", "UntilNextUpkeep(You())", "UntilNextUpkeep(Opponent)", "UntilNextUpkeep(HostController)", "UntilNextUpkeep(Ref_TargetPlayer)", "UntilNextUpkeep(You,You)"} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([Flying])],` + duration + `)]))])`
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", duration)
		}
	}
}

func TestRONNextUpkeepErhnamTargetsAndCombat(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "ARN", "Erhnam Djinn")
	source := g.PutOnBattlefield(card, a.PlayerID())
	wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 0, 5, mage.WithSubTypes("Wall")), b.PlayerID())
	land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
	own := g.PutOnBattlefield(mage.NewCreature("Own", "", 2, 5), a.PlayerID())
	shroud := g.PutOnBattlefield(mage.NewCreature("Shroud", "", 2, 5, mage.WithKeyword(core.Shroud)), b.PlayerID())
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 5), b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("triggered on opponent upkeep")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	obj := g.StackPeek()
	if obj == nil || len(obj.Targets) != 1 || obj.Targets[0] != victim.ID() {
		t.Fatal("illegal target was offered")
	}
	for _, excluded := range []*mage.Permanent{source, wall, land, own, shroud} {
		for _, id := range obj.TargetSpecs[0].Possible(a.PlayerID(), card, g) {
			if id == excluded.ID() {
				t.Fatal("target filter admitted an excluded permanent")
			}
		}
	}
	g.ResolveStack()
	blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 2, 5), a.PlayerID())
	if mage.HasLandwalkEvasion(g.FindPermanent(victim.ID()), a.PlayerID(), g) {
		t.Fatal("forestwalk prevented blocking without a Forest")
	}
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
	if !mage.HasLandwalkEvasion(g.FindPermanent(victim.ID()), a.PlayerID(), g) {
		t.Fatal("forestwalk did not prevent blocking")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.RunStepWithPriority(core.Upkeep)
	if mage.HasLandwalkEvasion(g.FindPermanent(victim.ID()), a.PlayerID(), g) || !mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(victim.ID()), g) {
		t.Fatal("blocking restriction survived expiry")
	}
}

func TestRONNextUpkeepInventory(t *testing.T) {
	source := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[Unsupported],UntilNextUpkeep(You))]))])`
	report, err := InspectFS(fstest.MapFS{"inventory.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 0 || len(report.Issues) != 1 {
		t.Fatal("unsupported modification accepted")
	}
	for _, signature := range report.Signatures {
		if signature.Name == "You" && signature.Role == "player_reference" {
			return
		}
	}
	t.Fatal("duration player lost its inventory role")
}

func TestRONNextUpkeepNative(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "ARN", "Erhnam Djinn")
	source := g.PutOnBattlefield(card, a.PlayerID())
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 5), b.PlayerID())
	g.RunStepWithPriority(core.Upkeep)
	if !g.FindPermanent(victim.ID()).HasKeyword(core.Forestwalk) || g.StackSize() != 0 {
		t.Fatal("native upkeep did not resolve the grant")
	}
	g.RunStepWithPriority(core.Cleanup)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Upkeep)
	if !g.FindPermanent(victim.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("grant ended before the next own upkeep")
	}
	g.SetActivePlayerIndex(0)
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(victim.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("grant ended at untap")
	}
	g.RunStepWithPriority(core.Upkeep)
	if g.FindPermanent(victim.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("grant survived next own upkeep")
	}
}

func TestRONNextUpkeepActivationInstances(t *testing.T) {
	source := `Card(Name:"Duration",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AdjustPT(+1,+2),AddAbility([Flying])],UntilNextUpkeep(You))])))])`
	nodes, err := Parse(strings.NewReader(source), "duration.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	q := g.PutOnBattlefield(factory(), b.PlayerID())
	x := g.PutOnBattlefield(mage.NewCreature("X", "", 2, 5), a.PlayerID())
	y := g.PutOnBattlefield(mage.NewCreature("Y", "", 2, 5), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{x.ID()}); err == nil || g.StackSize() != 0 {
		t.Fatal("unpaid activation accepted")
	}
	a.ManaPool().Add(core.Colorless, 1)
	b.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{x.ID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), q.ID(), 0, []uuid.UUID{y.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	clone := g.Clone()
	g.SetActivePlayerIndex(0)
	g.RunStepWithPriority(core.Upkeep)
	if g.FindPermanent(x.ID()).HasKeyword(core.Flying) || g.FindPermanent(x.ID()).CurrentPower(g) != 2 || !g.FindPermanent(y.ID()).HasKeyword(core.Flying) || g.FindPermanent(y.ID()).CurrentToughness(g) != 7 {
		t.Fatal("instances shared bindings or expiry players")
	}
	if !clone.FindPermanent(x.ID()).HasKeyword(core.Flying) || clone.FindPermanent(x.ID()).CurrentPower(clone) != 3 {
		t.Fatal("expiry changed cloned modifications")
	}
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Upkeep)
	if g.FindPermanent(y.ID()).HasKeyword(core.Flying) || g.FindPermanent(y.ID()).CurrentToughness(g) != 5 {
		t.Fatal("second instance did not expire")
	}
}

func TestRONNextUpkeepSpellAndDelayed(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		t.Run(map[bool]string{false: "spell", true: "delayed"}[delayed], func(t *testing.T) {
			action := `CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AddAbility([Flying])],UntilNextUpkeep(You))`
			if delayed {
				action = `CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + action + `]))`
			}
			source := `Card(Name:"Duration Spell",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[ManaCostU],Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + action + `])))])`
			nodes, err := Parse(strings.NewReader(source), "spell.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			p := g.PutOnBattlefield(mage.NewCreature("Recipient", "", 2, 5), b.PlayerID())
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(p.ID()).HasKeyword(core.Flying) == delayed {
				t.Fatal("duration started at the wrong resolution")
			}
			if delayed {
				g.RunStepWithPriority(core.EndStep)
			}
			if !g.FindPermanent(p.ID()).HasKeyword(core.Flying) {
				t.Fatal("spell source departure ended the grant")
			}
			g.SetActivePlayerIndex(1)
			g.RunStepWithPriority(core.Upkeep)
			if !g.FindPermanent(p.ID()).HasKeyword(core.Flying) {
				t.Fatal("recipient upkeep ended the grant")
			}
			g.SetActivePlayerIndex(0)
			g.RunStepWithPriority(core.Upkeep)
			if g.FindPermanent(p.ID()).HasKeyword(core.Flying) {
				t.Fatal("spell duration lost the resolving controller")
			}
		})
	}
}
