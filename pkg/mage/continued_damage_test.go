package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func continuedDamageFixture(t *testing.T) (*Game, *BasePlayer, *BasePlayer, *Permanent, DamageSource) {
	t.Helper()
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Selected", "{R}", 2, 10), b.PlayerID())
	card := NewInstant("Additional source", "{W}")
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID()}
	g.stack.Push(obj)
	selected, spell := BindDamageSource("selected"), BindDamageSource("spell")
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{selected: g.CaptureDamageSource(source.ID()), spell: g.CaptureSpellDamageSource(obj.ID)}
	g.resolution.damageCause = damageCauseFromStack(obj)
	effect := AddSpellDamageToNextSelectedSourceEvent(FromDamageSource(selected), You, FromDamageSource(spell), DamageSourceFilter{})
	if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.stack.Pop()
	g.resolution.damageCause = damageCause{}
	return g, a, b, source, g.resolution.damageSourceBindings[spell]
}

func TestContinuedDamageOrdering(t *testing.T) {
	for _, order := range []string{"before", "after", "additional", "redirected"} {
		t.Run(order, func(t *testing.T) {
			g, a, b, source, _ := continuedDamageFixture(t)
			saved := g.CaptureDamageSource(source.ID())
			var victim *Permanent
			switch order {
			case "before":
				g.AddPreventionShield(a.PlayerID(), 2)
				slices.Reverse(g.effects.replacements)
			case "after":
				g.AddPreventionShield(a.PlayerID(), 2)
			case "additional":
				g.AddPreventionShield(b.PlayerID(), 2)
			case "redirected":
				victim = g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
				g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
			}
			outcome := g.damage.dealDamageToPlayerFromSourceOutcome(g, a, 5, saved)
			if victim != nil && g.FindPermanent(victim.ID()).Damage != 5 {
				t.Fatal("added damage did not follow redirection")
			}
			wantA, wantB := 17, 15
			if order == "before" {
				wantB = 17
			}
			if order == "additional" {
				wantA, wantB = 15, 17
			}
			if order == "redirected" {
				wantA, wantB = 15, 20
			}
			if a.Life() != wantA || b.Life() != wantB || outcome.amount != 20-wantA || !outcome.source.SameObject(saved) {
				t.Fatalf("life %d %d outcome %+v", a.Life(), b.Life(), outcome)
			}
		})
	}
}

type continuedDamageDouble struct{ replacementBase }

func (*continuedDamageDouble) Matches(a Action, _ GameReader) bool { return damageActionAmount(a) > 0 }
func (*continuedDamageDouble) Replace(a Action, _ *Game) Action {
	return damageActionWithAmount(a, damageActionAmount(a)*2)
}
func (*continuedDamageDouble) IsActive(GameReader) bool   { return true }
func (r *continuedDamageDouble) Clone() ReplacementEffect { c := *r; return &c }

func TestContinuedDamageInheritedUses(t *testing.T) {
	g, a, b, source, _ := continuedDamageFixture(t)
	g.effects.PrependReplacement(&continuedDamageDouble{})
	outcomes := g.dealSimultaneousDamageOutcomes(PlayerDamageAssignment(a, 2, g.CaptureDamageSource(source.ID())))
	if a.Life() != 16 || b.Life() != 16 || len(outcomes) != 1 || outcomes[0].amount != 4 {
		t.Fatal("a prior replacement repeated on a descendant or extra damage entered the input result")
	}
}

