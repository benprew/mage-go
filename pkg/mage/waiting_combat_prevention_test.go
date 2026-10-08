package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingCombatPreventionReferences(t *testing.T) {
	for _, kind := range []string{"to-and-by", "source", "collection-turn", "collection-combat"} {
		for _, state := range []string{"live", "depart", "return", "phase", "cycle", "control", "damage", "combat-end", "cleanup"} {
			t.Run(kind+"/"+state, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Subject", "{R}", 1, 20)
				p := g.PutOnBattlefield(card, a.PlayerID())
				other := g.PutOnBattlefield(NewCreature("Other", "{U}", 1, 20), b.PlayerID())
				saved := g.CaptureDamageSource(p.ID())
				var effect ResolvingEffect
				switch kind {
				case "to-and-by":
					effect = PreventCombatDamageToAndBy(This)
				case "source":
					effect = PreventCombatDamageByThisTurn(This)
				case "collection-turn", "collection-combat":
					g.combat.AddAttackerIncarnation(other.ID(), other.incarnationID, a.PlayerID())
					g.combat.AddBlockerIncarnation(p.ID(), p.incarnationID, other.ID())
					if kind == "collection-turn" {
						effect = PreventCombatDamageByEachThisTurn(BlockersOf(This))
					} else {
						effect = PreventCombatDamageByEach(BlockersOf(This))
					}
				}
				source := p.ID()
				if kind == "collection-turn" || kind == "collection-combat" {
					source = other.ID()
				}
				if err := ApplyEffect(g, effect, source, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if got := g.WaitingPreventionPermanentSources(); len(got) != 1 || !got[0].SameObject(saved) {
					t.Fatal("missing installed combat reference")
				}
				g.combat.Reset()
				switch state {
				case "depart", "return":
					g.RemoveFromBattlefield(p)
					if state == "return" {
						g.PutOnBattlefield(card, b.PlayerID())
					}
					g.zones.ClearLKI()
				case "phase", "cycle":
					ids := g.PhaseOut(p.ID())
					if state == "cycle" {
						g.PhaseIn(ids)
					}
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: other.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "damage":
					g.damage.SetResolvingCombatDamage(true)
					g.DealDamageToPlayerFromSource(b, 3, saved)
					if b.Life() != 20 {
						t.Fatal("combat prevention failed")
					}
					g.damage.SetResolvingCombatDamage(false)
				case "combat-end":
					g.effects.RemoveEndOfCombat()
				case "cleanup":
					g.effects.ClearReplacementsEndOfTurn()
					g.effects.RemoveEndOfCombat()
				}
				clone := g.Clone()
				for _, game := range []*Game{g, clone} {
					got := game.WaitingPreventionPermanentSources()
					want := 1
					if state == "cleanup" || state == "combat-end" && kind == "collection-combat" {
						want = 0
					}
					if len(got) != want {
						t.Fatalf("references=%d want=%d", len(got), want)
					}
					if want == 1 && (!got[0].SameObject(saved) || state == "control" && got[0].Controller() != b.PlayerID()) {
						t.Fatal("incorrect retained identity or current information")
					}
				}
				g.effects.ClearReplacementsEndOfTurn()
				g.effects.RemoveEndOfCombat()
				if state != "cleanup" && (state != "combat-end" || kind != "collection-combat") && len(clone.WaitingPreventionPermanentSources()) != 1 {
					t.Fatal("original cleanup changed clone")
				}
			})
		}
	}
}

func TestWaitingCombatPreventionChoiceBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "{R}", 1, 20), a.PlayerID())
	for _, effect := range []Effect{PreventCombatDamageToAndBy(This), PreventCombatDamageByThisTurn(This), PreventCombatDamageByEachThisTurn(fixedTapSet{permanents: []*Permanent{p, p}})} {
		if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	g.effects.AddReplacement(&combatSourcePreventionReplacement{preventedID: p.ID()})
	g.effects.AddReplacement(&creatureCombatDamagePreventionReplacement{creatureID: uuid.New(), incarnation: uuid.New()})
	g.SetPreventCombatDamage()
	if got := g.WaitingPreventionPermanentSources(); len(got) != 1 {
		t.Fatal("duplicate, filter-only or invalid reference")
	}
	first, second := BindDamageSource("first"), BindDamageSource("second")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	for _, binding := range []*DamageSourceBinding{first, second} {
		if err := ChooseDamageSourceFromWaitingPreventionReferences(You, DamageSourceHasColor(core.Red), binding).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if !g.resolution.damageSourceBindings[binding].SameObject(g.CaptureDamageSource(p.ID())) {
			t.Fatal("filtered choice lost combat subject")
		}
	}
	if err := ChooseDamageSourceFromWaitingPreventionReferences(You, DamageSourceHasColor(core.Blue), first).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[first].Valid() || !g.resolution.damageSourceBindings[second].Valid() {
		t.Fatal("empty filtered choice changed independent result")
	}
	got := g.WaitingPreventionPermanentSources()
	got[0].card.SetOwner(b.PlayerID())
	if g.WaitingPreventionPermanentSources()[0].card.Owner() == b.PlayerID() {
		t.Fatal("query shares mutable card information")
	}
}
