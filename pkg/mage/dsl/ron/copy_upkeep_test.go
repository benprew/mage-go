package ron

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func copyUpkeepFactory(t *testing.T, body string) mage.CardFactory {
	t.Helper()
	return copyColorFactory(t, `[ManaCostU]`, "", `KeepColor,AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),`+body+`)])`)
}

func TestRONCopyUpkeepGameplay(t *testing.T) {
	g, a, b := ronGame()
	chooser := &ronEntryCopyPlayer{Player: a, accept: true}
	g.SetPlayerAt(0, chooser)
	donor := g.PutOnBattlefield(textCopyFactory(t, "Donor", "")(), b.PlayerID())
	chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return donor }
	factory := copyUpkeepFactory(t, `ActionList([TapPermanent(ThisPermanent),GainLife(Integer(1))])`)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	if p == nil || p.Name() != donor.Name() {
		t.Fatal("copy failed")
	}
	queueRONUpkeep(g, b)
	if g.StackSize() != 0 {
		t.Fatal("donor controller supplied You")
	}
	queueRONUpkeep(g, a)
	if g.StackSize() != 1 || g.StackPeek().SourceID != p.ID() {
		t.Fatal("recipient source")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || g.FindPermanent(p.ID()).Tapped || clone.GetPlayer(a.PlayerID()).Life() != 21 || !clone.FindPermanent(p.ID()).Tapped {
		t.Fatal("pending clone isolation")
	}
	g.BouncePermanentToHand(p)
	g.ResolveStack()
	if a.Life() != 21 || donor.Tapped {
		t.Fatal("pending departure")
	}
	card := a.Hand()[0]
	a.RemoveFromHand(card.ID())
	chooser.accept = false
	g.PutOnBattlefield(card, a.PlayerID())
	queueRONUpkeep(g, a)
	if g.StackSize() != 0 {
		t.Fatal("returned visit kept copy grant")
	}
}

func TestRONCopyUpkeepInvalid(t *testing.T) {
	for _, addition := range []string{
		`AddAbility`, `AddAbility()`, `AddAbility([])`, `AddAbility([Flying])`,
		`AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),ActionList([GainLife(Integer(1))]))])`,
		`AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([TapPermanent(HostPermanent)]))])`,
		`AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([TapPermanent(Ref_TargetPermanent)]))])`,
		`AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([If(CostWasPaid,[GainLife(Integer(1))])]))])`,
		`AddAbilityVariable(ThisAbility)`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[AsPermanentEnters(ThisPermanent,[MayActions([EnterAsACopyOfAPermanent(IsCardtype(Creature),CopyEffects([` + addition + `]))])])])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid addition accepted", addition)
		}
	}
}

func TestRONCopyUpkeepTargets(t *testing.T) {
	for _, change := range []string{"none", "illegal", "returned", "phase", "phase cycle", "source leaves", "control", "copy", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			donor := g.PutOnBattlefield(textCopyFactory(t, "Donor", "")(), b.PlayerID())
			victim := g.PutOnBattlefield(textCopyFactory(t, "Victim", "")(), b.PlayerID())
			chooser := &copySpellTargetPlayer{ronEntryCopyPlayer: ronEntryCopyPlayer{Player: a, accept: true}, target: victim.ID()}
			chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return donor }
			g.SetPlayerAt(0, chooser)
			factory := copyUpkeepFactory(t, `Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([TapPermanent(Ref_TargetPermanent)]))`)
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			if p == nil || victim.Tapped || chooser.offers != 1 {
				t.Fatal("entry made a trigger choice")
			}
			queueRONUpkeep(g, a)
			if g.StackSize() != 1 || len(g.StackPeek().Targets) != 1 || g.StackPeek().Targets[0] != victim.ID() || chooser.offers != 1 {
				t.Fatal("target or optional timing")
			}
			switch change {
			case "illegal":
				victim.GrantBaseAttr(core.Shroud)
			case "returned":
				g.BouncePermanentToHand(victim)
				card := b.Hand()[0]
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "phase":
				g.PhaseOut(victim.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(victim.ID()))
			case "source leaves":
				g.BouncePermanentToHand(p)
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: donor.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("stack copy")
				}
			case "clone":
				clone := g.Clone()
				clone.ResolveStack()
				if !clone.FindPermanent(victim.ID()).Tapped || g.FindPermanent(victim.ID()).Tapped {
					t.Fatal("target clone isolation")
				}
			}
			g.ResolveStack()
			want := change != "illegal" && change != "returned" && change != "phase"
			if g.FindPermanentIncludingPhased(victim.ID()).Tapped != want {
				t.Fatal("wrong target visit or optional result")
			}
		})
	}
}

