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

func TestRONGraveyardEventDingusEgg(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, change := range []string{"normal", "source-leave", "source-return", "source-control", "source-phase", "victim-return", "prevent"} {
			t.Run(set+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, set, "Dingus Egg")
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				land := repositoryCard(t, "LEB", "Forest")
				land.SetOwner(b.PlayerID())
				victim := g.PutOnBattlefield(land, a.PlayerID())
				g.DestroyPermanent(victim)
				if a.Life() != 20 || b.Life() != 20 || g.StackSize() != 0 {
					t.Fatal("damage did not wait for trigger placement")
				}
				switch change {
				case "source-leave", "source-return":
					g.DestroyPermanent(g.FindPermanent(source.ID()))
					if change == "source-return" {
						g.PutOnBattlefield(card, b.PlayerID())
					}
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "source-phase":
					g.PhaseOut(source.ID())
				case "victim-return":
					g.PutOnBattlefield(land, a.PlayerID())
				case "prevent":
					g.AddPreventionShield(a.PlayerID(), 1)
				}
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatalf("stack=%d", g.StackSize())
				}
				clone := g.Clone()
				g.ResolveStack()
				want := 18
				if change == "prevent" {
					want = 19
				}
				if a.Life() != want || b.Life() != 20 {
					t.Fatalf("damage used wrong last controller: %d/%d", a.Life(), b.Life())
				}
				if clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("resolution changed clone")
				}
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != want {
					t.Fatal("clone lost saved owner or source")
				}
			})
		}
	}
}

func TestRONGraveyardEventTablet(t *testing.T) {
	for _, change := range []string{"pay", "decline", "no-mana", "other-controller", "source-control-before", "source-control-after", "source-departed", "source-returned", "source-phased", "victim-phased", "ability-lost", "self", "batch"} {
		t.Run(change, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: change != "decline"}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			g.SetTurn(2)
			card := repositoryCard(t, "ATQ", "Tablet of Epityr")
			card.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			artifact := repositoryCard(t, "ATQ", "Ornithopter")
			artifact.SetOwner(b.PlayerID())
			controller := a.PlayerID()
			if change == "other-controller" {
				controller = b.PlayerID()
			}
			victim := g.PutOnBattlefield(artifact, controller)
			if change != "no-mana" {
				a.ManaPool().Add(core.Colorless, 3)
			}
			b.ManaPool().Add(core.Colorless, 3)
			want := 1
			switch change {
			case "source-control-before":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				want = 0
			case "source-phased":
				g.PhaseOut(source.ID())
				want = 0
			case "victim-phased":
				g.PhaseOut(victim.ID())
				want = 0
			case "ability-lost":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
				want = 0
			case "other-controller":
				want = 0
			}
			switch change {
			case "self":
				g.DestroyPermanent(source)
			case "batch":
				g.DestroyPermanents([]*mage.Permanent{source, victim, victim})
				want = 2
			default:
				if live := g.FindPermanent(victim.ID()); live != nil {
					g.DestroyPermanent(live)
				}
			}
			if change == "source-control-after" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if change == "source-departed" || change == "source-returned" {
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if change == "source-returned" {
					b.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			}
			g.PutTriggersOnStack()
			if g.StackSize() != want || a.offers != 0 || a.Life() != 20 {
				t.Fatalf("trigger timing/count: %d want %d", g.StackSize(), want)
			}
			g.ResolveStack()
			gains := want
			if change == "decline" || change == "no-mana" {
				gains = 0
			}
			if a.Life() != 20+gains || b.Life() != 20 || b.ManaPool().TotalMana() != 3 {
				t.Fatal("payment followed owner or new controller")
			}
			mana := 3 - gains
			if change == "no-mana" {
				mana = 0
			}
			if a.ManaPool().TotalMana() != mana {
				t.Fatal("wrong optional payment amount")
			}
		})
	}
}

