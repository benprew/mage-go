package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

type ronDefenderPlayer struct {
	*gametest.TestPlayer
	accept bool
	offers int
}

type ronDefenderTargetPlayer struct {
	*ronPaymentPlayer
	preferred uuid.UUID
}

func (p *ronDefenderTargetPlayer) ChooseTargets(possible []uuid.UUID, minimum, maximum int, g *mage.Game) []uuid.UUID {
	for _, id := range possible {
		if id == p.preferred {
			return []uuid.UUID{id}
		}
	}
	return p.BasePlayer.ChooseTargets(possible, minimum, maximum, g)
}

func TestRONDefendingPlayerInstancesAndCopy(t *testing.T) {
	n := hostUntapRecord(t, "FEM", "Necrite")
	factory, err := Compile(n)
	if err != nil {
		t.Fatal(err)
	}
	first, second := factory(), factory()
	if first.Abilities()[0].(*mage.GenericTriggered).Targets()[0] == second.Abilities()[0].(*mage.GenericTriggered).Targets()[0] {
		t.Fatal("shared target declarations")
	}
	a := &ronDefenderTargetPlayer{ronPaymentPlayer: &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}}
	b := &ronDefenderTargetPlayer{ronPaymentPlayer: &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}}
	g := mage.NewGame(a, b)
	one := g.PutOnBattlefield(first, a.PlayerID())
	two := g.PutOnBattlefield(second, a.PlayerID())
	firstVictim := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 3), b.PlayerID())
	secondVictim := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 3), b.PlayerID())
	a.preferred = firstVictim.ID()
	g.FireEvent(core.GameEvent{Type: core.EvtAttacksUnblocked, SourceID: one.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
	g.PutTriggersOnStack()
	a.preferred = secondVictim.ID()
	copied := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true)
	if copied == nil || copied.Targets[0] != secondVictim.ID() || g.StackObjects()[0].Targets[0] != firstVictim.ID() {
		t.Fatal("copy retargeting changed the original or lost defender")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(one.ID()) != nil || g.FindPermanent(secondVictim.ID()) != nil || g.FindPermanent(firstVictim.ID()) == nil {
		t.Fatal("copy used wrong source or target")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(firstVictim.ID()) == nil {
		t.Fatal("source paid twice")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtAttacksUnblocked, SourceID: two.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(two.ID()) != nil || g.FindPermanent(firstVictim.ID()) != nil {
		t.Fatal("second instance lost its own event or target")
	}
}

func TestRONDefendingPlayerScopeAndInventory(t *testing.T) {
	for _, event := range []string{"WhenACreatureAttacks", "WhenACreatureAttacksAndIsntBlocked"} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Defender contract",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:2),Rules:[TriggerA(`+event+`(SinglePermanent(ThisPermanent)),ActionList([PlayerAction(Trigger_DefendingPlayer,GainLife(Integer(1)))]))])`), "defender.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		card := factory()
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		source := g.PutOnBattlefield(card, a.PlayerID())
		eventType := core.EvtDeclaredAttacker
		if event == "WhenACreatureAttacksAndIsntBlocked" {
			eventType = core.EvtAttacksUnblocked
		}
		g.FireEvent(core.GameEvent{Type: eventType, SourceID: source.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
		g.PutTriggersOnStack()
		if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
			t.Fatal("changed-controller copy failed")
		}
		g.RemoveFromBattlefield(source)
		g.PutOnBattlefield(card, b.PlayerID())
		clone := g.Clone()
		g.ResolveStack()
		clone.ResolveStack()
		for _, game := range []*mage.Game{g, clone} {
			if game.GetPlayer(b.PlayerID()).Life() != 22 || game.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("defender followed source or copy controller")
			}
		}
	}
	var records []string
	for line := range strings.SplitSeq(mustRONSet(t, "FEM"), "\n") {
		if strings.Contains(line, `Name: "Mindstab Thrull"`) || strings.Contains(line, `Name: "Necrite"`) {
			records = append(records, line)
		}
	}
	files := fstest.MapFS{"a.ron": {Data: []byte(strings.Join(records, "\n"))}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatalf("isolated complete-record registry: %v", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, sig := range report.Signatures {
		if sig.Name == "Trigger_DefendingPlayer" && sig.Role != "player_reference" {
			t.Fatal("defender inventory lost its role")
		}
	}
}

func (p *ronDefenderPlayer) ChooseMayAbility(string) bool { p.offers++; return p.accept }

func TestRONDefendingPlayerNative(t *testing.T) {
	for _, name := range []string{"Mindstab Thrull", "Necrite"} {
		for _, scenario := range []string{"pay", "decline", "blocked", "no-target"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				a := &ronDefenderPlayer{TestPlayer: gametest.NewTestPlayer("A"), accept: scenario != "decline"}
				b := gametest.NewTestPlayer("B")
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				card := repositoryCard(t, "FEM", name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				g.SetStep(core.PrecombatMain)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("unpaid cast changed state")
				}
				a.ManaPool().Add(core.Black, 3)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				source := g.FindPermanent(card.ID())
				if source == nil || a.ManaPool().TotalMana() != 0 || a.offers != 0 {
					t.Fatal("cast or payment timing")
				}
				source.RevokeBaseAttr(core.AttrSummonSick)
				victimCard := mage.NewCreature("Victim", "", 0, 5)
				victimCard.SetOwner(b.PlayerID())
				victim := g.PutOnBattlefield(victimCard, b.PlayerID())
				if scenario == "no-target" {
					g.PhaseOut(victim.ID())
				}
				for range 4 {
					addRONHandCard(b.BasePlayer, mage.NewArtifact("Hand", ""))
				}
				a.SetAttackers(1, []string{name})
				if scenario == "blocked" {
					b.SetBlockers(1, map[string]string{"Victim": name})
				}
				g.RunStepWithPriority(core.DeclareAttackers)
				if !g.GetCombat().IsAttacking(source.ID()) || a.offers != 0 || len(b.Hand()) != 4 {
					t.Fatal("acted before blockers")
				}
				g.RunStepWithPriority(core.DeclareBlockers)
				paid := scenario != "decline" && scenario != "blocked" && (name != "Necrite" || scenario != "no-target")
				if (g.FindPermanent(source.ID()) == nil) != paid {
					t.Fatal("wrong sacrifice")
				}
				if name == "Mindstab Thrull" {
					want := 4
					if paid {
						want = 1
					}
					if len(b.Hand()) != want || len(a.Hand()) != 0 {
						t.Fatal("wrong discard player or count")
					}
				} else if (g.FindPermanentIncludingPhased(victim.ID()) == nil) != paid {
					t.Fatal("wrong destruction")
				}
			})
		}
	}
}

func TestRONDefendingPlayerDeclarations(t *testing.T) {
	const event = `WhenACreatureAttacksAndIsntBlocked(SinglePermanent(ThisPermanent))`
	const body = `Targeted([TargetPermanent(And([IsCardtype(Creature),ControlledByPlayer(Trigger_DefendingPlayer)]))],ActionList([TapPermanent(Ref_TargetPermanent)]))`
	for _, bad := range []string{
		strings.Replace(body, "Trigger_DefendingPlayer", "Trigger_DefendingPlayer()", 1),
		strings.Replace(body, "Trigger_DefendingPlayer", "Trigger_ThatCreature", 1),
		strings.Replace(body, "Trigger_DefendingPlayer", "Ref_TargetPlayer", 1),
		strings.Replace(body, "ControlledByPlayer(Trigger_DefendingPlayer)", "ControlledByPlayer()", 1),
		strings.Replace(body, "ControlledByPlayer(Trigger_DefendingPlayer)", "ControlledByPlayer(Trigger_DefendingPlayer,You)", 1),
		strings.Replace(body, "And([IsCardtype(Creature),ControlledByPlayer(Trigger_DefendingPlayer)])", "Or([IsCardtype(Creature),ControlledByPlayer(Trigger_DefendingPlayer)])", 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:2),Rules:[TriggerA(`+event+`,`+bad+`)])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	for _, rule := range []string{`Activated(PayMana([]),` + body + `)`, `TriggerA(WhenACreatureBlocks(SinglePermanent(ThisPermanent)),` + body + `)`, `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),` + body + `)`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:2),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted out-of-scope %s", rule)
		}
	}
}

