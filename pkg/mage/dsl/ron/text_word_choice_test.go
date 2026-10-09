package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"

	"github.com/benprew/mage-go/pkg/mage"
)

type ronTextWordTester struct {
	mage.Player
	calls int
}

func (p *ronTextWordTester) ChooseMode(options []string, _ string) int {
	p.calls++
	return len(options) - 1
}
func TestRONTextWordChoiceGameplay(t *testing.T) {
	for _, choice := range []string{"ChooseTwoColorWords", "ChooseTwoBasicLandTypes"} {
		g, base, _ := ronGame()
		a := &ronTextWordTester{Player: base}
		g.SetPlayerAt(0, a)
		card := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([`+choice+`,GainLife(Integer(1))]))`)()
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		if a.calls != 0 {
			t.Fatal("choice before resolution")
		}
		g.ResolveStack()
		if a.calls != 2 || a.Life() != 21 {
			t.Fatal("missing resolving pair choice")
		}
	}
}

func TestRONTextWordChoiceDeclarations(t *testing.T) {
	for _, body := range []string{`ChooseTwoColorWords()`, `ChooseTwoBasicLandTypes(Plains)`, `GainLife(TheFirstChosenColorWord)`, `GainLife(TheSecondChosenLandType)`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	for _, rule := range []string{`Activated(ChooseTwoColorWords,ActionList([GainLife(Integer(1))]))`, `AsPermanentEnters(ThisPermanent,[ChooseTwoBasicLandTypes])`, `PermanentLayerEffect(ThisPermanent,[ChooseTwoColorWords])`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		_, err := Compile(staticBaseRecord(t, "LEA", name))
		if err == nil || strings.Contains(err.Error(), "unsupported action (ChooseTwo") {
			t.Fatalf("wrong complete-card boundary for %s: %v", name, err)
		}
	}
}
func TestRONTextWordChoiceRealOperands(t *testing.T) {
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		record := staticBaseRecord(t, "LEA", name)
		rules := record.Fields["Rules"].Args
		branches := rules[0].Args[0].Args[0].Args
		for _, branch := range branches {
			parts := branch.Args[1].Args[0].Args
			build, plan, err := compileTextWordChoice(parts[0], scope{})
			if err != nil {
				t.Fatal(err)
			}
			operands := parts[1].Args[1].Args[0].Args
			bindings := instanceBindings{textWords: make(map[*textWordChoicePlan]textWordBindings)}
			other := instanceBindings{textWords: make(map[*textWordChoicePlan]textWordBindings)}
			effect := build(bindings)
			if bindings.textWords[plan] == other.textWords[plan] {
				t.Fatal("missing instance binding")
			}
			otherEffect := build(other)
			if effect == otherEffect || bindings.textWords[plan] == other.textWords[plan] {
				t.Fatal("shared declarations")
			}
			g, base, _ := ronGame()
			a := &ronTextWordTester{Player: base}
			g.SetPlayerAt(0, a)
			ctx := &mage.EffectContext{Game: g, Controller: a.PlayerID()}
			if plan.color {
				first, err := compileColorWordRef(operands[0], scope{colorWords: plan})
				if err != nil {
					t.Fatal(err)
				}
				second, err := compileColorWordRef(operands[1], scope{colorWords: plan})
				if err != nil {
					t.Fatal(err)
				}
				if _, ok := first(bindings).Resolve(ctx); ok {
					t.Fatal("read before producer")
				}
				if err := effect.Apply(ctx); err != nil {
					t.Fatal(err)
				}
				x, ok := first(bindings).Resolve(ctx)
				y, otherOK := second(bindings).Resolve(ctx)
				if !ok || !otherOK || x != core.Green || y != core.Red {
					t.Fatal("wrong real color operands")
				}
				if _, ok := first(other).Resolve(ctx); ok {
					t.Fatal("choice escaped instance")
				}
			} else {
				first, err := compileLandWordRef(operands[0], scope{landWords: plan})
				if err != nil {
					t.Fatal(err)
				}
				second, err := compileLandWordRef(operands[1], scope{landWords: plan})
				if err != nil {
					t.Fatal(err)
				}
				if err := effect.Apply(ctx); err != nil {
					t.Fatal(err)
				}
				x, ok := first(bindings).Resolve(ctx)
				y, otherOK := second(bindings).Resolve(ctx)
				if !ok || !otherOK || x != core.ForestLandType || y != core.MountainLandType {
					t.Fatal("wrong real land operands")
				}
				if _, ok := first(other).Resolve(ctx); ok {
					t.Fatal("choice escaped instance")
				}
			}
		}
	}
	for _, n := range []Node{{Kind: Symbol, Name: "TheFirstChosenColorWord"}, {Kind: Tuple, Name: "TheSecondChosenColorWord"}, {Kind: Symbol, Name: "TheFirstChosenLandType"}} {
		if _, err := compileColorWordRef(n, scope{}); err == nil {
			t.Fatal("unbound color word")
		}
		if _, err := compileLandWordRef(n, scope{}); err == nil {
			t.Fatal("unbound land word")
		}
	}
}
func TestRONTextWordChoiceCopyPaymentAndSource(t *testing.T) {
	for _, state := range []string{"departed", "phased"} {
		g, base, b := ronGame()
		a := &ronTextWordTester{Player: base}
		other := &ronTextWordTester{Player: b}
		g.SetPlayerAt(0, a)
		g.SetPlayerAt(1, other)
		factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),ActionList([ChooseTwoColorWords,PlayerActions(You,[ChooseTwoBasicLandTypes])]))`)
		p := g.PutOnBattlefield(factory(), a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || a.calls != 0 || g.StackSize() != 0 {
			t.Fatal("failed cost chose words")
		}
		a.ManaPool().Add(core.Colorless, 1)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
		if state == "departed" {
			g.BouncePermanentToHand(p)
		} else {
			g.PhaseOut(p.ID())
		}
		clone := g.Clone()
		ca := &ronTextWordTester{Player: clone.GetPlayer(a.PlayerID())}
		cb := &ronTextWordTester{Player: clone.GetPlayer(b.PlayerID())}
		clone.SetPlayerAt(0, ca)
		clone.SetPlayerAt(1, cb)
		g.ResolveStack()
		clone.ResolveStack()
		if a.calls != 4 || other.calls != 4 || ca.calls != 4 || cb.calls != 4 {
			t.Fatalf("copies/clones lost chooser: %d %d %d %d", a.calls, other.calls, ca.calls, cb.calls)
		}
	}
}

