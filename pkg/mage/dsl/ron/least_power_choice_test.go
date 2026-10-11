package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const ronLeastCreature = `And([IsCardtype(Creature),PowerIs(EqualTo(TheLeastPowerAmongPermanents(IsCardtype(Creature))))])`

func honeyFactory(t *testing.T) mage.CardFactory {
	t.Helper()
	f, err := Compile(staticBaseRecord(t, "ARN", "Drop of Honey"))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func honeyUpkeep(g *mage.Game, player uuid.UUID) {
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: player})
	g.PutTriggersOnStack()
}

func TestRONLeastPowerChoiceDropOfHoney(t *testing.T) {
	f := honeyFactory(t)
	for _, scenario := range []string{"ties", "negative", "live-power", "layer-power", "token-member", "ability-removal", "phase-member", "late-entry", "shroud", "indestructible", "regeneration", "exile", "source-departure", "source-control", "source-phase", "returned-member"} {
		t.Run(scenario, func(t *testing.T) {
			g, base, b := ronGame()
			a := &ronPermanentChoicePlayer{Player: base, choose: func(p []*mage.Permanent) *mage.Permanent { return p[len(p)-1] }}
			g.SetPlayerAt(0, a)
			power := 1
			if scenario == "negative" {
				power = -2
			}
			firstCard := mage.NewCreature("First", "", power, 5)
			firstCard.SetOwner(b.PlayerID())
			first := g.PutOnBattlefield(firstCard, b.PlayerID())
			secondCard := mage.NewCreature("Second", "", power, 5)
			secondCard.SetOwner(b.PlayerID())
			second := g.PutOnBattlefield(secondCard, b.PlayerID())
			high := g.PutOnBattlefield(mage.NewCreature("High", "", 4, 5), b.PlayerID())
			sourceCard := f()
			sourceCard.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			honeyUpkeep(g, b.PlayerID())
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep triggered")
			}
			honeyUpkeep(g, a.PlayerID())
			if g.StackSize() != 1 || a.calls != 0 {
				t.Fatal("choice must occur at resolution")
			}
			want := second.ID()
			switch scenario {
			case "live-power":
				g.MutablePermanent(second.ID()).AddCounter(core.P1P1, 2)
				want = first.ID()
			case "layer-power":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(-2), mage.Fixed(0))), high.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(2), mage.Fixed(0))), second.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want = first.ID()
			case "token-member":
				token := mage.NewToken("Token", -3, 5, []core.CardType{core.TypeCreature}, nil)
				want = g.PutOnBattlefield(token, b.PlayerID()).ID()
			case "ability-removal":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
			case "phase-member":
				g.PhaseOut(second.ID())
				want = first.ID()
			case "late-entry":
				want = g.PutOnBattlefield(mage.NewCreature("Late", "", -3, 5), b.PlayerID()).ID()
			case "shroud":
				g.MutablePermanent(second.ID()).GrantBaseAttr(core.Shroud)
			case "indestructible":
				g.MutablePermanent(second.ID()).GrantBaseAttr(core.Indestructible)
			case "regeneration":
				g.AddRegenerationShield(second.ID())
			case "exile":
				if err := mage.ApplyEffect(g, mage.ExileIfWouldDieThisTurn(mage.This), second.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "source-departure":
				g.BouncePermanentToHand(source)
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: high.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "source-phase":
				g.PhaseOut(source.ID())
			case "returned-member":
				g.BouncePermanentToHand(second)
				b.RemoveFromHand(second.ID())
				g.PutOnBattlefield(secondCard, b.PlayerID())
			}
			g.ResolveTopOfStack()
			if scenario == "indestructible" {
				if g.FindPermanent(second.ID()) == nil || g.FindPermanent(first.ID()) == nil {
					t.Fatal("indestructible choice destroyed another creature")
				}
			} else if g.FindPermanentIncludingPhased(want) != nil {
				t.Fatal("least-power creature survived")
			}
			if g.FindPermanent(high.ID()) == nil {
				t.Fatal("destroyed higher-power creature")
			}
			if scenario == "exile" && g.FindExiledCard(want) == nil {
				t.Fatal("death replacement omitted")
			}
			if scenario == "ties" || scenario == "shroud" || scenario == "negative" || scenario == "source-control" {
				if a.calls != 1 {
					t.Fatal("saved trigger controller did not choose")
				}
			}
		})
	}
}

