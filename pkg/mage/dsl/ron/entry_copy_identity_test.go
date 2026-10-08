package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONEntryCopyIdentityGameplay(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "FEM", "Icatian Javelineers"), b.PlayerID())
	values, err := g.CaptureCopiableValues(source)
	if err != nil {
		t.Fatal(err)
	}
	original := repositoryCard(t, "LEB", "Air Elemental")
	original.SetOwner(a.PlayerID())
	card, err := mage.BuildEntryCopy(original, values)
	if err != nil {
		t.Fatal(err)
	}
	if card.ID() != original.ID() || card.Owner() != a.PlayerID() || original.Name() != "Air Elemental" {
		t.Fatal("copy changed physical identity or source definition")
	}
	copied := g.PutOnBattlefield(card, b.PlayerID())
	if copied.Counters[core.Javelin] != 1 || copied.Name() != "Icatian Javelineers" || copied.CurrentPower(g) != 1 || copied.HasAttr(core.Flying) {
		t.Fatal("copy did not acquire fresh entry rules and characteristics")
	}
	copied.RevokeBaseAttr(core.AttrSummonSick)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), copied.ID(), 1, []uuid.UUID{uuid.New()}); err == nil || copied.Counters[core.Javelin] != 1 || copied.Tapped {
		t.Fatal("illegal target consumed an acquired cost")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), copied.ID(), 1, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(copied)
	if len(a.Hand()) != 1 || a.Hand()[0].Name() != "Air Elemental" || a.Hand()[0].ID() != original.ID() || !a.Hand()[0].HasType(core.TypeCreature) {
		t.Fatal("departure kept copied rules or used the controller's hand")
	}
	if info := g.LKI(copied.ID()); info == nil || info.Name != "Icatian Javelineers" {
		t.Fatal("departure lost copied last-known information")
	}
	g.ResolveStack()
	if a.Life() != 19 || source.Counters[core.Javelin] != 1 {
		t.Fatal("departure lost activation or shared payment state")
	}
	returned, _ := a.RemoveFromHand(original.ID())
	p := g.PutOnBattlefield(returned, a.PlayerID())
	if p.Name() != "Air Elemental" || !p.HasAttr(core.Flying) || p.Counters[core.Javelin] != 0 {
		t.Fatal("returned physical card retained copy characteristics")
	}
}

func TestRONEntryCopyIdentityEntryReplacements(t *testing.T) {
	for _, choice := range []int{0, 1} {
		a := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("A"), index: 1 - choice}
		b := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("B"), index: choice}
		g := mage.NewGame(a, b)
		source := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Triskelion"), a.PlayerID())
		values, err := g.CaptureCopiableValues(source)
		if err != nil {
			t.Fatal(err)
		}
		donor := g.PutOnBattlefield(mage.NewArtifact("Donor", ""), a.PlayerID())
		g.AddETBAdditionalCounters(donor.ID(), core.P1P1, 1, mage.IsCreature, false)
		g.AddCounterDoubler(donor.ID(), core.P1P1, mage.IsCreature)
		physical := repositoryCard(t, "FEM", "Icatian Javelineers")
		physical.SetOwner(a.PlayerID())
		card, err := mage.BuildEntryCopy(physical, values)
		if err != nil {
			t.Fatal(err)
		}
		p := g.PutOnBattlefield(card, b.PlayerID())
		want := 8
		if choice == 1 {
			want = 7
		}
		if int(p.Counters[core.P1P1]) != want || p.Counters[core.Javelin] != 0 || a.choices != 0 || b.choices != 1 || !p.HasType(core.TypeArtifact) {
			t.Fatal("copy entry used physical rules or a separate replacement pipeline")
		}
	}
}

