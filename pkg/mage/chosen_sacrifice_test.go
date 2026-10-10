package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestChosenSacrificePlayers(t *testing.T) {
	for _, state := range []string{"normal", "phased", "empty", "exile", "invalid", "control"} {
		t.Run(state, func(t *testing.T) {
			a := &lastSacrificeChoicePlayer{BasePlayer: NewBasePlayer("A")}
			b := &lastSacrificeChoicePlayer{BasePlayer: NewBasePlayer("B")}
			g := NewGame(a, b)
			first := g.PutOnBattlefield(NewLand("First"), a.PlayerID())
			card := NewLand("Second", WithKeyword(core.Indestructible))
			card.SetOwner(b.PlayerID())
			second := g.PutOnBattlefield(card, a.PlayerID())
			foreign := g.PutOnBattlefield(NewLand("Foreign"), b.PlayerID())
			g.AddRegenerationShield(second.ID())
			chooser := PlayerRef(ActivePlayer)
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			g.SetActivePlayerIndex(1)
			switch state {
			case "phased":
				g.PhaseOut(second.ID())
			case "empty":
				g.PhaseOut(first.ID())
				g.PhaseOut(second.ID())
			case "exile":
				g.AddExileIfWouldGoToGraveyardThisTurn(second.ID(), foreign.ID())
			case "invalid":
				chooser = FromPlayer(BindPlayer("absent"))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: foreign.ID(), TargetID: second.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			clone := g.Clone()
			if err := SacrificeChosenBy(You, chooser, MatchingPermanents(IsLand)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			selectedSecond := state == "normal" || state == "exile"
			if (g.FindPermanentIncludingPhased(second.ID()) == nil) != selectedSecond {
				t.Fatal("wrong sacrifice subject")
			}
			selectedFirst := state == "phased" || state == "control"
			if (g.FindPermanentIncludingPhased(first.ID()) == nil) != selectedFirst || g.FindPermanent(foreign.ID()) == nil {
				t.Fatal("wrong controlled collection")
			}
			if a.choices != 0 || (b.choices != 1 && selectedSecond) {
				t.Fatal("wrong chooser")
			}
			if state == "normal" && len(b.Graveyard()) != 1 {
				t.Fatal("owner destination")
			}
			if state == "exile" && len(g.GetExile()) != 1 {
				t.Fatal("replacement bypassed")
			}
			if clone.FindPermanent(second.ID()) == nil && state != "phased" && state != "empty" {
				t.Fatal("original changed clone")
			}
		})
	}
}

func TestChosenSacrificeDefinition(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "valid", Card: NewSorcery("Good", "", Spell(SacrificeChosenBy(You, You, MatchingPermanents(IsLand))))},
		{Name: "player", Card: NewSorcery("Bad", "", Spell(SacrificeChosenBy(nil, You, MatchingPermanents(IsLand)))), WantErr: true},
		{Name: "chooser", Card: NewSorcery("Bad", "", Spell(SacrificeChosenBy(You, nil, MatchingPermanents(IsLand)))), WantErr: true},
		{Name: "subjects", Card: NewSorcery("Bad", "", Spell(SacrificeChosenBy(You, You, nil))), WantErr: true},
		{Name: "unbound", Card: NewSorcery("Bad", "", Spell(SacrificeChosenBy(FromPlayer(BindPlayer("absent")), You, MatchingPermanents(IsLand)))), WantErr: true},
	})
}

type chosenSacrificePlayer struct {
	*BasePlayer
	choose func([]*Permanent) *Permanent
}

func (p *chosenSacrificePlayer) ChoosePermanent(candidates []*Permanent, _ string, _ GameReader) *Permanent {
	return p.choose(candidates)
}

func TestChosenSacrificeIdentity(t *testing.T) {
	for _, state := range []string{"return", "phase", "control", "invalid", "nil"} {
		t.Run(state, func(t *testing.T) {
			a := NewBasePlayer("A")
			b := &chosenSacrificePlayer{BasePlayer: NewBasePlayer("B")}
			g := NewGame(a, b)
			g.SetActivePlayerIndex(1)
			first := g.PutOnBattlefield(NewLand("First"), a.PlayerID())
			card := NewLand("Second")
			second := g.PutOnBattlefield(card, a.PlayerID())
			foreign := g.PutOnBattlefield(NewLand("Foreign"), b.PlayerID())
			b.choose = func(_ []*Permanent) *Permanent {
				switch state {
				case "return":
					g.RemoveFromBattlefield(second)
					g.PutOnBattlefield(card, a.PlayerID())
				case "phase":
					g.PhaseOut(second.ID())
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: foreign.ID(), TargetID: second.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "invalid":
					return foreign
				case "nil":
					return nil
				}
				return second
			}
			if err := SacrificeChosenBy(You, ActivePlayer, MatchingPermanents(IsLand)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			fallback := state == "invalid" || state == "nil"
			if (g.FindPermanent(first.ID()) == nil) != fallback || g.FindPermanentIncludingPhased(second.ID()) == nil || g.FindPermanent(foreign.ID()) == nil {
				t.Fatal("choice followed a new incarnation or illegal subject")
			}
		})
	}
}

func TestChosenSacrificeWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	body := SacrificeChosenBy(ControllerOf(This), You, SingletonPermanent(This))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || g.FindPermanent(source.ID()) == nil {
			t.Fatal("waiting query lost or sacrificed source")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(source.ID()) != nil {
		t.Fatal("delayed sacrifice failed")
	}
}
