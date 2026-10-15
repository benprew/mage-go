package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONEntryLifeLossLich(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := staticBaseRecord(t, set, "Lich")
			if _, err := Compile(record); err != nil {
				t.Fatalf("complete Lich rejected: %v", err)
			}
			entry := record.Fields["Rules"].Args[0]
			factory := staticBaseSubrule(t, record, entry)
			g, a, b := ronGame()
			c := factory()
			c.SetOwner(b.PlayerID())
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			observed := 0
			g.PutOnBattlefield(mage.NewArtifact("Observer", "", mage.WithAbility(mage.NewTriggered(core.EvtZoneChange, false).SetCondition(func(event *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
				if event.SourceID == c.ID() && event.ToZone == core.ZoneBattlefield {
					observed++
					if a.Life() != 0 {
						t.Fatal("entry preceded life loss")
					}
				}
				return false
			}))), a.PlayerID())
			g.AddPreventionShield(a.PlayerID(), 100)
			if g.PutOnBattlefield(c, a.PlayerID()) == nil || observed != 1 || a.Life() != 0 || b.Life() != 20 {
				t.Fatal("source entry clause failed")
			}
		})
	}
}

func TestRONEntryLifeLossResolution(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `AsPermanentEnters(ThisPermanent,[LoseLife(Integer(3))]),TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([GainLife(LifeTotalOfPlayer(You))]))`)
	g, a, b := ronGame()
	c := factory()
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 {
		t.Fatal("announcement lost life")
	}
	a.SetLife(11)
	clone := g.Clone()
	clone.GetPlayer(a.PlayerID()).SetLife(7)
	g.ResolveStack()
	clone.ResolveStack()
	if a.Life() != 16 || clone.GetPlayer(a.PlayerID()).Life() != 8 || b.Life() != 20 {
		t.Fatal("entry or queued trigger timing")
	}
}

func TestRONEntryLifeLossInvalid(t *testing.T) {
	for _, rule := range []string{
		`AsPermanentEnters(ThisPermanent,[LoseLife])`,
		`AsPermanentEnters(ThisPermanent,[LoseLife()])`,
		`AsPermanentEnters(ThisPermanent,[LoseLife(Integer(1),Integer(2))])`,
		`AsPermanentEnters(ThisPermanent,[LoseLife(1)])`,
		`AsPermanentEnters(ThisPermanent,[LoseLife(Integer(You))])`,
		`AsPermanentEnters(ThisPermanent,[LoseLife(LifeTotalOfPlayer(You()))])`,
		`AsPermanentEnters(ThisPermanent,[LoseLife(LifeTotalOfPlayer(Opponent))])`,
		`AsPermanentEnters(ThisPermanent,[LoseLife(LifeTotalOfPlayer(Ref_TargetPlayer))])`,
		`AsPermanentEnters(ThisPermanent,[LoseLife(LifeTotalOfPlayer(LoseLife_ThatPlayer))])`,
		`AsPermanentEnters(ThisPermanent,[LoseLife(ValueX)])`,
		`AsPermanentEnters(ThisPermanent,[LoseLife(Integer(2)),EntersTapped])`,
		`AsPermanentEnters(HostPermanent,[LoseLife(Integer(2))])`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[` + rule + `])`
		nodes, err := Parse(strings.NewReader(src), "invalid-entry-loss.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONEntryLifeLossInventory(t *testing.T) {
	src := `Card(Name:"Loss",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[AsPermanentEnters(ThisPermanent,[LoseLife(LifeTotalOfPlayer(You))])])`
	r, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if r.Compiled != 1 {
		t.Fatalf("entry rejected: %+v", r.Issues)
	}
	want := map[string]string{"LoseLife": "entry_operation", "LifeTotalOfPlayer": "entry_value", "You": "player_reference"}
	for _, s := range r.Signatures {
		if role, ok := want[s.Name]; ok && s.Role != role {
			t.Fatalf("%s role %s", s.Name, s.Role)
		}
	}
}

func TestRONEntryLifeLossFixedAndLifecycle(t *testing.T) {
	for _, amount := range []int{-3, 0, 2, 25} {
		t.Run(fmt.Sprint(amount), func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", fmt.Sprintf(`AsPermanentEnters(ThisPermanent,[LoseLife(Integer(%d))])`, amount))
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			want := 20 - max(0, amount)
			if p == nil || a.Life() != 20 || b.Life() != want {
				t.Fatal("fixed loss or entry controller")
			}
			g.PhaseIn(g.PhaseOut(p.ID()))
			if b.Life() != want {
				t.Fatal("phasing repeated loss")
			}
			clone := g.Clone()
			g.BouncePermanentToHand(p)
			a.RemoveFromHand(card.ID())
			if g.PutOnBattlefield(card, a.PlayerID()) == nil || a.Life() != want || b.Life() != want || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("return controller or clone")
			}
			if g.PutOnBattlefield(factory(), b.PlayerID()) == nil || b.Life() != 20-2*max(0, amount) {
				t.Fatal("factory instances shared entry")
			}
		})
	}
	for _, kind := range []string{"Instant", "Sorcery"} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[` + kind + `],Subtypes:[]),Rules:[AsPermanentEnters(ThisPermanent,[LoseLife(Integer(2))])])`
		nodes, err := Parse(strings.NewReader(src), "invalid-entry-loss.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("nonpermanent entry accepted")
		}
	}
}
