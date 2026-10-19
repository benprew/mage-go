package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONGlobalTargetQueryGameplay(t *testing.T) {
	for _, name := range []string{"Artifact Ward", "Anti-Magic Aura", "Bartel Runeaxe", "Tetsuo Umezawa", "Raiding Party"} {
		for _, state := range []string{"active", "removed", "phase", "departed"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				set := "LEG"
				if name == "Artifact Ward" {
					set = "ATQ"
				}
				if name == "Raiding Party" {
					set = "FEM"
				}
				source := g.PutOnBattlefield(repositoryCard(t, set, name), a.PlayerID())
				victim := source
				if name == "Artifact Ward" || name == "Anti-Magic Aura" {
					victim = g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 4), a.PlayerID())
					g.Attach(source.ID(), victim.ID())
				}
				var obj *mage.StackObject
				if name == "Artifact Ward" {
					factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(0)]),Targeted([TargetPermanent(AnyPermanent)],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`)
					actor := g.PutOnBattlefield(factory(), b.PlayerID())
					g.PhaseOut(source.ID())
					if err := g.ActivateAbilityByIndex(b.PlayerID(), actor.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
						t.Fatal(err)
					}
					g.PhaseIn([]uuid.UUID{source.ID()})
					obj = g.StackPeek()
				} else {
					factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`)
					if victim.ID() != source.ID() {
						g.PhaseOut(source.ID())
					}
					obj = castRONSpell(t, g, b, factory(), victim.ID())
					if victim.ID() != source.ID() {
						g.PhaseIn([]uuid.UUID{source.ID()})
					}
					if name == "Raiding Party" {
						colors := []core.Color{core.White}
						obj.ColorOverride = &colors
					}
				}
				switch state {
				case "removed":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), source.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "phase":
					g.PhaseOut(source.ID())
				case "departed":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				}
				instructions, ok := mage.InspectDirectDestruction(obj.Effects[0])
				if !ok {
					t.Fatal("missing destruction")
				}
				plan, ok := mage.InspectPermanentTarget(instructions[0].Subject)
				if !ok {
					t.Fatal("missing target plan")
				}
				want := state != "active"
				if name == "Bartel Runeaxe" || name == "Tetsuo Umezawa" {
					want = true
				}
				if victim.ID() == source.ID() && (state == "departed" || state == "phase") {
					want = false
				}
				for _, game := range []*mage.Game{g, g.Clone()} {
					got, known := plan.Select(game, game.StackPeek())
					if !known || (len(got) == 1) != want {
						t.Fatalf("got %v known %v want legal %v", got, known, want)
					}
					game.ResolveStack()
					absent := game.FindPermanentIncludingPhased(victim.ID()) == nil
					if absent != (want || (state == "departed" && victim.ID() == source.ID())) {
						t.Fatal("native resolution differs")
					}
				}
			})
		}
	}
}

func TestRONGlobalTargetQueryFilters(t *testing.T) {
	for _, syntax := range []string{`AnySpell`, `IsColor(Red)`, `IsCardtype(Sorcery)`, `IsEnchantmentType(Aura)`, `And([IsColor(Red),Or([IsCardtype(Sorcery),IsCardtype(Instant)])])`} {
		g, p, obj, binding := ronTargetQueryFixture(t)
		filter, err := compileSpellCardFilter(nameScopeNode(t, syntax), true)
		if err != nil {
			t.Fatal(err)
		}
		source := g.PutOnBattlefield(mage.NewAura("Rule", "", mage.WithCastTarget(mage.TargetLand()), mage.WithAbility(mage.Static(mage.CantBeTargetedBySpells(mage.AttachedPermanent, filter)))), obj.Controller)
		g.Attach(source.ID(), p.ID())
		got, known := binding.Select(g, obj)
		if !known || (len(got) == 1) != (syntax == `IsEnchantmentType(Aura)`) {
			t.Fatalf("%s: %v %v", syntax, got, known)
		}
	}
}

func ronTargetQueryFixture(t *testing.T) (*mage.Game, *mage.Permanent, *mage.StackObject, mage.PermanentTargetPredicate) {
	t.Helper()
	g, a, b := ronGame()
	p := g.PutOnBattlefield(mage.NewLand("Subject"), b.PlayerID())
	obj := castRONSpell(t, g, a, repositoryCard(t, "LEA", "Stone Rain"), p.ID())
	instructions, _ := mage.InspectDirectDestruction(obj.Effects[0])
	plan, ok := mage.InspectPermanentTarget(instructions[0].Subject)
	if !ok {
		t.Fatal("missing plan")
	}
	return g, p, obj, plan
}

func TestRONGlobalTargetQueryInvalid(t *testing.T) {
	for _, rule := range []string{
		`PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfSpells(AnySpell())])`,
		`PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfSpells(Or([]))])`,
		`PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfSpells(IsCardtype(Unknown))])`,
		`PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfAbilities(AbilityOfASource())])`,
		`PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfAbilities(AbilityOfASource(IsColor(Colorless)))])`,
		`PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfAbilities(AbilityOfASource(And([IsCardtype(Artifact),IsColor()])) )])`,
		`PermanentRuleEffect(Ref_TargetPermanent,[CantBeTheTargetOfSpells(AnySpell)])`,
	} {
		node := nameScopeNode(t, `Card(Name:"Invalid restriction",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)
		if _, err := Compile(node); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONGlobalTargetQueryAbilityFilters(t *testing.T) {
	for _, syntax := range []string{`IsCardtype(Sorcery)`, `IsCardtype(Artifact)`, `IsColor(Red)`, `And([IsCardtype(Sorcery),Or([IsColor(Red),IsColor(White)])])`} {
		g, p, obj, plan := ronTargetQueryFixture(t)
		obj.IsAbility = true
		filter, err := compileAbilityTargetingFilter(nameScopeNode(t, `AbilityOfASource(`+syntax+`)`))
		if err != nil {
			t.Fatal(err)
		}
		source := g.PutOnBattlefield(mage.NewAura("Rule", "", mage.WithCastTarget(mage.TargetLand()), mage.WithAbility(mage.Static(mage.CantBeTargetedByAbilities(mage.AttachedPermanent, filter)))), obj.Controller)
		g.Attach(source.ID(), p.ID())
		got, known := plan.Select(g, obj)
		if !known || (len(got) == 1) != (syntax == `IsCardtype(Artifact)`) {
			t.Fatalf("%s: %v %v", syntax, got, known)
		}
	}
}
