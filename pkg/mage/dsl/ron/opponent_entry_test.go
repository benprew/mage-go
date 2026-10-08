package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONOpponentEntryRealCards(t *testing.T) {
	for _, name := range []string{"Black Vise", "The Rack"} {
		for _, count := range []int{0, 2, 3, 4, 7} {
			t.Run(fmt.Sprintf("%s/%d", name, count), func(t *testing.T) {
				g, a, b := ronGame()
				set := "ATQ"
				if name == "Black Vise" {
					set = "LEB"
				}
				card := repositoryCard(t, set, name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
					t.Fatal("unpaid cast changed state")
				}
				a.ManaPool().Add(core.Colorless, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(card.ID()) != nil {
					t.Fatal("entry before spell resolution")
				}
				g.ResolveStack()
				if p := g.FindPermanent(card.ID()); p == nil || p.ChosenPlayer != b.PlayerID() || !p.HasType(core.TypeArtifact) || g.StackSize() != 0 {
					t.Fatal("entry choice must be set without a trigger")
				}
				queueRONUpkeep(g, a)
				if g.StackSize() != 0 {
					t.Fatal("unchosen upkeep triggered")
				}
				queueRONUpkeep(g, b)
				if g.StackSize() != 1 || b.Life() != 20 {
					t.Fatal("chosen upkeep timing")
				}
				fillRONHand(b, count)
				fillRONHand(a, 10)
				g.ResolveStack()
				damage := max(0, 3-count)
				if name == "Black Vise" {
					damage = max(0, count-4)
				}
				if b.Life() != 20-damage || a.Life() != 20 {
					t.Fatalf("resolution hand count: A=%d B=%d, damage=%d", a.Life(), b.Life(), damage)
				}
			})
		}
	}
}

func TestRONOpponentEntryIdentity(t *testing.T) {
	for _, name := range []string{"Black Vise", "The Rack"} {
		for _, change := range []string{"control-before", "control-after", "depart", "return", "phase-before", "phase-after", "phase-cycle", "ability-before", "ability-after", "copy", "prevent", "lifelink"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				set := "ATQ"
				if name == "Black Vise" {
					set = "LEB"
				}
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				if change == "lifelink" {
					g.MutablePermanent(p.ID()).GrantBaseAttr(core.Lifelink)
				}
				if change == "control-before" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				if change == "phase-before" {
					g.PhaseOut(p.ID())
				}
				if change == "phase-cycle" {
					g.PhaseIn(g.PhaseOut(p.ID()))
				}
				if change == "ability-before" {
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				queueRONUpkeep(g, b)
				if change == "phase-before" || change == "ability-before" {
					if g.StackSize() != 0 {
						t.Fatal("inactive source triggered")
					}
					return
				}
				if g.StackSize() != 1 {
					t.Fatal("choice changed with control or phasing")
				}
				switch change {
				case "control-after":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "depart", "return", "lifelink":
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					if change == "return" {
						a.RemoveFromHand(card.ID())
						returned := g.PutOnBattlefield(card, b.PlayerID())
						if returned.ChosenPlayer != a.PlayerID() {
							t.Fatal("returned source did not make a fresh entry choice")
						}
					}
				case "phase-after":
					g.PhaseOut(p.ID())
				case "ability-after":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				case "prevent":
					g.AddPreventionShield(b.PlayerID(), 1)
				}
				if name == "Black Vise" {
					fillRONHand(b, 7)
				}
				clone := g.Clone()
				g.ResolveStack()
				damage := 3
				if change == "copy" {
					damage = 6
				}
				if change == "prevent" {
					damage = 2
				}
				wantA := 20
				if change == "lifelink" {
					wantA = 23
				}
				if b.Life() != 20-damage || a.Life() != wantA || clone.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatalf("saved event identity, LKI, prevention or clone: A=%d B=%d", a.Life(), b.Life())
				}
				clone.ResolveStack()
				if clone.GetPlayer(b.PlayerID()).Life() != b.Life() {
					t.Fatal("clone lost pending event")
				}
			})
		}
	}
}

