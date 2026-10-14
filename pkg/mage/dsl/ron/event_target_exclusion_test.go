package ron

import (
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

type ronExclusionPaymentPlayer struct {
	mage.Player
	accept bool
	offers int
}

func (p *ronExclusionPaymentPlayer) ChooseMayAbility(string) bool {
	p.offers++
	return p.accept
}

func TestRONEventTargetExclusionMantle(t *testing.T) {
	for _, scenario := range []string{"paid", "declined", "prevented", "power", "target-phase", "target-return", "shroud", "reattach", "source-left", "host-control", "phase-cycle", "host-left", "host-return", "host-phase", "copy", "blocked", "no-target"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronActionPaymentPlayer{ronPaymentPlayer: &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "declined"}}
			b := &ronActionPaymentPlayer{ronPaymentPlayer: &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}}
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			g.SetStep(core.DeclareAttackers)
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 8), a.PlayerID())
			host.RevokeBaseAttr(core.AttrSummonSick)
			victimCard := mage.NewCreature("Victim", "", 0, 12)
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			c := repositoryCard(t, "FEM", "Farrel's Mantle")
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			g.SetStep(core.PrecombatMain)
			a.ManaPool().Add(core.White, 3)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			aura := g.FindPermanent(c.ID())
			if aura == nil || aura.AttachedTo != host.ID() || a.ManaPool().TotalMana() != 0 {
				t.Fatal("Aura did not cast and attach")
			}
			a.target, b.target = victim.ID(), victim.ID()
			if scenario == "no-target" {
				g.PhaseOut(victim.ID())
			}
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{host.ID()})
			if g.StackSize() != 0 || a.offers != 0 {
				t.Fatal("unblocked trigger ran during attack declaration")
			}
			var blocks []mage.BlockAssignment
			if scenario == "blocked" {
				blocks = []mage.BlockAssignment{{BlockerID: victim.ID(), AttackerID: host.ID()}}
			}
			if scenario == "blocked" {
				g.ExecuteBlockers(blocks)
			} else {
				g.FireEvent(core.GameEvent{Type: core.EvtAttacksUnblocked, SourceID: host.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
				g.PutTriggersOnStack()
			}
			if scenario == "blocked" || scenario == "no-target" {
				if g.StackSize() != 0 || a.offers != 0 {
					t.Fatal("blocked or targetless attack offered payment")
				}
				return
			}
			if g.StackSize() != 1 || !slices.Equal(g.StackPeek().Targets, []uuid.UUID{victim.ID()}) || a.offers != 0 {
				t.Fatalf("wrong target or payment timing: stack=%d offers=%d", g.StackSize(), a.offers)
			}
			damage := 4
			switch scenario {
			case "prevented":
				g.AddPreventionShield(victim.ID(), 4)
				damage = 0
			case "power":
				g.AddContinuousEffect(mage.TemporaryBoost(host.ID(), 3, 0))
				damage = 7
			case "target-phase":
				g.PhaseOut(victim.ID())
			case "target-return":
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(victimCard, b.PlayerID())
			case "shroud":
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
			case "reattach":
				g.Attach(aura.ID(), victim.ID())
			case "source-left":
				g.RemoveFromBattlefield(aura)
			case "host-left", "host-return":
				g.RemoveFromBattlefield(host)
				if scenario == "host-return" {
					g.PutOnBattlefield(host.Card, b.PlayerID())
				}
			case "host-phase":
				g.PhaseOut(host.ID())
			case "host-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				damage *= 2
			}
			clone := g.Clone()
			cloneA := &ronExclusionPaymentPlayer{Player: clone.GetPlayer(a.PlayerID()), accept: a.accept}
			cloneB := &ronExclusionPaymentPlayer{Player: clone.GetPlayer(b.PlayerID()), accept: b.accept}
			clone.SetPlayerAt(0, cloneA)
			clone.SetPlayerAt(1, cloneB)
			g.ResolveStack()
			if clone.FindPermanentIncludingPhased(victim.ID()).Damage != 0 {
				t.Fatal("live payment changed clone")
			}
			clone.ResolveStack()
			legal := scenario != "target-phase" && scenario != "target-return" && scenario != "shroud"
			available := legal && scenario != "host-phase"
			paid := available && scenario != "declined"
			if !paid {
				damage = 0
			}
			for _, game := range []*mage.Game{g, clone} {
				if game.FindPermanentIncludingPhased(victim.ID()).Damage != damage {
					t.Fatalf("damage=%d want=%d", game.FindPermanentIncludingPhased(victim.ID()).Damage, damage)
				}
				game.ExecuteCombatDamage()
				wantLife := 20
				if !paid && scenario != "host-control" && scenario != "host-phase" {
					wantLife -= game.FindPermanent(host.ID()).CurrentPower(game)
				}
				if game.GetPlayer(b.PlayerID()).Life() != wantLife {
					t.Fatal("paid host still assigned combat damage, or unpaid host lost it")
				}
			}
			if available {
				wantA, wantB := 1, 0
				if scenario == "copy" {
					wantA = 2
				}
				if scenario == "host-control" {
					wantA, wantB = 0, 1
				}
				if a.offers != wantA || b.offers != wantB || cloneA.offers != wantA || cloneB.offers != wantB {
					t.Fatalf("payer offers=%d,%d want=%d,%d", a.offers, b.offers, wantA, wantB)
				}
			} else if a.offers != 0 || b.offers != 0 {
				t.Fatal("illegal target offered payment")
			}
		})
	}
}

