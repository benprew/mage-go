package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronPlayerChoiceTester struct {
	mage.Player
	answer func([]uuid.UUID) []uuid.UUID
	calls  int
}

func (p *ronPlayerChoiceTester) ChooseTargets(candidates []uuid.UUID, _, _ int, _ *mage.Game) []uuid.UUID {
	p.calls++
	return p.answer(candidates)
}

func TestRONPlayerChoiceGameplay(t *testing.T) {
	for _, response := range []string{"valid", "nil", "invalid", "multiple"} {
		t.Run(response, func(t *testing.T) {
			g, base, b := ronGame()
			a := &ronPlayerChoiceTester{Player: base, answer: func(ids []uuid.UUID) []uuid.UUID {
				switch response {
				case "nil":
					return nil
				case "invalid":
					return []uuid.UUID{uuid.New()}
				case "multiple":
					return ids
				}
				return []uuid.UUID{b.PlayerID()}
			}}
			g.SetPlayerAt(0, a)
			factory := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([ChooseAPlayer(AnyPlayer),PlayerAction(TheChosenPlayer,GainLife(Integer(2)))]))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if a.calls != 0 {
				t.Fatal("choice occurred at announcement")
			}
			clone := g.Clone()
			clonePlayer := &ronPlayerChoiceTester{Player: clone.GetPlayer(a.PlayerID()), answer: func(_ []uuid.UUID) []uuid.UUID { return []uuid.UUID{a.PlayerID()} }}
			clone.SetPlayerAt(0, clonePlayer)
			g.ResolveStack()
			clone.ResolveStack()
			wantA, wantB := 22, 20
			if response == "valid" {
				wantA, wantB = 20, 22
			}
			if a.Life() != wantA || b.Life() != wantB || a.calls != 1 {
				t.Fatal("wrong chosen player")
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("pending clone shared choice")
			}
			if factory().Abilities()[0] == card.Abilities()[0] {
				t.Fatal("factory shared declaration")
			}
		})
	}
}

func TestRONPlayerChoiceSequenceAndDelay(t *testing.T) {
	g, a, b := ronGame()
	card := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([ChooseAPlayer(Opponent),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerAction(TheChosenPlayer,GainLife(Integer(3)))])),ChooseAPlayer(CastASpellThisTurn(IsCardtype(Sorcery))),PlayerAction(TheChosenPlayer,GainLife(Integer(9))),GainLife(Integer(1))]))`)()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("empty choice or delay execution failed")
	}
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if game.GetPlayer(b.PlayerID()).Life() != 23 || game.GetPlayer(a.PlayerID()).Life() != 21 {
			t.Fatal("later empty choice changed delayed capture")
		}
	}
}

func TestRONPlayerChoiceCastHistory(t *testing.T) {
	for _, state := range []string{"pending", "countered", "failed", "new-turn", "instant", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			g.SetActivePlayerIndex(1)
			spell := permanentScopeFactory(t, "Sorcery", `SpellActions(ActionList([GainLife(Integer(1))]))`)()
			if state == "instant" {
				spell = permanentScopeFactory(t, "Instant", `SpellActions(ActionList([GainLife(Integer(1))]))`)()
			}
			if state == "failed" {
				spell = mage.NewSorcery("Costly", "{9}")
			}
			spell.SetOwner(b.PlayerID())
			b.AddToHand(spell)
			err := g.CastSpellByID(b.PlayerID(), spell.ID(), nil, 0)
			if (err != nil) != (state == "failed") {
				t.Fatalf("cast=%v", err)
			}
			if state == "copy" && g.CopySpellOnStack(spell.ID(), a.PlayerID(), false) == nil {
				t.Fatal("spell copy failed")
			}
			if state == "countered" {
				g.CounterSpellOnStack(spell.ID())
			}
			if state == "new-turn" {
				g.ResolveStack()
				g.RunStepWithPriority(core.Cleanup)
			}
			g.SetActivePlayerIndex(0)
			g.SetStep(core.PrecombatMain)
			choice := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([ChooseAPlayer(CastASpellThisTurn(IsCardtype(Sorcery))),PlayerAction(TheChosenPlayer,GainLife(Integer(2)))]))`)()
			choice.SetOwner(a.PlayerID())
			a.AddToHand(choice)
			if err := g.CastSpellByID(a.PlayerID(), choice.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			before := b.Life()
			g.ResolveTopOfStack()
			want := before
			if state == "pending" || state == "countered" || state == "copy" {
				want += 2
			}
			if b.Life() != want {
				t.Fatalf("life=%d want=%d", b.Life(), want)
			}
		})
	}
}

