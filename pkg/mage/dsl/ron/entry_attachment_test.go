package ron

import (
	"maps"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func entryAttachmentProjection(t *testing.T, set string) Node {
	t.Helper()
	record := chosenDiscardRecord(t, set, "Animate Dead")
	rules := record.Fields["Rules"]
	trigger := rules.Args[1]
	trigger.Args = append([]Node(nil), trigger.Args...)
	body := trigger.Args[2]
	body.Args = []Node{{Kind: List, Args: append([]Node(nil), body.Args[0].Args[:3]...)}}
	trigger.Args[2] = body
	record.Fields = maps.Clone(record.Fields)
	record.Fields["Rules"] = Node{Kind: List, Args: []Node{rules.Args[0], trigger, rules.Args[2]}}
	return record
}

func TestRONEntryAttachmentAnimateDead(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			factory, err := Compile(entryAttachmentProjection(t, set))
			if err != nil {
				t.Fatal(err)
			}
			for _, mode := range []string{"normal", "shroud", "hexproof", "protection", "stale", "left", "phased", "cycle", "control", "copy", "blocked", "cast"} {
				t.Run(mode, func(t *testing.T) {
					g, a, b := ronGame()
					guest := mage.NewCreature("Guest", "", 3, 4)
					guest.SetOwner(b.PlayerID())
					if mode == "blocked" {
						guest.AddType(core.TypeLand)
						g.PutOnBattlefield(mage.NewArtifact("Entry rule", "", mage.WithAbility(mage.Static(mage.PreventLandEntry()))), a.PlayerID())
					}
					if mode == "shroud" {
						mage.WithKeyword(core.Shroud)(guest)
					}
					if mode == "hexproof" {
						mage.WithKeyword(core.Hexproof)(guest)
					}
					if mode == "protection" {
						mage.WithAbility(mage.ProtectionFromColors(core.Black))(guest)
					}
					b.AddToGraveyard(guest)
					card := factory()
					card.SetOwner(a.PlayerID())
					if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
						t.Fatal(err)
					}
					var aura *mage.Permanent
					if mode == "cast" {
						a.AddToHand(card)
						a.ManaPool().Add(core.Black, 1)
						a.ManaPool().Add(core.Colorless, 1)
						if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{guest.ID()}, 0); err != nil {
							t.Fatal(err)
						}
						g.ResolveTopOfStack()
						aura = g.FindPermanent(card.ID())
					} else {
						aura = g.PutOnBattlefield(card, a.PlayerID())
					}
					g.PutTriggersOnStack()
					if g.StackSize() != 1 {
						t.Fatal("entry did not queue")
					}
					switch mode {
					case "stale":
						moved, ok := g.MoveFromGraveyard(b.PlayerID(), guest.ID(), core.ZoneHand)
						if !ok {
							t.Fatal("move failed")
						}
						b.AddToGraveyard(moved)
					case "left":
						g.BouncePermanentToHand(aura)
					case "phased":
						g.PhaseOut(aura.ID())
					case "cycle":
						g.PhaseIn(g.PhaseOut(aura.ID()))
					case "control":
						g.AddControlEffect(mage.ControlEffectSpec{TargetID: aura.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					case "copy":
						if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
							t.Fatal("copy failed")
						}
					}
					clone := g.Clone()
					for _, game := range []*mage.Game{g, clone} {
						game.ResolveStack()
						p := game.FindPermanent(guest.ID())
						if mode == "stale" || mode == "left" || mode == "phased" || mode == "blocked" {
							if p != nil {
								t.Fatal("invalid visit returned")
							}
							continue
						}
						if p == nil || p.ControllerID() != a.PlayerID() || p.Card.Owner() != b.PlayerID() {
							t.Fatal("return lost controller or owner")
						}
						ap := game.FindPermanent(card.ID())
						if mode == "protection" {
							if ap != nil {
								t.Fatal("illegal unattached Aura survived")
							}
							continue
						}
						if ap == nil || ap.AttachedTo != p.ID() || p.CurrentPower(game) != 2 {
							t.Fatal("exact entrant attachment or host layer failed")
						}
					}
				})
			}
			if _, err := Compile(chosenDiscardRecord(t, set, "Animate Dead")); err == nil {
				t.Fatal("incomplete full record accepted")
			}
		})
	}
}

