package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONActivationDefinedXDoll(t *testing.T) {
	for _, change := range []string{"live", "count-change", "leave", "return", "phase", "phase-cycle", "copy", "illegal-target", "phase-before"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "LEG", "Voodoo Doll")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Pin] = 2
			a.ManaPool().Add(core.Colorless, 4)
			g.SetXValue(77)
			target := b.PlayerID()
			var victim *mage.Permanent
			if change == "illegal-target" {
				victim = g.PutOnBattlefield(mage.NewCreature("Victim", "", 9, 9), b.PlayerID())
				target = victim.ID()
			}
			if change == "phase-before" {
				g.PhaseOut(p.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, []uuid.UUID{target}); err == nil {
					t.Fatal("phased source activated")
				}
				if a.ManaPool().TotalMana() != 4 || g.StackSize() != 0 || g.XValue() != 77 {
					t.Fatal("failed activation changed state")
				}
				return
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, []uuid.UUID{target}); err != nil {
				t.Fatal(err)
			}
			if g.StackPeek().XValue != 2 || a.ManaPool().TotalMana() != 0 || g.XValue() != 77 || !g.FindPermanent(p.ID()).Tapped {
				t.Fatal("constrained payment")
			}
			damage := 2
			switch change {
			case "count-change":
				g.MutablePermanent(p.ID()).Counters[core.Pin] = 5
				damage = 5
			case "leave", "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if change == "return" {
					a.RemoveFromHand(c.ID())
					q := g.PutOnBattlefield(c, b.PlayerID())
					g.MutablePermanent(q.ID()).Counters[core.Pin] = 11
				}
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "copy":
				copyObj := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
				if copyObj == nil || copyObj.XValue != 2 {
					t.Fatal("copy lost announcement")
				}
				damage = 4
			case "illegal-target":
				g.BouncePermanentToHand(victim)
				damage = 0
			}
			clone := g.Clone()
			g.ResolveStack()
			if clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("clone changed")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if game.GetPlayer(b.PlayerID()).Life() != 20-damage {
					t.Fatalf("life %d", game.GetPlayer(b.PlayerID()).Life())
				}
			}
		})
	}
}

func TestRONActivationDefinedXInvalid(t *testing.T) {
	count := `TheTotalNumberOfCountersOfTypeAmongPermanents(PinCounter,SinglePermanent(ThisPermanent))`
	for _, modifier := range []string{`XIs`, `XIs()`, `XIs(` + count + `,` + count + `)`, `XIs(Integer(2))`, `XIs(ValueX)`, `XIs(TheTotalNumberOfCountersOfTypeAmongPermanents(PinCounter,SinglePermanent(Ref_TargetPermanent)))`, `XIs(TheTotalNumberOfCountersOfTypeAmongPermanents(PinCounter,EachPermanent(AnyPermanent)))`, `XIs(TheTotalNumberOfCountersOfTypeAmongPermanents(UnknownCounter,SinglePermanent(ThisPermanent)))`, `And([XIs(` + count + `),XIs(` + count + `)])`} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[ActivatedWithModifiers(PayManaX([ManaCostX],ValueX),ActionList([GainLife(Integer(1))]),` + modifier + `)])`
		if registry, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(src)}}, "*.ron"); err == nil || registry != nil {
			t.Fatal("invalid declaration published", modifier)
		}
	}
	for _, rules := range []string{
		`ActivatedWithModifiers(PayMana([ManaCostGeneric(1)]),ActionList([GainLife(Integer(1))]),XIs(` + count + `))`,
		`ActivatedWithModifiers(PayManaX([ManaCostX],ValueX),ActionList([GainLife(ValueX)]),XIs(` + count + `))`,
		`ActivatedWithModifiers(PayManaX([ManaCostX],ValueX),Targeted([TargetPermanent(ManaValueIs(EqualTo(ValueX)))],ActionList([TapPermanent(Ref_TargetPermanent)])),XIs(` + count + `))`,
	} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rules + `])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("unsupported constrained X context accepted")
		}
	}
}

