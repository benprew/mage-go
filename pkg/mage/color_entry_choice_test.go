package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type colorEntryPlayer struct {
	*BasePlayer
	answer core.Color
	calls  int
}

func (*colorEntryPlayer) ChooseMayAbility(string) bool { return true }

func TestChooseColorAsEntersCopyReplacementAndBatch(t *testing.T) {
	a := &colorEntryPlayer{BasePlayer: NewBasePlayer("A"), answer: core.Black}
	b := &colorEntryPlayer{BasePlayer: NewBasePlayer("B"), answer: core.Red}
	g := NewGame(a, b)
	donorPlan, err := NewCardDefinitionPlan(func() Card {
		return NewCreature("Color donor", "", 2, 2, WithAbility(ChooseColorAsEnters()))
	})
	if err != nil {
		t.Fatal(err)
	}
	donorCard, err := donorPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	donor := g.PutOnBattlefield(donorCard, b.PlayerID())
	copyPlan, err := NewCardDefinitionPlan(func() Card {
		return NewCreature("Physical copy", "", 1, 1, WithAbility(ChooseColorAsEnters()), WithAbility(MayCopyAsEnters(IsCreature)))
	})
	if err != nil {
		t.Fatal(err)
	}
	copyCard, err := copyPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	copied := g.PutOnBattlefield(copyCard, a.PlayerID())
	if copied == nil || copied.Name() != "Color donor" || copied.ChosenColor != core.Black || donor.ChosenColor != core.Red || a.calls != 1 || b.calls != 1 {
		t.Fatal("copy replacement did not precede one fresh color choice")
	}
	observed := 0
	watcher := NewArtifact("Watcher", "", WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(evt *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		if evt.ToZone == core.ZoneBattlefield {
			p := reader.FindPermanent(evt.SourceID)
			if p != nil && p.Name() == "Batch color" {
				observed++
				var members []*Permanent
				for _, candidate := range reader.AllBattlefield() {
					if candidate.Name() == "Batch color" {
						members = append(members, candidate)
					}
				}
				if len(members) != 2 || a.calls != 3 {
					t.Fatal("entry observer ran before all choices and entries")
				}
				for _, member := range members {
					if member.ChosenColor != core.Green {
						t.Fatal("entry event did not see saved color")
					}
				}
			}
		}
		return false
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	a.answer = core.Green
	entries := g.putTokensOnBattlefieldTogether([]Card{
		NewArtifact("Batch color", "", WithAbility(ChooseColorAsEnters())),
		NewArtifact("Batch color", "", WithAbility(ChooseColorAsEnters())),
	}, a.PlayerID())
	if len(entries) != 2 || observed != 2 {
		t.Fatal("batch entry or event count")
	}
}

func (p *colorEntryPlayer) ChooseManaColor(string) core.Color {
	p.calls++
	return p.answer
}

func TestChooseColorAsEntersTimingAndIdentity(t *testing.T) {
	for _, answer := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green, core.Colorless, core.AnyColor, -1, 99} {
		t.Run(answer.String(), func(t *testing.T) {
			a := &colorEntryPlayer{BasePlayer: NewBasePlayer("A"), answer: answer}
			b := &colorEntryPlayer{BasePlayer: NewBasePlayer("B"), answer: core.Green}
			g := NewGame(a, b)
			want := answer
			if answer < core.White || answer > core.Green {
				want = core.White
			}
			checks := 0
			probe := &opponentEntryProbe{check: func(g *Game, p *Permanent) {
				checks++
				if p.ChosenColor != want {
					t.Fatal("choice unavailable to later entry operations")
				}
				for _, live := range g.AllBattlefield() {
					if live.ID() == p.ID() {
						t.Fatal("choice ran after battlefield registration")
					}
				}
			}}
			card := NewArtifact("Color entry", "", WithAbility(ChooseColorAsEnters()), WithAbility(probe))
			card.SetOwner(b.PlayerID())
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			if checks != 1 || a.calls != 1 || b.calls != 0 || g.StackSize() != 0 {
				t.Fatal("wrong chooser, repeated choice or stacked entry operation")
			}
			g.PhaseIn(g.PhaseOut(p.ID()))
			g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			if a.calls != 1 || b.calls != 0 || g.FindPermanent(p.ID()).ChosenColor != want {
				t.Fatal("control or phase change repeated choice")
			}
			clone := g.Clone()
			clone.MutablePermanent(p.ID()).ChosenColor = core.Colorless
			if g.FindPermanent(p.ID()).ChosenColor != want {
				t.Fatal("clone shares choice storage")
			}
			firstVisit := p.incarnationID
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			b.RemoveFromHand(card.ID())
			want = core.Green
			returned := g.PutOnBattlefield(card, b.PlayerID())
			if returned.ChosenColor != want || returned.incarnationID == firstVisit || b.calls != 1 || checks != 2 {
				t.Fatal("return did not make a fresh choice for a new visit")
			}
		})
	}
}

func TestChooseColorAsEntersCopyAndMissingPlayer(t *testing.T) {
	a := &colorEntryPlayer{BasePlayer: NewBasePlayer("A"), answer: core.Red}
	b := &colorEntryPlayer{BasePlayer: NewBasePlayer("B"), answer: core.Blue}
	g := NewGame(a, b)
	plan, err := NewCardDefinitionPlan(func() Card { return NewCreature("Color donor", "", 2, 2, WithAbility(ChooseColorAsEnters())) })
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	donor := g.PutOnBattlefield(card, a.PlayerID())
	values, err := g.CaptureCopiableValues(donor)
	if err != nil {
		t.Fatal(err)
	}
	physicalPlan, err := NewCardDefinitionPlan(func() Card { return NewCreature("Copy", "", 1, 1) })
	if err != nil {
		t.Fatal(err)
	}
	physical, err := physicalPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	copied, err := BuildEntryCopy(physical, values)
	if err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(copied, b.PlayerID())
	if p.ChosenColor != core.Blue || donor.ChosenColor != core.Red || a.calls != 1 || b.calls != 1 {
		t.Fatal("copy inherited donor choice or changed donor")
	}
	missing := NewPermanent(NewArtifact("Missing player", ""), physical.ID())
	missing.ChosenColor = core.Red
	if !ChooseColorAsEnters().OnEnter(g, missing) || missing.ChosenColor != core.White {
		t.Fatal("absent chooser kept an old or invalid color")
	}
	if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithAbility(ChooseColorAsEnters()))); err == nil {
		t.Fatal("nonpermanent entry choice accepted")
	}
}
