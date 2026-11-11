package mage

import (
	"slices"
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestImportContextFiltersAndTriggers(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	source := g.PutOnBattlefield(NewCreature("Source", "{W}", 1, 1), owner)
	other := g.PutOnBattlefield(NewCreature("Other", "{G}", 2, 3, WithSubTypes("Elf")), owner)
	other.Card.SetOwner(owner)
	ctx := AbilityContext{SourceID: source.ID(), Controller: owner}
	filter := ObjectAnd(IsCardType(TypeCreature), OtherThan(SourceObject()), ObjectControlledBy(SelectController()), ObjectOwnedBy(SelectController()))
	if filter.Match(source, g, ctx) || !filter.Match(other, g, ctx) {
		t.Fatal("relative filter has incorrect bindings")
	}
	spec := TriggerOr(WhenPermanentEntersBattlefield(filter), WhenCreatureAttacks(filter), WhenPermanentDies(filter))
	trigger := NewTriggeredAbility(spec, WithTarget(TargetPlayer()), GainLife(1))
	if len(trigger.Targets()) != 1 || len(trigger.Effects()) != 1 {
		t.Fatal("trigger parts were lost")
	}
	for _, et := range []EventType{EvtZoneChange, EvtDeclaredAttacker} {
		if !trigger.CheckEventType(et) {
			t.Fatal("event union lost a type")
		}
	}
	if trigger.CheckEventType(EvtUpkeep) {
		t.Fatal("event union accepts unrelated type")
	}
	g.DestroyPermanent(other)
	evt := &GameEvent{Type: EvtZoneChange, SourceID: other.ID(), FromZone: ZoneBattlefield, ToZone: ZoneGraveyard}
	if !filter.Match(g.LookupObject(other.ID()), g, ctx) || !trigger.CheckTrigger(evt, ctx, g) {
		t.Fatal("filter did not use LKI")
	}
	clone := g.Clone()
	if !trigger.CheckTrigger(evt, ctx, clone) {
		t.Fatal("trigger did not survive clone")
	}
}

func TestImportPermanentTargetUsesContext(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	source := g.PutOnBattlefield(NewCreature("Source", "{G}", 1, 1), owner)
	other := g.PutOnBattlefield(NewCreature("Other", "{G}", 1, 1), owner)
	g.PutOnBattlefield(NewCreature("Enemy", "{G}", 1, 1), g.players[1].PlayerID())
	target := TargetPermanent(ObjectAnd(IsCardType(TypeCreature), OtherThan(SourceObject()), ObjectControlledBy(SelectController())))
	if got := target.Possible(owner, source.Card, g); !slices.Equal(got, []uuid.UUID{other.ID()}) {
		t.Fatalf("unexpected targets: %v", got)
	}
}

type unsafeInspectionValue struct{}

func (unsafeInspectionValue) Text() string                   { return "dynamic" }
func (unsafeInspectionValue) Resolve(ctx *EffectContext) int { return ctx.Game.players[0].Life() }

func TestImportStaticInspection(t *testing.T) {
	if got, ok := StaticInt(Add(Fixed(2), Mul(Fixed(3), Fixed(4)))); !ok || got != 14 {
		t.Fatal("constant expression was not inspected")
	}
	if got, ok := StaticInt(Add(unsafeInspectionValue{}, Fixed(3))); ok || got != 0 {
		t.Fatal("partial expression was treated as static")
	}
	if _, ok := StaticInt(unsafeInspectionValue{}); ok {
		t.Fatal("dynamic value was treated as static")
	}
	effect := Boost(unsafeInspectionValue{}, Fixed(3))
	if effect.Text() == "" || effect.Properties().ToughnessBoost != 3 || effect.Properties().PowerBoost != 0 {
		t.Fatal("incorrect dynamic boost metadata")
	}
	var _ = TargetGroup(0)
	var _ = FirstTarget()
	var _ = VarObject("player")
}

func TestImportManaQualification(t *testing.T) {
	for _, tc := range []struct {
		name   string
		action *ActionDefinition
		want   AbilityType
	}{
		{"mana", NewActivated(Tap(), AddMana(Green, 1)), AbilityMana},
		{"extra effect", NewActivated(Tap(), AddMana(Green, 1), GainLife(1)), AbilityMana},
		{"no tap", NewActivated(LifePayCost(1), AddMana(Green, 1)), AbilityMana},
		{"target", NewActivated(Tap(), AddMana(Green, 1), WithTarget(TargetPlayer())), AbilityActivated},
		{"loyalty", NewActivated(Tap(), AddMana(Green, 1), WithLoyaltyAbility()), AbilityActivated},
		{"spell", NewSpell(AddMana(Green, 1)), AbilitySpell},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.action.Type() != tc.want {
				t.Fatalf("got %v, want %v", tc.action.Type(), tc.want)
			}
		})
	}
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	ability := NewActivated(LifePayCost(1), AddMana(Green, 1), GainLife(2), WithOncePerTurn())
	source := g.PutOnBattlefield(NewArtifact("Source", "{1}", WithAbility(ability)), owner)
	if err := g.ActivateAbilityByIndex(owner, source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !g.stack.IsEmpty() || g.players[0].Life() != 21 {
		t.Fatal("mana ability did not resolve immediately")
	}
	if err := g.ActivateAbilityByIndex(owner, source.ID(), 0, nil); err == nil {
		t.Fatal("activation limit was ignored")
	}
}

