package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const targetHostFilter = `And([IsEnchantmentType(Aura),AttachedToAPermanent(Or([IsCardtype(Creature),IsCardtype(Land)]))])`

func TestRONTargetHostGameplay(t *testing.T) {
	for _, change := range []string{"current", "move", "shroud", "target-return", "target-phase", "host-phase", "cycle", "copy", "retarget"} {
		t.Run(change, func(t *testing.T) {
			factory := singularGroupsFactory(t, `Targeted([TargetPermanent(`+targetHostFilter+`)],ActionList([TapPermanent(HostPermanent),GainLife(Integer(1))]))`, false)
			g, a, b := ronGame()
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 3, 8), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 7, 8), b.PlayerID())
			auraCard := mage.NewEnchantment("Aura", "", mage.WithSubTypes("Aura"))
			aura := g.PutOnBattlefield(auraCard, b.PlayerID())
			g.Attach(aura.ID(), host.ID())
			spell := factory()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{aura.ID()}, 0); err == nil || g.StackSize() != 0 || host.Tapped {
				t.Fatal("unpaid spell changed state")
			}
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{host.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 1 {
				t.Fatal("non-Aura target spent mana")
			}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{aura.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			wantHost, wantOther, wantLife := true, false, 21
			switch change {
			case "move":
				g.Attach(aura.ID(), other.ID())
				wantHost, wantOther = false, true
			case "shroud":
				g.MutablePermanent(aura.ID()).GrantBaseAttr(core.Shroud)
				wantHost, wantLife = false, 20
			case "target-return":
				g.RemoveFromBattlefield(aura)
				returned := g.PutOnBattlefield(auraCard, b.PlayerID())
				g.Attach(returned.ID(), other.ID())
				wantHost, wantLife = false, 20
			case "target-phase":
				g.PhaseOut(aura.ID())
				wantHost, wantLife = false, 20
			case "host-phase":
				g.PhaseOut(host.ID())
				wantHost, wantLife = false, 20
			case "cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				wantLife = 22
			case "retarget":
				replacement := g.PutOnBattlefield(mage.NewEnchantment("Other Aura", "", mage.WithSubTypes("Aura")), b.PlayerID())
				g.Attach(replacement.ID(), other.ID())
				phased := g.PhaseOut(aura.ID())
				copied := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true)
				if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != replacement.ID() {
					t.Fatal("copy did not select replacement Aura")
				}
				g.PhaseIn(phased)
				wantOther, wantLife = true, 22
			}
			clone := g.Clone()
			g.ResolveStack()
			if clone.FindPermanentIncludingPhased(host.ID()).Tapped || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("resolution changed clone")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if game.FindPermanentIncludingPhased(host.ID()).Tapped != wantHost || game.FindPermanent(other.ID()).Tapped != wantOther || game.GetPlayer(a.PlayerID()).Life() != wantLife {
					t.Fatal("host, legality or copy binding mismatch")
				}
			}
		})
	}
}

func TestRONTargetHostInvalid(t *testing.T) {
	for _, body := range []string{
		`ActionList([TapPermanent(HostPermanent)])`,
		`Targeted([TargetPermanent(AnyPermanent)],ActionList([TapPermanent(HostPermanent)]))`,
		`Targeted([TargetPermanent(Or([IsEnchantmentType(Aura),IsCardtype(Creature)]))],ActionList([TapPermanent(HostPermanent)]))`,
		`Targeted([TargetPermanent(IsEnchantmentType(Aura)),TargetPermanent(IsEnchantmentType(Aura))],ActionList([TapPermanent(HostPermanent)]))`,
		`Targeted([TargetPermanent(IsEnchantmentType(Aura))],ActionList([TapPermanent(HostPermanent())]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(`+body+`)])`), "invalid-host.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", body)
		}
	}
	for _, ctx := range []scope{{targetHost: &permanentTargetPlan{}, enteringSource: true}, {targetHost: &permanentTargetPlan{}, drawReplacementBody: true}} {
		if _, err := compilePermanentRef(Node{Kind: Symbol, Name: "HostPermanent"}, ctx); err == nil {
			t.Fatal("host read accepted in a foreign operation context")
		}
	}
	for _, rule := range []string{
		`Activated(PayMana([]),Targeted([TargetPermanent(IsEnchantmentType(Aura))],ActionList([TapPermanent(HostPermanent)])))`,
		`TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),Targeted([TargetPermanent(IsEnchantmentType(Aura))],ActionList([TapPermanent(HostPermanent)])))`,
	} {
		if _, err := Compile(typeComparisonRecord(t, rule, false)); err == nil {
			t.Fatal("nonspell declaration inferred a target host")
		}
	}
}