func TestRONTextWordChoiceTargetLegality(t *testing.T) {
	for _, illegal := range []bool{false, true} {
		g, base, b := ronGame()
		a := &ronTextWordTester{Player: base}
		other := &ronTextWordTester{Player: b}
		g.SetPlayerAt(0, a)
		g.SetPlayerAt(1, other)
		subject := g.PutOnBattlefield(mage.NewArtifact("subject", ""), b.PlayerID())
		card := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([PlayerActions(ControllerOfPermanent(Ref_TargetPermanent),[ChooseTwoColorWords])])))`)()
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{subject.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		if illegal {
			g.PhaseOut(subject.ID())
		}
		g.ResolveStack()
		want := 2
		if illegal {
			want = 0
		}
		if a.calls != 0 || other.calls != want {
			t.Fatalf("explicit player or legality: %d %d", a.calls, other.calls)
		}
	}
}

type ronRealWordTester struct {
	mage.Player
	mode, choices int
}

func (p *ronRealWordTester) ChooseMode(options []string, _ string) int {
	if len(options) == 2 {
		return p.mode
	}
	p.choices++
	return len(options) - 1
}
func TestRONTextWordChoiceRealGameplay(t *testing.T) {
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		for mode := range 2 {
			record := staticBaseRecord(t, "LEA", name)
			branches := record.Fields["Rules"].Args[0].Args[0].Args[0].Args
			for i := range branches {
				body := &branches[i].Args[1].Args[0]
				body.Args = body.Args[:1]
			}
			// The target union and text consumer need separate contracts.
			record.Fields["Rules"].Args[0].Args[0] = branches[mode]
			compiled, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			g, base, b := ronGame()
			a := &ronRealWordTester{Player: base, mode: mode}
			g.SetPlayerAt(0, a)
			subject := g.PutOnBattlefield(mage.NewArtifact("subject", ""), b.PlayerID())
			target := subject.ID()
			if mode == 0 {
				spell := mage.NewInstant("subject spell", "", mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1))))
				b.AddToHand(spell)
				if err := g.CastSpellByID(b.PlayerID(), spell.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				target = g.StackPeek().ID
			}
			card := compiled()
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
				t.Fatal(err)
			}
			if a.choices != 0 {
				t.Fatal("real choice ran during announcement")
			}
			g.ResolveStack()
			if a.choices != 2 {
				t.Fatalf("real %s mode %d made %d choices", name, mode, a.choices)
			}
		}
	}
}
