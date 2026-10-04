package ron

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSpellTargetTypedDeclarations(t *testing.T) {
	for _, filter := range []string{"AnySpell", "IsCardtype(Artifact)", "IsColor(Green)", "And([IsCardtype(Creature), IsColor(Black)])", "Or([IsCardtype(Instant), IsCardtype(Sorcery)])"} {
		t.Run(filter, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Instant", fmt.Sprintf(`SpellActions(Targeted([TargetSpell(%s)], ActionList([CounterSpell(Ref_TargetSpell)])))`, filter))
			first, second := factory(), factory()
			action := first.Abilities()[0].(*mage.ActionDefinition)
			if action.Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
				t.Fatal("shared target declaration")
			}
			for _, card := range []mage.Card{first, second} {
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
			}
			reads := 0
			var walk func(mage.DefinitionMetadata)
			walk = func(n mage.DefinitionMetadata) {
				if n.Role == "stack-object-ref" && n.Text == "Ref_TargetSpell" {
					reads++
				}
				for _, child := range n.Children {
					walk(child)
				}
			}
			walk(mage.InspectDefinition(action))
			if reads != 1 {
				t.Fatalf("typed spell reads = %d", reads)
			}
			invalid := mage.NewInstant("Invalid", "", mage.WithAbility(mage.NewAction(mage.ActionSpell, mage.WithEffects(action.Effects()...))))
			if mage.ValidateCardDefinition(invalid.Name(), invalid) == nil {
				t.Fatal("missing declaration accepted")
			}
		})
	}
}