func TestRONCopyUpkeepLifecycle(t *testing.T) {
	for _, change := range []string{"later copy", "two additions", "control", "phase", "phase cycle", "removed", "departed"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			chooser := &ronEntryCopyPlayer{Player: a, accept: true}
			g.SetPlayerAt(0, chooser)
			donor := g.PutOnBattlefield(textCopyFactory(t, "Donor", "")(), b.PlayerID())
			chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return donor }
			factory := copyUpkeepFactory(t, `ActionList([GainLife(Integer(1))])`)
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			if p == nil {
				t.Fatal("entry")
			}
			player := mage.Player(a)
			want := 1
			switch change {
			case "later copy", "two additions":
				chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return p }
				var next *mage.Permanent
				if change == "later copy" {
					next = g.PutOnBattlefield(textCopyFactory(t, "Later", `AsPermanentEnters(ThisPermanent,[MayActions([EnterAsACopyOfAPermanent(IsCardtype(Creature),NoCopyEffects)])])`)(), a.PlayerID())
					want = 2
				} else {
					next = g.PutOnBattlefield(factory(), a.PlayerID())
					want = 3
				}
				if next == nil {
					t.Fatal("later copy")
				}
				if p.Card.Abilities()[0].AbilityID() == next.Card.Abilities()[0].AbilityID() {
					t.Fatal("shared copied rule")
				}
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: donor.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				queueRONUpkeep(g, a)
				if g.StackSize() != 0 {
					t.Fatal("stale controller")
				}
				player = b
			case "phase":
				g.PhaseOut(p.ID())
				want = 0
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "removed":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want = 0
			case "departed":
				g.BouncePermanentToHand(p)
				want = 0
			}
			queueRONUpkeep(g, player)
			if g.StackSize() != want {
				t.Fatalf("got %d triggers, want %d", g.StackSize(), want)
			}
			g.ResolveStack()
			if player.Life() != 20+want {
				t.Fatal("ability controller or multiplicity")
			}
		})
	}
}

func TestRONCopyUpkeepVesuvanClause(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
			if err != nil {
				t.Fatal(err)
			}
			nodes, err := Parse(strings.NewReader(string(data)), set)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, n := range nodes {
				if n.Fields["Name"].String != "Vesuvan Doppelganger" {
					continue
				}
				found = true
				if _, err := Compile(n); err != nil {
					t.Fatal("full record blocker", err)
				}
				rewrite := func(node Node) Node { return node }
				rewrite = func(node Node) Node {
					if node.Name == "MayAction" {
						return Node{Kind: Tuple, Name: "TapPermanent", Args: []Node{{Kind: Symbol, Name: "Ref_TargetPermanent"}}, Position: node.Position}
					}
					for i, child := range node.Args {
						node.Args[i] = rewrite(child)
					}
					for key, child := range node.Fields {
						node.Fields[key] = rewrite(child)
					}
					return node
				}
				n = rewrite(n)
				factory, err := Compile(n)
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				donor := g.PutOnBattlefield(textCopyFactory(t, "Donor", "")(), b.PlayerID())
				victim := g.PutOnBattlefield(textCopyFactory(t, "Victim", "")(), b.PlayerID())
				chooser := &copySpellTargetPlayer{ronEntryCopyPlayer: ronEntryCopyPlayer{Player: a, accept: true}, target: victim.ID()}
				chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return donor }
				g.SetPlayerAt(0, chooser)
				card := factory()
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || chooser.offers != 0 {
					t.Fatal("unpaid entry choice")
				}
				a.ManaPool().Add(core.Blue, 2)
				a.ManaPool().Add(core.Colorless, 3)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				if chooser.offers != 0 {
					t.Fatal("entry before resolution")
				}
				g.ResolveStack()
				p := g.FindPermanent(card.ID())
				if p == nil || !slices.Contains(p.Colors(), core.Blue) || p.Name() != "Donor" {
					t.Fatal("real entry clause")
				}
				queueRONUpkeep(g, a)
				if g.StackSize() != 1 || g.StackPeek().SourceID != p.ID() {
					t.Fatal("real grant clause")
				}
				g.ResolveStack()
				if !g.FindPermanent(victim.ID()).Tapped || g.FindPermanent(donor.ID()).Tapped {
					t.Fatal("real target binding")
				}
			}
			if !found {
				t.Fatal("missing record")
			}
		})
	}
}