func TestImportTriggerSpecsRejectUnrelatedEvents(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	enemy := g.players[1].PlayerID()
	card := NewCreature("Creature", "{G}", 1, 1)
	card.SetOwner(owner)
	permanent := g.PutOnBattlefield(card, owner)
	ctx := AbilityContext{SourceID: uuid.New(), Controller: owner}
	creature := IsCardType(TypeCreature)
	for _, tc := range []struct {
		name  string
		spec  TriggerSpec
		event GameEvent
	}{
		{"entry", WhenPermanentEntersBattlefield(creature), GameEvent{Type: EvtZoneChange, SourceID: permanent.ID(), FromZone: ZoneHand, ToZone: ZoneBattlefield}},
		{"attack", WhenCreatureAttacks(creature), GameEvent{Type: EvtDeclaredAttacker, SourceID: permanent.ID()}},
		{"upkeep", AtBeginningOfUpkeep(SelectController()), GameEvent{Type: EvtUpkeep, PlayerID: owner}},
		{"end step", AtBeginningOfEndStep(SelectController()), GameEvent{Type: EvtEndStep, PlayerID: owner}},
		{"cast", WhenPlayerCastsSpell(SelectController(), ObjectAnd(creature, ObjectControlledBy(SelectController()))), GameEvent{Type: EvtSpellCast, SourceID: card.ID(), PlayerID: owner}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.spec.MatchesEvent(&tc.event, ctx, g) {
				t.Fatal("matching event was rejected")
			}
			wrong := tc.event
			wrong.Type = EvtLifeGained
			if tc.spec.MatchesEvent(&wrong, ctx, g) {
				t.Fatal("wrong event type was accepted")
			}
			wrong = tc.event
			if wrong.Type == EvtUpkeep || wrong.Type == EvtEndStep || wrong.Type == EvtSpellCast {
				wrong.PlayerID = enemy
			} else {
				wrong.SourceID = uuid.New()
			}
			if tc.spec.MatchesEvent(&wrong, ctx, g) {
				t.Fatal("wrong event subject was accepted")
			}
		})
	}
	g.DestroyPermanent(permanent)
	for _, spec := range []TriggerSpec{WhenPermanentDies(creature), WhenPermanentLeavesBattlefield(creature)} {
		event := &GameEvent{Type: EvtZoneChange, SourceID: permanent.ID(), FromZone: ZoneBattlefield, ToZone: ZoneGraveyard}
		if !spec.MatchesEvent(event, ctx, g) {
			t.Fatal("leave event was rejected")
		}
		event.FromZone = ZoneHand
		if spec.MatchesEvent(event, ctx, g) {
			t.Fatal("non-battlefield departure was accepted")
		}
	}
}

func TestImportComposedTriggerDispatchAndClone(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	filter := ObjectAnd(IsCardType(TypeCreature), OtherThan(SourceObject()))
	entry := WhenPermanentEntersBattlefield(filter)
	ability := NewTriggeredAbility(TriggerOr(entry, entry), GainLifeAmount(Fixed(1)))
	g.PutOnBattlefield(NewCreature("Source", "{W}", 1, 1, WithAbility(ability)), owner)
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.players[0].Life() != 20 {
		t.Fatal("source entry triggered another-object filter")
	}
	cloned := g.Clone()
	for _, game := range []*Game{g, cloned} {
		game.PutOnBattlefield(NewCreature("Visitor", "{G}", 1, 1), owner)
		game.PutTriggersOnStack()
		game.ResolveStack()
		if game.players[0].Life() != 21 {
			t.Fatal("composed trigger did not resolve exactly once")
		}
	}
}