func TestRONGraveyardEventBatchInstancesAndCopy(t *testing.T) {
	g, a, b := ronGame()
	for range 2 {
		g.PutOnBattlefield(repositoryCard(t, "LEB", "Dingus Egg"), a.PlayerID())
	}
	first := repositoryCard(t, "LEB", "Forest")
	first.SetOwner(a.PlayerID())
	second := repositoryCard(t, "LEB", "Island")
	second.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(first, b.PlayerID())
	q := g.PutOnBattlefield(second, a.PlayerID())
	g.DestroyPermanents([]*mage.Permanent{p, q, p})
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatalf("stack=%d want 4", g.StackSize())
	}
	top := g.StackPeek()
	if g.CopyStackObjectDirect(top, b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveTopOfStack()
	if a.Life()+b.Life() != 38 {
		t.Fatal("copied trigger lost owner")
	}
	g.ResolveStack()
	if a.Life()+b.Life() != 30 || (a.Life() != 14 && b.Life() != 14) {
		t.Fatalf("independent owners lost: %d/%d", a.Life(), b.Life())
	}
}

func TestRONGraveyardEventNonEvents(t *testing.T) {
	for _, change := range []string{"bounce", "exile", "discard", "phase", "nonland", "source-phased", "regenerate"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "LEB", "Dingus Egg"), a.PlayerID())
			card := repositoryCard(t, "LEB", "Forest")
			if change == "nonland" {
				card = repositoryCard(t, "ATQ", "Ornithopter")
			}
			card.SetOwner(b.PlayerID())
			if change == "discard" {
				b.AddToHand(card)
				g.PlayerDiscardByEffect(b, card.ID(), source.ID())
			} else {
				victim := g.PutOnBattlefield(card, b.PlayerID())
				switch change {
				case "bounce":
					g.BouncePermanentToHand(victim)
				case "exile":
					g.ExilePermanent(victim)
				case "phase":
					g.PhaseOut(victim.ID())
				case "source-phased":
					g.PhaseOut(source.ID())
					g.DestroyPermanent(victim)
				case "regenerate":
					g.AddRegenerationShield(victim.ID())
					g.DestroyPermanent(victim)
				default:
					g.DestroyPermanent(victim)
				}
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 0 || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("non-death or non-land event triggered")
			}
		})
	}
}

func TestRONGraveyardEventDeclarations(t *testing.T) {
	valid := `WhenAPermanentIsPutIntoAPlayersGraveyard(IsCardtype(Land), AnyPlayer)`
	for _, event := range []string{valid, `WhenAPermanentIsPutIntoAPlayersGraveyard(And([ControlledByAPlayer(SinglePlayer(You)), IsCardtype(Artifact)]), AnyPlayer)`} {
		c := graveyardEventDeclaration(t, event, `PlayerAction(Trigger_ThatPlayer, GainLife(Integer(1)))`)
		if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
			t.Fatal(err)
		}
	}
	for _, event := range []string{
		"WhenAPermanentIsPutIntoAPlayersGraveyard", "WhenAPermanentIsPutIntoAPlayersGraveyard()",
		`WhenAPermanentIsPutIntoAPlayersGraveyard(IsCardtype(Land))`,
		`WhenAPermanentIsPutIntoAPlayersGraveyard(IsCardtype(Land), SinglePlayer(You))`,
		`WhenAPermanentIsPutIntoAPlayersGraveyard(IsCardtype(Land), AnyPlayer())`,
		`WhenAPermanentIsPutIntoAPlayersGraveyard(IsCardtype(Instant), AnyPlayer)`,
		`WhenAPermanentIsPutIntoAPlayersGraveyard(And([]), AnyPlayer)`,
		`WhenAPermanentIsPutIntoAPlayersGraveyard(And([IsCardtype(Land), IsCardtype(Artifact)]), AnyPlayer)`,
		`WhenAPermanentIsPutIntoAPlayersGraveyard(And([IsCardtype(Land), ControlledByAPlayer(Opponent)]), AnyPlayer)`,
		`WhenAPermanentIsPutIntoAPlayersGraveyard(And([IsCardtype(Land), ControlledByAPlayer(SinglePlayer(You)), ControlledByAPlayer(SinglePlayer(You))]), AnyPlayer)`,
	} {
		text := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [TriggerA(%s, ActionList([GainLife(Integer(1))]))])`, event)
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid event accepted", event)
		}
	}
	for _, action := range []string{`PermanentDealsDamage(Trigger_ThatPermanent, Integer(1), Player(You))`, `PlayerAction(OwnerOfDeadPermanent, GainLife(Integer(1)))`} {
		text := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [TriggerA(%s, ActionList([%s]))])`, valid, action)
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("unbound event value accepted")
		}
	}
}