func TestRONActivationDefinedXDollTriggers(t *testing.T) {
	for _, change := range []string{"normal", "tapped-before", "tapped-after", "untapped-after", "indestructible", "regenerate", "leave", "return", "control", "prevention", "phase-before", "phase-after"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "LEG", "Voodoo Doll")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			queueRONUpkeep(g, b)
			g.ResolveStack()
			if g.FindPermanent(p.ID()).Counters[core.Pin] != 0 {
				t.Fatal("opponent upkeep added counter")
			}
			queueRONUpkeep(g, a)
			g.ResolveStack()
			queueRONUpkeep(g, a)
			g.ResolveStack()
			if g.FindPermanent(p.ID()).Counters[core.Pin] != 2 {
				t.Fatal("upkeep did not add counters")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent end step triggered")
			}
			if change == "tapped-before" || change == "untapped-after" {
				g.MutablePermanent(p.ID()).Tapped = true
			}
			if change == "phase-before" {
				g.PhaseOut(p.ID())
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			triggered := change != "tapped-before" && change != "untapped-after" && change != "phase-before"
			if (g.StackSize() == 1) != triggered {
				t.Fatal("detection did not check state")
			}
			switch change {
			case "tapped-after":
				g.MutablePermanent(p.ID()).Tapped = true
			case "untapped-after":
				g.MutablePermanent(p.ID()).Tapped = false
			case "indestructible":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Indestructible)
			case "regenerate":
				g.AddRegenerationShield(p.ID())
			case "leave", "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if change == "return" {
					a.RemoveFromHand(c.ID())
					q := g.PutOnBattlefield(c, b.PlayerID())
					g.MutablePermanent(q.ID()).Counters[core.Pin] = 11
				}
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "prevention":
				g.PreventAllDamageFrom(p.ID())
			case "phase-after":
				g.PhaseOut(p.ID())
			}
			g.ResolveStack()
			damage := 2
			if !triggered || change == "tapped-after" || change == "prevention" {
				damage = 0
			}
			if a.Life() != 20-damage || b.Life() != 20 {
				t.Fatalf("damage a=%d b=%d", a.Life(), b.Life())
			}
			destroyed := triggered && change != "tapped-after" && change != "phase-after" && change != "indestructible" && change != "regenerate" && change != "return"
			if change == "leave" {
				destroyed = true
			}
			if (g.FindPermanentIncludingPhased(p.ID()) == nil) != destroyed {
				t.Fatal("destruction or source identity")
			}
		})
	}
}

func TestRONActivationDefinedXDollCastAndRollback(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "LEG", "Voodoo Doll")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	a.ManaPool().Add(core.Colorless, 5)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
		t.Fatal("underpaid cast")
	}
	if len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 5 || g.StackSize() != 0 {
		t.Fatal("casting rollback")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(c.ID())
	if p == nil || c.ManaCost().CMC() != 6 {
		t.Fatal("cast metadata or entry")
	}
	g.SetXValue(7)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.StackPeek().XValue != 0 || g.XValue() != 7 {
		t.Fatal("zero constrained X")
	}
	g.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("zero counter damage")
	}
	g.MutablePermanent(p.ID()).Tapped = false
	g.MutablePermanent(p.ID()).Counters[core.Pin] = 2
	g.SetXValue(7)
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, []uuid.UUID{b.PlayerID()}); err == nil {
		t.Fatal("underpaid activation")
	}
	if g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 3 || g.StackSize() != 0 || g.XValue() != 7 {
		t.Fatal("activation rollback")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, []uuid.UUID{uuid.New()}); err == nil {
		t.Fatal("invalid target accepted")
	}
	if g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 4 || g.StackSize() != 0 {
		t.Fatal("illegal target spent resources")
	}
	g.RunStepWithPriority(core.Upkeep)
	if g.FindPermanent(p.ID()).Counters[core.Pin] != 3 {
		t.Fatal("native upkeep")
	}
	g.RunStepWithPriority(core.EndStep)
	if g.FindPermanent(p.ID()) != nil || a.Life() != 17 {
		t.Fatal("native end step")
	}
}

func TestRONActivationDefinedXInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Voodoo Doll"`) {
			record = line
		}
	}
	report, err := InspectFS(fstest.MapFS{"doll.ron": {Data: []byte(record)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("complete record inventory", err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "TheTotalNumberOfCountersOfTypeAmongPermanents" && signature.Role == "activation_x_value" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing announcement value role")
	}
}
