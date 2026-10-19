package ron

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronChainPlayer struct {
	*ronPaymentPlayer
	target      uuid.UUID
	targetCalls int
	declineCopy bool
	onPayment   func()
}

func (p *ronChainPlayer) ChooseMayAbility(label string) bool {
	p.offers++
	if p.offers == 1 && p.onPayment != nil {
		p.onPayment()
	}
	return p.accept && (!p.declineCopy || p.offers <= 1)
}

func (p *ronChainPlayer) ChooseTargets(_ []uuid.UUID, _, _ int, _ *mage.Game) []uuid.UUID {
	p.targetCalls++
	if p.target == uuid.Nil {
		return nil
	}
	return []uuid.UUID{p.target}
}

func TestRONRecipientSpellCopyChain(t *testing.T) {
	for _, state := range []string{"paid", "declined", "short", "keep", "creature", "control", "phase", "return", "prevention", "clone", "chain", "counter-copy", "paid-no-copy", "payment-control", "redirect"} {
		t.Run(state, func(t *testing.T) {
			factory, err := Compile(staticBaseRecord(t, "LEG", "Chain Lightning"))
			if err != nil {
				t.Fatal(err)
			}
			if err = mage.ValidateCardDefinition("Chain Lightning", factory()); err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			chooser := &ronChainPlayer{ronPaymentPlayer: &ronPaymentPlayer{BasePlayer: b, accept: state != "declined"}, target: a.PlayerID(), declineCopy: state == "paid-no-copy"}
			g.SetPlayerAt(1, chooser)
			g.SetStep(core.PrecombatMain)
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			victim := b.PlayerID()
			var p *mage.Permanent
			if state == "creature" || state == "control" || state == "phase" || state == "return" || state == "payment-control" || state == "redirect" {
				victimCard := mage.NewCreature("Victim", "", 1, 2)
				if state == "payment-control" {
					victimCard = mage.NewCreature("Victim", "", 1, 10)
				}
				victimCard.SetOwner(b.PlayerID())
				p = g.PutOnBattlefield(victimCard, b.PlayerID())
				victim = p.ID()
			}
			if g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim}, 0) == nil {
				t.Fatal("unpaid cast")
			}
			a.ManaPool().Add(core.Red, 1)
			if err = g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim}, 0); err != nil {
				t.Fatal(err)
			}
			mana := 2
			if state == "short" {
				mana = 1
			}
			b.ManaPool().Add(core.Red, mana)
			if chooser.offers != 0 || chooser.targetCalls != 0 {
				t.Fatal("early resolution choice")
			}
			switch state {
			case "redirect":
				g.SetCreatureDamageRedirect(p.ID(), a.PlayerID())
			case "payment-control":
				chooser.onPayment = func() {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				}
			case "keep":
				chooser.target = uuid.Nil
			case "phase":
				g.PhaseOut(p.ID())
			case "return":
				g.DestroyPermanent(p)
				b.RemoveFromGraveyard(p.Card.ID())
				g.PutOnBattlefield(p.Card, b.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "prevention":
				if err = mage.ApplyEffect(g, mage.PreventNextDamage(mage.You, mage.Fixed(3)), uuid.New(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if state == "chain" {
				payA := &ronChainPlayer{ronPaymentPlayer: &ronPaymentPlayer{BasePlayer: a, accept: true}, target: b.PlayerID()}
				g.SetPlayerAt(0, payA)
				a.ManaPool().Add(core.Red, 2)
			}
			var clone *mage.Game
			if state == "clone" {
				clone = g.Clone()
			}
			g.ResolveTopOfStack()
			copied := state != "declined" && state != "short" && state != "phase" && state != "return" && state != "control" && state != "paid-no-copy"
			if (g.StackSize() == 1) != copied {
				t.Fatalf("copy=%t stack=%d offers=%d", copied, g.StackSize(), chooser.offers)
			}
			if (state == "short" || state == "declined") && b.ManaPool().TotalMana() != mana {
				t.Fatal("failed payment spent mana")
			}
			if (state == "phase" || state == "return") && chooser.offers != 0 {
				t.Fatal("illegal target offered payment")
			}
			if state == "paid-no-copy" && (b.ManaPool().TotalMana() != 0 || chooser.offers != 2) {
				t.Fatal("paid optional copy")
			}
			if copied {
				obj := g.StackPeek()
				if !obj.IsCopy || obj.Controller != b.PlayerID() || chooser.targetCalls != 1 || b.ManaPool().TotalMana() != 0 {
					t.Fatal("copy control, choices or payment")
				}
				if state == "counter-copy" {
					g.CounterStackObject(obj.ID)
				} else {
					g.ResolveTopOfStack()
					if state == "chain" {
						if g.StackSize() != 1 || g.StackPeek().Controller != a.PlayerID() || g.StackPeek().Targets[0] != b.PlayerID() {
							t.Fatal("copy of resolving copy")
						}
						g.ResolveTopOfStack()
						if g.StackSize() != 0 {
							t.Fatal("unpaid repeated copy")
						}
					}
				}
			}
			if len(a.Graveyard()) != 1 || len(b.Graveyard()) != map[bool]int{true: 1, false: 0}[state == "creature" || state == "control"] {
				t.Fatal("copy zone or original graveyard")
			}
			wantA, wantB := 20, 17
			if state == "creature" || state == "control" || state == "phase" || state == "return" || state == "prevention" || state == "payment-control" || state == "redirect" {
				wantB = 20
			}
			if copied && state != "counter-copy" {
				if state == "keep" {
					wantB -= 3
				} else {
					wantA -= 3
				}
			}
			if state == "redirect" {
				wantA -= 3
			}
			if state == "chain" {
				wantB -= 3
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life %d/%d want %d/%d", a.Life(), b.Life(), wantA, wantB)
			}
			if clone != nil {
				if clone.StackSize() != 1 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("clone changed")
				}
				clone.ResolveTopOfStack()
				if g.StackSize() != 0 {
					t.Fatal("clone leaked")
				}
			}
		})
	}
}

