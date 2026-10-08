package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedReturnGameplay(t *testing.T) {
	for _, entry := range []string{"EntersNormally", "EntersUnderPlayersControl(You)", "EntersUnderPlayersControl(ControllerOfPermanent(ThisPermanent))"} {
		for _, state := range []string{"live", "control", "departed", "returned", "phased", "cycle", "stale-card", "copy"} {
			t.Run(entry+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				rule := `Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetGraveyardCard(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[` + entry + `])]))])))`
				factory := permanentScopeFactory(t, "Artifact", rule)
				card := factory()
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				saved := g.CaptureDamageSource(source.ID())
				victim := waitingCombatFactory(t, `AsPermanentEnters(ThisPermanent,[EntersTapped]),TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([GainLife(Integer(2))]))`)()
				victim.SetOwner(b.PlayerID())
				b.AddToGraveyard(victim)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err == nil || g.StackSize() != 0 || len(b.Graveyard()) != 1 {
					t.Fatal("unpaid return changed state")
				}
				a.ManaPool().Add(core.Colorless, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
					t.Fatal(err)
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				if len(g.WaitingDelayedPermanentSources()) != 0 || g.FindPermanent(victim.ID()) != nil {
					t.Fatal("body ran at announcement")
				}
				g.ResolveStack()
				switch state {
				case "control", "departed", "returned", "phased", "cycle":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				switch state {
				case "departed", "returned":
					g.BouncePermanentToHand(g.FindPermanent(source.ID()))
					if state == "returned" {
						a.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, a.PlayerID())
					}
				case "phased":
					g.PhaseOut(source.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(source.ID()))
				case "stale-card":
					moved, ok := g.MoveFromGraveyard(b.PlayerID(), victim.ID(), core.ZoneHand)
					if !ok {
						t.Fatal("move failed")
					}
					b.AddToHand(moved)
					b.RemoveFromHand(victim.ID())
					b.AddToGraveyard(moved)
				}
				explicitSource := strings.Contains(entry, "ThisPermanent")
				for range 3 {
					refs := g.WaitingDelayedPermanentSources()
					want := 0
					if explicitSource {
						want = 1
					}
					if len(refs) != want || want == 1 && !refs[0].SameObject(saved) || g.FindPermanent(victim.ID()) != nil || a.Life() != 20 || b.Life() != 20 {
						t.Fatal("query lost source or ran return", refs)
					}
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{clone, g} {
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(game.WaitingDelayedPermanentSources()) != 0 || game.FindPermanent(victim.ID()) != nil {
						t.Fatal("detection ran body or retained waiting reference")
					}
					game.PutTriggersOnStack()
					game.ResolveTopOfStack()
					p := game.FindPermanent(victim.ID())
					if state == "stale-card" {
						if p != nil || len(game.GetPlayer(b.PlayerID()).Graveyard()) != 1 {
							t.Fatal("return followed a new graveyard visit")
						}
						continue
					}
					controller := a.PlayerID()
					if state == "copy" && !explicitSource {
						controller = b.PlayerID()
					}
					if explicitSource && (state == "control" || state == "departed" || state == "returned" || state == "phased" || state == "cycle") {
						controller = b.PlayerID()
					}
					if p == nil || !p.Tapped || p.ControllerID() != controller || p.Card.Owner() != b.PlayerID() || game.GetPlayer(controller).Life() != 20 {
						t.Fatalf("return lost entry replacement, exact controller or trigger timing: p=%+v wantController=%v life=%d graveyard=%d", p, controller, game.GetPlayer(controller).Life(), len(game.GetPlayer(b.PlayerID()).Graveyard()))
					}
					game.PutTriggersOnStack()
					game.ResolveStack()
					if game.GetPlayer(controller).Life() != 22 {
						t.Fatal("entry trigger used wrong player or copy returned twice")
					}
					if game == clone && (g.FindPermanent(victim.ID()) != nil || len(g.WaitingDelayedPermanentSources()) != boolCount(explicitSource)) {
						t.Fatal("clone changed original state")
					}
				}
			})
		}
	}
}

func boolCount(value bool) int {
	if value {
		return 1
	}
	return 0
}

func TestRONWaitingDelayedReturnInvalid(t *testing.T) {
	for _, entry := range []string{
		`EntersUnderPlayersControl`, `EntersUnderPlayersControl()`, `EntersUnderPlayersControl(You,You)`,
		`EntersUnderPlayersControl(You())`, `EntersUnderPlayersControl(ThisPermanent)`,
		`EntersUnderPlayersControl(Ref_TargetPlayer)`, `EntersUnderPlayersControl(Trigger_ThatPlayer)`,
		`EntersUnderPlayersControl(ControllerOfPermanent(Ref_TargetPermanent))`, `EntersUnderPlayersControl(HostController)`,
		`EntersUnderPlayersControl(You),EntersNormally`, `EntersUnderPlayersControl(You),EntersUnderPlayersControl(You)`,
		`EntersUnderPlayersControl(You),EntersTapped`,
	} {
		assertGraveyardInvalid(t, `SpellActions(Targeted([TargetGraveyardCard(IsCardtype(Creature))],ActionList([PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[`+entry+`])])))`)
	}
	assertGraveyardInvalid(t, `SpellActions(Targeted([TargetGraveyardCard(IsCardtype(Instant))],ActionList([PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[EntersUnderPlayersControl(You)])])))`)
}

func TestRONWaitingDelayedReturnInstances(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetGraveyardCard(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[EntersUnderPlayersControl(ControllerOfPermanent(ThisPermanent))])]))])))`)
	victims := make([]mage.Card, 0, 2)
	for _, player := range []*mage.BasePlayer{a, b} {
		card := factory()
		card.SetOwner(player.PlayerID())
		source := g.PutOnBattlefield(card, player.PlayerID())
		victim := repositoryCard(t, "LEB", "Savannah Lions")
		victim.SetOwner(player.PlayerID())
		player.AddToGraveyard(victim)
		if err := g.ActivateAbilityByIndex(player.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		victims = append(victims, victim)
	}
	if len(g.WaitingDelayedPermanentSources()) != 2 {
		t.Fatal("factories shared source bindings")
	}
	clone := g.Clone()
	for _, game := range []*mage.Game{clone, g} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		for _, victim := range victims {
			p := game.FindPermanent(victim.ID())
			if p == nil || p.ControllerID() != victim.Owner() {
				t.Fatal("return used another instance's controller or target")
			}
		}
		if game == clone && len(g.WaitingDelayedPermanentSources()) != 2 {
			t.Fatal("clone consumed original delays")
		}
	}
}

func TestRONWaitingDelayedReturnHost(t *testing.T) {
	g, a, b := ronGame()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Host return",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),Activated(PayMana([]),Targeted([TargetGraveyardCard(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[EntersUnderPlayersControl(HostController)])]))])))])`), "host.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	card := factory()
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	host := g.PutOnBattlefield(repositoryCard(t, "LEB", "Savannah Lions"), a.PlayerID())
	later := g.PutOnBattlefield(repositoryCard(t, "LEB", "Savannah Lions"), b.PlayerID())
	g.Attach(source.ID(), host.ID())
	victim := repositoryCard(t, "LEB", "Savannah Lions")
	victim.SetOwner(a.PlayerID())
	a.AddToGraveyard(victim)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.Attach(source.ID(), later.ID())
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("query inferred implicit attachment")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	p := g.FindPermanent(victim.ID())
	if p == nil || p.ControllerID() != b.PlayerID() || p.Card.Owner() != a.PlayerID() {
		t.Fatal("return did not read current host controller")
	}
}