func TestRONPlayerChoiceInvalid(t *testing.T) {
	for _, body := range []string{
		`ChooseAPlayer`, `ChooseAPlayer()`, `ChooseAPlayer(Opponent,AnyPlayer)`, `ChooseAPlayer(Opponent())`,
		`ChooseAPlayer(CastASpellThisTurn(IsCardtype(Creature)))`, `ChooseAPlayer(CastASpellThisTurn(IsCardtype(Sorcery),IsCardtype(Sorcery)))`,
		`PlayerAction(TheChosenPlayer,GainLife(Integer(1)))`,
		`MayAction(ChooseAPlayer(Opponent)),PlayerAction(TheChosenPlayer,GainLife(Integer(1)))`,
		`ChooseAPlayer(Opponent),PlayerAction(TheChosenPlayer(),GainLife(Integer(1)))`,
		`PlayerAction(You,ChooseAPlayer(Opponent)),PlayerAction(TheChosenPlayer,GainLife(Integer(1)))`,
	} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + body + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONPlayerChoiceCompleteRecords(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEG", "Backdraft"}} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), tc.set)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, node := range nodes {
			if node.Fields["Name"].String == tc.name {
				found = true
				factory, err := Compile(node)
				if err != nil {
					t.Fatalf("%s: %v", tc.name, err)
				}
				if err := mage.ValidateCardDefinition(tc.name, factory()); err != nil {
					t.Fatal(err)
				}
			}
		}
		if !found {
			t.Fatal("missing card")
		}
	}
}

func TestRONPlayerChoiceRecordPrerequisites(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEG", "Backdraft"}, {"FEM", "Rainbow Vale"}} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
			if err != nil {
				t.Fatal(err)
			}
			nodes, err := Parse(strings.NewReader(string(data)), tc.set)
			if err != nil {
				t.Fatal(err)
			}
			var choice Node
			var find func(Node)
			find = func(n Node) {
				if n.Name == "ChooseAPlayer" {
					choice = n
				}
				for _, child := range n.Args {
					find(child)
				}
			}
			for _, node := range nodes {
				if node.Fields["Name"].String == tc.name {
					find(node.Fields["Rules"])
				}
			}
			if choice.Name == "" {
				t.Fatal("missing real choice declaration")
			}
			observer, err := Parse(strings.NewReader(`PlayerAction(TheChosenPlayer,GainLife(Integer(2)))`), "observer")
			if err != nil {
				t.Fatal(err)
			}
			builds, err := compileActionSequence([]Node{choice, observer[0]}, scope{})
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			if tc.name == "Backdraft" {
				g.SetActivePlayerIndex(1)
				spell := repositoryCard(t, "LEB", "Hurricane")
				spell.SetOwner(b.PlayerID())
				b.AddToHand(spell)
				b.ManaPool().Add(core.Green, 1)
				if err := g.CastSpellByID(b.PlayerID(), spell.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
			}
			effects := make([]mage.Effect, 0, len(builds))
			for _, build := range builds {
				effects = append(effects, build(instanceBindings{}))
			}
			if err := mage.ApplyEffect(g, mage.Sequence(effects...), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 20 || b.Life() != 22 {
				t.Fatal("real record choice selected wrong player")
			}
		})
	}
}