func TestRONDefendingPlayerControllerFilter(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(And([IsCardtype(Creature),And([ControlledByPlayer(You),Other(ThisPermanent)])]))],ActionList([TapPermanent(Ref_TargetPermanent)])))`)
	card := factory()
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	source := g.PutOnBattlefield(card, a.PlayerID())
	own := g.PutOnBattlefield(mage.NewCreature("Own", "", 2, 3), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 3), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{other.ID()}); err == nil || g.StackSize() != 0 {
		t.Fatal("opponent target accepted")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{own.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !g.FindPermanent(own.ID()).Tapped || g.FindPermanent(other.ID()).Tapped {
		t.Fatal("controller filter or nested conjunction")
	}
}

func TestRONDefendingPlayerPending(t *testing.T) {
	for _, name := range []string{"Mindstab Thrull", "Necrite"} {
		for _, scenario := range []string{"pay", "decline", "empty-hand", "short-hand", "late-hand", "removed-before", "phased-before", "source-control", "source-return", "source-left", "source-phase", "source-cycle", "target-control", "target-return", "target-phase", "target-cycle", "target-shroud", "removed-after", "copy", "replaced", "regeneration", "indestructible"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "decline"}
				b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
				g := mage.NewGame(a, b)
				card := repositoryCard(t, "FEM", name)
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				victimCard := mage.NewCreature("Victim", "", 2, 5)
				victimCard.SetOwner(b.PlayerID())
				victim := g.PutOnBattlefield(victimCard, b.PlayerID())
				hand := 4
				if scenario == "empty-hand" {
					hand = 0
				}
				if scenario == "short-hand" || scenario == "late-hand" {
					hand = 2
				}
				for range hand {
					addRONHandCard(b.BasePlayer, mage.NewArtifact("Hand", ""))
				}
				if scenario == "removed-before" {
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
					g.ApplyContinuousEffects()
				}
				if scenario == "phased-before" {
					g.PhaseOut(source.ID())
				}
				g.FireEvent(core.GameEvent{Type: core.EvtAttacksUnblocked, SourceID: source.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
				g.PutTriggersOnStack()
				if scenario == "removed-before" || scenario == "phased-before" {
					if g.StackSize() != 0 || a.offers != 0 {
						t.Fatal("inactive source triggered")
					}
					return
				}
				if g.StackSize() != 1 || a.offers != 0 {
					t.Fatal("trigger or payment timing")
				}
				if name == "Necrite" && (len(g.StackPeek().Targets) != 1 || g.StackPeek().Targets[0] != victim.ID()) {
					t.Fatal("wrong defending-player target")
				}
				if scenario == "late-hand" {
					for range 3 {
						addRONHandCard(b.BasePlayer, mage.NewArtifact("Late", ""))
					}
					hand += 3
				}
				switch scenario {
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: victim.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "source-return", "source-left":
					g.RemoveFromBattlefield(source)
					if scenario == "source-return" {
						g.PutOnBattlefield(card, a.PlayerID())
					}
				case "source-phase":
					g.PhaseOut(source.ID())
				case "source-cycle":
					g.PhaseIn(g.PhaseOut(source.ID()))
				case "target-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "target-return":
					g.RemoveFromBattlefield(victim)
					g.PutOnBattlefield(victimCard, b.PlayerID())
				case "target-phase":
					g.PhaseOut(victim.ID())
				case "target-cycle":
					g.PhaseIn(g.PhaseOut(victim.ID()))
				case "target-shroud":
					victim.GrantBaseAttr(core.Shroud)
				case "removed-after":
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
					g.ApplyContinuousEffects()
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				case "replaced":
					g.AddExileIfWouldGoToGraveyardThisTurn(source.ID(), uuid.New())
				case "regeneration":
					g.AddRegenerationShield(victim.ID())
				case "indestructible":
					victim.GrantBaseAttr(core.Indestructible)
				}
				paid := scenario != "decline" && scenario != "source-control" && scenario != "source-return" && scenario != "source-left" && scenario != "source-phase"
				if name == "Necrite" && (scenario == "target-control" || scenario == "target-return" || scenario == "target-phase" || scenario == "target-shroud") {
					paid = false
				}
				clone := g.Clone()
				g.ResolveStack()
				if clone.StackSize() == 0 || len(clone.GetPlayer(b.PlayerID()).Hand()) != hand {
					t.Fatal("resolution changed clone")
				}
				if name == "Mindstab Thrull" {
					want := hand
					if paid {
						want = max(hand-3, 0)
					}
					if len(b.Hand()) != want {
						t.Fatal("defender capture or sacrifice result")
					}
				} else {
					destroyed := paid && scenario != "indestructible"
					if (g.FindPermanentIncludingPhased(victim.ID()) == nil) != destroyed {
						t.Fatal("legality or no-regeneration destruction")
					}
				}
				if paid && g.FindPermanentIncludingPhased(source.ID()) != nil {
					t.Fatal("paid source remains")
				}
				if !paid && scenario != "source-left" && g.FindPermanentIncludingPhased(source.ID()) == nil {
					t.Fatal("unpaid source lost")
				}
				if scenario == "replaced" && (len(a.Graveyard()) != 0 || len(g.GetExile()) != 1) {
					t.Fatal("sacrifice replacement")
				}
				clone.ResolveStack()
				if g.StackSize() != 0 {
					t.Fatal("clone changed live stack")
				}
			})
		}
	}
}
