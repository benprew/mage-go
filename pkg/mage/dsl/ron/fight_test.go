package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONFightArena(t *testing.T) {
	for _, state := range []string{"live", "returned", "phase", "cycle", "shroud", "control", "noncreature", "source-leaves", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := opponentTargetGame()
			card := repositoryCard(t, "PROMOS", "Arena")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			first := g.PutOnBattlefield(mage.NewCreature("Mine", "", 2, 10), a.PlayerID())
			other := mage.NewCreature("Theirs", "", 3, 10)
			other.SetOwner(b.PlayerID())
			second := g.PutOnBattlefield(other, b.PlayerID())
			b.requested = []uuid.UUID{second.ID()}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{first.ID()}); err == nil || g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
				t.Fatal("unpaid fight changed state")
			}
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{first.ID()}); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source.ID()).Tapped || g.FindPermanent(first.ID()).Tapped || g.FindPermanent(second.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
				t.Fatal("cost or body timing")
			}
			switch state {
			case "returned":
				g.BouncePermanentToHand(second)
				b.RemoveFromHand(other.ID())
				second = g.PutOnBattlefield(other, b.PlayerID())
			case "phase":
				g.PhaseOut(second.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(second.ID()))
			case "shroud":
				g.MutablePermanent(second.ID()).GrantBaseAttr(core.Shroud)
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: second.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "noncreature":
				g.MutablePermanent(second.ID()).RevokeBaseAttr(core.AttrIsCreature)
			case "source-leaves":
				g.BouncePermanentToHand(source)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{clone, g} {
				game.ResolveStack()
				wantFirst, wantSecond := 3, 2
				illegal := state == "returned" || state == "phase" || state == "shroud" || state == "control" || state == "noncreature"
				if illegal {
					wantFirst, wantSecond = 0, 0
				}
				if state == "copy" {
					wantFirst, wantSecond = 6, 4
				}
				p, q := game.FindPermanent(first.ID()), game.FindPermanentIncludingPhased(second.ID())
				if p.Damage != wantFirst || q.Damage != wantSecond || !p.Tapped || q.Tapped == illegal {
					t.Fatalf("fight/tap result: %d/%d tapped %v/%v", p.Damage, q.Damage, p.Tapped, q.Tapped)
				}
				if game == clone && g.FindPermanent(first.ID()).Damage != 0 {
					t.Fatal("clone changed original")
				}
			}
		})
	}
}

func TestRONFightInvalid(t *testing.T) {
	for _, action := range []string{`Fight`, `Fight()`, `Fight(ThisPermanent)`, `Fight(ThisPermanent,ThisPermanent,ThisPermanent)`, `Fight(You,ThisPermanent)`, `Fight(ThisPermanent,You)`, `Fight(ThisPermanent(),ThisPermanent)`, `Fight(HostPermanent,ThisPermanent)`, `Fight(Ref_TargetPermanent,ThisPermanent)`, `Fight(ThisPermanent,Ref_TargetPermanent2)`, `Fight(Trigger_ThatPermanent,ThisPermanent)`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+action+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid fight", action)
		}
	}
}

