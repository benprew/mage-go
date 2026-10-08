package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type drawStepLifePlayer struct {
	*mage.BasePlayer
	accept bool
	offers int
}

func (p *drawStepLifePlayer) ChooseMayAbility(string) bool { p.offers++; return p.accept }

func TestRONDrawStepLifeFasting(t *testing.T) {
	for _, state := range []string{"accept", "decline", "phase", "depart", "control", "remove-ability", "clone"} {
		t.Run(state, func(t *testing.T) {
			a := &drawStepLifePlayer{BasePlayer: mage.NewBasePlayer("A"), accept: state != "decline"}
			b := &drawStepLifePlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			g.SetTurn(3)
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, "DRK", "Fasting")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			a.ManaPool().Add(core.White, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			if source == nil {
				t.Fatal("no source")
			}
			a.AddToLibrary(mage.NewLand("Draw"))
			b.AddToLibrary(mage.NewLand("Draw"))
			switch state {
			case "phase":
				g.PhaseOut(source.ID())
			case "depart":
				g.DestroyPermanent(source)
			case "remove-ability":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.SetActivePlayerIndex(1)
			}
			game := g
			if state == "clone" {
				game = g.Clone()
			}
			skip := state == "accept" || state == "control" || state == "clone"
			game.RunStepWithPriority(core.Upkeep)
			game.RunStepWithPriority(core.Draw)
			player := a.PlayerID()
			if state == "control" {
				player = b.PlayerID()
			}
			if skip {
				if len(game.GetPlayer(player).Hand()) != 0 || game.GetPlayer(player).Life() != 20 {
					t.Fatal("skipped step drew or gained early")
				}
				game.DestroyPermanent(game.FindPermanent(source.ID()))
			} else if len(game.GetPlayer(player).Hand()) != 1 {
				t.Fatal("normal draw missing")
			}
			game.RunStepWithPriority(core.PrecombatMain)
			want := 20
			if skip {
				want = 22
			}
			if game.GetPlayer(player).Life() != want {
				t.Fatal("life gain timing")
			}
			if state == "decline" && game.FindPermanent(source.ID()) != nil {
				t.Fatal("draw trigger did not destroy Fasting")
			}
			if state == "clone" && (a.Life() != 20 || g.FindPermanent(source.ID()) == nil) {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestRONDrawStepLifeInvalid(t *testing.T) {
	valid := `ReplaceWouldBeginDrawStep(PlayerWouldBeginTheirDrawStep(You),[MayCost(SkipThisDrawStep),If(CostWasPaid,[GainLife(Integer(2))])])`
	for _, rule := range []string{
		`ReplaceWouldBeginDrawStep`, `ReplaceWouldBeginDrawStep()`,
		strings.Replace(valid, "PlayerWouldBeginTheirDrawStep", "PlayerWouldDrawDuringTheirDrawStep", 1),
		strings.Replace(valid, "PlayerWouldBeginTheirDrawStep(You)", "PlayerWouldBeginTheirDrawStep()", 1),
		strings.Replace(valid, "MayCost(SkipThisDrawStep)", "MayCost()", 1),
		strings.Replace(valid, "If(CostWasPaid", "Unless(CostWasPaid", 1),
		strings.Replace(valid, "CostWasPaid", "CostWasPaid()", 1),
		strings.Replace(valid, "GainLife(Integer(2))", "DrawACard", 1),
		strings.Replace(valid, "GainLife(Integer(2))", "GainLife()", 1),
		strings.Replace(valid, "Integer(2)", "Integer()", 1),
		strings.Replace(valid, "Integer(2)", "Integer(2,3)", 1),
		strings.Replace(valid, "Integer(2)", "Integer(You)", 1),
		strings.Replace(valid, "You", "Opponent", 1), strings.Replace(valid, "You", "You()", 1),
		strings.Replace(valid, "SkipThisDrawStep", "SkipThisDrawStep()", 1),
		strings.Replace(valid, "MayCost", "PlayerMayCost", 1),
		strings.Replace(valid, "CostWasPaid", "TheLifePaid", 1),
		strings.Replace(valid, "Integer(2)", "Integer(-1)", 1),
		strings.Replace(valid, "Integer(2)", "X", 1),
		strings.Replace(valid, "GainLife(Integer(2))", "GainLife(Integer(2)),DrawACard", 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration", rule)
		}
	}
}

func TestRONDrawStepLifeFastingUpkeep(t *testing.T) {
	a := &drawStepLifePlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	g := mage.NewGame(a, mage.NewBasePlayer("B"))
	g.SetTurn(3)
	card := repositoryCard(t, "DRK", "Fasting")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	for turn := 3; turn <= 7; turn++ {
		g.SetTurn(turn)
		g.RunStepWithPriority(core.Upkeep)
		if turn < 7 {
			if g.FindPermanent(source.ID()) == nil || len(g.WaitingDrawStepLifePermanentSources()) != 1 {
				t.Fatal("early destruction")
			}
			g.RunStepWithPriority(core.Draw)
			if a.Life() != 20+(turn-3)*2 {
				t.Fatal("early life gain")
			}
			g.RunStepWithPriority(core.PrecombatMain)
		} else if g.FindPermanent(source.ID()) != nil || len(g.WaitingDrawStepLifePermanentSources()) != 0 || len(a.Graveyard()) != 1 {
			t.Fatal("fifth upkeep did not end replacement")
		}
	}
	if a.Life() != 28 || a.offers != 4 {
		t.Fatalf("life=%d offers=%d", a.Life(), a.offers)
	}
}

func TestRONDrawStepLifeProtection(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
	}{
		{"LEB", "Circle of Protection: Black", core.Black}, {"LEA", "Circle of Protection: Blue", core.Blue},
		{"LEA", "Circle of Protection: Green", core.Green}, {"LEA", "Circle of Protection: Red", core.Red},
		{"LEA", "Circle of Protection: White", core.White}, {"ATQ", "Circle of Protection: Artifacts", core.White},
		{"LEG", "Greater Realm of Preservation", core.Black},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := staticBaseRecord(t, tc.set, tc.name)
			actions := record.Fields["Rules"].Args[0].Args[1].Args[0].Args
			plan, err := compileChosenDamageSourceChoice(actions[0])
			if err != nil {
				t.Fatal(err)
			}
			shield, err := compileExactSourcePrevention(actions[1], scope{chosenDamageSource: plan})
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "DRK", "Fasting"), a.PlayerID())
			mods := []mage.ContinuousModification{mage.SetColors(mage.This, tc.color)}
			if tc.name == "Circle of Protection: Artifacts" {
				mods = append(mods, mage.AddCardType(mage.This, core.TypeArtifact))
			}
			if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mods...), source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			saved := g.CaptureDamageSource(source.ID())
			binding := mage.BindDamageSource("chosen")
			bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
			body := mage.Sequence(plan.waitingDrawStepLifeChoice(bindings), shield(bindings), mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(2)))
			if err := mage.ValidateCardDefinition("Consumer", mage.NewArtifact("Consumer", "", mage.WithAbility(mage.Activate(mage.Costs(), body)))); err != nil {
				t.Fatal(err)
			}
			for _, game := range []*mage.Game{g, g.Clone()} {
				if err := mage.ApplyEffect(game, body, uuid.Nil, b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if game.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("choice prevention failed")
				}
				game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 1, saved)
				if game.GetPlayer(b.PlayerID()).Life() != 19 {
					t.Fatal("shield did not consume")
				}
			}
			if _, err := Compile(record); err != nil {
				t.Fatal("complete source-choice record rejected", err)
			}
			if err := mage.ValidateCardDefinition("Invalid", mage.NewArtifact("Invalid", "", mage.WithAbility(mage.Activate(mage.Costs(), plan.waitingDrawStepLifeChoice(instanceBindings{}))))); err == nil {
				t.Fatal("missing binding accepted")
			}
		})
	}
}