func TestContinuedDamageIdentityAndWaiting(t *testing.T) {
	g, a, b, source, spell := continuedDamageFixture(t)
	saved := g.CaptureDamageSource(source.ID())
	g.RemoveFromBattlefield(source)
	g.PutOnBattlefield(source.Card, b.PlayerID())
	g.DealDamageToPlayer(a, 2, source.ID())
	if b.Life() != 20 {
		t.Fatal("return inherited the rule")
	}
	refs := g.WaitingDamageExpansionSources()
	if len(refs) != 2 || !refs[0].SameObject(saved) || !refs[1].SameObject(spell) {
		t.Fatal("waiting references lost")
	}
	refs[1].card.SetOwner(b.PlayerID())
	if g.WaitingDamageExpansionSources()[1].card.Owner() != a.PlayerID() {
		t.Fatal("waiting query shared card state")
	}
	if !slices.ContainsFunc(g.DamageSourcesFor(a.PlayerID()), spell.SameObject) {
		t.Fatal("saved spell was not a legal source choice")
	}
	g.DealDamageToPlayerFromSource(a, 3, saved)
	if a.Life() != 15 || b.Life() != 17 || len(g.WaitingDamageExpansionSources()) != 0 {
		t.Fatal("saved visit or consumption failed")
	}
}

func TestContinuedDamageSpellContinuity(t *testing.T) {
	for _, state := range []string{"live", "copy", "returned", "departed"} {
		t.Run(state, func(t *testing.T) {
			g, a, b, _, _ := continuedDamageFixture(t)
			card := NewCreature("Selected spell", "{R}", 2, 10)
			card.SetOwner(b.PlayerID())
			obj := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID()}
			g.stack.Push(obj)
			rule := g.effects.replacements[0].(*continuedDamageReplacement)
			chosen, mirror := BindDamageSource("selected spell"), BindDamageSource("saved mirror")
			g.resolution.damageSourceBindings[chosen] = g.CaptureSpellDamageSource(obj.ID)
			g.resolution.damageSourceBindings[mirror] = rule.spell
			g.effects.replacements = nil
			if err := AddSpellDamageToNextSelectedSourceEvent(FromDamageSource(chosen), You, FromDamageSource(mirror), DamageSourceFilter{}).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if state == "copy" {
				g.CopyStackObjectDirect(obj, b.PlayerID(), false)
			}
			g.ResolveStack()
			var selected *Permanent
			for _, p := range g.AllBattlefield() {
				if p.Name() == card.Name() && (p.ID() == card.ID()) == (state != "copy") {
					selected = p
					break
				}
			}
			if selected == nil {
				t.Fatal("missing permanent spell result")
			}
			saved := g.CaptureDamageSource(selected.ID())
			if state == "returned" || state == "departed" {
				g.RemoveFromBattlefield(selected)
				if state == "returned" {
					selected = g.PutOnBattlefield(selected.Card, b.PlayerID())
					saved = g.CaptureDamageSource(selected.ID())
				}
			}
			g.DealDamageToPlayerFromSource(a, 3, saved)
			want := 17
			if state == "copy" || state == "returned" {
				want = 20
			}
			if a.Life() != 17 || b.Life() != want {
				t.Fatalf("life %d %d", a.Life(), b.Life())
			}
		})
	}
}