func TestRONEntryAttachmentInvalid(t *testing.T) {
	const attach = `AttachPermanentToPermanent(ThisPermanent,ThisPermanent)`
	for _, bad := range []string{`AttachPermanentToPermanent`, `AttachPermanentToPermanent()`, `AttachPermanentToPermanent(ThisPermanent)`, `AttachPermanentToPermanent(ThisPermanent,ThisPermanent,ThisPermanent)`, `AttachPermanentToPermanent(You,ThisPermanent)`, `AttachPermanentToPermanent(ThisPermanent,You)`, `AttachPermanentToPermanent(ThisPermanent,ThePermanentPutOnTheBattlefieldThisWay)`, `AttachPermanentToPermanent(ThisPermanent(),ThisPermanent)`} {
		_, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([`+bad+`]))])`))
		if err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	node := nameScopeNode(t, attach)
	for _, ctx := range []scope{{}, {enteringSource: true}, {resolving: true, drawReplacementBody: true}} {
		if _, err := compileExactAttachment(node, ctx); err == nil {
			t.Fatal("unsupported context accepted")
		}
	}
	for _, pair := range [][2]string{{"ThePermanentPutOnTheBattlefieldThisWay", "ThePermanentPutOnTheBattlefieldThisWay()"}, {"EnchantedGraveyardCard", "You"}} {
		bad := strings.Replace(`PutGraveyardCardOntoBattlefield(EnchantedGraveyardCard,[EntersUnderPlayersControl(You)]),AttachPermanentToPermanent(ThisPermanent,ThePermanentPutOnTheBattlefieldThisWay)`, pair[0], pair[1], 1)
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantGraveyardCard(IsCardtype(Creature),AnyPlayer),TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([`+bad+`]))])`)); err == nil {
			t.Fatal("invalid result accepted")
		}
	}
}

type entryAttachmentPlayer struct{ mage.Player }

func (p *entryAttachmentPlayer) ChooseCardFromGraveyard(cards []mage.Card, _ string, _ mage.GameReader) mage.Card {
	for _, card := range cards {
		if card.Name() == "Second" {
			return card
		}
	}
	return cards[0]
}

func TestRONEntryAttachmentInstances(t *testing.T) {
	factory, err := Compile(entryAttachmentProjection(t, "LEA"))
	if err != nil {
		t.Fatal(err)
	}
	g, base, b := ronGame()
	a := &entryAttachmentPlayer{Player: base}
	g.SetPlayerAt(0, a)
	first := mage.NewCreature("First", "", 3, 4)
	first.SetOwner(b.PlayerID())
	b.AddToGraveyard(first)
	one := factory()
	one.SetOwner(a.PlayerID())
	source1 := g.PutOnBattlefield(one, a.PlayerID())
	second := mage.NewCreature("Second", "", 5, 6)
	second.SetOwner(b.PlayerID())
	b.AddToGraveyard(second)
	two := factory()
	two.SetOwner(a.PlayerID())
	source2 := g.PutOnBattlefield(two, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("two entries did not queue")
	}
	clone := g.Clone()
	g.BouncePermanentToHand(source1)
	g.ResolveStack()
	if g.FindPermanent(first.ID()) != nil || g.FindPermanent(source2.ID()).AttachedTo != second.ID() {
		t.Fatal("pending entries shared a result")
	}
	clone.ResolveStack()
	if clone.FindPermanent(source1.ID()).AttachedTo != first.ID() || clone.FindPermanent(source2.ID()).AttachedTo != second.ID() {
		t.Fatal("clone lost separate entry results")
	}
}

func TestRONEntryAttachmentOrdinaryBody(t *testing.T) {
	factory, err := Compile(nameScopeNode(t, `Card(Name:"Mover",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([AttachPermanentToPermanent(ThisPermanent,Ref_TargetPermanent)])))])`))
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	old := g.PutOnBattlefield(mage.NewCreature("Old", "", 2, 4), a.PlayerID())
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 4, mage.WithKeyword(core.Hexproof)), b.PlayerID())
	card := factory()
	aura := g.PutOnBattlefield(card, a.PlayerID())
	g.Attach(aura.ID(), old.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), aura.ID(), 0, []uuid.UUID{host.ID()}); err == nil {
		t.Fatal("targeting ignored hexproof")
	}
	host.RevokeBaseAttr(core.Hexproof)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), aura.ID(), 0, []uuid.UUID{host.ID()}); err != nil {
		t.Fatal(err)
	}
	host.GrantBaseAttr(core.Shroud)
	g.ResolveStack()
	if g.FindPermanent(aura.ID()).AttachedTo != old.ID() {
		t.Fatal("illegal target resolved")
	}
	host.RevokeBaseAttr(core.Shroud)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), aura.ID(), 0, []uuid.UUID{host.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(aura.ID()).AttachedTo != host.ID() {
		t.Fatal("ordinary action did not attach")
	}
}

func TestRONEntryAttachmentFailedResult(t *testing.T) {
	record := entryAttachmentProjection(t, "LEA")
	rules := record.Fields["Rules"]
	parts := rules.Args[1].Args[2].Args[0].Args
	second := parts[1]
	second.Position.Column++
	rules.Args[1].Args[2].Args[0].Args = []Node{parts[0], parts[1], second, parts[2]}
	record.Fields["Rules"] = rules
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	guest := mage.NewCreature("Guest", "", 3, 4)
	guest.SetOwner(b.PlayerID())
	b.AddToGraveyard(guest)
	card := factory()
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(guest.ID()) == nil || g.FindPermanent(card.ID()) != nil {
		t.Fatal("failed producer reused an earlier entrant")
	}
}
