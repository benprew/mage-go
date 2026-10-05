package ron

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func numberedTargetVictim(name string) mage.Card {
	if name == "Dust to Dust" {
		return mage.NewArtifact("Victim", "")
	}
	return mage.NewCreature("Victim", "", 2, 3)
}

func TestRONNumberedTargetsRealCards(t *testing.T) {
	for _, name := range []string{"Ashes to Ashes", "Dust to Dust", "Word of Binding"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "DRK", name)
			if err := mage.ValidateCardDefinition(name, c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			p := g.PutOnBattlefield(numberedTargetVictim(name), a.PlayerID())
			q := g.PutOnBattlefield(numberedTargetVictim(name), b.PlayerID())
			r := g.PutOnBattlefield(numberedTargetVictim(name), b.PlayerID())
			color := core.Black
			if name == "Dust to Dust" {
				color = core.White
			}
			a.ManaPool().Add(color, 2)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{p.ID(), q.ID()}, 2); err == nil {
				t.Fatal("unpaid spell accepted")
			}
			if a.ManaPool().TotalMana() != 2 || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed payment changed state")
			}
			generic := 1
			if name == "Word of Binding" {
				generic = 2
			}
			a.ManaPool().Add(core.Colorless, generic)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{p.ID(), q.ID()}, 2); err != nil {
				t.Fatal(err)
			}
			if p.Tapped || g.FindPermanent(q.ID()) == nil || a.Life() != 20 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("effect ran before resolution or wrong payment")
			}
			clone := g.Clone()
			g.ResolveStack()
			if clone.FindPermanent(q.ID()) == nil || clone.FindPermanent(q.ID()).Tapped || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("resolution changed pending clone")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if game.FindPermanent(r.ID()) == nil || game.FindPermanent(r.ID()).Tapped {
					t.Fatal("unselected permanent affected")
				}
				for _, id := range []uuid.UUID{p.ID(), q.ID()} {
					if name == "Word of Binding" {
						if !game.FindPermanent(id).Tapped {
							t.Fatal("target not tapped")
						}
					} else if game.FindPermanent(id) != nil {
						t.Fatal("target not exiled")
					}
				}
				wantLife := 20
				if name == "Ashes to Ashes" {
					wantLife = 15
				}
				if game.GetPlayer(a.PlayerID()).Life() != wantLife || game.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("wrong damage recipient or amount")
				}
				if name != "Word of Binding" && len(game.GetExile()) != 2 {
					t.Fatal("wrong exile count")
				}
			}
		})
	}
}