func TestRONTargetHostInstances(t *testing.T) {
	factory := singularGroupsFactory(t, `Targeted([TargetPermanent(IsEnchantmentType(Aura))],ActionList([TapPermanent(HostPermanent)]))`, false)
	g, a, b := ronGame()
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 3, 8), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 8), b.PlayerID())
	for _, host := range []*mage.Permanent{first, second} {
		aura := g.PutOnBattlefield(mage.NewEnchantment("Aura", "", mage.WithSubTypes("Aura")), b.PlayerID())
		g.Attach(aura.ID(), host.ID())
		spell := factory()
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		a.ManaPool().Add(core.Blue, 1)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{aura.ID()}, 0); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped {
		t.Fatal("separate factory instances shared host bindings")
	}
	g.ResolveStack()
	if !g.FindPermanent(first.ID()).Tapped {
		t.Fatal("earlier spell lost its host binding")
	}
}

func TestRONTargetHostRepositoryComparison(t *testing.T) {
	record := chosenDiscardRecord(t, "LEG", "Enchantment Alteration")
	var comparison Node
	var find func(Node)
	find = func(n Node) {
		if n.Name == "SharesACardtypeWithPermanent" {
			comparison = n
		}
		for _, child := range n.Args {
			find(child)
		}
		for _, child := range n.Fields {
			find(child)
		}
	}
	find(record)
	if comparison.Name == "" {
		t.Fatal("missing repository comparison")
	}
	if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "AttachPermanentToAPermanent") {
		t.Fatalf("full-card attachment blocker changed: %v", err)
	}
	targetPlan := &permanentTargetPlan{name: "Ref_TargetPermanent"}
	build, err := compilePermanentTypeComparison(comparison, scope{spellTargets: true, permanentTarget: targetPlan, targetHost: targetPlan})
	if err != nil {
		t.Fatal(err)
	}
	bindings := initializeActionBindings(instanceBindings{})
	target := mage.DeclarePermanentTarget("aura", mage.And(mage.IsEnchantment, mage.HasSubType("Aura")))
	bindings.permanents[targetPlan] = target
	card := mage.NewInstant("Repository expression", "{U}", mage.Spell(mage.TapEach(build(bindings)), mage.WithTargets(target)))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 3, 8), b.PlayerID())
	member := g.PutOnBattlefield(mage.NewCreature("Member", "", 1, 5), b.PlayerID())
	other := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
	aura := g.PutOnBattlefield(mage.NewEnchantment("Aura", "", mage.WithSubTypes("Aura")), b.PlayerID())
	g.Attach(aura.ID(), host.ID())
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{aura.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !host.Tapped || !member.Tapped || other.Tapped || aura.Tapped {
		t.Fatal("repository comparison read the spell instead of the target host")
	}
}

func TestRONTargetHostDepartureAndDelay(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		t.Run(map[bool]string{false: "departure", true: "delay"}[delayed], func(t *testing.T) {
			body := `PutPermanentIntoItsOwnersHand(Ref_TargetPermanent),GainLife(PowerOfPermanent(HostPermanent))`
			if delayed {
				body = `CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainLife(PowerOfPermanent(HostPermanent))]))`
			}
			factory := singularGroupsFactory(t, `Targeted([TargetPermanent(IsEnchantmentType(Aura))],ActionList([`+body+`]))`, false)
			g, a, b := ronGame()
			hostCard := mage.NewCreature("Host", "", 3, 8)
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			auraCard := mage.NewEnchantment("Aura", "", mage.WithSubTypes("Aura"))
			aura := g.PutOnBattlefield(auraCard, b.PlayerID())
			g.Attach(aura.ID(), host.ID())
			spell := factory()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{aura.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if !delayed {
				if a.Life() != 23 || g.FindPermanent(aura.ID()) != nil {
					t.Fatal("departure lost retained host")
				}
				return
			}
			g.RemoveFromBattlefield(aura)
			g.RemoveFromBattlefield(host)
			returned := g.PutOnBattlefield(auraCard, b.PlayerID())
			other := g.PutOnBattlefield(hostCard, b.PlayerID())
			other.AddCounter(core.P1P1, 4)
			g.Attach(returned.ID(), other.ID())
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				game.ResolveStack()
				if game.GetPlayer(a.PlayerID()).Life() != 23 {
					t.Fatal("delayed host followed a returned visit")
				}
			}
		})
	}
}
