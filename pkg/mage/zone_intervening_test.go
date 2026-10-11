package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestZoneInterveningInformation(t *testing.T) {
	for _, change := range []string{"live", "lost", "source-left", "source-returned", "host-left", "host-returned", "host-phase", "cycle", "reattach"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			hostCard := NewCreature("Host", "", 2, 4, WithKeyword(Flying))
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			sourceCard := NewAura("Source", "")
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			g.Attach(source.ID(), host.ID())
			ctx := StateContext{SourceID: source.ID(), ControllerID: a.PlayerID(), SourceIncarnation: source.incarnationID}
			condition := PermanentMatchesFilterWithLKI(AttachedPermanent, HasKeywordFilter(Flying))
			switch change {
			case "lost":
				g.AddContinuousEffect(RemoveAllAbilities(host.ID(), Indefinite))
				g.ApplyContinuousEffects()
			case "source-left", "source-returned":
				g.RemoveFromBattlefield(source)
				if change == "source-returned" {
					q := g.PutOnBattlefield(sourceCard, a.PlayerID())
					r := g.PutOnBattlefield(NewCreature("Other", "", 2, 4), b.PlayerID())
					g.Attach(q.ID(), r.ID())
				}
			case "host-left", "host-returned":
				g.RemoveFromBattlefield(host)
				if change == "host-returned" {
					q := g.PutOnBattlefield(hostCard, b.PlayerID())
					g.AddContinuousEffect(RemoveAllAbilities(q.ID(), Indefinite))
					g.ApplyContinuousEffects()
				}
			case "host-phase":
				g.PhaseOut(host.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "reattach":
				q := g.PutOnBattlefield(NewCreature("Other", "", 2, 4), b.PlayerID())
				g.Attach(source.ID(), q.ID())
			}
			want := change != "lost" && change != "reattach"
			if condition.checkState(g, ctx) != want || condition.checkState(g.Clone(), ctx) != want {
				t.Fatal("condition lost exact attachment information")
			}
			ctx.SourceIncarnation = uuid.New()
			if condition.checkState(g, ctx) {
				t.Fatal("unknown source supplied information")
			}
		})
	}
}

func TestZoneInterveningHistoricalCause(t *testing.T) {
	for _, sacrificed := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		c := NewArtifact("Watcher", "", WithAbility(WhenEvent(PermanentDies().OfType(TypeArtifact), GainLifeFor(You, Fixed(2)), WithDeathNotSacrificed())))
		if err := ValidateCardDefinition(c.Name(), c); err != nil {
			t.Fatal(err)
		}
		c.SetOwner(a.PlayerID())
		source := g.PutOnBattlefield(c, a.PlayerID())
		victimCard := NewArtifact("Victim", "")
		victimCard.SetOwner(a.PlayerID())
		victim := g.PutOnBattlefield(victimCard, a.PlayerID())
		if sacrificed {
			g.DoSacrifice(victim)
		} else {
			g.DestroyPermanent(victim)
		}
		g.DoSacrifice(g.FindPermanent(source.ID()))
		g.PutTriggersOnStack()
		want := 0
		if !sacrificed {
			want = 1
		}
		if g.StackSize() != want {
			t.Fatalf("stack=%d want %d", g.StackSize(), want)
		}
		clone := g.Clone()
		g.ResolveStack()
		clone.ResolveStack()
		if a.Life() != 20+2*want || clone.GetPlayer(a.PlayerID()).Life() != 20+2*want {
			t.Fatal("later sacrifice overwrote death cause")
		}
	}
}

func TestZoneInterveningInvalidDefinitions(t *testing.T) {
	for _, ability := range []TriggeredAbility{
		WhenEvent(Enters(This), GainLifeFor(You, Fixed(1)), WithDeathNotSacrificed()),
		At(YourUpkeep, GainLifeFor(You, Fixed(1)), WithDeathNotSacrificed()),
		WhenEvent(PermanentDies(), GainLifeFor(You, Fixed(1)), WithDeathNotSacrificed(), WithDeathNotSacrificed()),
		WhenEvent(PermanentDies(), GainLifeFor(You, Fixed(1)), WithDeathNotSacrificed(), WithInterveningIf(Controls(IsLand))),
	} {
		card := NewArtifact("Invalid", "", WithAbility(ability))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid event context accepted")
		}
	}
	for _, condition := range []StateCondition{PermanentMatchesFilterWithLKI(From(BindPermanent("unbound")), IsCreature)} {
		card := NewArtifact("Invalid", "", WithAbility(WhenEvent(Enters(This), GainLifeFor(You, Fixed(1)), WithInterveningIf(condition))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid information condition accepted")
		}
	}
}

func TestZoneInterveningIndependentEventsAndCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 2 {
		card := NewArtifact("Watcher", "", WithAbility(WhenEvent(PermanentDies().OfType(TypeArtifact), GainLifeFor(You, Fixed(1)), WithDeathNotSacrificed())))
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	card := NewArtifact("Victim", "")
	card.SetOwner(a.PlayerID())
	victim := g.PutOnBattlefield(card, a.PlayerID())
	g.DestroyPermanent(victim)
	g.PutOnBattlefield(card, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(card.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("death and sacrifice histories were conflated")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 21 {
		t.Fatal("copy lost event or controller")
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("clone shared resolution")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 21 {
		t.Fatal("clone lost historical cause")
	}
}

func TestZoneInterveningRejectsModalAndUnguardedBodies(t *testing.T) {
	for _, change := range []string{"modal", "append"} {
		trigger := WhenEvent(PermanentDies(), GainLifeFor(You, Fixed(1)), WithDeathNotSacrificed()).(*GenericTriggered)
		if change == "modal" {
			trigger.WithModes(Mode{Effects: []Effect{GainLifeFor(You, Fixed(1))}}, Mode{Effects: []Effect{GainLifeFor(You, Fixed(2))}})
		} else {
			trigger.AddEffect(GainLifeFor(You, Fixed(2)))
		}
		card := NewArtifact("Invalid", "", WithAbility(trigger))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("unguarded trigger body accepted", change)
		}
	}
}