func TestRONEventTargetExclusionInvalid(t *testing.T) {
	event := `WhenACreatureAttacksAndIsntBlocked(SinglePermanent(HostPermanent))`
	for _, ref := range []string{"Trigger_ThatCreature()", "Trigger_ThatPlayer", "Ref_TargetPermanent", "TheChosenPermanent", "HostPermanent", "Trigger_ThatOtherCreature", "You"} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[TriggerA(` + event + `,Targeted([TargetPermanent(Other(` + ref + `))],ActionList([TapPermanent(Ref_TargetPermanent)])))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid exclusion", ref)
		}
	}
	for _, rule := range []string{
		`Activated(PayMana([]),Targeted([TargetPermanent(Other(Trigger_ThatCreature))],ActionList([TapPermanent(Ref_TargetPermanent)])))`,
		`TriggerA(` + event + `,ActionList([GainLife(Integer(1))])),Activated(PayMana([]),Targeted([TargetPermanent(Other(Trigger_ThatCreature))],ActionList([TapPermanent(Ref_TargetPermanent)])))`,
		`TriggerA(` + event + `,Targeted([TargetPermanent(Other())],ActionList([TapPermanent(Ref_TargetPermanent)])))`,
		`TriggerA(` + event + `,Targeted([TargetPermanent(Or([Other(Trigger_ThatCreature),IsCardtype(Creature)]))],ActionList([TapPermanent(Ref_TargetPermanent)])))`,
	} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[` + rule + `])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted escaped or malformed exclusion", rule)
		}
	}
}