func graveyardEventDeclaration(t *testing.T, event, action string) mage.Card {
	t.Helper()
	text := fmt.Sprintf(`Card(Name: "Graveyard observer", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [TriggerA(%s, ActionList([%s]))])`, event, action)
	nodes, err := Parse(strings.NewReader(text), "observer.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	return factory()
}

func TestRONGraveyardEventNativeLoad(t *testing.T) {
	files := fstest.MapFS{}
	for _, tc := range []struct{ set, name string }{{"LEA", "Dingus Egg"}, {"LEB", "Dingus Egg"}, {"2ED", "Dingus Egg"}, {"3ED", "Dingus Egg"}, {"CED", "Dingus Egg"}, {"CEI", "Dingus Egg"}, {"ATQ", "Tablet of Epityr"}} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.HasPrefix(line, `Card(Name: "`+tc.name+`"`) {
				files[tc.set+".ron"] = &fstest.MapFile{Data: []byte(line)}
				break
			}
		}
	}
	if len(files) != 7 {
		t.Fatal("missing complete records")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("reprints were counted as distinct cards")
	}
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	for _, name := range []string{"Dingus Egg", "Tablet of Epityr"} {
		card, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
			t.Fatal("unpaid artifact cast succeeded")
		}
		if len(a.Hand()) != 1 || g.StackSize() != 0 {
			t.Fatal("failed casting changed state")
		}
		a.ManaPool().Add(core.Colorless, card.ManaCost().CMC())
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(card.ID()) != nil {
			t.Fatal("artifact entered before resolution")
		}
		g.ResolveStack()
	}
	a.ManaPool().Add(core.Colorless, 1)
	land := repositoryCard(t, "LEB", "Forest")
	land.SetOwner(b.PlayerID())
	artifact := repositoryCard(t, "ATQ", "Ornithopter")
	artifact.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(land, a.PlayerID())
	q := g.PutOnBattlefield(artifact, a.PlayerID())
	g.DestroyPermanents([]*mage.Permanent{p, q})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 20 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("loaded cards did not execute their full rules")
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [TriggerA(WhenAPermanentIsPutIntoAPlayersGraveyard(IsCardtype(Land), SinglePlayer(You)), ActionList([GainLife(Integer(1))]))])`)}
	if loaded, err := LoadFS(files, "*.ron"); err == nil || loaded != nil {
		t.Fatal("invalid collection published a registry")
	}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("failed loading changed existing registry")
	}
}

func TestRONGraveyardEventDeathKindsAndLKI(t *testing.T) {
	for _, change := range []string{"destroy", "sacrifice", "graveyard", "token", "lost-type", "gained-type", "phase-cycle", "simultaneous-source"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "LEB", "Dingus Egg"), a.PlayerID())
			card := mage.NewLand("Land fixture")
			if change == "gained-type" {
				card = mage.NewArtifact("Artifact fixture", "")
			}
			if change == "token" {
				card = mage.NewToken("Land token", 0, 0, []core.CardType{core.TypeLand}, nil)
			}
			card.SetOwner(b.PlayerID())
			victim := g.PutOnBattlefield(card, a.PlayerID())
			if change == "lost-type" {
				g.AddContinuousEffect(mage.FuncContinuousEffect(core.LayerType, core.EndOfTurn, func(game *mage.Game, _ uuid.UUID) error {
					game.GetEffects().RevokeAttr(victim.ID(), core.AttrIsLand)
					return nil
				}))
				g.ApplyContinuousEffects()
				if g.FindPermanent(victim.ID()).HasType(core.TypeLand) {
					t.Fatal("fixture did not remove land type")
				}
			}
			if change == "gained-type" {
				if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mage.AddCardType(mage.This, core.TypeLand)), victim.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if change == "phase-cycle" {
				g.PhaseIn(g.PhaseOut(victim.ID()))
			}
			switch change {
			case "sacrifice":
				g.DoSacrifice(g.FindPermanent(victim.ID()))
			case "graveyard":
				g.PutPermanentIntoGraveyard(g.FindPermanent(victim.ID()))
			case "simultaneous-source":
				g.DestroyPermanents([]*mage.Permanent{source, victim})
			default:
				g.DestroyPermanent(g.FindPermanent(victim.ID()))
			}
			g.PutTriggersOnStack()
			want := 1
			if change == "lost-type" {
				want = 0
			}
			if g.StackSize() != want {
				t.Fatalf("stack=%d want %d", g.StackSize(), want)
			}
			g.ResolveStack()
			if a.Life() != 20-2*want || b.Life() != 20 {
				t.Fatal("death used printed type or current controller")
			}
		})
	}
}

func TestRONGraveyardEventInventoryAndScopes(t *testing.T) {
	text := `Card(Name: "Observer", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [TriggerA(WhenAPermanentIsPutIntoAPlayersGraveyard(And([IsCardtype(Artifact), ControlledByAPlayer(SinglePlayer(You))]), AnyPlayer), ActionList([PlayerAction(Trigger_ThatPlayer, GainLife(Integer(1)))]))])`
	files := fstest.MapFS{"observer.ron": {Data: []byte(text)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	var typeSeen, playerSeen, ownerSeen bool
	for _, s := range report.Signatures {
		if s.Name == "IsCardtype" && s.Role == "permanent_filter" {
			typeSeen = true
		}
		if s.Name == "AnyPlayer" && s.Role == "player_filter" {
			playerSeen = true
		}
		if s.Name == "Trigger_ThatPlayer" && s.Role == "player_reference" {
			ownerSeen = true
		}
	}
	if report.Compiled != 1 || !typeSeen || !playerSeen || !ownerSeen {
		t.Fatal("inventory lost graveyard event roles")
	}
	for _, rule := range []string{
		`SpellActions(ActionList([PlayerAction(Trigger_ThatPlayer, GainLife(Integer(1)))]))`,
		`TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), ActionList([PlayerAction(Trigger_ThatPlayer, GainLife(Integer(1)))]))`,
		`TriggerI(WhenAPermanentIsPutIntoAPlayersGraveyard(IsCardtype(Artifact), AnyPlayer), PermanentPutInGraveyardPassesFilter(WasntSacrificed()), ActionList([GainLife(Integer(1))]))`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("out-of-scope player or unsupported intervening-if accepted")
		}
	}
}

func TestRONGraveyardEventOwnerControllerAndAPNAP(t *testing.T) {
	g, a, b := ronGame()
	event := `WhenAPermanentIsPutIntoAPlayersGraveyard(IsCardtype(Land), AnyPlayer)`
	card := graveyardEventDeclaration(t, event, `PlayerAction(Trigger_ThatPlayer, GainLife(Integer(1))), PlayerAction(ControllerOfDeadPermanent, GainLife(Integer(2)))`)
	g.PutOnBattlefield(card, a.PlayerID())
	land := repositoryCard(t, "LEB", "Forest")
	land.SetOwner(b.PlayerID())
	victim := g.PutOnBattlefield(land, a.PlayerID())
	g.DestroyPermanent(victim)
	g.PutOnBattlefield(land, b.PlayerID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 21 {
		t.Fatal("event owner and last controller were conflated or followed return")
	}
	g, a, b = ronGame()
	for _, controller := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
		g.PutOnBattlefield(repositoryCard(t, "LEB", "Dingus Egg"), controller)
	}
	land = repositoryCard(t, "LEB", "Forest")
	land.SetOwner(a.PlayerID())
	victim = g.PutOnBattlefield(land, b.PlayerID())
	g.DestroyPermanent(victim)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 || g.StackPeek().Controller != b.PlayerID() {
		t.Fatal("graveyard triggers lost APNAP order")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 16 {
		t.Fatal("trigger controller replaced captured land controller")
	}
}
