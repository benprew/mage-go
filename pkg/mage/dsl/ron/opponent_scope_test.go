package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONOpponentCitanulDruid(t *testing.T) {
	for _, scenario := range []string{"normal", "own-cast", "nonartifact", "failed-cast", "copy-spell", "source-control-before", "new-opponent", "source-control-after", "source-depart", "source-return", "source-phase-before", "source-phase-after", "source-phase-cycle", "two-events", "copy-trigger"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ATQ", "Citanul Druid")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			current := source.ID()
			if scenario == "source-control-before" || scenario == "new-opponent" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if scenario == "source-phase-before" {
				g.PhaseOut(source.ID())
			}
			caster := b
			if scenario == "own-cast" || scenario == "new-opponent" {
				caster = a
			}
			name := "Sol Ring"
			if scenario == "nonartifact" {
				name = "Dark Ritual"
			}
			spell := repositoryCard(t, "LEB", name)
			spell.SetOwner(caster.PlayerID())
			caster.AddToHand(spell)
			g.SetActivePlayerIndex(1)
			if caster == a {
				g.SetActivePlayerIndex(0)
			}
			if scenario != "failed-cast" {
				caster.ManaPool().Add(core.Colorless, 1)
				caster.ManaPool().Add(core.Black, 1)
			}
			err := g.CastSpellByID(caster.PlayerID(), spell.ID(), nil, 0)
			if scenario == "failed-cast" {
				g.PutTriggersOnStack()
				if err == nil || !g.Stack().IsEmpty() || len(caster.Hand()) != 1 || source.Counters[core.P1P1] != 0 {
					t.Fatal("failed cast changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "copy-spell" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.PutTriggersOnStack()
			triggered := scenario != "own-cast" && scenario != "nonartifact" && scenario != "source-control-before" && scenario != "source-phase-before"
			wantStack := 1
			if triggered {
				wantStack++
			}
			if scenario == "copy-spell" {
				wantStack++
			}
			if g.StackSize() != wantStack || source.Counters[core.P1P1] != 0 {
				t.Fatal("event filtering or timing")
			}
			if !triggered {
				return
			}
			count := 1
			switch scenario {
			case "source-control-after":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "source-depart", "source-return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if scenario == "source-return" {
					current = g.PutOnBattlefield(card, b.PlayerID()).ID()
				}
			case "source-phase-after":
				g.PhaseOut(source.ID())
			case "source-phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "two-events":
				firstTrigger := g.GetStack().RemoveByID(g.StackPeek().ID)
				g.ResolveTopOfStack()
				castRONSpell(t, g, b, repositoryCard(t, "LEB", "Sol Ring"))
				g.PushStack(firstTrigger)
				g.PutTriggersOnStack()
				count = 2
			case "copy-trigger":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("trigger copy failed")
				}
				count = 2
			}
			clone := g.Clone()
			g.ResolveStack()
			expected := count
			if scenario == "source-return" || scenario == "source-phase-after" {
				expected = 0
			}
			if p := g.FindPermanentIncludingPhased(current); p != nil {
				if int(p.Counters[core.P1P1]) != expected || p.CurrentPower(g) != 1+expected || p.CurrentToughness(g) != 1+expected {
					t.Fatal("counter missed original incarnation")
				}
			} else if scenario != "source-depart" {
				t.Fatal("source missing")
			}
			if p := clone.FindPermanentIncludingPhased(current); p != nil && p.Counters[core.P1P1] != 0 {
				t.Fatal("resolution changed clone")
			}
			clone.ResolveStack()
			if p := clone.FindPermanentIncludingPhased(current); p != nil && int(p.Counters[core.P1P1]) != expected {
				t.Fatal("clone lost event")
			}
		})
	}
}