func TestRONOpponentEntryInvalid(t *testing.T) {
	choice := `AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)])`
	upkeep := `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(TheChosenPlayer)),ActionList([GainLife(Integer(1))]))`
	for _, rules := range []string{
		upkeep,
		choice + `,` + choice,
		`AsPermanentEnters(ThisPermanent,[ChooseAPlayer])`,
		`AsPermanentEnters(ThisPermanent,[ChooseAPlayer()])`,
		`AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent())])`,
		`AsPermanentEnters(ThisPermanent,[ChooseAPlayer(You)])`,
		`AsPermanentEnters(ThisPermanent,[ChooseAPlayer(AnyPlayer)])`,
		`AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent),EntersTapped])`,
		`AsPermanentEnters(ThisPermanent(),[ChooseAPlayer(Opponent)])`,
		choice + `,TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(TheChosenPlayer())),ActionList([GainLife(Integer(1))]))`,
		choice + `,TriggerA(AtTheBeginningOfAPlayersEndStep(SinglePlayer(TheChosenPlayer)),ActionList([GainLife(Integer(1))]))`,
		choice + `,Activated(PayMana([]),ActionList([PlayerAction(TheChosenPlayer,GainLife(Integer(1)))]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rules+`])`), "invalid-entry.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid entry declaration: %s", rules)
		}
	}
}

func TestRONOpponentEntryInstancesAndLoading(t *testing.T) {
	for _, name := range []string{"Black Vise", "The Rack"} {
		g, a, b := ronGame()
		set := "ATQ"
		if name == "Black Vise" {
			set = "LEB"
		}
		first := g.PutOnBattlefield(repositoryCard(t, set, name), a.PlayerID())
		second := g.PutOnBattlefield(repositoryCard(t, set, name), b.PlayerID())
		if first.ChosenPlayer != b.PlayerID() || second.ChosenPlayer != a.PlayerID() {
			t.Fatal("entry choices shared factory state")
		}
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
		g.MutablePermanent(first.ID()).ChosenPlayer = a.PlayerID()
		g.PutTriggersOnStack()
		if g.StackSize() != 2 {
			t.Fatal("pending events rechecked a changed entry choice")
		}
		if name == "Black Vise" {
			fillRONHand(a, 5)
			fillRONHand(b, 7)
		} else {
			fillRONHand(a, 2)
		}
		g.ResolveStack()
		if a.Life() != 19 || b.Life() != 17 {
			t.Fatal("independent event player or hand values were lost")
		}
	}
	files := fstest.MapFS{}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI", "ATQ"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), set)
		if err != nil {
			t.Fatal(err)
		}
		var records []string
		lines := strings.Split(string(data), "\n")
		for _, node := range nodes {
			if name := node.Fields["Name"].String; name == "Black Vise" || name == "The Rack" {
				records = append(records, lines[node.Position.Line-1])
			}
		}
		if len(records) > 0 {
			files[set+".ron"] = &fstest.MapFile{Data: []byte(strings.Join(records, "\n"))}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || registry == nil {
		t.Fatal("complete reprints failed strict loading", err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Unbound",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(TheChosenPlayer)),ActionList([GainLife(Integer(1))]))])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("unbound entry choice installed a partial collection")
	}
}

type ronOpponentEntryProbe struct {
	mage.BaseAbility
	check func(*mage.Game, *mage.Permanent)
}

func (a *ronOpponentEntryProbe) OnEnter(g *mage.Game, p *mage.Permanent) bool {
	a.check(g, p)
	return true
}

func TestRONOpponentEntryNativeAndCopy(t *testing.T) {
	for _, name := range []string{"Black Vise", "The Rack"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			set := "ATQ"
			if name == "Black Vise" {
				set = "LEB"
			}
			card := repositoryCard(t, set, name)
			checked := 0
			card.AddAbility(&ronOpponentEntryProbe{check: func(g *mage.Game, p *mage.Permanent) {
				checked++
				if p.ChosenPlayer != g.GetOpponent(p.ControllerID()).PlayerID() {
					t.Fatal("RON chose the opponent after other entry operations")
				}
				for _, live := range g.AllBattlefield() {
					if live.ID() == p.ID() {
						t.Fatal("RON choice ran after battlefield registration")
					}
				}
			}})
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.CopySpellOnStack(card.ID(), b.PlayerID(), false) == nil {
				t.Fatal("permanent spell copy failed")
			}
			g.ResolveStack()
			if checked != 2 {
				t.Fatal("copied declarations lost entry operation")
			}
			var copyID uuid.UUID
			for _, p := range g.AllBattlefield() {
				if p.Name() == name && p.ControllerID() == b.PlayerID() {
					copyID = p.ID()
					if !p.IsToken || p.ChosenPlayer != a.PlayerID() {
						t.Fatal("copy did not choose for its entry controller")
					}
				}
			}
			if copyID == uuid.Nil || g.FindPermanent(card.ID()).ChosenPlayer != b.PlayerID() {
				t.Fatal("copy or original choice missing")
			}
			if name == "Black Vise" {
				fillRONHand(a, 7)
				fillRONHand(b, 7)
			}
			g.RunStepWithPriority(core.Upkeep)
			if a.Life() != 17 || b.Life() != 20 {
				t.Fatal("native upkeep used the wrong chosen player")
			}
		})
	}
}

func TestRONOpponentEntryDeclarationScopeAndInventory(t *testing.T) {
	choice := `AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)])`
	trigger := `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(TheChosenPlayer)),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))`
	factory := permanentScopeFactory(t, "Artifact", trigger+`,`+choice)
	first, second := factory(), factory()
	if first.Abilities()[0] == second.Abilities()[0] || first.Abilities()[1] == second.Abilities()[1] {
		t.Fatal("factories shared choice or trigger ability identity")
	}
	if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{"scope.ron": &fstest.MapFile{Data: []byte(`Card(Name:"Scope",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + choice + `,` + trigger + `])`)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory lost accepted choice", err)
	}
	roles := map[string]string{"ChooseAPlayer": "entry_operation", "Opponent": "player_filter", "TheChosenPlayer": "player_reference"}
	for name, role := range roles {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role %s", name, role)
		}
	}
	for _, rules := range []string{
		`SpellActions(ActionList([AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)])]))`,
		choice + `,SpellActions(ActionList([GainLife(Integer(1))]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+rules+`])`), "nonpermanent.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("nonpermanent or resolving entry choice accepted")
		}
	}
}
