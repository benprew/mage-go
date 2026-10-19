package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCollectionExclusionNested(t *testing.T) {
	for _, excluded := range []string{
		`And([IsCardtype(Creature),IsColor(Red)])`,
		`Or([IsColor(Red),IsCardtype(Artifact)])`,
		`ExceptFor(ExceptFor(IsColor(Red)))`,
	} {
		t.Run(excluded, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([TapEachPermanent(And([IsCardtype(Creature),ExceptFor(`+excluded+`)]))]))`)
			g, a, b := ronGame()
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			red := g.PutOnBattlefield(mage.NewCreature("Red", "{R}", 1, 3), b.PlayerID())
			green := g.PutOnBattlefield(mage.NewCreature("Green", "{G}", 1, 3), a.PlayerID())
			phased := g.PutOnBattlefield(mage.NewCreature("Phased", "{G}", 1, 3), a.PlayerID())
			g.PhaseOut(phased.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			g.ResolveStack()
			if clone.FindPermanent(green.ID()).Tapped {
				t.Fatal("resolution changed clone")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if game.FindPermanent(red.ID()).Tapped || !game.FindPermanent(green.ID()).Tapped || game.FindPermanent(p.ID()).Tapped || game.FindPermanentIncludingPhased(phased.ID()).Tapped {
					t.Fatal("wrong nested exclusion members")
				}
			}
		})
	}
}

func TestRONCollectionExclusionDelayedIdentity(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "phase-cycle", "control", "copy", "late-entry"} {
		t.Run(change, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapEachPermanent(And([IsCardtype(Creature),ExceptFor(SinglePermanent(Ref_TargetPermanent))]))]))])))`)
			g, a, b := ronGame()
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			c := mage.NewCreature("Excluded", "", 1, 3)
			c.SetOwner(a.PlayerID())
			excluded := g.PutOnBattlefield(c, a.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 3), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{excluded.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			switch change {
			case "return":
				g.RemoveFromBattlefield(excluded)
				excluded = g.PutOnBattlefield(c, a.PlayerID())
			case "phase":
				g.PhaseOut(excluded.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(excluded.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{TargetID: excluded.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "late-entry":
				other = g.PutOnBattlefield(mage.NewCreature("Late", "", 1, 3), b.PlayerID())
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("missing delayed trigger")
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.ResolveStack()
				if game.FindPermanentIncludingPhased(excluded.ID()).Tapped != (change == "return") || !game.FindPermanent(other.ID()).Tapped {
					t.Fatal("exclusion lost exact visit or live membership")
				}
			}
		})
	}
}

func TestRONCollectionExclusionPlayerScope(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([TapEachPermanent(And([IsCardtype(Creature),ExceptFor(ControlledByAPlayer(SinglePlayer(You)))]))]))`)
	g, a, b := ronGame()
	first, second := factory(), factory()
	if first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("shared ability instance")
	}
	p := g.PutOnBattlefield(first, a.PlayerID())
	q := g.PutOnBattlefield(second, b.PlayerID())
	own := g.PutOnBattlefield(mage.NewCreature("Own", "", 1, 3), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 3), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveTopOfStack()
	if !g.FindPermanent(own.ID()).Tapped || g.FindPermanent(other.ID()).Tapped {
		t.Fatal("copy used original controller")
	}
	g.ResolveStack()
	if !g.FindPermanent(other.ID()).Tapped {
		t.Fatal("original used copied controller")
	}
	g.UntapPermanent(g.FindPermanent(own.ID()))
	g.UntapPermanent(g.FindPermanent(other.ID()))
	if err := g.ActivateAbilityByIndex(b.PlayerID(), q.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !g.FindPermanent(own.ID()).Tapped || g.FindPermanent(other.ID()).Tapped {
		t.Fatal("factory shared player scope")
	}
}

func TestRONCollectionExclusionInvalid(t *testing.T) {
	for _, filter := range []string{
		`ExceptFor`, `ExceptFor()`, `ExceptFor(IsTapped,IsUntapped)`,
		`ExceptFor([])`, `ExceptFor(And([]))`, `ExceptFor(Or([]))`,
		`ExceptFor(SinglePermanent(Ref_TargetPermanent))`,
		`ExceptFor(SinglePermanent(Trigger_ThatPermanent))`,
		`ExceptFor(ControlledByAPlayer(SinglePlayer(ActionPlayer)))`,
		`ExceptFor(TheTokensCreatedThisWay)`, `ExceptFor(Unknown)`,
	} {
		t.Run(filter, func(t *testing.T) {
			nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([TapEachPermanent(`+filter+`)]))])`), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid exclusion accepted")
			}
		})
	}
}

func TestRONCollectionExclusionContexts(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`ExceptFor(IsTapped)`), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{}, {resolving: true, enteringSource: true}, {resolving: true, drawReplacementBody: true}} {
		if _, err := compileContextualCollection(nodes[0], ctx); err == nil {
			t.Fatal("unsupported exclusion context accepted")
		}
	}
}

func TestRONCollectionExclusionAtomicLoad(t *testing.T) {
	const record = `Card(Name:"Exclusion",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([TapEachPermanent(ExceptFor(And([IsCardtype(Creature),IsUntapped])))]))])`
	files := fstest.MapFS{"good.ron": {Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.Replace(record, "Exclusion", "Bad", 1), "IsUntapped", "CouldntAttackThisTurn()"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 || !registry.CardRegistered("Exclusion") {
		t.Fatal("malformed history published a partial registry")
	}
	if _, err := Compile(repositoryRecord(t, "DRK", "Season of the Witch", "season.ron")); err != nil {
		t.Fatal("Season history must compile", err)
	}
}
