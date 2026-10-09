package mage

import "testing"

type cardNameReadProbe struct {
	ref  CardNameRef
	want string
	t    *testing.T
}

func (*cardNameReadProbe) Text() string                 { return "read saved name" }
func (*cardNameReadProbe) Properties() EffectProperties { return EffectProperties{} }
func (e *cardNameReadProbe) Apply(ctx *EffectContext) error {
	name, ok := e.ref.Resolve(ctx)
	if !ok || name != e.want {
		e.t.Fatalf("name=%q present=%v want=%q", name, ok, e.want)
	}
	return nil
}
func (e *cardNameReadProbe) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", children: []definitionEdge{{"name", e.ref}}}
}

func TestCardNameRefContract(t *testing.T) {
	base, b := NewBasePlayer("A"), NewBasePlayer("B")
	a := &cardNameChoiceTester{Player: base, name: "Ice"}
	g := NewGame(a, b)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	binding := BindCardName("chosen")
	ref := FromCardName(binding)
	if _, ok := ref.Resolve(ctx); ok {
		t.Fatal("unwritten name exists")
	}
	if err := ChooseCardName(You, OracleCardNameCatalog(), binding).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	a.name = "Fire"
	if err := ChooseCardName(You, OracleCardNameCatalog(), binding).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if name, ok := ref.Resolve(&EffectContext{Game: clone}); !ok || name != "Ice" {
		t.Fatal("clone shared name state")
	}
	if name, ok := ref.Resolve(ctx); !ok || name != "Fire" || a.calls != 2 {
		t.Fatal("read changed choice or lost name")
	}
	clone.resolution.Clear()
	if _, ok := ref.Resolve(&EffectContext{Game: clone}); ok {
		t.Fatal("read survived cleared frame")
	}
	if err := ChooseCardName(ControllerOf(This), OracleCardNameCatalog(), binding).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := ref.Resolve(ctx); ok {
		t.Fatal("absent chooser retained name")
	}
}

func TestCardNameRefDeclarations(t *testing.T) {
	binding := BindCardName("chosen")
	choice := ChooseCardName(You, OracleCardNameCatalog(), binding)
	read := &cardNameReadProbe{ref: FromCardName(binding), want: "Ice", t: t}
	for _, effect := range []Effect{read, Sequence(read, choice), &cardNameReadProbe{ref: FromCardName(nil), t: t}, &cardNameReadProbe{ref: FromCardName(BindCardName("")), t: t}} {
		card := NewInstant("Invalid", "", Spell(effect))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid name read accepted")
		}
	}
	card := NewInstant("Valid", "", Spell(Sequence(choice, read)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestCardNameRefFrames(t *testing.T) {
	base, b := NewBasePlayer("A"), NewBasePlayer("B")
	a := &cardNameChoiceTester{Player: base, name: "Ice"}
	g := NewGame(a, b)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	binding := BindCardName("chosen")
	ref := FromCardName(binding)
	choice := ChooseCardName(You, OracleCardNameCatalog(), binding)
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	a.name = "Fire"
	body := Sequence(choice, &cardNameReadProbe{ref: ref, want: "Fire", t: t})
	if err := ApplyEffect(g, body, ctx.SourceID, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if name, _ := ref.Resolve(ctx); name != "Ice" {
		t.Fatal("nested frame replaced outer name")
	}
	g.PutOnBattlefield(NewArtifact("one", ""), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("two", ""), b.PlayerID())
	if err := ForEach(MatchingPermanents(AnyPermanent), BindPermanent("member"), body).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if name, _ := ref.Resolve(ctx); name != "Ice" || a.calls != 4 {
		t.Fatal("loop read escaped member frame")
	}
}

func TestCardNameRefPendingResolution(t *testing.T) {
	for _, state := range []string{"departed", "phased"} {
		t.Run(state, func(t *testing.T) {
			base, b := NewBasePlayer("A"), NewBasePlayer("B")
			a := &cardNameChoiceTester{Player: base, name: "Ice"}
			g := NewGame(a, b)
			binding := BindCardName("chosen")
			body := Sequence(ChooseCardName(You, OracleCardNameCatalog(), binding), &cardNameReadProbe{ref: FromCardName(binding), want: "Ice", t: t})
			card := NewArtifact("source", "", WithAbility(Activate(Costs(), body)))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if state == "departed" {
				g.BouncePermanentToHand(p)
			} else {
				g.PhaseOut(p.ID())
			}
			if a.calls != 0 {
				t.Fatal("announcement chose name")
			}
			clone := g.Clone()
			cloneChooser := &cardNameChoiceTester{Player: clone.GetPlayer(a.PlayerID()), name: "Ice"}
			clone.SetPlayerAt(0, cloneChooser)
			g.ResolveStack()
			clone.ResolveStack()
			if a.calls != 1 || cloneChooser.calls != 1 {
				t.Fatalf("choices=%d clone=%d", a.calls, cloneChooser.calls)
			}
			if _, ok := FromCardName(binding).Resolve(&EffectContext{Game: g}); ok {
				t.Fatal("completed action retained name")
			}
		})
	}
}