func TestRONSpellScopeInvalidDeclarations(t *testing.T) {
	for _, rule := range []string{
		`SpellActions(ActionList([CounterSpell(Ref_TargetSpell)]))`,
		`Activated(PayMana([]), Targeted([TargetSpell(AnySpell)], ActionList([]))), Activated(PayMana([]), ActionList([CounterSpell(Ref_TargetSpell)]))`,
		`SpellActions(Targeted([TargetPermanent(AnyPermanent)], ActionList([CounterSpell(Ref_TargetSpell)])))`,
		`SpellActions(Targeted([TargetSpell(AnySpell)], ActionList([CounterSpell(Ref_TargetSpell())])))`,
		`SpellActions(Targeted([TargetSpell(AnySpell)], ActionList([CounterSpell(Ref_TargetSpell1)])))`,
		`SpellActions(Targeted([TargetSpell(AnySpell)], ActionList([CounterSpell(ThisSpell)])))`,
		`SpellActions(Targeted([TargetSpell(AnySpell)], ActionList([]))), SpellActions(ActionList([CounterSpell(Ref_TargetSpell)]))`,
		`SpellActions(Targeted([TargetSpell(And([]))], ActionList([CounterSpell(Ref_TargetSpell)])))`,
		`SpellActions(Targeted([TargetSpell(Or([]))], ActionList([CounterSpell(Ref_TargetSpell)])))`,
		`SpellActions(Targeted([TargetSpell(IsColor(Purple))], ActionList([CounterSpell(Ref_TargetSpell)])))`,
		`SpellActions(Targeted([TargetSpell(IsCardtype(Creature()))], ActionList([CounterSpell(Ref_TargetSpell)])))`,
		`SpellActions(Targeted([TargetSpell(IsCardtype(Creature, Artifact))], ActionList([CounterSpell(Ref_TargetSpell)])))`,
		`SpellActions(Targeted([TargetSpell(And([AnySpell, ControlledByAPlayer(SinglePlayer(You))]))], ActionList([CounterSpell(Ref_TargetSpell)])))`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [%s])`, rule)), "invalid-spell.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("invalid declaration accepted: %s", rule)
		}
	}
}

func castRONSpell(t *testing.T, g *mage.Game, player *mage.BasePlayer, card mage.Card, targets ...uuid.UUID) *mage.StackObject {
	t.Helper()
	card.SetOwner(player.PlayerID())
	player.AddToHand(card)
	for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green, core.Colorless} {
		player.ManaPool().Add(color, 20)
	}
	if err := g.CastSpellByID(player.PlayerID(), card.ID(), targets, 0); err != nil {
		t.Fatal(err)
	}
	return g.StackPeek()
}

func TestRONRealFilteredCounters(t *testing.T) {
	for _, tc := range []struct {
		set, name, victim, victimSet string
		activated                    bool
	}{
		{"ATQ", "Artifact Blast", "Sol Ring", "LEB", false},
		{"LEG", "Flash Counter", "Jump", "LEB", false},
		{"LEG", "Remove Soul", "Air Elemental", "LEB", false},
		{"LEB", "Deathgrip", "Llanowar Elves", "LEB", true},
		{"LEB", "Lifeforce", "Frozen Shade", "LEB", true},
		{"LEB", "Counterspell", "Sol Ring", "LEB", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			victim := repositoryCard(t, tc.victimSet, tc.victim)
			var targets []uuid.UUID
			if tc.victim == "Jump" {
				creature := repositoryCard(t, "LEB", "Air Elemental")
				creature.SetOwner(a.PlayerID())
				targets = []uuid.UUID{g.PutOnBattlefield(creature, a.PlayerID()).ID()}
			}
			target := castRONSpell(t, g, a, victim, targets...)
			counter := repositoryCard(t, tc.set, tc.name)
			if tc.activated {
				counter.SetOwner(b.PlayerID())
				source := g.PutOnBattlefield(counter, b.PlayerID())
				if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID}); err == nil {
					t.Fatal("unpaid activation succeeded")
				}
				if g.StackSize() != 1 || g.FindPermanent(source.ID()) == nil {
					t.Fatal("failed payment changed state")
				}
				b.ManaPool().Add(core.Black, 2)
				b.ManaPool().Add(core.Green, 2)
				if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID}); err != nil {
					t.Fatal(err)
				}
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			} else {
				castRONSpell(t, g, b, counter, target.ID)
			}
			if g.GetStack().FindByID(target.ID) == nil {
				t.Fatal("counter acted before resolution")
			}
			clone := g.Clone()
			clone.ResolveTopOfStack()
			if clone.GetStack().FindByID(target.ID) != nil || g.GetStack().FindByID(target.ID) == nil {
				t.Fatal("clone shared target state")
			}
			g.ResolveTopOfStack()
			if g.GetStack().FindByID(target.ID) != nil || !slices.ContainsFunc(a.Graveyard(), func(c mage.Card) bool { return c.ID() == victim.ID() }) {
				t.Fatal("spell not countered into owner's graveyard")
			}
		})
	}
}

func TestRONSpellFiltersAndAnnouncement(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetSpell(And([Or([IsCardtype(Artifact), IsCardtype(Creature)]), IsColor(Green)]))], ActionList([CounterSpell(Ref_TargetSpell)])))`)
	g, a, b := ronGame()
	mixed := mage.NewCreature("Mixed", "{G}{B}", 2, 2, mage.WithCardType(core.TypeArtifact))
	mixed.SetOwner(b.PlayerID())
	good := &mage.StackObject{ID: uuid.New(), SourceID: mixed.ID(), Card: mixed, Controller: b.PlayerID()}
	g.PushStack(good)
	wrong := mage.NewInstant("Wrong", "{G}")
	wrong.SetOwner(b.PlayerID())
	bad := &mage.StackObject{ID: uuid.New(), SourceID: wrong.ID(), Card: wrong, Controller: b.PlayerID()}
	g.PushStack(bad)
	ability := &mage.StackObject{ID: uuid.New(), SourceID: mixed.ID(), Card: mixed, Controller: b.PlayerID(), IsAbility: true}
	g.PushStack(ability)
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	spec := card.Abilities()[0].(*mage.ActionDefinition).Targets()[0]
	before := g.StackSize()
	for range 2 {
		possible := spec.Possible(a.PlayerID(), card, g)
		if !slices.Equal(possible, []uuid.UUID{good.ID}) {
			t.Fatalf("possible = %v", possible)
		}
	}
	for _, id := range []uuid.UUID{bad.ID, ability.ID, uuid.New()} {
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{id}, 0); err == nil {
			t.Fatal("illegal spell target accepted")
		}
		if g.StackSize() != before || !slices.ContainsFunc(a.Hand(), func(c mage.Card) bool { return c.ID() == card.ID() }) {
			t.Fatal("invalid target changed state")
		}
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{good.ID}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if g.GetStack().FindByID(good.ID) != nil || g.GetStack().FindByID(bad.ID) == nil || g.GetStack().FindByID(ability.ID) == nil {
		t.Fatal("counter changed unrelated objects")
	}
}

