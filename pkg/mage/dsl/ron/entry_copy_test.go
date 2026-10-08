package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronEntryCopyPlayer struct {
	mage.Player
	accept          bool
	offers, choices int
	pick            func([]*mage.Permanent, mage.GameReader) *mage.Permanent
}

func (p *ronEntryCopyPlayer) ChooseMayAbility(string) bool { p.offers++; return p.accept }
func (p *ronEntryCopyPlayer) ChoosePermanent(c []*mage.Permanent, _ string, g mage.GameReader) *mage.Permanent {
	p.choices++
	if p.pick != nil {
		return p.pick(c, g)
	}
	return c[0]
}

func TestRONEntryCopyGameplay(t *testing.T) {
	g, a, b := ronGame()
	chooser := &ronEntryCopyPlayer{Player: a, accept: true}
	g.SetPlayerAt(0, chooser)
	source := g.PutOnBattlefield(repositoryCard(t, "FEM", "Icatian Javelineers"), b.PlayerID())
	source.GrantBaseAttr(core.Shroud)
	card := repositoryCard(t, "LEB", "Clone")
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid Clone cast succeeded")
	}
	if chooser.offers != 0 {
		t.Fatal("failed payment made an entry choice")
	}
	a.ManaPool().Add(core.Blue, 1)
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if chooser.offers != 0 {
		t.Fatal("copy choice at announcement")
	}
	pending := g.Clone()
	other := &ronEntryCopyPlayer{Player: pending.GetPlayer(a.PlayerID()), accept: false}
	pending.SetPlayerAt(0, other)
	g.ResolveStack()
	copied := g.FindPermanent(card.ID())
	if copied == nil || copied.Name() != "Icatian Javelineers" || copied.Counters[core.Javelin] != 1 || copied.HasAttr(core.Shroud) || chooser.offers != 1 || chooser.choices != 1 {
		t.Fatal("copy values, counters, nontarget choice or timing")
	}
	copied.RevokeBaseAttr(core.AttrSummonSick)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), copied.ID(), 1, nil); err == nil || copied.Tapped || copied.Counters[core.Javelin] != 1 {
		t.Fatal("copied cost rollback")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), copied.ID(), 1, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.PhaseIn(g.PhaseOut(copied.ID()))
	if chooser.offers != 1 {
		t.Fatal("phasing repeated copy choice")
	}
	g.BouncePermanentToHand(g.FindPermanent(card.ID()))
	if a.Hand()[0].Name() != "Clone" || g.LKI(card.ID()).Name != "Icatian Javelineers" {
		t.Fatal("physical departure or copied LKI")
	}
	g.ResolveStack()
	if b.Life() != 19 || source.Counters[core.Javelin] != 1 {
		t.Fatal("copied pending ability or independent binding")
	}
	pending.ResolveStack()
	if pending.FindPermanent(card.ID()) != nil || pending.GetPlayer(a.PlayerID()).Graveyard()[0].Name() != "Clone" || other.offers != 1 || other.choices != 0 {
		t.Fatal("declined copy or clone isolation")
	}
}

