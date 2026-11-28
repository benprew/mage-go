package mtgish

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func compileFixture(t *testing.T, name string) *mage.CardDefinition {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "testdata", "mtgish", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	card, err := ParseMtgishJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	definition, err := Compile(card)
	if err != nil {
		t.Fatal(err)
	}
	return definition
}

func TestCompiledLightningBolt(t *testing.T) {
	definition := compileFixture(t, "lightning_bolt")
	for _, target := range []string{"player", "creature", "planeswalker"} {
		t.Run(target, func(t *testing.T) {
			tg := gametest.NewTestGame(t)
			caster, opponent := tg.GetPlayer(gametest.PlayerA), tg.GetPlayer(gametest.PlayerB)
			bolt := definition.NewInstance(caster.PlayerID())
			caster.AddToHand(bolt)
			caster.ManaPool().Add(core.Red, 1)

			targetID := opponent.PlayerID()
			switch target {
			case "creature":
				targetID = tg.PutOnBattlefield(mage.NewCreature("Creature", "{G}", 1, 4), opponent.PlayerID()).ID()
			case "planeswalker":
				targetID = tg.PutOnBattlefield(mage.NewPlaneswalker("Planeswalker", "{U}", 5), opponent.PlayerID()).ID()
			}
			if err := tg.CastSpellByID(caster.PlayerID(), bolt.ID(), []uuid.UUID{targetID}, 0); err != nil {
				t.Fatal(err)
			}
			if tg.StackSize() != 1 || caster.ManaPool().Count(core.Red) != 0 {
				t.Fatal("Lightning Bolt did not use the stack or pay its mana cost")
			}
			tg.ResolveStack()

			switch target {
			case "player":
				tg.AssertLife(gametest.PlayerB, 17)
			case "creature":
				if damage := tg.FindPermanent(targetID).Damage; damage != 3 {
					t.Fatalf("creature took %d damage, want 3", damage)
				}
			case "planeswalker":
				if loyalty := tg.FindPermanent(targetID).Counters[core.Loyalty]; loyalty != 2 {
					t.Fatalf("planeswalker has %d loyalty, want 2", loyalty)
				}
			}
			tg.AssertGraveyardCount(gametest.PlayerA, "Lightning Bolt", 1)
		})
	}
}