func TestContinuedDamageInvalid(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		AddSpellDamageToNextSelectedSourceEvent(nil, You, This, DamageSourceFilter{}),
		AddSpellDamageToNextSelectedSourceEvent(This, nil, This, DamageSourceFilter{}),
		AddSpellDamageToNextSelectedSourceEvent(This, You, nil, DamageSourceFilter{}),
	} {
		card := NewInstant("Invalid", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestContinuedDamageCauseAndCombat(t *testing.T) {
	g, a, b, source, spell := continuedDamageFixture(t)
	originalCause := damageCause{stackID: uuid.New()}
	action := NewDamageToPlayerAction(source.ID(), a.PlayerID(), 3, true).withDamageSource(g.CaptureDamageSource(source.ID()))
	action.cause = originalCause
	actions, _ := g.effects.prepareDamageBatchReplacements([]Action{action}, g)
	if len(actions) != 2 {
		t.Fatal("event did not expand")
	}
	additional := actions[1].(*DamageToPlayerAction)
	if damageActionCause(actions[0]).stackID != originalCause.stackID || additional.cause.stackID != spell.StackID() || additional.IsCombatDamage() || !additional.DamageSource().SameObject(spell) {
		t.Fatal("source, cause or combat status crossed packets")
	}
	g.damage.SetResolvingCombatDamage(true)
	flags := map[string]bool{}
	g.damage.SetOnDamageDealt(func(name, _ string, _ int, combat bool) { flags[name] = combat })
	g.damage.executePreparedDamageActions(g, actions)
	if !flags[source.Name()] || flags[spell.Name()] || a.Life() != 17 || b.Life() != 17 {
		t.Fatal("combat event classification changed")
	}
}

func TestContinuedDamageInstructionPrevention(t *testing.T) {
	g, a, b, source, _ := continuedDamageFixture(t)
	effect := DealDamageWithPreventionFrom(DamageSourceOf(This), You, Fixed(5), Fixed(2))
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 17 || b.Life() != 15 {
		t.Fatalf("instruction prevention crossed into added spell damage: %d %d", a.Life(), b.Life())
	}
}

func TestContinuedDamageEvent(t *testing.T) {
	for _, batch := range []bool{false, true} {
		t.Run(map[bool]string{false: "single", true: "batch"}[batch], func(t *testing.T) {
			g, a, b, source, spell := continuedDamageFixture(t)
			observed := 0
			additional := 0
			g.damage.SetOnDamageDealt(func(name, _ string, amount int, _ bool) {
				observed++
				if name == spell.Name() {
					additional++
					if amount != 5 {
						t.Error("additional amount was not grouped")
					}
				}
				if a.Life() != 15 || b.Life() != 15 {
					t.Errorf("partial event: %d %d", a.Life(), b.Life())
				}
			})
			selected := g.CaptureDamageSource(source.ID())
			if batch {
				g.DealSimultaneousDamage(PlayerDamageAssignment(a, 2, selected), PlayerDamageAssignment(a, 3, selected))
			} else {
				g.DealDamageToPlayerFromSource(a, 5, selected)
			}
			if additional != 1 || observed < 2 || a.Life() != 15 || b.Life() != 15 {
				t.Fatalf("event failed: %d %d %d", observed, a.Life(), b.Life())
			}
			if g.DamageTakenByPlayer(b.PlayerID()) != 5 {
				t.Fatal("additional packet bypassed damage history")
			}
			if spell.StackID() == uuid.Nil {
				t.Fatal("fixture lacks spell identity")
			}
			g.damage.SetOnDamageDealt(nil)
			g.DealDamageToPlayerFromSource(a, 1, selected)
			if a.Life() != 14 || b.Life() != 15 {
				t.Fatal("rule was not consumed")
			}
		})
	}
}

func TestContinuedDamageLifecycle(t *testing.T) {
	for _, state := range []string{"clone", "departure", "return", "phase", "control", "cleanup", "zero", "prevented"} {
		t.Run(state, func(t *testing.T) {
			g, a, b, source, _ := continuedDamageFixture(t)
			saved := g.CaptureDamageSource(source.ID())
			switch state {
			case "departure", "return":
				g.RemoveFromBattlefield(source)
				if state == "return" {
					g.PutOnBattlefield(source.Card, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: source.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				saved = g.CaptureDamageSource(source.ID())
			case "cleanup":
				g.effects.ClearReplacementsEndOfTurn()
			case "zero":
				g.DealDamageToPlayerFromSource(a, 0, saved)
			case "prevented":
				g.AddPreventionShield(a.PlayerID(), 2)
				g.effects.PrependReplacement(g.effects.replacements[len(g.effects.replacements)-1])
				g.DealDamageToPlayerFromSource(a, 2, saved)
			}
			baseline := g.Clone()
			for _, game := range []*Game{g, g.Clone()} {
				game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 3, saved)
				wantA, wantB := 17, 17
				if state == "cleanup" {
					wantB = 20
				}
				if state == "control" {
					wantA, wantB = 14, 20
				}
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatalf("life %d %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
				}
			}
			if baseline.GetPlayer(a.PlayerID()).Life() != 20 || baseline.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("clone mutated")
			}
		})
	}
}