func TestRONLeastPowerChoiceAbsence(t *testing.T) {
	f := honeyFactory(t)
	for _, scenario := range []string{"last-creature", "empty-entry", "pending-entry", "source-phase", "member-phase"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			var creature *mage.Permanent
			if scenario != "empty-entry" && scenario != "pending-entry" {
				creature = g.PutOnBattlefield(mage.NewCreature("Only", "", 1, 4), b.PlayerID())
			}
			source := g.PutOnBattlefield(f(), a.PlayerID())
			switch scenario {
			case "last-creature":
				honeyUpkeep(g, a.PlayerID())
				g.ResolveTopOfStack()
			case "pending-entry":
				g.PutTriggersOnStack()
				g.PutOnBattlefield(mage.NewCreature("Late", "", 1, 4), b.PlayerID())
			case "source-phase":
				group := g.PhaseOut(source.ID())
				g.BouncePermanentToHand(creature)
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("phased source triggered")
				}
				g.PhaseIn(group)
			case "member-phase":
				g.PhaseOut(creature.ID())
			}
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatalf("absence triggers=%d", g.StackSize())
			}
			g.ResolveStack()
			if g.FindPermanent(source.ID()) != nil {
				t.Fatal("absence sacrifice failed")
			}
		})
	}
}

func TestRONLeastPowerChoiceInvalid(t *testing.T) {
	for _, body := range []string{
		`DestroyAPermanentNoRegen`, `DestroyAPermanentNoRegen()`, `DestroyAPermanentNoRegen(AnyPermanent,AnyPermanent)`,
		`DestroyAPermanentNoRegen(Ref_TargetPermanent)`,
		`DestroyAPermanentNoRegen(AnyPermanent),TapPermanent(TheChosenPermanent)`,
		`DestroyAPermanentNoRegen(PowerIs(EqualTo(TheLeastPowerAmongPermanents)))`, `DestroyAPermanentNoRegen(And([]))`,
		`DestroyAPermanentNoRegen(PowerIs(EqualTo(TheLeastPowerAmongPermanents())))`,
		`DestroyAPermanentNoRegen(PowerIs(EqualTo(TheLeastPowerAmongPermanents(AnyPermanent,AnyPermanent))))`,
		`DestroyAPermanentNoRegen(PowerIs(EqualTo(TheLeastPowerAmongPermanents(Ref_TargetPermanent))))`,
		`DestroyAPermanentNoRegen(PowerIs(GreaterThanOrEqualTo(TheLeastPowerAmongPermanents(AnyPermanent))))`,
		`DestroyAPermanentNoRegen(PowerIs(EqualTo(TheLeastPowerAmongPermanents(AnyPermanent),Integer(1))))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", body)
		}
	}
}

func TestRONLeastPowerChoiceRecordsAndInventory(t *testing.T) {
	n := staticBaseRecord(t, "ARN", "Drop of Honey")
	f, err := Compile(n)
	if err != nil {
		t.Fatal(err)
	}
	first, second := f(), f()
	if first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
		t.Fatal("shared factory declarations")
	}
	src := `Card(Name:"Choice",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([DestroyAPermanentNoRegen(` + ronLeastCreature + `)]))])`
	files := fstest.MapFS{"choice.ron": {Data: []byte(src)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	for _, s := range report.Signatures {
		if s.Name == "TheLeastPowerAmongPermanents" && s.Role != "power_comparison_value" {
			t.Fatal("least-power role", s.Role)
		}
		if s.Name == "IsCardtype" && s.Role != "permanent_filter" {
			t.Fatal("comparison set role", s.Role)
		}
	}
	registry := mage.NewRegistry()
	registry.Register("Sentinel", func() mage.Card { return mage.NewArtifact("Sentinel", "") })
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(src, "Name:\"Choice\"", "Name:\"Invalid\"", 1) + ` Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[Unknown])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("accepted partial collection")
	}
	if _, err := registry.CreateCard("Sentinel"); err != nil {
		t.Fatal("registry lost sentinel", err)
	}
}