func TestRONPlayerChoiceActivationAndExplicitChooser(t *testing.T) {
	g, base, other := ronGame()
	a := &ronPlayerChoiceTester{Player: base, answer: func(ids []uuid.UUID) []uuid.UUID { return ids[:1] }}
	b := &ronPlayerChoiceTester{Player: other, answer: func(ids []uuid.UUID) []uuid.UUID { return ids[1:] }}
	g.SetPlayerAt(0, a)
	g.SetPlayerAt(1, b)
	source := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),ActionList([ChooseAPlayer(AnyPlayer),PlayerAction(TheChosenPlayer,GainLife(Integer(2)))]))`)(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || a.calls != 0 || g.StackSize() != 0 {
		t.Fatal("failed payment made a choice")
	}
	a.ManaPool().Add(core.Colorless, 2)
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	if a.calls != 0 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("choice occurred during payment")
	}
	g.ResolveStack()
	if a.calls != 2 || a.Life() != 24 || b.Life() != 20 {
		t.Fatal("pending choices lost their results")
	}
	card := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([ChooseAPlayer(AnyPlayer),PlayerAction(Ref_TargetPlayer,ChooseAPlayer(AnyPlayer)),PlayerAction(TheChosenPlayer,GainLife(Integer(1)))])))`)()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.calls != 3 || b.calls != 1 || a.Life() != 25 || b.Life() != 20 {
		t.Fatal("explicit chooser changed the outer result")
	}
}

func TestRONPlayerChoiceLateCastAndIndependentBindings(t *testing.T) {
	g, base, b := ronGame()
	a := &ronPlayerChoiceTester{Player: base, answer: func(_ []uuid.UUID) []uuid.UUID { return []uuid.UUID{b.PlayerID()} }}
	g.SetPlayerAt(0, a)
	card := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([ChooseAPlayer(AnyPlayer),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerAction(TheChosenPlayer,GainLife(Integer(3)))])),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([ChooseAPlayer(CastASpellThisTurn(IsCardtype(Sorcery))),PlayerAction(TheChosenPlayer,GainLife(Integer(2)))]))]))`)()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	spell := permanentScopeFactory(t, "Sorcery", `SpellActions(ActionList([GainLife(Integer(1))]))`)()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 || a.calls != 1 {
		t.Fatal("choice did not read resolution-time history")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 23 {
		t.Fatal("later choice replaced the delayed binding")
	}
}

func TestRONPlayerChoiceSourceLifecycle(t *testing.T) {
	for _, state := range []string{"control", "phase", "return"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([ChooseAPlayer(Opponent),PlayerAction(TheChosenPlayer,GainLife(Integer(2)))]))]))`)(), a.PlayerID())
			source.Card.SetOwner(a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			switch state {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: uuid.New(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseOut(source.ID())
			case "return":
				g.BouncePermanentToHand(source)
				a.RemoveFromHand(source.ID())
				g.PutOnBattlefield(source.Card, b.PlayerID())
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if a.Life() != 20 || b.Life() != 22 {
				t.Fatal("source change replaced delayed chooser")
			}
		})
	}
}

func TestRONPlayerChoiceInventoryAndAbilityScope(t *testing.T) {
	src := `Card(Name:"Choice",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([ChooseAPlayer(CastASpellThisTurn(IsCardtype(Sorcery))),PlayerAction(TheChosenPlayer,GainLife(Integer(1)))]))])`
	report, err := InspectFS(fstest.MapFS{"choice.ron": &fstest.MapFile{Data: []byte(src)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("choice inventory rejected", err)
	}
	for _, signature := range report.Signatures {
		roles := map[string]string{"CastASpellThisTurn": "player_filter", "IsCardtype": "spell_filter", "TheChosenPlayer": "player_reference"}
		if want, ok := roles[signature.Name]; ok && signature.Role != want {
			t.Fatalf("%s role=%s", signature.Name, signature.Role)
		}
	}
	for _, src := range []string{
		`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([ChooseAPlayer(Opponent)])),Activated(PayMana([]),ActionList([PlayerAction(TheChosenPlayer,GainLife(Integer(1)))]))])`,
		`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)]),Activated(PayMana([]),ActionList([PlayerAction(TheChosenPlayer,GainLife(Integer(1)))]))])`,
	} {
		nodes, err := Parse(strings.NewReader(src), "invalid")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("choice escaped its ability or entry context")
		}
	}
}