func TestRONRecipientSpellCopyInvalid(t *testing.T) {
	base := `SpellActions(Targeted([AnyTarget],ActionList([PlayerAction(PlayerOrControllerOfPermanent(Ref_AnyTarget),CopySpellAndMayChooseNewTargets(ThisSpell))])))`
	validNodes, parseErr := Parse(strings.NewReader(`Card(Name:"Valid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[`+base+`])`), "valid.ron")
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	if _, err := Compile(validNodes[0]); err != nil {
		t.Fatal(err)
	}
	for _, rule := range []string{
		strings.Replace(base, "ThisSpell)", "ThisSpell())", 1),
		strings.Replace(base, "Ref_AnyTarget)", "Ref_AnyTarget())", 1),
		strings.Replace(base, "Ref_AnyTarget)", "Ref_TargetPermanent)", 1),
		strings.Replace(base, "CopySpellAndMayChooseNewTargets(ThisSpell)", "CopySpellAndMayChooseNewTargets(Ref_TargetSpell)", 1),
		`Activated(PayMana([]),ActionList([CopySpellAndMayChooseNewTargets(ThisSpell)]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[`+rule+`])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration", rule)
		}
	}
}

func TestRONRecipientSpellCopyInventory(t *testing.T) {
	report, err := InspectFS(os.DirFS("../../../../cards/mtgish"), "LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, entry := range report.Signatures {
		if !slices.Contains(entry.Cards, "Chain Lightning") {
			continue
		}
		for name, role := range map[string]string{"PlayerOrControllerOfPermanent": "player_reference", "CopySpellAndMayChooseNewTargets": "action", "Ref_AnyTarget": "recipient"} {
			if entry.Name == name && entry.Role == role {
				seen[name] = true
			}
		}
	}
	if len(seen) != 3 {
		t.Fatal("missing semantic roles", seen)
	}
}
