package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONExileInstructionRealCards(t *testing.T) {
	for _, name := range []string{"Dust to Dust", "Ashes to Ashes"} {
		for _, state := range []string{"normal", "one illegal", "all illegal", "phase", "phase cycle", "returned", "copy", "retarget", "clone", "unpaid", "bad count", "duplicate", "protection", "indestructible"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				a := mage.NewBasePlayer("A")
				b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B")}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				makeSubject := func(label string) *mage.Permanent {
					var card mage.Card = mage.NewArtifact(label, "")
					if name == "Ashes to Ashes" {
						card = mage.NewCreature(label, "", 2, 2)
					}
					card.SetOwner(b.PlayerID())
					return g.PutOnBattlefield(card, b.PlayerID())
				}
				p, q, r := makeSubject("one"), makeSubject("two"), makeSubject("three")
				card := repositoryCard(t, "DRK", name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				ids := []uuid.UUID{p.ID(), q.ID()}
				if state != "unpaid" {
					a.ManaPool().Add(core.White, 10)
					a.ManaPool().Add(core.Black, 10)
				}
				if state == "bad count" {
					ids = ids[:1]
				}
				if state == "duplicate" {
					ids[1] = ids[0]
				}
				err := g.CastSpellByID(a.PlayerID(), card.ID(), ids, 0)
				if state == "unpaid" || state == "bad count" || state == "duplicate" {
					if err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 || g.FindPermanent(p.ID()) == nil || a.Life() != 20 {
						t.Fatal("invalid casting changed state")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				obj := g.StackPeek()
				first, second, third := true, true, false
				switch state {
				case "one illegal":
					p.GrantBaseAttr(core.Shroud)
					first = false
				case "all illegal":
					p.GrantBaseAttr(core.Shroud)
					q.GrantBaseAttr(core.Shroud)
					first, second = false, false
				case "phase":
					g.PhaseOut(p.ID())
					first = false
				case "phase cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "returned":
					g.RemoveFromBattlefield(p)
					g.PutOnBattlefield(p.Card, b.PlayerID())
					first = false
				case "copy", "retarget":
					b.targets = []uuid.UUID{r.ID(), q.ID()}
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), state == "retarget")
					if obj == nil {
						t.Fatal("copy failed")
					}
					if state == "retarget" {
						first, third = false, true
					}
				case "protection":
					if err := mage.ApplyEffect(g, mage.PreventNextDamage(mage.You, mage.Fixed(3)), uuid.Nil, a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "indestructible":
					p.GrantBaseAttr(core.Indestructible)
					g.AddRegenerationShield(p.ID())
				}
				exile, ok := mage.InspectPermanentExileInstructions(mage.Sequence(obj.Effects...))
				if name == "Dust to Dust" && (!ok || len(exile) != 1 || exile[0].Subjects == nil) {
					t.Fatal("RON lost native exile group")
				}
				query, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
				watched := query.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
				for _, game := range []*mage.Game{g, g.Clone()} {
					for range 2 {
						if certain, known := game.InspectSpellDestruction(game.StackPeek(), watched); certain || !known {
							t.Fatal("real exile query unknown")
						}
					}
					if len(game.GetExile()) != 0 || game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 {
						t.Fatal("query executed an action")
					}
					game.ResolveTopOfStack()
					for i, expect := range []bool{first, second, third} {
						id := []*mage.Permanent{p, q, r}[i].ID()
						exiled := false
						for _, entry := range game.GetExile() {
							exiled = exiled || entry.Card.ID() == id
						}
						if exiled != expect {
							t.Fatalf("subject %d exile=%v want=%v", i, exiled, expect)
						}
					}
					lifeA, lifeB := 20, 20
					if name == "Ashes to Ashes" && state != "all illegal" {
						if state == "copy" || state == "retarget" {
							lifeB = 15
						} else {
							lifeA = 15
						}
						if state == "protection" {
							lifeA = 18
						}
					}
					if game.GetPlayer(a.PlayerID()).Life() != lifeA || game.GetPlayer(b.PlayerID()).Life() != lifeB {
						t.Fatal("wrong damage recipient, prevention or all-illegal gate")
					}
				}
			})
		}
	}
}

func TestRONExileInstructionBindings(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([Exile([Permanent(Ref_TargetPermanent)],[])])))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(mage.NewLand("one"), b.PlayerID())
	q := g.PutOnBattlefield(mage.NewLand("two"), b.PlayerID())
	first := castRONSpell(t, g, a, factory(), p.ID())
	second := castRONSpell(t, g, b, factory(), q.ID())
	for _, obj := range []*mage.StackObject{first, second} {
		if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
			t.Fatal("independent singular binding unknown")
		}
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(p.ID()) == nil || g.FindPermanent(q.ID()) != nil {
		t.Fatal("factory bindings crossed")
	}
	g.ResolveStack()
	if len(g.GetExile()) != 2 {
		t.Fatal("singular exile failed")
	}
}

func TestRONExileInstructionInvalid(t *testing.T) {
	for _, action := range []string{`Exile()`, `Exile([Permanent(Ref_TargetPermanent)])`, `Exile([],[])`, `Exile([Permanent(You)],[])`, `Exile([Permanent(Ref_TargetPermanent2)],[])`, `Exile([Permanent(Ref_TargetPermanent)], [Unknown])`, `Exile([EachPermanent(Ref_TargetPermanent)],[])`, `Exile([EachPermanent(Unknown)],[])`, `Exile([Permanent()],[])`} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Invalid exile",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([`+action+`])))])`)); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}