func TestRONDrawStepLifeInstances(t *testing.T) {
	a := &drawStepLifePlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &drawStepLifePlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
	g := mage.NewGame(a, b)
	g.SetTurn(3)
	record := staticBaseRecord(t, "DRK", "Fasting")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	first, second := factory(), factory()
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Howling Mine"), b.PlayerID())
	a.AddToLibrary(mage.NewLand("Card"))
	b.AddToLibrary(mage.NewLand("Card"))
	if len(g.WaitingDrawStepLifePermanentSources()) != 2 {
		t.Fatal("instances shared source")
	}
	clone := g.Clone()
	g.RunStepWithPriority(core.Draw)
	if len(a.Hand()) != 0 || g.StackSize() != 0 || len(g.WaitingDrawStepLifePermanentSources()) != 2 {
		t.Fatal("skipped step detected Howling Mine or consumed static rule")
	}
	g.DestroyPermanent(g.FindPermanent(p.ID()))
	g.RunStepWithPriority(core.PrecombatMain)
	if a.Life() != 22 || b.Life() != 20 || len(g.WaitingDrawStepLifePermanentSources()) != 1 {
		t.Fatal("independent replacement lifetime")
	}
	clone.SetActivePlayerIndex(1)
	clone.RunStepWithPriority(core.Draw)
	clone.RunStepWithPriority(core.PrecombatMain)
	if clone.GetPlayer(b.PlayerID()).Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 20 || len(clone.WaitingDrawStepLifePermanentSources()) != 2 {
		t.Fatal("clone or player isolation")
	}
}

func TestRONDrawStepLifeFixedAmounts(t *testing.T) {
	for _, amount := range []string{"0", "3"} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Rule",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[ReplaceWouldBeginDrawStep(PlayerWouldBeginTheirDrawStep(You),[MayCost(SkipThisDrawStep),If(CostWasPaid,[GainLife(Integer(`+amount+`))])])])`), "amount.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		card := factory()
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		a := &drawStepLifePlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
		g := mage.NewGame(a, mage.NewBasePlayer("B"))
		g.SetTurn(3)
		g.PutOnBattlefield(card, a.PlayerID())
		g.RunStepWithPriority(core.Draw)
		g.RunStepWithPriority(core.PrecombatMain)
		want := 20
		if amount == "3" {
			want = 23
		}
		if a.Life() != want || a.offers != 1 {
			t.Fatal("fixed amount or optional zero skip")
		}
		nodes[0].Fields["Typeline"].Fields["Cardtypes"] = Node{Kind: List, Args: []Node{{Kind: Symbol, Name: "Instant"}}}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("replacement on instant")
		}
	}
}