func TestRONEntryCopyInvalid(t *testing.T) {
	for _, rule := range []string{
		`AsPermanentEnters(ThisPermanent,[MayActions([EnterAsACopyOfAPermanent(IsCardtype(Creature))])])`,
		`AsPermanentEnters(ThisPermanent,[MayActions([EnterAsACopyOfAPermanent(IsCardtype(Creature),NoCopyEffects())])])`,
		`AsPermanentEnters(ThisPermanent,[MayActions([EnterAsACopyOfAPermanent(IsCardtype(Creature),CopyEffects([]))])])`,
		`AsPermanentEnters(ThisPermanent,[MayActions([EntersTapped,EnterAsACopyOfAPermanent(IsCardtype(Creature),NoCopyEffects)])])`,
		`AsPermanentEnters(ThisPermanent,[EnterAsACopyOfAPermanent(IsCardtype(Creature),NoCopyEffects)])`,
		`AsPermanentEnters(HostPermanent,[MayActions([EnterAsACopyOfAPermanent(IsCardtype(Creature),NoCopyEffects)])])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONEntryCopySources(t *testing.T) {
	for _, mode := range []string{"normal", "copy", "token", "facedown", "phase", "empty"} {
		t.Run(mode, func(t *testing.T) {
			g, a, b := ronGame()
			chooser := &ronEntryCopyPlayer{Player: a, accept: true}
			g.SetPlayerAt(0, chooser)
			source := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), b.PlayerID())
			switch mode {
			case "copy":
				source = g.PutOnBattlefield(repositoryCard(t, "LEB", "Clone"), a.PlayerID())
			case "token":
				hive := g.PutOnBattlefield(repositoryCard(t, "LEB", "The Hive"), b.PlayerID())
				b.ManaPool().Add(core.Colorless, 5)
				if err := g.ActivateAbilityByIndex(b.PlayerID(), hive.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				for _, p := range g.AllBattlefield() {
					if p.IsToken {
						source = p
					}
				}
			case "facedown":
				source.FaceDown = true
			case "phase":
				g.PhaseOut(source.ID())
			case "empty":
				g.BouncePermanentToHand(source)
			}
			chooser.offers = 0
			chooser.choices = 0
			chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return source }
			card := repositoryCard(t, "LEB", "Clone")
			p := g.PutOnBattlefield(card, a.PlayerID())
			if p == nil {
				t.Fatal("copy entry absent")
			}
			if mode == "phase" || mode == "empty" {
				if p.Name() != "Clone" || chooser.offers != 0 || chooser.choices != 0 {
					t.Fatal("absent source offered")
				}
				return
			}
			want := "Air Elemental"
			if mode == "token" {
				want = "Wasp"
			}
			if mode == "facedown" {
				want = ""
			}
			if p.Name() != want || p.IsToken || p.FaceDown || chooser.offers != 1 || chooser.choices != 1 {
				t.Fatal("copy source or physical status")
			}
			if mode == "facedown" && (p.CurrentPower(g) != 2 || p.Card.(*mage.BaseCard).HasManaCost() || p.HasAttr(core.Flying)) {
				t.Fatal("face-down values")
			}
			values, err := g.CaptureCopiableValues(p)
			if err != nil {
				t.Fatal(err)
			}
			rebuilt, err := values.Build()
			if err != nil || rebuilt.Name() != want {
				t.Fatal("copied provenance", err)
			}
		})
	}
}

func TestRONEntryCopyAcquiredChoiceAndObserver(t *testing.T) {
	g, a, b := ronGame()
	chooser := &ronEntryCopyPlayer{Player: a, accept: true}
	g.SetPlayerAt(0, chooser)
	nodes, err := Parse(strings.NewReader(`Card(Name:"Donor",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)]),TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([GainLife(Integer(2))]))])`), "donor.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	donor := factory()
	source := g.PutOnBattlefield(donor, b.PlayerID())
	g.ResolveStack()
	if source.ChosenPlayer != a.PlayerID() {
		t.Fatal("donor choice")
	}
	observer := mage.NewArtifact("Observer", "", mage.WithAbility(mage.WhenEvent(mage.CreatureYouControlEnters(), mage.GainLifeFor(mage.You, mage.Fixed(1)))))
	g.PutOnBattlefield(observer, a.PlayerID())
	card := repositoryCard(t, "LEB", "Clone")
	p := g.PutOnBattlefield(card, a.PlayerID())
	if p == nil || p.ChosenPlayer != b.PlayerID() {
		t.Fatal("copy retained donor choice")
	}
	g.ResolveStack()
	if a.Life() != 23 {
		t.Fatal("acquired ETB or external observer missed copy entry")
	}
}
