package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestHandOperationsCurrentPlayerAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land, white, blue := NewLand("Land"), NewCreature("White", "{W}", 1, 1), NewCreature("Blue", "{U}", 1, 1)
	for _, c := range []Card{land, white, blue} {
		c.SetOwner(b.PlayerID())
		b.AddToHand(c)
	}
	a.AddToHand(NewLand("Other"))
	target := DeclarePlayerTarget("player")
	card := NewSorcery("Hand operations", "", WithAbility(Spell(Sequence(RevealHandFor(target), DiscardMatchingHand(target, NewCardFilter("nonland", func(c Card) bool { return !c.HasType(core.TypeLand) }))), WithTargets(target))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
	count := CountMatchingHand(You, HasColorCardFilter(core.White))
	if count.resolveValue(ctx) != 1 || len(g.PublicReveals()) != 0 {
		t.Fatal("query changed state or count")
	}
	effect := Sequence(RevealHandFor(You), DiscardMatchingHand(You, NewCardFilter("nonland", func(c Card) bool { return !c.HasType(core.TypeLand) })))
	clone := g.Clone()
	if err := ApplyEffect(clone, effect, uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(b.Hand()) != 3 || len(g.PublicReveals()) != 0 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 1 {
		t.Fatal("clone isolation")
	}
	seen := 0
	watcher := NewTriggered(core.EvtDiscard, false).SetCondition(func(evt *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		seen++
		if len(b.Hand()) != 1 || len(b.Graveyard()) != 2 || len(g.PublicReveals()) != 3 {
			t.Error("event observed partial discard or early reveal")
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(watcher)), a.PlayerID())
	if err := ApplyEffect(g, effect, uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if seen != 2 || len(a.Hand()) != 1 || b.Hand()[0].ID() != land.ID() || count.resolveValue(ctx) != 0 {
		t.Fatal("wrong player, filter, or current count")
	}
	reveals := g.PublicReveals()
	reveals[1].Types[0] = core.TypeLand
	if g.PublicReveals()[1].Types[0] != core.TypeCreature {
		t.Fatal("history aliases caller")
	}
	if err := ApplyEffect(g, RevealHandFor(You), uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.PublicReveals()) != 4 || len(b.Hand()) != 1 {
		t.Fatal("repeat reveal moved cards")
	}
}

func TestHandOperationsDeclarations(t *testing.T) {
	filter := NewCardFilter("all", func(Card) bool { return true })
	for _, effect := range []Effect{RevealHandFor(nil), RevealHandFor(DeclarePlayerTarget("missing")), DiscardMatchingHand(nil, filter), GainLifeFor(You, CountMatchingHand(nil, filter))} {
		c := NewSorcery("Invalid", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("accepted invalid hand declaration")
		}
	}
	g := NewGame(NewBasePlayer("A"), NewBasePlayer("B"))
	missing := ControllerOf(This)
	if err := ApplyEffect(g, Sequence(RevealHandFor(missing), DiscardMatchingHand(missing, filter), GainLifeFor(You, CountMatchingHand(missing, filter))), uuid.New(), g.ActivePlayerObj().PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.PublicReveals()) != 0 || g.ActivePlayerObj().Life() != 20 {
		t.Fatal("absent player did work")
	}
}

func TestHandOperationsDiscardReplacements(t *testing.T) {
	for _, mode := range []string{"library", "prevent"} {
		t.Run(mode, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first, second := NewCreature("First", "", 1, 1), NewCreature("Second", "", 1, 1)
			for _, c := range []Card{first, second} {
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
			}
			g.effects.AddReplacement(&handOperationReplacement{card: second.ID(), prevent: mode == "prevent"})
			if err := ApplyEffect(g, DiscardMatchingHand(You, IsCreatureCard), uuid.New(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != first.ID() {
				t.Fatal("batch replacement missed original hand")
			}
			if mode == "library" && (len(a.Hand()) != 0 || len(a.Library()) != 1) {
				t.Fatal("redirection")
			}
			if mode == "prevent" && (len(a.Hand()) != 1 || a.Hand()[0].ID() != second.ID()) {
				t.Fatal("prevention")
			}
		})
	}
}

type handOperationReplacement struct {
	replacementBase
	card    uuid.UUID
	prevent bool
}

func (r *handOperationReplacement) Matches(a Action, g GameReader) bool {
	d, ok := a.(*DiscardAction)
	return ok && d.CausedByEffect() && d.CardID() == r.card && len(g.GetPlayer(d.PlayerID()).Hand()) == 2
}
func (r *handOperationReplacement) Replace(a Action, g *Game) Action {
	if r.prevent {
		return nil
	}
	return (&allDiscardToLibraryReplacement{}).Replace(a, g)
}
func (*handOperationReplacement) IsActive(GameReader) bool { return true }
func (r *handOperationReplacement) Clone() ReplacementEffect {
	return &handOperationReplacement{card: r.card, prevent: r.prevent}
}