func TestRONOpponentPlayerTargets(t *testing.T) {
	for _, context := range []string{"spell", "activation", "trigger", "mode"} {
		for _, copy := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/copy=%v", context, copy), func(t *testing.T) {
				const actions = `Targeted([TargetPlayer(Opponent)],ActionList([PlayerAction(Ref_TargetPlayer,LoseLife(Integer(2)))]))`
				rule := `SpellActions(` + actions + `)`
				kind := "Instant"
				switch context {
				case "activation":
					kind = "Artifact"
					rule = `Activated(And([PayMana([ManaCostGeneric(1)]),TapPermanent(ThisPermanent)]),` + actions + `)`
				case "trigger":
					kind = "Artifact"
					rule = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),` + actions + `)`
				case "mode":
					rule = `SpellActions(Modal_ChooseOne([` + actions + `,ActionList([GainLife(Integer(3))])]))`
				}
				factory := permanentScopeFactory(t, kind, rule)
				first, second := factory(), factory()
				if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
					t.Fatal(err)
				}
				if first.Abilities()[0] == second.Abilities()[0] {
					t.Fatal("shared ability")
				}
				g, a, b := ronGame()
				first.SetOwner(a.PlayerID())
				if context == "spell" || context == "mode" {
					a.AddToHand(first)
					if err := g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{a.PlayerID()}, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
						t.Fatal("self target accepted or rollback failed")
					}
					if err := g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
						t.Fatal(err)
					}
				} else {
					source := g.PutOnBattlefield(first, a.PlayerID())
					if context == "activation" {
						if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil || source.Tapped {
							t.Fatal("unpaid activation changed state")
						}
						a.ManaPool().Add(core.Colorless, 1)
						if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{a.PlayerID()}); err == nil || g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 1 {
							t.Fatal("illegal target paid cost")
						}
						if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
							t.Fatal(err)
						}
					} else {
						g.PutTriggersOnStack()
						if g.StackSize() != 1 || g.StackPeek().Targets[0] != b.PlayerID() {
							t.Fatal("trigger target")
						}
					}
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				}
				if b.Life() != 20 {
					t.Fatal("effect ran at announcement")
				}
				if copy {
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					g.ResolveTopOfStack()
					if b.Life() != 20 || a.Life() != 20 {
						t.Fatalf("copy did not recheck opponent: a=%d b=%d stack=%d", a.Life(), b.Life(), g.StackSize())
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				if a.Life() != 20 || b.Life() != 18 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("controller or clone scope")
				}
				clone.ResolveStack()
				if clone.GetPlayer(b.PlayerID()).Life() != 18 {
					t.Fatal("clone binding")
				}
			})
		}
	}
}

func TestRONOpponentPermanentTargets(t *testing.T) {
	for _, change := range []string{"normal", "target-control", "source-control", "depart", "return", "phase", "phase-cycle", "copy"} {
		t.Run(change, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(And([IsCardtype(Creature),And([ControlledByAPlayer(Opponent),AnyPermanent])]))],ActionList([TapPermanent(Ref_TargetPermanent)])))`)
			g, a, b := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			own := g.PutOnBattlefield(mage.NewCreature("Own", "", 2, 2), a.PlayerID())
			targetCard := mage.NewCreature("Opponent", "", 2, 2)
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
			for _, id := range []uuid.UUID{own.ID(), land.ID()} {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{id}); err == nil || g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
					t.Fatal("illegal target paid cost")
				}
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "target-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "depart":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "return":
				g.RemoveFromBattlefield(target)
				target = g.PutOnBattlefield(targetCard, b.PlayerID())
			case "phase":
				g.PhaseOut(target.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveTopOfStack()
				if g.FindPermanent(target.ID()).Tapped {
					t.Fatal("copy legality")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			want := change != "target-control" && change != "return" && change != "phase"
			if g.FindPermanentIncludingPhased(target.ID()).Tapped != want || clone.FindPermanentIncludingPhased(target.ID()).Tapped {
				t.Fatal("target legality or clone")
			}
			clone.ResolveStack()
			if clone.FindPermanentIncludingPhased(target.ID()).Tapped != want {
				t.Fatal("clone lost target")
			}
		})
	}
}

func TestRONOpponentInvalidDeclarations(t *testing.T) {
	for _, rule := range []string{
		`SpellActions(Targeted([TargetPlayer(Opponent())],ActionList([])))`,
		`SpellActions(Targeted([TargetPlayer(SinglePlayer(Opponent))],ActionList([])))`,
		`SpellActions(Targeted([TargetPermanent(ControlledByAPlayer(Opponent()))],ActionList([])))`,
		`SpellActions(Targeted([TargetPermanent(Or([AnyPermanent,ControlledByAPlayer(Opponent)]))],ActionList([])))`,
		`SpellActions(Targeted([TargetPermanent(ControlledByAPlayer(Opponent,AnyPlayer))],ActionList([])))`,
		`TriggerA(WhenAPlayerCastsASpell(Opponent(),AnySpell),ActionList([]))`,
		`TriggerA(WhenAPlayerCastsASpell(SinglePlayer(Opponent),AnySpell),ActionList([]))`,
		`SpellActions(ActionList([PlayerAction(Opponent,LoseLife(Integer(1)))]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONOpponentCopyRetarget(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(Opponent)],ActionList([PlayerAction(Ref_TargetPlayer,LoseLife(Integer(2)))])))`)
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	cp := g.CopyStackObjectDirect(original, b.PlayerID(), true)
	if cp == nil || len(cp.Targets) != 1 || cp.Targets[0] != a.PlayerID() || original.Targets[0] != b.PlayerID() {
		t.Fatal("copy did not keep separate legal target")
	}
	clone := g.Clone()
	g.ResolveTopOfStack()
	if a.Life() != 18 || b.Life() != 20 {
		t.Fatal("copy used original player binding")
	}
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 18 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("original or clone binding")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 18 || clone.GetPlayer(b.PlayerID()).Life() != 18 {
		t.Fatal("clone lost independent copy targets")
	}
}