func TestRONWaitingDelayedReturnInventory(t *testing.T) {
	src := `Card(Name:"Return",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),Targeted([TargetGraveyardCard(IsCardtype(Creature))],ActionList([PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[EntersUnderPlayersControl(ControllerOfPermanent(ThisPermanent))])])))] )`
	files := fstest.MapFS{"return.ron": &fstest.MapFile{Data: []byte(src)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal("valid controller entry rejected", report)
	}
	for name, role := range map[string]string{"EntersUnderPlayersControl": "entry_operation", "ControllerOfPermanent": "player_reference", "ThisPermanent": "permanent_reference"} {
		found := false
		for _, sig := range report.Signatures {
			found = found || sig.Name == name && sig.Role == role
		}
		if !found {
			t.Fatal("missing inventory role", name, role)
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.ReplaceAll(src, `Name:"Return"`, `Name:"Invalid"`), "ControllerOfPermanent(ThisPermanent)", "Ref_TargetPlayer"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid controller replaced installed collection")
	}
}

func TestRONWaitingDelayedReturnProtection(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
	}{
		{"LEB", "Circle of Protection: Black", core.Black}, {"LEA", "Circle of Protection: Blue", core.Blue}, {"LEA", "Circle of Protection: Green", core.Green}, {"LEA", "Circle of Protection: Red", core.Red}, {"LEA", "Circle of Protection: White", core.White}, {"ATQ", "Circle of Protection: Artifacts", core.Colorless}, {"LEG", "Greater Realm of Preservation", core.Black},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := staticBaseRecord(t, tc.set, tc.name)
			rule := record.Fields["Rules"].Args[0]
			actions := rule.Args[1].Args[0].Args
			plan, err := compileChosenDamageSourceChoice(actions[0])
			if err != nil {
				t.Fatal(err)
			}
			shield, err := compileExactSourcePrevention(actions[1], scope{chosenDamageSource: plan})
			if err != nil {
				t.Fatal(err)
			}
			costs, _, err := compileCosts(rule.Args[0])
			if err != nil {
				t.Fatal(err)
			}
			a := &ronSourceChoicePlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			factory := waitingCombatFactory(t, `Activated(PayMana([]),Targeted([TargetGraveyardCard(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[EntersUnderPlayersControl(ControllerOfPermanent(ThisPermanent))])]))])))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.ChangeObjectColor(p.ID(), tc.color)
			saved := g.CaptureDamageSource(p.ID())
			victim := repositoryCard(t, "LEB", "Savannah Lions")
			victim.SetOwner(b.PlayerID())
			b.AddToGraveyard(victim)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			g.BouncePermanentToHand(p)
			binding := mage.BindDamageSource("waiting shield")
			bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
			payment := make([]mage.Cost, len(costs))
			for i, cost := range costs {
				payment[i] = cost()
			}
			consumer := mage.NewArtifact("Consumer", "", mage.WithAbility(mage.Activate(mage.Costs(payment...), mage.Sequence(plan.waitingDelayedPermanentChoice(bindings), shield(bindings)))))
			if err := mage.ValidateCardDefinition(consumer.Name(), consumer); err != nil {
				t.Fatal(err)
			}
			q := g.PutOnBattlefield(consumer, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err == nil || a.calls != 0 || g.StackSize() != 0 {
				t.Fatal("unpaid choice changed state")
			}
			a.ManaPool().Add(core.White, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.calls != 0 {
				t.Fatal("early source selection")
			}
			g.ResolveStack()
			if a.calls != 1 || !a.source.SameObject(saved) {
				t.Fatal("waiting shield body omitted departed source")
			}
			g.DealDamageToPlayerFromSource(a, 2, saved)
			g.DealDamageToPlayerFromSource(a, 1, saved)
			if a.Life() != 19 {
				t.Fatal("shield prevention or consumption failed")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if len(g.WaitingDelayedPermanentSources()) != 0 || g.FindPermanent(victim.ID()) == nil || g.FindPermanent(victim.ID()).ControllerID() != a.PlayerID() {
				t.Fatal("return delay remained waiting")
			}
			if _, err := Compile(record); err != nil {
				t.Fatal("complete source-choice record rejected", err)
			}
		})
	}
}