func TestImportTargetSelectorsApplyPlayerEffects(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	enemy := g.players[1].PlayerID()
	ctx := NewEffectContext(g, uuid.New(), owner, []uuid.UUID{enemy})
	ctx.TargetGroups = [][]uuid.UUID{{enemy}}
	if err := GainLifeForPlayers(TargetGroup(0), Fixed(3)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.players[1].Life() != 23 || g.players[0].Life() != 20 {
		t.Fatal("target group selected the wrong player")
	}
	ids := TargetGroup(0).Select(ctx)
	ids[0] = owner
	if ctx.TargetGroups[0][0] != enemy {
		t.Fatal("target selector exposed context storage")
	}
}

func TestImportManaRestrictionsAndPlanning(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	action := NewActivated(Tap(), AddMana(Green, 1), GainLife(1))
	card := NewArtifact("Source", "{1}", WithAbility(action))
	permanent := g.PutOnBattlefield(card, owner)
	permanent.GrantBaseAttr(AttrCantActivateNonManaAbilities)
	if len(g.GetActivatableAbilities(owner)) != 1 {
		t.Fatal("manual mana ability was hidden")
	}
	if len(action.ManaProfile(g, permanent.ID()).Productions) != 0 {
		t.Fatal("planner would omit the extra effect")
	}
	if err := g.ActivateAbilityByIndex(owner, permanent.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !g.stack.IsEmpty() || g.players[0].Life() != 21 {
		t.Fatal("mana ability restriction or resolution is incorrect")
	}
	permanent.Tapped = false
	permanent.GrantBaseAttr(AttrCantActivate)
	if err := g.ActivateAbilityByIndex(owner, permanent.ID(), 0, nil); err == nil {
		t.Fatal("all-ability restriction was ignored")
	}
}

func TestImportObjectViewIncludesCurrentPowerAndToughness(t *testing.T) {
	g := newPriorityTestGame()
	permanent := g.PutOnBattlefield(NewCreature("Creature", "{G}", 1, 2), g.players[0].PlayerID())
	permanent.powerBonus = 3
	permanent.toughBonus = 4
	var view ObjectView = permanent
	if view.ViewOwner() != g.LookupObject(permanent.ID()).ViewOwner() {
		t.Fatal("object views disagree on owner")
	}
	if view.ViewPower() != 4 || view.ViewToughness() != 6 {
		t.Fatal("object view omitted current bonuses")
	}
}

func TestImportFilterPowerWithoutGame(t *testing.T) {
	g := newPriorityTestGame()
	p := g.PutOnBattlefield(NewCreature("Creature", "{G}", 1, 2), g.players[0].PlayerID())
	p.powerBonus, p.toughBonus = 3, 4
	filter := CharacteristicFilter("current stats", func(o ObjectView, _ GameReader) bool { return o.ViewPower() == 4 && o.ViewToughness() == 6 })
	if !filter.Match(p, nil) || !filter.Match(p, g) {
		t.Fatal("filter views disagree with permanent stats")
	}
}

func TestImportTapForManaRestrictionsAndTracking(t *testing.T) {
	for _, blocked := range []bool{true, false} {
		t.Run(map[bool]string{true: "condition", false: "tracking"}[blocked], func(t *testing.T) {
			g := newPriorityTestGame()
			owner := g.players[0].PlayerID()
			opts := []any{AddMana(Green, 1), WithOncePerTurn()}
			if blocked {
				opts = append(opts, WithActivationCondition(func(*Game, *Permanent, uuid.UUID) bool { return false }))
			}
			action := NewActivated(Tap(), opts...)
			p := g.PutOnBattlefield(NewArtifact("Mana Source", "{1}", WithAbility(action)), owner)
			g.PutOnBattlefield(NewEnchantment("Watcher", "{0}", WithAbility(NewTriggered(EvtAbilityActivated, false, GainLife(1)))), owner)
			err := g.TapForMana(owner, p.ID())
			if blocked {
				if err == nil || p.Tapped {
					t.Fatal("blocked mana ability was activated")
				}
				return
			}
			g.PutTriggersOnStack()
			g.ResolveStack()
			if g.players[0].Life() != 21 {
				t.Fatal("activation event was not emitted")
			}
			p.Tapped = false
			if err := g.TapForMana(owner, p.ID()); err == nil {
				t.Fatal("activation limit was bypassed")
			}
		})
	}
}

func TestImportCreatureTargetMetadata(t *testing.T) {
	creature, land := IsCardType(TypeCreature), IsCardType(TypeLand)
	for _, tc := range []struct {
		filter ObjectFilter
		want   bool
	}{
		{creature, true}, {ObjectAnd(creature, OtherThan(SourceObject())), true},
		{ObjectOr(creature, ObjectAnd(creature, ObjectHasColor(Green))), true},
		{ObjectOr(creature, land), false}, {ObjectNot(creature), false}, {ObjectFilter{}, false},
	} {
		if TargetsOnlyCreatures(TargetPermanent(tc.filter)) != tc.want {
			t.Errorf("incorrect metadata for %s", tc.filter.Text())
		}
	}
	if !TargetsOnlyCreatures(TargetNCreatures(2)) {
		t.Fatal("target count wrapper lost creature restriction")
	}
}

func TestImportPowerSinkTapsCompiledManaLand(t *testing.T) {
	g := newPriorityTestGame()
	defender := g.players[1]
	land := g.PutOnBattlefield(NewLand("Compiled Land", WithAbility(NewActivated(Tap(), AddMana(Green, 1)))), defender.PlayerID())
	spell := NewInstant("Spell", "{0}")
	spell.SetOwner(defender.PlayerID())
	g.pushStack(&StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: defender.PlayerID()})
	ctx := NewEffectContext(g, uuid.New(), g.players[0].PlayerID(), []uuid.UUID{spell.ID()})
	ctx.XValue = 2
	if err := PowerSinkEffect().Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if !land.Tapped || g.FindStackObject(spell.ID()) != nil {
		t.Fatal("Power Sink did not tap the compiled mana land and counter the spell")
	}
}
