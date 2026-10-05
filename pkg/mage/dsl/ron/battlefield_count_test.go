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

func TestRONBattlefieldCountKarma(t *testing.T) {
	for _, change := range []string{"none", "control", "return", "phase-before", "phase-after", "copy", "prevent", "lifelink", "member-control", "member-phase", "member-return"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEB", "Karma")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			memberCard := repositoryCard(t, "LEB", "Swamp")
			memberCard.SetOwner(a.PlayerID())
			member := g.PutOnBattlefield(memberCard, b.PlayerID())
			if change == "phase-before" {
				g.PhaseOut(source.ID())
			}
			if change == "lifelink" {
				source.GrantBaseAttr(core.Lifelink)
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			wantStack := 1
			if change == "phase-before" {
				wantStack = 0
			}
			if g.StackSize() != wantStack || b.Life() != 20 {
				t.Fatal("wrong trigger or evaluation timing")
			}
			g.PutOnBattlefield(deathHistoryCard(t, "LEB", "Swamp", b.PlayerID()), b.PlayerID())
			g.PutOnBattlefield(deathHistoryCard(t, "LEB", "Swamp", a.PlayerID()), a.PlayerID())
			switch change {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "phase-after":
				g.PhaseOut(source.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "prevent":
				g.AddPreventionShield(b.PlayerID(), 1)
			case "member-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: member.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "member-phase":
				g.PhaseOut(member.ID())
			case "member-return":
				g.BouncePermanentToHand(g.FindPermanent(member.ID()))
				a.RemoveFromHand(memberCard.ID())
				g.PutOnBattlefield(memberCard, b.PlayerID())
			}
			clone := g.Clone()
			clone.PutOnBattlefield(deathHistoryCard(t, "LEB", "Swamp", b.PlayerID()), b.PlayerID())
			clone.ResolveStack()
			if b.Life() != 20 {
				t.Fatal("clone changed live game")
			}
			g.ResolveStack()
			damage, extra := 2, 1
			switch change {
			case "phase-before":
				damage, extra = 0, 0
			case "copy":
				damage, extra = 4, 2
			case "prevent", "member-control", "member-phase":
				damage = 1
			}
			wantA := 20
			if change == "lifelink" {
				wantA += damage
			}
			if b.Life() != 20-damage || a.Life() != wantA || clone.GetPlayer(b.PlayerID()).Life() != 20-damage-extra {
				t.Fatalf("lives: live %d/%d, clone B %d", a.Life(), b.Life(), clone.GetPlayer(b.PlayerID()).Life())
			}
		})
	}
}

func TestRONBattlefieldCountJovialEvil(t *testing.T) {
	for _, change := range []string{"none", "phase", "phase-cycle", "return", "control", "copy", "prevent", "illegal-copy", "color", "token"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			memberCard := repositoryCard(t, "LEB", "Savannah Lions")
			memberCard.SetOwner(a.PlayerID())
			member := g.PutOnBattlefield(memberCard, b.PlayerID())
			g.PutOnBattlefield(deathHistoryCard(t, "LEB", "White Knight", a.PlayerID()), a.PlayerID())
			g.PutOnBattlefield(deathHistoryCard(t, "LEB", "White Ward", b.PlayerID()), b.PlayerID())
			g.PutOnBattlefield(deathHistoryCard(t, "LEB", "Llanowar Elves", b.PlayerID()), b.PlayerID())
			spell := repositoryCard(t, "LEG", "Jovial Evil")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{b.PlayerID()}, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed payment changed state")
			}
			a.ManaPool().Add(core.Black, 3)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{a.PlayerID()}, 0); err == nil {
				t.Fatal("own player target accepted")
			}
			if a.ManaPool().TotalMana() != 3 || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("illegal target spent mana")
			}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
			if b.Life() != 20 {
				t.Fatal("count applied at announcement")
			}
			g.PutOnBattlefield(deathHistoryCard(t, "LEB", "White Knight", b.PlayerID()), b.PlayerID())
			switch change {
			case "phase":
				g.PhaseOut(member.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(member.ID()))
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(member.ID()))
				a.RemoveFromHand(memberCard.ID())
				g.PutOnBattlefield(memberCard, b.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: member.ID(), TargetID: member.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "copy", "illegal-copy":
				if g.CopySpellOnStack(spell.ID(), b.PlayerID(), change == "copy") == nil {
					t.Fatal("copy failed")
				}
			case "prevent":
				g.AddPreventionShield(b.PlayerID(), 1)
			case "color":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.Blue)), member.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "token":
				if err := mage.ApplyEffect(g, mage.CreateTokenCopies(mage.You, mage.Fixed(1), mage.DefineToken("White token", mage.TokenPT(1, 1), mage.TokenTypes(core.TypeCreature), mage.TokenColors(core.White))), member.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			wantA, wantB := 20, 16
			if change == "phase" || change == "control" || change == "color" {
				wantB = 18
			}
			if change == "copy" {
				wantA = 18
			}
			if change == "prevent" {
				wantB = 17
			}
			if change == "token" {
				wantB = 14
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("lives = %d/%d, want %d/%d", a.Life(), b.Life(), wantA, wantB)
			}
		})
	}
}