func TestRONSpellCounterResolutionLegality(t *testing.T) {
	for _, change := range []string{"color", "removed", "recast", "uncounterable", "source-phased"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			victim := repositoryCard(t, "LEB", "Llanowar Elves")
			target := castRONSpell(t, g, a, victim)
			counter := repositoryCard(t, "LEB", "Deathgrip")
			counter.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(counter, b.PlayerID())
			b.ManaPool().Add(core.Black, 2)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID}); err != nil {
				t.Fatal(err)
			}
			var returned *mage.StackObject
			switch change {
			case "color":
				target.Card.(*mage.BaseCard).SetColorOverride([]core.Color{core.Blue})
			case "removed":
				g.GetStack().RemoveByID(target.ID)
			case "recast":
				g.GetStack().RemoveByID(target.ID)
				returned = &mage.StackObject{ID: uuid.New(), SourceID: victim.ID(), Card: victim, Controller: a.PlayerID()}
			case "uncounterable":
				mage.WithUncounterable()(target.Card.(*mage.BaseCard))
			case "source-phased":
				g.PhaseOut(source.ID())
			}
			if change == "recast" {
				obj := g.StackPeek()
				g.GetStack().RemoveByID(obj.ID)
				g.PushStack(returned)
				g.PushStack(obj)
			}
			g.ResolveTopOfStack()
			if change == "source-phased" {
				if g.GetStack().FindByID(target.ID) != nil {
					t.Fatal("source phasing ended the ability")
				}
			} else if change == "color" || change == "uncounterable" {
				if g.GetStack().FindByID(target.ID) == nil {
					t.Fatal("illegal or uncounterable spell countered")
				}
			} else if len(a.Graveyard()) != 0 {
				t.Fatal("absent target moved to graveyard")
			}
			if returned != nil && g.GetStack().FindByID(returned.ID) == nil {
				t.Fatal("old target followed a new cast")
			}
		})
	}
}

func TestRONSpellCounterCopyRetargetAndClone(t *testing.T) {
	g, a, b := ronGame()
	creature := repositoryCard(t, "LEB", "Air Elemental")
	creature.SetOwner(a.PlayerID())
	host := g.PutOnBattlefield(creature, a.PlayerID())
	first := castRONSpell(t, g, a, repositoryCard(t, "LEB", "Jump"), host.ID())
	second := castRONSpell(t, g, a, repositoryCard(t, "LEB", "Jump"), host.ID())
	counter := castRONSpell(t, g, b, repositoryCard(t, "LEG", "Flash Counter"), first.ID)
	cloned := g.Clone()
	copied := g.CopyStackObjectDirect(counter, b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy failed")
	}
	copied.Targets[0] = second.ID
	copied.TargetZones[second.ID] = core.ZoneStack
	g.ResolveTopOfStack()
	if g.GetStack().FindByID(second.ID) != nil || g.GetStack().FindByID(first.ID) == nil || counter.Targets[0] != first.ID {
		t.Fatal("copy lost its target identity")
	}
	g.ResolveTopOfStack()
	if g.GetStack().FindByID(first.ID) != nil || len(a.Graveyard()) != 2 {
		t.Fatal("original lost its target")
	}
	cloned.ResolveTopOfStack()
	if cloned.GetStack().FindByID(first.ID) != nil || cloned.GetStack().FindByID(second.ID) == nil || len(cloned.GetPlayer(a.PlayerID()).Graveyard()) != 1 {
		t.Fatal("clone shared copied target")
	}
}

func TestRONSpellTargetInTrigger(t *testing.T) {
	g, a, b := ronGame()
	target := castRONSpell(t, g, a, repositoryCard(t, "LEB", "Sol Ring"))
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), Targeted([TargetSpell(IsCardtype(Artifact))], ActionList([CounterSpell(Ref_TargetSpell)])))`)
	card := factory()
	card.SetOwner(b.PlayerID())
	source := g.PutOnBattlefield(card, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 || g.StackPeek().Targets[0] != target.ID {
		t.Fatal("trigger did not bind the spell at placement")
	}
	g.DestroyPermanent(g.FindPermanent(source.ID()))
	g.ResolveTopOfStack()
	if g.GetStack().FindByID(target.ID) != nil {
		t.Fatal("departed source ended the trigger")
	}
}
