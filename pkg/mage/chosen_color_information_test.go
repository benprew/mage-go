package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestChosenColorInformationIdentity(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "unknown", "unset", "wildcard", "invalid", "member-phase", "layer"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Source", "")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.ChosenColor = core.Red
			red := g.PutOnBattlefield(NewCreature("Red", "{R}", 1, 5), b.PlayerID())
			blue := g.PutOnBattlefield(NewCreature("Blue", "{U}", 1, 5), b.PlayerID())
			multi := g.PutOnBattlefield(NewCreature("Multi", "{R}{U}", 1, 5), b.PlayerID())
			colorless := g.PutOnBattlefield(NewArtifact("Colorless", ""), b.PlayerID())
			ctx := &EffectContext{Game: g, SourceID: source.ID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
			wantRed, wantBlue, wantMulti := true, false, true
			switch state {
			case "departed", "returned":
				g.BouncePermanentToHand(source)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID()).ChosenColor = core.Blue
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "unknown":
				ctx.sourceIncarnation = uuid.New()
				wantRed, wantMulti = false, false
			case "unset":
				source.ChosenColor = core.Colorless
				wantRed, wantMulti = false, false
			case "wildcard":
				source.ChosenColor = core.AnyColor
				wantRed, wantMulti = false, false
			case "invalid":
				source.ChosenColor = core.Color(99)
				wantRed, wantMulti = false, false
			case "member-phase":
				g.PhaseOut(red.ID())
				wantRed = false
			case "layer":
				if err := ApplyEffect(g, Until(core.EndOfTurn, SetColors(This, core.Blue)), red.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				wantRed = false
			}
			set := PermanentsOfChosenColor(This, MatchingPermanents(AnyPermanent))
			for _, game := range []*Game{g, g.Clone()} {
				local := *ctx
				local.Game = game
				game.SetRandomResults([]int{7})
				got := set.resolvePermanents(&local)
				for _, check := range []struct {
					id   uuid.UUID
					want bool
				}{{red.ID(), wantRed}, {blue.ID(), wantBlue}, {multi.ID(), wantMulti}, {colorless.ID(), false}} {
					found := false
					for _, p := range got {
						found = found || p.ID() == check.id
					}
					if found != check.want {
						t.Fatalf("member %s = %v, want %v", check.id, found, check.want)
					}
				}
				if game.RandIntn(10) != 7 || red.Tapped {
					t.Fatal("query changed state")
				}
			}
		})
	}
}

func TestChosenColorInformationDeclarationsAndWaiting(t *testing.T) {
	for _, set := range []PermanentSet{PermanentsOfChosenColor(nil, MatchingPermanents(AnyPermanent)), PermanentsOfChosenColor(This, nil), PermanentsOfChosenColor(From(BindPermanent("unbound")), MatchingPermanents(AnyPermanent))} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), TapEach(set))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	source.ChosenColor = core.Red
	reads := 0
	set := PermanentsOfChosenColor(This, MatchingPermanents(NewPermanentFilter("query", func(*Permanent, *Game) bool { reads++; return true })))
	if err := ApplyEffect(g, AtNext(EachEndStep, TapEach(set)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 {
			t.Fatal("waiting query lost source or executed predicate")
		}
	}
	clone := g.Clone()
	clone.MutablePermanent(source.ID()).ChosenColor = core.Blue
	if g.FindPermanent(source.ID()).ChosenColor != core.Red {
		t.Fatal("clone shares choice")
	}
}

func TestChosenColorInformationSavedReferences(t *testing.T) {
	for _, kind := range []string{"target", "saved", "host"} {
		t.Run(kind, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Choice", "")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			p.ChosenColor = core.Red
			red := g.PutOnBattlefield(NewCreature("Red", "{R}", 1, 5), b.PlayerID())
			blue := g.PutOnBattlefield(NewCreature("Blue", "{U}", 1, 5), b.PlayerID())
			target := DeclarePermanentTarget("choice", AnyPermanent)
			binding := BindPermanent("saved")
			ctx := &EffectContext{Game: g, SourceID: p.ID(), sourceCaptured: true, sourceIncarnation: p.incarnationID, targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{p.ID(), p.incarnationID}}}}
			if err := Snapshot(target, binding).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			var ref PermanentRef = target
			if kind == "saved" {
				ref = From(binding)
			}
			if kind == "host" {
				aura := g.PutOnBattlefield(NewEnchantment("Aura", "", WithSubTypes("Aura")), a.PlayerID())
				g.Attach(aura.ID(), p.ID())
				ctx.SourceID, ctx.sourceIncarnation = aura.ID(), aura.incarnationID
				ref = AttachedPermanent
			}
			g.BouncePermanentToHand(p)
			a.RemoveFromHand(card.ID())
			g.PutOnBattlefield(card, a.PlayerID()).ChosenColor = core.Blue
			if err := TapEach(PermanentsOfChosenColor(ref, MatchingPermanents(AnyPermanent))).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if !red.Tapped || blue.Tapped {
				t.Fatal("saved reference followed returned visit")
			}
		})
	}
}