func TestRONEntryCopyIdentityDeathAndFreshEntryChoices(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "ARN", "Rukh Egg"), a.PlayerID())
	values, err := g.CaptureCopiableValues(source)
	if err != nil {
		t.Fatal(err)
	}
	physical := repositoryCard(t, "LEB", "Air Elemental")
	physical.SetOwner(a.PlayerID())
	card, err := mage.BuildEntryCopy(physical, values)
	if err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, b.PlayerID())
	g.DestroyPermanent(p)
	if a.Graveyard()[0].Name() != "Air Elemental" || len(g.DelayedTriggers()) != 0 {
		t.Fatal("destination definition or death timing")
	}
	pending := g.Clone()
	g.ResolveStack()
	g.RunStepWithPriority(core.EndStep)
	if len(g.AllBattlefield()) != 2 {
		t.Fatal("copied death rule did not create its delayed token")
	}
	for _, permanent := range g.AllBattlefield() {
		if permanent.IsToken && (permanent.ControllerID() != b.PlayerID() || permanent.CurrentPower(g) != 4 || !permanent.HasAttr(core.Flying)) {
			t.Fatal("copied death rule used the physical owner")
		}
	}
	if len(pending.AllBattlefield()) != 1 || len(pending.DelayedTriggers()) != 0 {
		t.Fatal("death resolution changed cloned pending trigger")
	}
	pending.ResolveStack()
	pending.RunStepWithPriority(core.EndStep)
	if len(pending.AllBattlefield()) != 2 {
		t.Fatal("clone lost copied death rule")
	}
	entryFactory := permanentScopeFactory(t, "Artifact", `AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)])`)
	chosen := g.PutOnBattlefield(entryFactory(), a.PlayerID())
	choiceValues, err := g.CaptureCopiableValues(chosen)
	if err != nil {
		t.Fatal(err)
	}
	card, err = mage.BuildEntryCopy(repositoryCard(t, "LEB", "Wall of Wood"), choiceValues)
	if err != nil {
		t.Fatal(err)
	}
	newChoice := g.PutOnBattlefield(card, b.PlayerID())
	if chosen.ChosenPlayer != b.PlayerID() || newChoice.ChosenPlayer != a.PlayerID() {
		t.Fatal("copy retained the source's entry choice")
	}
}

func TestRONEntryCopyIdentityDeparture(t *testing.T) {
	for _, destination := range []string{"destroy", "sacrifice", "graveyard", "exile", "bounce"} {
		t.Run(destination, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), b.PlayerID())
			values, err := g.CaptureCopiableValues(source)
			if err != nil {
				t.Fatal(err)
			}
			original := repositoryCard(t, "LEB", "Wall of Wood")
			original.SetOwner(a.PlayerID())
			card, err := mage.BuildEntryCopy(original, values)
			if err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(card, b.PlayerID())
			g.PhaseOut(p.ID())
			g.PhaseIn([]uuid.UUID{p.ID()})
			if p.Name() != "Air Elemental" {
				t.Fatal("phase cycle ended copy")
			}
			pending := g.Clone()
			switch destination {
			case "destroy":
				g.DestroyPermanent(p)
			case "sacrifice":
				g.DoSacrifice(p)
			case "graveyard":
				g.PutPermanentIntoGraveyard(p)
			case "exile":
				g.ExilePermanent(p)
			case "bounce":
				g.BouncePermanentToHand(p)
			}
			var departed mage.Card
			switch destination {
			case "exile":
				departed = g.FindExiledCard(p.ID()).Card
			case "bounce":
				departed = a.Hand()[0]
			default:
				departed = a.Graveyard()[0]
			}
			if departed.ID() != original.ID() || departed.Owner() != a.PlayerID() || departed.Name() != "Wall of Wood" || departed.AttrSeeds()[core.Defender] == 0 {
				t.Fatal("destination lost the physical definition")
			}
			if pending.FindPermanent(p.ID()).Name() != "Air Elemental" {
				t.Fatal("departure changed cloned battlefield")
			}
			pending.DoSacrifice(pending.FindPermanent(p.ID()))
			if pending.GetPlayer(a.PlayerID()).Graveyard()[0].Name() != "Wall of Wood" {
				t.Fatal("clone lost physical definition")
			}
		})
	}
}