func TestRONFightDelayed(t *testing.T) {
	for _, operand := range []string{"Fight(ThisPermanent,Ref_TargetPermanent)", "Fight(Ref_TargetPermanent,ThisPermanent)"} {
		for _, state := range []string{"live", "departed", "returned", "phase", "cycle", "control", "copy"} {
			t.Run(operand+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				factory := waitingCombatFactory(t, `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+operand+`]))])))`)
				card := factory()
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				target := g.PutOnBattlefield(mage.NewCreature("Target", "", 2, 10), b.PlayerID())
				savedSource, savedTarget := g.CaptureDamageSource(source.ID()), g.CaptureDamageSource(target.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
					t.Fatal(err)
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				if len(g.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("early waiting reference")
				}
				g.ResolveStack()
				switch state {
				case "departed", "returned":
					g.BouncePermanentToHand(source)
					if state == "returned" {
						a.RemoveFromHand(card.ID())
						source = g.PutOnBattlefield(card, a.PlayerID())
					}
				case "phase":
					g.PhaseOut(source.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(source.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: target.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
				for range 3 {
					refs := g.WaitingDelayedPermanentSources()
					haveSource, haveTarget := false, false
					for _, ref := range refs {
						haveSource = haveSource || ref.SameObject(savedSource)
						haveTarget = haveTarget || ref.SameObject(savedTarget)
					}
					if len(refs) != 2 || !haveSource || !haveTarget || g.FindPermanent(target.ID()).Damage != 0 {
						t.Fatal("waiting query lost operand or dealt damage")
					}
				}
				for _, game := range []*mage.Game{g.Clone(), g} {
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(game.WaitingDelayedPermanentSources()) != 0 || game.FindPermanent(target.ID()).Damage != 0 {
						t.Fatal("detection retained wait or fought early")
					}
					game.PutTriggersOnStack()
					game.ResolveStack()
					want := source.CurrentPower(game)
					if state == "departed" || state == "returned" || state == "phase" {
						want = 0
					}
					if state == "copy" {
						want *= 2
					}
					if game.FindPermanent(target.ID()).Damage != want {
						t.Fatalf("delayed damage=%d want=%d", game.FindPermanent(target.ID()).Damage, want)
					}
				}
			})
		}
	}
}

func TestRONFightSelfAndNoncombat(t *testing.T) {
	for _, power := range []int{-1, 0, 2} {
		g, a, _ := ronGame()
		factory := waitingCombatFactory(t, `Activated(PayMana([]),ActionList([Fight(ThisPermanent,ThisPermanent)]))`)
		p := g.PutOnBattlefield(factory(), a.PlayerID())
		if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetBasePower(mage.This, mage.Fixed(power))), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.SetPreventCombatDamage()
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if g.FindPermanent(p.ID()).Damage != max(0, 2*power) {
			t.Fatal("self fight amount or combat prevention")
		}
	}
}

func TestRONFightHost(t *testing.T) {
	g, a, b := ronGame()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Host",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([Fight(HostPermanent,HostPermanent)]))]))])`), "host.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	card := factory()
	source := g.PutOnBattlefield(card, a.PlayerID())
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 1, 10), a.PlayerID())
	later := g.PutOnBattlefield(mage.NewCreature("Later", "", 2, 10), b.PlayerID())
	g.Attach(source.ID(), host.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("inferred implicit host")
	}
	g.Attach(source.ID(), later.ID())
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(host.ID()).Damage != 0 || g.FindPermanent(later.ID()).Damage != 4 {
		t.Fatal("host did not use resolution attachment")
	}
}

func TestRONFightWaitingProtection(t *testing.T) {
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
			factory := waitingCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([Fight(ThisPermanent,ThisPermanent)]))]))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.ChangeObjectColor(p.ID(), tc.color)
			saved := g.CaptureDamageSource(p.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
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
			if len(g.WaitingDelayedPermanentSources()) != 0 {
				t.Fatal("fight delay remained waiting")
			}
			if _, err := Compile(record); err != nil {
				t.Fatal("complete source-choice record rejected", err)
			}
		})
	}
}

func TestRONFightInstances(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature)),TargetPermanent(IsCardtype(Creature))],ActionList([Fight(Ref_TargetPermanent1,Ref_TargetPermanent2)])))`)
	pairs := make([][2]uuid.UUID, 0, 2)
	for _, player := range []*mage.BasePlayer{a, b} {
		first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 10), player.PlayerID())
		second := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 10), player.PlayerID())
		source := g.PutOnBattlefield(factory(), player.PlayerID())
		if err := g.ActivateAbilityByIndex(player.PlayerID(), source.ID(), 0, []uuid.UUID{first.ID(), second.ID()}); err != nil {
			t.Fatal(err)
		}
		pairs = append(pairs, [2]uuid.UUID{first.ID(), second.ID()})
	}
	for _, game := range []*mage.Game{g.Clone(), g} {
		game.ResolveTopOfStack()
		if game.FindPermanent(pairs[0][0]).Damage != 0 || game.FindPermanent(pairs[1][0]).Damage != 3 || game.FindPermanent(pairs[1][1]).Damage != 2 {
			t.Fatal("factories shared target bindings")
		}
		game.ResolveStack()
		if game.FindPermanent(pairs[0][0]).Damage != 3 || game.FindPermanent(pairs[0][1]).Damage != 2 {
			t.Fatal("first factory lost bindings")
		}
	}
}

func TestRONFightInventoryAndStrictLoad(t *testing.T) {
	src := `Card(Name:"Fight",Typeline:(Supertypes:[],Cardtypes:[Artifact,Creature],Subtypes:[]),CardPT:(Power:2,Toughness:5),Rules:[Activated(PayMana([]),ActionList([Fight(ThisPermanent,ThisPermanent)]))])`
	files := fstest.MapFS{"fight.ron": {Data: []byte(src)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	refs := 0
	for _, sig := range report.Signatures {
		if sig.Name == "ThisPermanent" {
			if sig.Role != "permanent_reference" {
				t.Fatal("wrong fight operand role", sig.Role)
			}
			refs++
		}
	}
	if refs == 0 || report.Compiled != 1 {
		t.Fatal("missing fight inventory evidence")
	}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(src, "Fight(ThisPermanent,ThisPermanent)", "Fight(ThisPermanent,You)"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection changed registry")
	}
}
