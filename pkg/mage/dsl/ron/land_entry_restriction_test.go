package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLandEntryRestrictionWorms(t *testing.T) {
	record := staticBaseRecord(t, "DRK", "Worms of the Earth")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"damage", "prevent", "sacrifice", "decline"} {
		for _, active := range []int{0, 1} {
			t.Run(state+string(rune('A'+active)), func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
				b := &ronAlternativePlayer{ronPaymentPlayer: &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: state != "decline"}, mode: 1}
				if state == "sacrifice" {
					b.mode = 0
				}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				for range 2 {
					g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
				}
				card := factory()
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(core.Black, 5)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				g.SetActivePlayerIndex(active)
				for _, p := range []mage.Player{a, b} {
					land := repositoryCard(t, "LEB", "Forest")
					p.AddToHand(land)
					if g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), p.PlayerID()) != nil {
						t.Fatal("land entered")
					}
					if err := g.PlayLand(p.PlayerID(), land.ID()); err == nil || len(p.Hand()) != 1 {
						t.Fatal("land play bypassed rule")
					}
				}
				if state == "prevent" {
					g.AddPreventionShield(b.PlayerID(), 5)
				}
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: g.ActivePlayerObj().PlayerID()})
				g.ResolveStack()
				wantLife := 20
				if state == "damage" {
					wantLife = 15
				}
				if b.Life() != wantLife || (g.FindPermanent(card.ID()) != nil) != (state == "decline") {
					t.Fatal("payment/destruction", b.Life())
				}
				if state == "sacrifice" && len(g.FilterBattlefield(mage.IsLand)) != 0 {
					t.Fatal("payment lands remain")
				}
				entered := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
				if (entered != nil) != (state != "decline") {
					t.Fatal("restriction lifetime")
				}
			})
		}
	}
}

func TestRONLandEntryRestrictionInvalid(t *testing.T) {
	for _, rule := range []string{`CardsCantEnterTheBattlefield`, `CardsCantEnterTheBattlefield()`, `CardsCantEnterTheBattlefield(IsCardtype(Land),IsCardtype(Land))`, `CardsCantEnterTheBattlefield(IsCardtype)`, `CardsCantEnterTheBattlefield(IsCardtype(Land()))`, `CardsCantEnterTheBattlefield(IsCardtype(Creature))`, `CardsCantEnterTheBattlefield(ThisPermanent)`, `CardsCantEnterTheBattlefield(And([IsCardtype(Land)]))`, `Activated(PayMana([]),ActionList([CardsCantEnterTheBattlefield(IsCardtype(Land))]))`} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[` + rule + `])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", rule)
		}
	}
	nodes, _ := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[CardsCantEnterTheBattlefield(IsCardtype(Land))])`), "instant.ron")
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("nonpermanent rule accepted")
	}
}

func TestRONLandEntryRestrictionAtomicLoad(t *testing.T) {
	valid := `Card(Name:"Entry rule",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[CardsCantEnterTheBattlefield(IsCardtype(Land))])`
	files := fstest.MapFS{"valid.ron": {Data: []byte(valid)}}
	if _, err := LoadFS(files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(valid, "IsCardtype(Land)", "Unbound", 1))}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 0 {
		t.Fatal("partial registry published")
	}
}

func TestRONLandEntryRestrictionRealEntryPaths(t *testing.T) {
	for _, path := range []string{"hand", "library", "clone"} {
		t.Run(path, func(t *testing.T) {
			a := &ronHandEntryPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			g := mage.NewGame(a, mage.NewBasePlayer("B"))
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			var touch *mage.Permanent
			if path == "hand" {
				touch = g.PutOnBattlefield(repositoryCard(t, "DRK", "Gaea's Touch"), a.PlayerID())
			}
			worms := repositoryCard(t, "DRK", "Worms of the Earth")
			worms.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(worms, a.PlayerID())
			land := repositoryCard(t, "LEB", "Forest")
			land.SetOwner(a.PlayerID())
			switch path {
			case "hand":
				a.AddToHand(land)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), touch.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if len(a.Hand()) != 1 || a.Hand()[0].ID() != land.ID() {
					t.Fatal("Gaea's Touch consumed blocked land")
				}
			case "library":
				a.SetLibrary([]mage.Card{land})
				spell := repositoryCard(t, "LEG", "Untamed Wilds")
				a.AddToHand(spell)
				a.ManaPool().Add(core.Green, 3)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if len(a.Library()) != 1 || a.Library()[0].ID() != land.ID() {
					t.Fatal("Untamed Wilds consumed blocked land")
				}
			case "clone":
				clone := g.Clone()
				clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
				if clone.PutOnBattlefield(land, a.PlayerID()) == nil || g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID()) != nil {
					t.Fatal("clone shared restriction state")
				}
			}
			if g.FindPermanent(land.ID()) != nil {
				t.Fatal("blocked entry")
			}
		})
	}
}