func TestRONLeastPowerChoiceCopiesAndClones(t *testing.T) {
	g, base, other := ronGame()
	a := &ronPermanentChoicePlayer{Player: base, choose: func(p []*mage.Permanent) *mage.Permanent { return p[len(p)-1] }}
	b := &ronPermanentChoicePlayer{Player: other, choose: func(p []*mage.Permanent) *mage.Permanent { return p[0] }}
	g.SetPlayerAt(0, a)
	g.SetPlayerAt(1, b)
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 4), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 4), b.PlayerID())
	third := g.PutOnBattlefield(mage.NewCreature("Third", "", 1, 4), b.PlayerID())
	card := honeyFactory(t)()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || g.StackSize() != 0 {
		t.Fatal("cast without mana succeeded")
	}
	a.ManaPool().Add(core.Green, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	source := g.FindPermanent(card.ID())
	if source == nil || a.ManaPool().TotalMana() != 0 {
		t.Fatal("RON enchantment did not enter after payment")
	}
	honeyUpkeep(g, a.PlayerID())
	clone := g.Clone()
	ca := &ronPermanentChoicePlayer{Player: clone.GetPlayer(a.PlayerID()), choose: func(p []*mage.Permanent) *mage.Permanent { return p[0] }}
	clone.SetPlayerAt(0, ca)
	g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	g.BouncePermanentToHand(source)
	g.ResolveStack()
	if g.FindPermanent(first.ID()) != nil || g.FindPermanent(third.ID()) != nil || g.FindPermanent(second.ID()) == nil || a.calls != 1 || b.calls != 1 {
		t.Fatal("copies shared choice or controller")
	}
	clone.ResolveStack()
	if clone.FindPermanent(first.ID()) != nil || clone.FindPermanent(second.ID()) == nil || clone.FindPermanent(third.ID()) == nil || ca.calls != 1 {
		t.Fatal("clone choice affected original")
	}
}

func TestRONLeastPowerChoicePlayerScope(t *testing.T) {
	g, base, other := ronGame()
	a := &ronPermanentChoicePlayer{Player: base, choose: func(p []*mage.Permanent) *mage.Permanent { return p[0] }}
	b := &ronPermanentChoicePlayer{Player: other, choose: func(p []*mage.Permanent) *mage.Permanent { return p[len(p)-1] }}
	g.SetPlayerAt(0, a)
	g.SetPlayerAt(1, b)
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 4), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 4), b.PlayerID())
	c := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,DestroyAPermanentNoRegen(`+ronLeastCreature+`)),GainLife(Integer(2))])))`)()
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	if a.calls != 0 || b.calls != 0 {
		t.Fatal("announcement made a choice")
	}
	g.ResolveStack()
	if a.calls != 0 || b.calls != 1 || g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) != nil || a.Life() != 22 || b.Life() != 20 {
		t.Fatal("explicit player scope escaped")
	}
}

func TestRONLeastPowerChoiceActivationAndEmpty(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(map[bool]string{false: "creature", true: "empty"}[empty], func(t *testing.T) {
			g, base, b := ronGame()
			a := &ronPermanentChoicePlayer{Player: base, choose: func(p []*mage.Permanent) *mage.Permanent { return p[0] }}
			g.SetPlayerAt(0, a)
			var victim *mage.Permanent
			if !empty {
				victim = g.PutOnBattlefield(mage.NewCreature("Only", "", 1, 4), b.PlayerID())
			}
			source := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),ActionList([DestroyAPermanentNoRegen(`+ronLeastCreature+`),GainLife(Integer(1))]))`)(), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || a.calls != 0 || g.StackSize() != 0 || a.Life() != 20 {
				t.Fatal("failed payment changed the game")
			}
			a.ManaPool().Add(core.Colorless, 2)
			for range 2 {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			if a.calls != 0 || a.Life() != 20 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("choice occurred during payment")
			}
			g.ResolveStack()
			if a.calls != 0 || a.Life() != 22 || (victim != nil && g.FindPermanent(victim.ID()) != nil) {
				t.Fatal("sole or empty choice failed")
			}
		})
	}
}
