package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type tapChoicePlayer struct {
	*BasePlayer
	accept  bool
	prompts []string
}

func (p *tapChoicePlayer) ChooseMayAbility(prompt string) bool {
	p.prompts = append(p.prompts, prompt)
	return p.accept
}

func TestComposableTapChoice(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		tapped, accept, stun, want bool
	}{
		{"tap", false, true, false, true},
		{"untap", true, true, false, false},
		{"decline tap", false, false, false, false},
		{"decline untap", true, false, false, true},
		{"stun replacement", true, true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &tapChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: tc.accept}
			b := &tapChoicePlayer{BasePlayer: NewBasePlayer("B"), accept: !tc.accept}
			g := NewGame(a, b)
			c := NewCreature("Subject", "", 1, 1)
			p := g.PutOnBattlefield(c, b.PlayerID())
			p.Tapped = tc.tapped
			if tc.stun {
				p.AddCounter(Stun, 1)
			}
			if err := ApplyEffect(g, MayTapOrUntap(This), c.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(c.ID()).Tapped != tc.want || len(a.prompts) != 1 || len(b.prompts) != 0 {
				t.Fatal("wrong action or choosing player")
			}
			verb := "tap "
			if tc.tapped {
				verb = "untap "
			}
			if !strings.HasPrefix(a.prompts[0], verb) {
				t.Fatalf("prompt = %q", a.prompts[0])
			}
			if tc.stun && g.FindPermanent(c.ID()).Counters[Stun] != 0 {
				t.Fatal("stun replacement bypassed")
			}
		})
	}
}

func TestComposableTapChoiceSavedLifetime(t *testing.T) {
	for _, change := range []string{"none", "return", "absent", "phase", "phase cycle", "control"} {
		t.Run(change, func(t *testing.T) {
			a := &tapChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Subject", "", 1, 1)
			g.PutOnBattlefield(c, a.PlayerID())
			saved := BindPermanent("subject")
			body := Sequence(Snapshot(This, saved), FuncEffect("change subject", EffectProperties{}, func(g *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
				switch change {
				case "return", "absent":
					g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
					if change == "return" {
						g.PutOnBattlefield(c, a.PlayerID())
					}
				case "phase":
					g.PhaseOut(c.ID())
				case "phase cycle":
					g.PhaseIn(g.PhaseOut(c.ID()))
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: c.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
				}
				return nil
			}), MayTapOrUntap(From(saved)))
			if err := ApplyEffect(g, body, c.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			acted := change == "none" || change == "phase cycle" || change == "control"
			if (len(a.prompts) == 1) != acted {
				t.Fatal("choice used an absent incarnation")
			}
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{c.ID()})
			}
			if p := g.FindPermanent(c.ID()); p != nil && p.Tapped != acted {
				t.Fatal("wrong saved subject")
			}
		})
	}
}

func TestComposableTapChoiceEvents(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		tapped, accept, stun bool
		wantLife             int
	}{
		{"tap", false, true, false, 21},
		{"untap", true, true, false, 22},
		{"decline tap", false, false, false, 20},
		{"decline untap", true, false, false, 20},
		{"replace untap", true, true, true, 20},
		{"decline with stun", true, false, true, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &tapChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: tc.accept}
			g := NewGame(a, NewBasePlayer("B"))
			c := NewCreature("Subject", "", 1, 1)
			p := g.PutOnBattlefield(c, a.PlayerID())
			p.Tapped = tc.tapped
			if tc.stun {
				p.AddCounter(Stun, 1)
			}
			watcher := NewEnchantment("Watcher", "",
				WithAbility(NewTriggered(EvtTapped, false, GainLifeFor(You, Fixed(1)))),
				WithAbility(NewTriggered(EvtBecameUntapped, false, GainLifeFor(You, Fixed(2)))))
			g.PutOnBattlefield(watcher, a.PlayerID())
			if err := ApplyEffect(g, MayTapOrUntap(This), c.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			resolvePendingTriggers(g)
			if a.Life() != tc.wantLife {
				t.Fatalf("life = %d, want %d from action events", a.Life(), tc.wantLife)
			}
			wantStun := uint8(0)
			if tc.stun && !tc.accept {
				wantStun = 1
			}
			if got := g.FindPermanent(c.ID()).Counters[Stun]; got != wantStun {
				t.Fatalf("stun counters = %d, want %d", got, wantStun)
			}
		})
	}
}