func TestRONNumberedTargetsInvalid(t *testing.T) {
	for _, body := range []string{
		`Targeted([NumberTargetPermanents(Integer(-1),AnyPermanent)],ActionList([TapEachPermanent(Ref_TargetPermanents)]))`,
		`Targeted([NumberTargetPermanents(Integer(1))],ActionList([TapEachPermanent(Ref_TargetPermanents)]))`,
		`Targeted([NumberTargetPermanents(Integer(1),AnyPermanent,AnyPermanent)],ActionList([TapEachPermanent(Ref_TargetPermanents)]))`,
		`Targeted([NumberTargetPermanents(Plus(Integer(1),Integer(1)),AnyPermanent)],ActionList([TapEachPermanent(Ref_TargetPermanents)]))`,
		`Targeted([NumberTargetPermanents(ValueX(),AnyPermanent)],ActionList([TapEachPermanent(Ref_TargetPermanents)]))`,
		`Targeted([NumberTargetPermanents(Integer("2"),AnyPermanent)],ActionList([TapEachPermanent(Ref_TargetPermanents)]))`,
		`Targeted([NumberTargetPermanents(Integer(1),AnyPermanent)],ActionList([TapEachPermanent(Ref_TargetPermanent)]))`,
		`Targeted([TargetPermanent(AnyPermanent)],ActionList([Exile([EachPermanent(Ref_TargetPermanents)],[])]))`,
		`ActionList([TapEachPermanent(Ref_TargetPermanents)])`,
		`Targeted([NumberTargetPermanents(Integer(1),AnyPermanent)],ActionList([Exile([EachPermanent(Ref_TargetPermanents)],[Unknown])]))`,
		`Targeted([NumberTargetPermanents(Integer(1),AnyPermanent)],ActionList([Exile([EachPermanent(Ref_TargetPermanents())],[])]))`,
		`Targeted([NumberTargetPermanents(Integer(1),AnyPermanent)],ActionList([Exile([EachPermanent()],[])]))`,
		`Targeted([NumberTargetPermanents(Integer(1),AnyPermanent)],ActionList([CreateFutureTrigger(AtTheNextEndOfCombat,ActionList([Exile([EachPermanent(Ref_TargetPermanents)],[])]))]))`,
		`Targeted([NumberTargetPermanents(Integer(1),AnyPermanent)],ActionList([Exile([EachPermanent(Ref_TargetPermanents),Permanent(ThisPermanent)],[])]))`,
		`Modal_ChooseOne([Targeted([NumberTargetPermanents(Integer(1),AnyPermanent)],ActionList([TapEachPermanent(Ref_TargetPermanents)])),ActionList([TapEachPermanent(Ref_TargetPermanents)])])`,
		`Targeted([NumberTargetPermanents(Integer(1),AnyPermanent)],ActionList([CreateFutureTrigger(AtTheNextEndOfCombat,ActionList([TapEachPermanent(Ref_TargetPermanents)]))]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(`+body+`)])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONNumberedTargetsInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([NumberTargetPermanents(Integer(2),IsCardtype(Creature))],ActionList([Exile([EachPermanent(Ref_TargetPermanents)],[]),TapEachPermanent(Ref_TargetPermanents),UnknownAction])))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"Integer": "announcement_value", "IsCardtype": "permanent_filter", "Ref_TargetPermanents": "permanent_collection_reference"}
	for _, entry := range report.Signatures {
		if role, ok := want[entry.Name]; ok {
			if entry.Role != role {
				t.Fatalf("%s role=%s want=%s", entry.Name, entry.Role, role)
			}
			delete(want, entry.Name)
		}
	}
	if len(want) != 0 || report.Compiled != 0 || len(report.Issues) != 1 {
		t.Fatal("lost nested inventory")
	}
}

func TestRONNumberedTargetsLegality(t *testing.T) {
	for _, name := range []string{"Ashes to Ashes", "Dust to Dust", "Word of Binding"} {
		for _, change := range []string{"shroud", "all-illegal", "return", "phase", "phase-cycle", "control", "copy"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				c := repositoryCard(t, "DRK", name)
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
				victim := numberedTargetVictim(name)
				p := g.PutOnBattlefield(victim, b.PlayerID())
				q := g.PutOnBattlefield(numberedTargetVictim(name), a.PlayerID())
				a.ManaPool().Add(core.Black, 2)
				a.ManaPool().Add(core.White, 2)
				a.ManaPool().Add(core.Colorless, 2)
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{p.ID(), q.ID()}, 2); err != nil {
					t.Fatal(err)
				}
				switch change {
				case "shroud":
					p.GrantBaseAttr(core.Shroud)
				case "all-illegal":
					p.GrantBaseAttr(core.Shroud)
					q.GrantBaseAttr(core.Shroud)
				case "return":
					g.RemoveFromBattlefield(p)
					g.PutOnBattlefield(victim, b.PlayerID())
				case "phase":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: q.ID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				g.AddPreventionShield(a.PlayerID(), 2)
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					game.ResolveStack()
					if change == "phase" {
						game.PhaseIn([]uuid.UUID{p.ID()})
					}
					pLegal := change == "phase-cycle" || change == "control" || change == "copy"
					qLegal := change != "all-illegal"
					for _, target := range []struct {
						id    uuid.UUID
						legal bool
					}{{p.ID(), pLegal}, {q.ID(), qLegal}} {
						current := game.FindPermanent(target.id)
						if name == "Word of Binding" {
							if current == nil || current.Tapped != target.legal {
								t.Fatal("tap ignored target legality or incarnation")
							}
						} else if (current == nil) != target.legal {
							t.Fatal("exile ignored target legality or incarnation")
						}
					}
					wantA, wantB := 20, 20
					if name == "Ashes to Ashes" && change != "all-illegal" {
						wantA = 17
						if change == "copy" {
							wantB = 15
							wantA = 20
						}
					}
					if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
						t.Fatal("damage, prevention, copy controller or all-illegal resolution failed")
					}
				}
			})
		}
	}
}

func TestRONNumberedTargetsAnnouncement(t *testing.T) {
	for _, name := range []string{"Ashes to Ashes", "Dust to Dust", "Word of Binding"} {
		for _, bad := range []string{"empty", "few", "many", "duplicate", "wrong-type", "artifact-creature", "shroud", "phase"} {
			t.Run(name+"/"+bad, func(t *testing.T) {
				g, a, b := ronGame()
				c := repositoryCard(t, "DRK", name)
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
				p := g.PutOnBattlefield(numberedTargetVictim(name), a.PlayerID())
				q := g.PutOnBattlefield(numberedTargetVictim(name), b.PlayerID())
				ids := []uuid.UUID{p.ID(), q.ID()}
				switch bad {
				case "empty":
					ids = nil
				case "few":
					ids = ids[:1]
				case "many":
					r := g.PutOnBattlefield(numberedTargetVictim(name), b.PlayerID())
					ids = []uuid.UUID{p.ID(), q.ID(), r.ID()}
				case "duplicate":
					ids = []uuid.UUID{p.ID(), p.ID()}
				case "wrong-type":
					r := g.PutOnBattlefield(mage.NewLand("Land"), a.PlayerID())
					ids[0] = r.ID()
				case "artifact-creature":
					if name != "Ashes to Ashes" {
						return
					}
					r := g.PutOnBattlefield(mage.NewCreature("Artifact creature", "", 1, 1, mage.WithCardType(core.TypeArtifact)), a.PlayerID())
					ids[0] = r.ID()
				case "shroud":
					p.GrantBaseAttr(core.Shroud)
				case "phase":
					g.PhaseOut(p.ID())
				}
				a.ManaPool().Add(core.Black, 2)
				a.ManaPool().Add(core.White, 2)
				a.ManaPool().Add(core.Colorless, 2)
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), ids, 2); err == nil {
					t.Fatal("invalid target selection accepted")
				}
				if a.ManaPool().TotalMana() != 6 || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("target failure changed state")
				}
			})
		}
	}
	for _, x := range []int{0, 1, 3} {
		g, a, b := ronGame()
		c := repositoryCard(t, "DRK", "Word of Binding")
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		ids := make([]uuid.UUID, x)
		for i := range ids {
			ids[i] = g.PutOnBattlefield(numberedTargetVictim("Word of Binding"), b.PlayerID()).ID()
		}
		a.ManaPool().Add(core.Black, 2)
		a.ManaPool().Add(core.Colorless, x)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), ids, x); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 {
			t.Fatal("wrong X payment")
		}
		for _, id := range ids {
			if !g.FindPermanent(id).Tapped {
				t.Fatal("X target not tapped")
			}
		}
	}
}

func TestRONNumberedTargetsBatch(t *testing.T) {
	for _, name := range []string{"Dust to Dust", "Word of Binding"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefield(numberedTargetVictim(name), a.PlayerID())
			q := g.PutOnBattlefield(numberedTargetVictim(name), b.PlayerID())
			var events, intermediate int
			observer := mage.NewLand("Observer", mage.WithAbility(mage.NewTriggered(core.EvtTapped, false, mage.GainLifeFor(mage.You, mage.Fixed(1))).SetCondition(
				func(event *core.GameEvent, game mage.GameReader, _, _ uuid.UUID) bool {
					if event.SourceID == p.ID() || event.SourceID == q.ID() {
						events++
						if !game.FindPermanent(p.ID()).Tapped || !game.FindPermanent(q.ID()).Tapped {
							intermediate++
						}
					}
					return false
				})), mage.WithAbility(mage.NewStateTriggered(false, mage.Sacrifice(mage.This)).SetCondition(
				func(_ *core.GameEvent, game mage.GameReader, _, _ uuid.UUID) bool {
					return game.FindPermanent(p.ID()) == nil && game.FindPermanent(q.ID()) != nil
				})))
			watcher := g.PutOnBattlefield(observer, a.PlayerID())
			c := repositoryCard(t, "DRK", name)
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			a.ManaPool().Add(core.White, 2)
			a.ManaPool().Add(core.Black, 2)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{p.ID(), q.ID()}, 2); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(watcher.ID()) == nil || intermediate != 0 || (name == "Word of Binding" && events != 2) {
				t.Fatal("batch exposed an intermediate state")
			}
		})
	}
}

func TestRONNumberedTargetsContexts(t *testing.T) {
	for _, action := range []string{`TapEachPermanent(Ref_TargetPermanents)`, `Exile([EachPermanent(Ref_TargetPermanents)],[])`} {
		for _, context := range []string{"mode", "activation", "trigger"} {
			t.Run(context+"/"+action, func(t *testing.T) {
				body := `Targeted([NumberTargetPermanents(Integer(2),IsCardtype(Creature))],ActionList([` + action + `]))`
				rule, kind := `Activated(PayMana([]),`+body+`)`, "Artifact"
				if context == "mode" {
					rule, kind = `SpellActions(Modal_ChooseOne([`+body+`,ActionList([GainLife(Integer(1))])]))`, "Instant"
				}
				if context == "trigger" {
					rule = `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),` + body + `)`
				}
				nodes, err := Parse(strings.NewReader(`Card(Name:"Context",Typeline:(Supertypes:[],Cardtypes:[`+kind+`],Subtypes:[]),Rules:[`+rule+`])`), "context.ron")
				if err != nil {
					t.Fatal(err)
				}
				factory, err := Compile(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				p := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 2), a.PlayerID())
				q := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 2), b.PlayerID())
				card := factory()
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				switch context {
				case "mode":
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID(), q.ID()}, 0); err != nil {
						t.Fatal(err)
					}
				case "activation":
					source := g.PutOnBattlefield(card, a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID(), q.ID()}); err != nil {
						t.Fatal(err)
					}
					g.RemoveFromBattlefield(source)
				case "trigger":
					g.PutOnBattlefield(card, a.PlayerID())
					g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
					g.PutTriggersOnStack()
				}
				if g.StackSize() != 1 {
					t.Fatal("missing action")
				}
				g.ResolveStack()
				for _, id := range []uuid.UUID{p.ID(), q.ID()} {
					current := g.FindPermanent(id)
					if strings.HasPrefix(action, "Exile") {
						if current != nil {
							t.Fatal("target not exiled")
						}
					} else if current == nil || !current.Tapped {
						t.Fatal("target not tapped")
					}
				}
			})
		}
	}
}

func TestRONNumberedTargetsInstancesAndCopy(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"X tap",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[ManaCostX],Rules:[SpellActions(Targeted([NumberTargetPermanents(ValueX,IsCardtype(Creature))],ActionList([TapEachPermanent(Ref_TargetPermanents)])))])`), "x.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	a := &pluralColorPlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	ids := make([]uuid.UUID, 4)
	for i := range ids {
		ids[i] = g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 2), b.PlayerID()).ID()
	}
	for _, targets := range [][]uuid.UUID{ids[:1], ids[1:3]} {
		c := factory()
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		a.ManaPool().Add(core.Colorless, len(targets))
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), targets, len(targets)); err != nil {
			t.Fatal(err)
		}
	}
	a.targets = ids[2:4]
	cp := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true)
	if cp == nil || !slices.Equal(cp.Targets, a.targets) {
		t.Fatal("copy lost X bound or retargeting")
	}
	g.ResolveStackObject(cp)
	if g.FindPermanent(ids[0]).Tapped || g.FindPermanent(ids[1]).Tapped || !g.FindPermanent(ids[2]).Tapped || !g.FindPermanent(ids[3]).Tapped {
		t.Fatal("copy changed original target bindings")
	}
	g.ResolveStack()
	for _, id := range ids {
		if !g.FindPermanent(id).Tapped {
			t.Fatal("factory instances lost their own X or targets")
		}
	}
}

func TestRONNumberedTargetsFixedZero(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Zero",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([NumberTargetPermanents(Integer(0),AnyPermanent)],ActionList([Exile([EachPermanent(Ref_TargetPermanents)],[]),TapEachPermanent(Ref_TargetPermanents),GainLife(Integer(1))])))])`), "zero.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	p := g.PutOnBattlefield(mage.NewCreature("Untargeted", "", 2, 2), b.PlayerID())
	c := factory()
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 21 || g.FindPermanent(p.ID()) == nil || p.Tapped {
		t.Fatal("empty target collection stopped resolution or selected a permanent")
	}
}