func TestCompiledLightningBoltChecksTargetLegality(t *testing.T) {
	definition := compileFixture(t, "lightning_bolt")
	tg := gametest.NewTestGame(t)
	caster, opponent := tg.GetPlayer(gametest.PlayerA), tg.GetPlayer(gametest.PlayerB)
	bolt := definition.NewInstance(caster.PlayerID())
	caster.AddToHand(bolt)
	caster.ManaPool().Add(core.Red, 1)
	artifact := tg.PutOnBattlefield(mage.NewArtifact("Artifact", "{1}"), opponent.PlayerID())
	if err := tg.CastSpellByID(caster.PlayerID(), bolt.ID(), []uuid.UUID{artifact.ID()}, 0); err == nil {
		t.Fatal("Lightning Bolt accepted a noncreature artifact")
	}

	creature := mage.NewCreature("Creature", "{G}", 1, 4)
	permanent := tg.PutOnBattlefield(creature, opponent.PlayerID())
	if err := tg.CastSpellByID(caster.PlayerID(), bolt.ID(), []uuid.UUID{permanent.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	tg.ExilePermanent(permanent)
	returnedCard, ok := tg.RemoveFromExile(creature.ID())
	if !ok {
		t.Fatal("could not return the target from exile")
	}
	returned := tg.PutOnBattlefield(returnedCard, opponent.PlayerID())
	tg.ResolveStack()
	if returned.Damage != 0 {
		t.Fatalf("Lightning Bolt dealt %d damage to the returned permanent", returned.Damage)
	}
	tg.AssertGraveyardCount(gametest.PlayerA, "Lightning Bolt", 1)
}

func TestCompiledProdigalSorcerer(t *testing.T) {
	definition := compileFixture(t, "prodigal_sorcerer")
	tg := gametest.NewTestGame(t)
	owner := tg.GetPlayer(gametest.PlayerA).PlayerID()
	opponent := tg.GetPlayer(gametest.PlayerB).PlayerID()
	first := tg.PutOnBattlefield(definition.NewInstance(owner), owner)
	second := tg.PutOnBattlefield(definition.NewInstance(owner), owner)
	target := []uuid.UUID{opponent}
	if err := tg.ActivateAbilityByIndex(owner, first.ID(), 0, target); err == nil {
		t.Fatal("activated while the Sorcerer had summoning sickness")
	}
	first.RevokeBaseAttr(core.AttrSummonSick)
	second.RevokeBaseAttr(core.AttrSummonSick)
	if err := tg.ActivateAbilityByIndex(owner, first.ID(), 0, target); err != nil {
		t.Fatal(err)
	}
	if !tg.FindPermanent(first.ID()).Tapped || tg.StackSize() != 1 {
		t.Fatal("activation did not tap the source and use the stack")
	}
	tg.AssertLife(gametest.PlayerB, 20)
	if err := tg.ActivateAbilityByIndex(owner, first.ID(), 0, target); err == nil {
		t.Fatal("activated while the Sorcerer was tapped")
	}

	clone := tg.Clone()
	if err := clone.ActivateAbilityByIndex(owner, second.ID(), 0, []uuid.UUID{owner}); err != nil {
		t.Fatal(err)
	}
	if tg.FindPermanent(second.ID()).Tapped || tg.StackSize() != 1 {
		t.Fatal("activation changed the original game")
	}
	tg.DestroyPermanent(tg.FindPermanent(first.ID()))
	tg.ResolveStack()
	tg.AssertLife(gametest.PlayerA, 20)
	tg.AssertLife(gametest.PlayerB, 19)

	clone.ResolveStack()
	if a, b := clone.GetPlayer(owner).Life(), clone.GetPlayer(opponent).Life(); a != 19 || b != 19 {
		t.Fatalf("clone resolved damage to the wrong targets: life %d, %d", a, b)
	}
}

func TestCompiledLlanowarElves(t *testing.T) {
	definition := compileFixture(t, "llanowar_elves")
	tg := gametest.NewTestGame(t)
	player := tg.GetPlayer(gametest.PlayerA)
	owner := player.PlayerID()
	elves := tg.PutOnBattlefield(definition.NewInstance(owner), owner)
	if ability := elves.RuntimeAbilities[0]; ability.Type() != core.AbilityMana {
		t.Fatalf("Llanowar Elves has an ability of type %v, want mana", ability.Type())
	}
	if err := tg.ActivateAbilityByIndex(owner, elves.ID(), 0, nil); err == nil {
		t.Fatal("activated while the Elves had summoning sickness")
	}
	if tg.TryPayMana(owner, "{G}") {
		t.Fatal("planner used an Elf with summoning sickness")
	}
	elves.RevokeBaseAttr(core.AttrSummonSick)
	clone := tg.Clone()
	if !clone.TryPayMana(owner, "{G}") || !clone.FindPermanent(elves.ID()).Tapped {
		t.Fatal("planner did not use Llanowar Elves to pay for green mana")
	}
	if err := tg.ActivateAbilityByIndex(owner, elves.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !tg.FindPermanent(elves.ID()).Tapped || tg.StackSize() != 0 || player.ManaPool().Count(core.Green) != 1 {
		t.Fatal("mana ability did not tap the Elves and resolve immediately")
	}
}

func TestCompiledDarkRitual(t *testing.T) {
	definition := compileFixture(t, "dark_ritual")
	tg := gametest.NewTestGame(t)
	player := tg.GetPlayer(gametest.PlayerA)
	ritual := definition.NewInstance(player.PlayerID())
	player.AddToHand(ritual)
	player.ManaPool().Add(core.Black, 1)
	if err := tg.CastSpellByID(player.PlayerID(), ritual.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if tg.StackSize() != 1 || player.ManaPool().Count(core.Black) != 0 {
		t.Fatal("Dark Ritual produced mana before resolution")
	}
	tg.ResolveStack()
	if amount := player.ManaPool().Count(core.Black); amount != 3 {
		t.Fatalf("Dark Ritual added %d black mana, want 3", amount)
	}
	tg.AssertGraveyardCount(gametest.PlayerA, "Dark Ritual", 1)
}

func TestCompileExecutableRuleDiagnostics(t *testing.T) {
	spell := func(actions string) string {
		return `{"_Rule":"SpellActions","args":` + actions + `}`
	}
	activated := func(cost, actions string) string {
		return `{"_Rule":"Activated","args":[` + cost + `,` + actions + `]}`
	}
	list := func(actions ...string) string {
		return `{"_Actions":"ActionList","args":[` + strings.Join(actions, ",") + `]}`
	}
	targeted := func(targets, actions string) string {
		return `{"_Actions":"Targeted","args":[` + targets + `,` + actions + `]}`
	}
	const (
		anyTarget       = `[{"_Target":"AnyTarget"}]`
		selfTap         = `{"_Cost":"TapPermanent","args":{"_Permanent":"ThisPermanent"}}`
		addMana         = `{"_Action":"AddMana","args":{"_ManaProduce":"ManaProduceB"}}`
		spellDamage     = `{"_Action":"SpellDealsDamage","args":[{"_Spell":"ThisSpell"},{"_GameNumber":"Integer","args":3},{"_DamageRecipient":"Ref_AnyTarget"}]}`
		permanentDamage = `{"_Action":"PermanentDealsDamage","args":[{"_Permanent":"ThisPermanent"},{"_GameNumber":"Integer","args":1},{"_DamageRecipient":"Ref_AnyTarget"}]}`
	)
	for _, test := range []struct {
		name, rules       string
		code              DiagnosticCode
		path, family, tag string
	}{
		{"missing actions", `{"_Rule":"SpellActions"}`, InvalidInput, "/0/args", "_Rule", "SpellActions"},
		{"wrong action family", spell(list(`{"_Rule":"AddMana"}`)), InvalidInput, "/0/args/args/0/_Rule", "_Action", ""},
		{"null action", spell(list("null")), InvalidInput, "/0/args/args/0", "", ""},
		{"extra activation argument", `{"_Rule":"Activated","args":[` + selfTap + "," + list(addMana) + ",false]}", InvalidInput, "/0/args", "_Rule", "Activated"},
		{"extra damage argument", spell(targeted(anyTarget, list(strings.Replace(spellDamage, `}]}`, `},3]}`, 1)))), InvalidInput, "/0/args/args/1/args/0/args", "_Action", "SpellDealsDamage"},
		{"tap another permanent", activated(strings.Replace(selfTap, "ThisPermanent", "Ref_TargetPermanent", 1), list(addMana)), UnsupportedConstruct, "/0/args/0/args", "_Permanent", "Ref_TargetPermanent"},
		{"unknown mana choice", spell(list(strings.Replace(addMana, "ManaProduceB", "AnyManaColor", 1))), UnsupportedConstruct, "/0/args/args/0/args", "_ManaProduce", "AnyManaColor"},
		{"unsupported mana child", spell(list(`{"_Action":"AddMana","args":{"_ManaProduce":"And","args":[{"_ManaProduce":"ManaProduceB"},{"_ManaProduce":"AnyManaColor"}]}}`)), UnsupportedConstruct, "/0/args/args/0/args/args/1", "_ManaProduce", "AnyManaColor"},
		{"multiple target groups", spell(targeted(`[{"_Target":"AnyTarget"},{"_Target":"AnyTarget"}]`, list(spellDamage))), UnsupportedConstruct, "/0/args/args/0", "_Actions", "Targeted"},
		{"nested targeting", spell(targeted(anyTarget, targeted(anyTarget, list(spellDamage)))), UnsupportedConstruct, "/0/args/args/1", "_Actions", "Targeted"},
		{"unbound target", spell(list(spellDamage)), InvalidInput, "/0/args/args/0/args/2", "_DamageRecipient", "Ref_AnyTarget"},
		{"numbered target", spell(targeted(anyTarget, list(strings.Replace(spellDamage, "Ref_AnyTarget", "Ref_AnyTarget1", 1)))), UnsupportedConstruct, "/0/args/args/1/args/0/args/2", "_DamageRecipient", "Ref_AnyTarget1"},
		{"wrong source context", activated(selfTap, targeted(anyTarget, list(spellDamage))), InvalidInput, "/0/args/1/args/1/args/0/args/0", "_Spell", "ThisSpell"},
		{"source does not accept args", spell(targeted(anyTarget, list(strings.Replace(spellDamage, `"ThisSpell"}`, `"ThisSpell","args":null}`, 1)))), InvalidInput, "/0/args/args/1/args/0/args/0/args", "_Spell", "ThisSpell"},
		{"damage overflow", spell(targeted(anyTarget, list(strings.Replace(spellDamage, `"args":3`, `"args":2147483648`, 1)))), InvalidInput, "/0/args/args/1/args/0/args/1/args", "_GameNumber", "Integer"},
		{"target in an earlier ability", activated(selfTap, targeted(anyTarget, list(permanentDamage))) + "," + activated(selfTap, list(permanentDamage)), InvalidInput, "/1/args/1/args/0/args/2", "_DamageRecipient", "Ref_AnyTarget"},
		{"multiple spell sections", spell(list(addMana)) + "," + spell(list(addMana)), UnsupportedConstruct, "/1", "_Rule", "SpellActions"},
	} {
		t.Run(test.name, func(t *testing.T) {
			card := sourceCard(t, "["+test.rules+"]")
			definition, err := Compile(card)
			if definition != nil {
				t.Fatal("returned a partial definition")
			}
			result := requireDiagnostic(t, err, test.code, "/Rules"+test.path, test.family, test.tag)
			if result.Card != card.Name {
				t.Fatalf("diagnostic lost card name: %v", result)
			}
		})
	}
}