func TestComposableTapChoiceValidation(t *testing.T) {
	for _, subject := range []PermanentRef{nil, This, DeclarePermanentTarget("undeclared", IsCreature), From(BindPermanent("unwritten"))} {
		c := NewInstant("Invalid", "", WithAbility(Spell(MayTapOrUntap(subject))))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("invalid subject accepted")
		}
	}
}

func TestTypedOpponentTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	own, other := NewArtifact("Own", ""), NewArtifact("Other", "")
	g.PutOnBattlefield(own, a.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	target := DeclarePermanentTarget("artifact", IsArtifact, TargetOpponentControls())
	ability := Spell(TapSubject(target), WithTargets(target))
	card := NewInstant("Spell", "", WithAbility(ability))
	possible := ability.Targets()[0].Possible(a.PlayerID(), card, g)
	if len(possible) != 1 || possible[0] != other.ID() {
		t.Fatal("wrong controller restriction")
	}
	g.pushStack(&StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{other.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	g.AddControlEffect(ControlEffectSpec{SourceID: own.ID(), TargetID: other.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
	g.ResolveStack()
	if g.FindPermanent(other.ID()).Tapped {
		t.Fatal("target remained legal after control change")
	}
}

func TestComposableTapChoiceTargetClone(t *testing.T) {
	for _, returned := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		first, second := NewCreature("First", "", 1, 1), NewCreature("Second", "", 1, 1)
		g.PutOnBattlefield(first, b.PlayerID())
		g.PutOnBattlefield(second, b.PlayerID())
		x, y := DeclarePermanentTarget("first", IsCreature), DeclarePermanentTarget("second", IsCreature)
		ability := Spell(Sequence(MayTapOrUntap(x), MayTapOrUntap(y)), WithTargets(x, y))
		card := NewInstant("Choice", "", WithAbility(ability))
		g.pushStack(&StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID(), second.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
		if returned {
			g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
			g.PutOnBattlefield(first, b.PlayerID())
		}
		clone := g.Clone()
		chooser := &tapChoicePlayer{BasePlayer: extractBasePlayer(clone.GetPlayer(a.PlayerID())), accept: true}
		clone.players[0] = chooser
		clone.TapPermanent(clone.FindPermanent(second.ID()))
		clone.ResolveStack()
		if clone.FindPermanent(first.ID()).Tapped == returned || clone.FindPermanent(second.ID()).Tapped {
			t.Fatal("wrong target incarnation or resolution-time state")
		}
		if g.FindPermanent(first.ID()).Tapped || g.FindPermanent(second.ID()).Tapped {
			t.Fatal("clone changed original")
		}
		want := 2
		if returned {
			want = 1
		}
		if len(chooser.prompts) != want {
			t.Fatal("illegal target received a choice")
		}
	}
}

func TestTypedOpponentTargetConflicts(t *testing.T) {
	for _, collection := range []bool{false, true} {
		t.Run(map[bool]string{false: "singular", true: "collection"}[collection], func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("conflicting restrictions accepted")
				}
			}()
			if collection {
				DeclarePermanentTargets("targets", 0, 2, IsCreature, TargetYouControl(), TargetOpponentControls())
			} else {
				DeclarePermanentTarget("target", IsCreature, TargetOpponentControls(), TargetYouControl())
			}
		})
	}
}

func TestComposableTapChoiceCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	target := DeclarePermanentTarget("artifact", IsArtifact)
	ability := Spell(MayTapOrUntap(target), WithTargets(target))
	card := NewInstant("Choice", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	original := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(original)
	a.chooseQueue = [][]uuid.UUID{{second.ID()}}
	copied := g.CopySpellOnStack(card.ID(), a.PlayerID(), true)
	chooser := &tapChoicePlayer{BasePlayer: a.BasePlayer, accept: true}
	g.players[0] = chooser
	g.ResolveStackObject(copied)
	if !g.FindPermanent(second.ID()).Tapped || g.FindPermanent(first.ID()).Tapped {
		t.Fatal("copy lost its new target")
	}
	chooser.accept = false
	g.ResolveStackObject(original)
	if g.FindPermanent(first.ID()).Tapped || len(chooser.prompts) != 2 {
		t.Fatal("copy choice affected original")
	}
}