func TestRONEventTargetExclusionStrictLoad(t *testing.T) {
	record := repositoryRecord(t, "FEM", "Farrel's Mantle", "FEM.lines.ron")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Farrel's Mantle", factory()); err != nil {
		t.Fatal(err)
	}
	registry := mage.NewRegistry()
	data, err := fs.ReadFile(repositoryFixtures.files, "FEM.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var source string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Farrel's Mantle"`) {
			source = line
			break
		}
	}
	if source == "" {
		t.Fatal("missing complete record")
	}

	if err := LoadInto(registry, fstest.MapFS{"valid.ron": {Data: []byte(source)}}, "*.ron"); err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(source, "Other(Trigger_ThatCreature)", "Other(Trigger_ThatPlayer)", 1)
	if err := LoadInto(registry, fstest.MapFS{"bad.ron": {Data: []byte(bad)}}, "*.ron"); err == nil {
		t.Fatal("invalid collection loaded")
	}
	if !registry.CardRegistered("Farrel's Mantle") || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("failed load changed registry")
	}
}

func TestRONEventTargetExclusionNativeCombat(t *testing.T) {
	for _, state := range []string{"accept", "decline", "blocked"} {
		t.Run(state, func(t *testing.T) {
			a := &ronActionNativePlayer{TestPlayer: gametest.NewTestPlayer("A"), accept: state != "decline"}
			b := gametest.NewTestPlayer("B")
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 8), a.PlayerID())
			host.RevokeBaseAttr(core.AttrSummonSick)
			victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 0, 8), b.PlayerID())
			a.target = victim.ID()
			aura := g.PutOnBattlefield(repositoryCard(t, "FEM", "Farrel's Mantle"), a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			a.SetAttackers(1, []string{"Host"})
			if state == "blocked" {
				b.SetBlockers(1, map[string]string{"Victim": "Host"})
			}
			g.RunStepWithPriority(core.DeclareAttackers)
			if victim.Damage != 0 {
				t.Fatal("early damage payment")
			}
			g.RunStepWithPriority(core.DeclareBlockers)
			wantDamage := 0
			if state == "accept" {
				wantDamage = 4
			}
			if g.FindPermanent(victim.ID()).Damage != wantDamage {
				t.Fatal("native unblocked event or optional payment")
			}
			g.RunStepWithPriority(core.CombatDamage)
			wantLife := 20
			if state == "decline" {
				wantLife = 18
			}
			if b.Life() != wantLife {
				t.Fatal("native combat damage suppression")
			}
			g.RunStepWithPriority(core.EndCombat)
			g.RunStepWithPriority(core.Cleanup)
			g.GetCombat().AddAttacker(host.ID(), b.PlayerID())
			g.ExecuteCombatDamage()
			if b.Life() != wantLife-2 {
				t.Fatal("combat damage suppression survived cleanup")
			}
		})
	}
}

func TestRONEventTargetExclusionBindings(t *testing.T) {
	for _, event := range []string{"WhenACreatureAttacks", "WhenACreatureBlocks", "WhenACreatureAttacksAndIsntBlocked", "WhenAPermanentBecomesTapped"} {
		t.Run(event, func(t *testing.T) {
			src := `Card(Name:"Watcher",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),TriggerA(` + event + `(SinglePermanent(HostPermanent)),Targeted([TargetPermanent(And([IsCardtype(Creature),Other(Trigger_ThatPermanent)])),TargetPermanent(Other(Trigger_ThatPermanent))],ActionList([TapPermanent(Ref_TargetPermanent1),TapPermanent(Ref_TargetPermanent2)])))])`
			factory, err := Compile(nameScopeNode(t, src))
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition("Watcher", factory()); err != nil {
				t.Fatal(err)
			}
			g, a, b := opponentTargetGame()
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 8), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 8), b.PlayerID())
			victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 8), b.PlayerID())
			for _, host := range []*mage.Permanent{first, second} {
				aura := g.PutOnBattlefield(factory(), a.PlayerID())
				g.Attach(aura.ID(), host.ID())
				kind := core.EvtDeclaredAttacker
				if event == "WhenACreatureBlocks" {
					kind = core.EvtDeclaredBlocker
				}
				if event == "WhenACreatureAttacksAndIsntBlocked" {
					kind = core.EvtAttacksUnblocked
				}
				if event == "WhenAPermanentBecomesTapped" {
					g.TapPermanent(host)
				} else {
					g.FireEvent(core.GameEvent{Type: kind, SourceID: host.ID(), Flag: true})
				}
			}
			a.requested = []uuid.UUID{victim.ID()}
			g.PutTriggersOnStack()
			if g.StackSize() != 2 || len(a.candidates) != 4 {
				t.Fatal("numbered targets or factory instances lost their event binding")
			}
			for i, candidates := range a.candidates {
				excluded, allowed := second.ID(), first.ID()
				if i >= 2 {
					excluded, allowed = first.ID(), second.ID()
				}
				if slices.Contains(candidates, excluded) || !slices.Contains(candidates, allowed) {
					t.Fatal("factory event exclusion crossed")
				}
			}
			g.ResolveStack()
			if !g.FindPermanent(victim.ID()).Tapped {
				t.Fatal("numbered targets failed to resolve")
			}
		})
	}
}

func TestRONEventTargetExclusionCombatOpponent(t *testing.T) {
	for _, ref := range []string{"Trigger_ThatOtherCreature", "Trigger_ThatOtherPermanent"} {
		g, a, b := opponentTargetGame()
		factory, err := Compile(nameScopeNode(t, `Card(Name:"Watcher",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[TriggerA(WhenACreatureBlocksACreature(SinglePermanent(ThisPermanent),IsCardtype(Creature)),Targeted([TargetPermanent(And([IsCardtype(Creature),Other(`+ref+`)]))],ActionList([TapPermanent(Ref_TargetPermanent)])))],CardPT:(Power:2,Toughness:8))`))
		if err != nil {
			t.Fatal(err)
		}
		source := g.PutOnBattlefield(factory(), a.PlayerID())
		attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 8), b.PlayerID())
		victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 8), b.PlayerID())
		a.requested = []uuid.UUID{victim.ID()}
		g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: source.ID(), TargetID: attacker.ID(), Flag: true})
		g.PutTriggersOnStack()
		if g.StackSize() != 1 || len(a.candidates) != 1 || slices.Contains(a.candidates[0], attacker.ID()) || !slices.Contains(a.candidates[0], source.ID()) {
			t.Fatal("opponent binding excluded the wrong event subject")
		}
		g.ResolveStack()
		if !g.FindPermanent(victim.ID()).Tapped {
			t.Fatal("combat opponent exclusion failed to resolve")
		}
	}
}