func TestRONBattlefieldCountSequenceAndContexts(t *testing.T) {
	count := `TheNumberOfPermanentsOnTheBattlefield(And([IsCardtype(Artifact), And([ControlledByAPlayer(SinglePlayer(You)), AnyPermanent])]))`
	factory := handSizeFactory(t, `Activated(PayMana([]),ActionList([GainLife(`+count+`),SacrificePermanent(ThisPermanent),GainLife(Twice(`+count+`))]))`)
	g, a, b := ronGame()
	g.PutOnBattlefield(mage.NewArtifact("Other", ""), a.PlayerID())
	g.PutOnBattlefield(mage.NewArtifact("Foreign", ""), b.PlayerID())
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 24 || b.Life() != 20 || g.FindPermanent(source.ID()) != nil {
		t.Fatal("count did not update after the first instruction")
	}
	for _, rule := range []string{
		`SpellActions(ActionList([GainLife(Twice(TheNumberOfPermanentsOnTheBattlefield(IsCardtype(Creature))))]))`,
		`SpellActions(Modal_ChooseOne([ActionList([GainLife(Twice(TheNumberOfPermanentsOnTheBattlefield(AnyPermanent)))]),ActionList([GainLife(Integer(1))])]))`,
	} {
		c := permanentScopeFactory(t, "Instant", rule)()
		if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRONBattlefieldCountInvalid(t *testing.T) {
	for _, value := range []string{
		`TheNumberOfPermanentsOnTheBattlefield`, `TheNumberOfPermanentsOnTheBattlefield()`,
		`TheNumberOfPermanentsOnTheBattlefield(AnyPermanent,AnyPermanent)`,
		`TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer)`,
		`TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(You))`,
		`TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(SinglePlayer(ThisPermanent)))`,
		`TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(SinglePlayer(Trigger_ThatPlayer)))`,
		`TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer)))`,
		`TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(SinglePlayer(DealsDamage_ThatPlayer)))`,
		`TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(SinglePlayer(You())))`,
		`TheNumberOfPermanentsOnTheBattlefield(And([ControlledByAPlayer(SinglePlayer(You)),ControlledByAPlayer(SinglePlayer(You))]))`,
		`TheNumberOfPermanentsOnTheBattlefield(Or([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]))`,
		`TheNumberOfPermanentsOnTheBattlefield(And(AnyPermanent))`,
		`Twice`, `Twice()`, `Twice(Integer(1),Integer(2))`, `Twice(TheNumberOfCardsInPlayersHand(Ref_TargetPlayer))`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([GainLife(%s)]))])`, value)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
	for _, rules := range []string{
		`Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([]))),Activated(PayMana([]),ActionList([GainLife(TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer))))]))`,
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),ActionList([])),Activated(PayMana([]),ActionList([GainLife(TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(SinglePlayer(Trigger_ThatPlayer))))]))`,
		`SpellActions(ActionList([GainLife(TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(SinglePlayer(ControllerOfPermanent(ThisPermanent)))))]))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[` + rules + `])`
		if !strings.HasPrefix(rules, "SpellActions") {
			src = strings.Replace(src, "[Instant]", "[Artifact]", 1)
		}
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted source or sibling binding leak", rules)
		}
	}
}

func TestRONBattlefieldCountBindingsAndSignedValues(t *testing.T) {
	rule := `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([GainLife(TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer))))])))`
	factory := permanentScopeFactory(t, "Instant", rule)
	first, second := factory(), factory()
	action := first.Abilities()[0].(*mage.ActionDefinition)
	if action.Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("factory instances share target bindings")
	}
	if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
		t.Fatal(err)
	}
	invalid := mage.NewInstant("Missing target", "", mage.WithAbility(mage.NewAction(mage.ActionSpell, mage.WithEffects(action.Effects()...))))
	if mage.ValidateCardDefinition(invalid.Name(), invalid) == nil {
		t.Fatal("count concealed its target declaration")
	}
	g, a, b := ronGame()
	g.PutOnBattlefield(mage.NewArtifact("A", ""), a.PlayerID())
	for range 3 {
		g.PutOnBattlefield(mage.NewArtifact("B", ""), b.PlayerID())
	}
	castRONSpell(t, g, a, first, a.PlayerID())
	castRONSpell(t, g, a, second, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 24 || b.Life() != 20 {
		t.Fatal("independent counts used the wrong target")
	}
	for _, amount := range []string{`TheNumberOfPermanentsOnTheBattlefield(IsLandType(Swamp))`, `Twice(Minus(Integer(0),Integer(2)))`} {
		card := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([GainLife(`+amount+`)]))`)()
		castRONSpell(t, g, a, card)
		g.ResolveStack()
		if a.Life() != 24 {
			t.Fatal("empty count or signed multiplication changed life")
		}
	}
}

func TestRONBattlefieldCountInventory(t *testing.T) {
	value := `Twice(TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer))))`
	src := fmt.Sprintf(`Card(Name:"Count",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([GainLife(%s)])))])`, value)
	report, err := InspectFS(fstest.MapFS{"count.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory: %v, report: %+v", err, report)
	}
	for name, role := range map[string]string{
		"Twice": "value", "TheNumberOfPermanentsOnTheBattlefield": "value", "ControlledByAPlayer": "permanent_filter", "SinglePlayer": "player_filter", "Ref_TargetPlayer": "player_reference",
	} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Errorf("missing %s role %s", name, role)
		}
	}
}

func TestRONBattlefieldCountReprintsAndNativeUpkeep(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI", "LEG"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for record := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(record, `Name: "Karma"`) || strings.Contains(record, `Name: "Jovial Evil"`) {
				files[set+".ron"] = &fstest.MapFile{Data: []byte(record)}
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatalf("complete records: %v", err)
	}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, set, "Karma")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid Karma accepted")
			}
			a.ManaPool().Add(core.White, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Swamp"), a.PlayerID())
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Swamp"), b.PlayerID())
			g.RunStepWithPriority(core.Upkeep)
			if a.Life() != 19 || b.Life() != 20 {
				t.Fatal("controller upkeep")
			}
			g.SetActivePlayerIndex(1)
			g.RunStepWithPriority(core.Upkeep)
			if a.Life() != 19 || b.Life() != 19 {
				t.Fatal("opponent upkeep")
			}
		})
	}
}
