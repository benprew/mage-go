package ron

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type generalRONSourcePlayer struct {
	*mage.BasePlayer
	wanted uuid.UUID
	calls  int
	seen   []mage.DamageSource
	chosen mage.DamageSource
}

func (p *generalRONSourcePlayer) ChooseDamageSource(sources []mage.DamageSource, _ *mage.Game) int {
	p.calls++
	p.seen = sources
	for i, source := range sources {
		if source.ID() == p.wanted {
			p.chosen = source
			return i
		}
	}
	return 0
}

var generalRONProtectionCards = []struct{ set, name, sourceSet, source string }{
	{"LEB", "Circle of Protection: Black", "LEB", "Frozen Shade"},
	{"LEA", "Circle of Protection: Blue", "LEA", "Prodigal Sorcerer"},
	{"LEA", "Circle of Protection: Green", "LEA", "Llanowar Elves"},
	{"LEA", "Circle of Protection: Red", "LEA", "Orcish Artillery"},
	{"LEA", "Circle of Protection: White", "LEA", "Samite Healer"},
	{"ATQ", "Circle of Protection: Artifacts", "ATQ", "Grapeshot Catapult"},
	{"LEG", "Greater Realm of Preservation", "LEA", "Orcish Artillery"},
}

func TestRONGeneralDamageSourceProtection(t *testing.T) {
	for _, tc := range generalRONProtectionCards {
		for _, state := range []string{"live", "departed", "phased", "return", "recheck", "copy", "cleanup", "spell"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				a := &generalRONSourcePlayer{BasePlayer: mage.NewBasePlayer("A")}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				consumer := repositoryCard(t, tc.set, tc.name)
				consumer.SetOwner(a.PlayerID())
				q := g.PutOnBattlefield(consumer, a.PlayerID())
				card := repositoryCard(t, tc.sourceSet, tc.source)
				card.SetOwner(b.PlayerID())
				a.wanted = card.ID()
				var p *mage.Permanent
				if state == "spell" {
					g.SetActivePlayerIndex(1)
					castRONSpell(t, g, b, card)
				} else {
					p = g.PutOnBattlefield(card, b.PlayerID())
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err == nil || a.calls != 0 {
					t.Fatal("unpaid activation chose a source")
				}
				a.ManaPool().Add(core.White, 2)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if a.calls != 0 {
					t.Fatal("choice ran at announcement")
				}
				if state == "departed" || state == "phased" {
					if err := mage.ApplyEffect(g, mage.AtNext(mage.EachEndStep, mage.TapSubject(mage.This)), p.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				if state == "departed" {
					g.BouncePermanentToHand(p)
				}
				if state == "phased" {
					g.PhaseOut(p.ID())
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveTopOfStack()
				if state == "copy" {
					g.ResolveTopOfStack()
				}
				wantCalls := 1
				if state == "copy" {
					wantCalls = 2
				}
				if a.calls != wantCalls || !a.chosen.Valid() {
					t.Fatal("resolution did not select source")
				}
				saved := a.chosen
				if state == "spell" {
					g.ResolveTopOfStack()
					p = g.FindPermanent(card.ID())
				}
				if state == "return" {
					g.BouncePermanentToHand(p)
					b.RemoveFromHand(card.ID())
					p = g.PutOnBattlefield(card, b.PlayerID())
				}
				if state == "recheck" {
					if tc.name == "Circle of Protection: Artifacts" {
						g.MutablePermanent(p.ID()).FaceDown = true
					} else {
						g.ChangeObjectColor(p.ID(), core.Colorless)
					}
				}
				if state == "cleanup" {
					g.GetEffects().ClearReplacementsEndOfTurn()
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					recipient := game.GetPlayer(a.PlayerID())
					damageSource := saved
					if state == "spell" || state == "return" || state == "recheck" {
						damageSource = game.CaptureDamageSource(p.ID())
					}
					game.DealDamageToPlayerFromSource(recipient, 2, damageSource)
					want := 20
					if state == "return" || state == "recheck" || state == "cleanup" {
						want = 18
					}
					if recipient.Life() != want {
						t.Fatal("shield identity, recheck or lifetime", recipient.Life(), want)
					}
					if state == "recheck" {
						if tc.name == "Circle of Protection: Artifacts" {
							game.MutablePermanent(p.ID()).FaceDown = false
						} else {
							game.ChangeObjectColor(p.ID(), savedColor(saved))
						}
						game.DealDamageToPlayer(recipient, 2, p.ID())
						if recipient.Life() != want {
							t.Fatal("nonmatching damage consumed shield")
						}
					}
					game.DealDamageToPlayerFromSource(recipient, 1, damageSource)
					if state == "copy" {
						if recipient.Life() != want {
							t.Fatal("copy lost separate shield")
						}
						game.DealDamageToPlayerFromSource(recipient, 1, damageSource)
					}
					if recipient.Life() != want-1 {
						t.Fatal("one-use shield consumption", recipient.Life(), want)
					}
				}
			})
		}
	}
}
func savedColor(source mage.DamageSource) core.Color {
	for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green} {
		if source.HasColor(color) {
			return color
		}
	}
	return core.Colorless
}

func TestRONGeneralDamageSourceInvalid(t *testing.T) {
	for _, actions := range []string{
		`ChooseADamageSource`, `ChooseADamageSource()`, `ChooseADamageSource(AnyDamageSource,You)`,
		`ChooseADamageSource(Or([]))`,
		`If(PermanentPassesFilter(ThisPermanent,IsUntapped),ActionList([ChooseADamageSource(AnyDamageSource)])),CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Player(You)),[PreventThatDamage])`, `ChooseADamageSource(IsColor(Colorless))`,
		`CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Player(You)),[PreventThatDamage]),ChooseADamageSource(AnyDamageSource)`,
		`ChooseADamageSource(AnyDamageSource),CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource(),Player(You)),[PreventThatDamage])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+actions+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", actions)
		}
	}
}

func TestRONGeneralDamageSourceReprints(t *testing.T) {
	count := 0
	names := make(map[string]bool)
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI", "ATQ", "LEG"} {
		nodes, err := Parse(strings.NewReader(mustRONSet(t, set)), set+".ron")
		if err != nil {
			t.Fatal(err)
		}
		for _, record := range nodes {
			name := record.Fields["Name"].String
			if name != "Jade Monolith" && !slices.ContainsFunc(generalRONProtectionCards, func(tc struct{ set, name, sourceSet, source string }) bool { return tc.name == name }) {
				continue
			}
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(name, err)
			}
			first, second := factory(), factory()
			if err := mage.ValidateCardDefinition(name, first); err != nil {
				t.Fatal(err)
			}
			if first.ID() == second.ID() {
				t.Fatal("factory shares identity")
			}
			count++
			names[name] = true
		}
	}
	if count != 37 || len(names) != 8 {
		t.Fatal("affected inventory changed", count, len(names))
	}
}

func TestRONGeneralDamageSourceJade(t *testing.T) {
	for _, state := range []string{"live", "departed", "return", "phased", "illegal", "control", "cleanup"} {
		t.Run(state, func(t *testing.T) {
			a := &generalRONSourcePlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			jade := g.PutOnBattlefield(repositoryCard(t, "LEA", "Jade Monolith"), a.PlayerID())
			target := g.PutOnBattlefield(mage.NewCreature("Recipient", "", 1, 20), b.PlayerID())
			card := repositoryCard(t, "LEA", "Prodigal Sorcerer")
			source := g.PutOnBattlefield(card, b.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			a.wanted = source.ID()
			if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), jade.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
				t.Fatal("unpaid Jade activation succeeded")
			}
			a.ManaPool().Add(core.White, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), jade.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if a.calls != 0 {
				t.Fatal("Jade chose at announcement")
			}
			switch state {
			case "departed":
				g.BouncePermanentToHand(source)
			case "phased":
				g.PhaseOut(source.ID())
			case "illegal":
				g.PhaseOut(target.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: target.ID(), TargetID: jade.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.ResolveTopOfStack()
			if state == "return" {
				g.BouncePermanentToHand(source)
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			}
			if state == "cleanup" {
				g.GetEffects().ClearReplacementsEndOfTurn()
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.ResolveTopOfStack()
				want := 19
				if state == "illegal" || state == "cleanup" {
					want = 20
				}
				if game.GetPlayer(a.PlayerID()).Life() != want || game.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("Jade changed wrong recipient", state)
				}
				damage := 0
				if state == "cleanup" {
					damage = 1
				}
				if game.FindPermanentIncludingPhased(target.ID()).Damage != damage {
					t.Fatal("Jade target legality or replacement")
				}
			}
			if state == "illegal" {
				if a.calls != 0 {
					t.Fatal("illegal target made a choice")
				}
			} else if a.calls != 1 || !a.chosen.Valid() {
				t.Fatal("Jade choice failed")
			}
		})
	}
}

func TestRONGeneralDamageSourceBindingsAndBatch(t *testing.T) {
	text := `Card(Name:"Choices",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([
 ChooseADamageSource(IsColor(Red)),CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Player(You)),[PreventThatDamage]),
 ChooseADamageSource(IsColor(Blue)),CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Player(You)),[PreventThatDamage])]))])`
	nodes, err := Parse(strings.NewReader(text), "choices.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	red := g.PutOnBattlefield(mage.NewCreature("Red", "{R}", 1, 20), a.PlayerID())
	blue := g.PutOnBattlefield(mage.NewCreature("Blue", "{U}", 1, 20), b.PlayerID())
	for _, player := range []*mage.BasePlayer{a, b} {
		q := g.PutOnBattlefield(factory(), player.PlayerID())
		if err := g.ActivateAbilityByIndex(player.PlayerID(), q.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	for _, game := range []*mage.Game{g, g.Clone()} {
		for _, source := range []*mage.Permanent{red, blue} {
			body := mage.DealDamagePackets(mage.This,
				mage.DamagePacketTo(mage.You, mage.Fixed(2)), mage.DamagePacketTo(mage.You, mage.Fixed(3)))
			for _, player := range []*mage.BasePlayer{a, b} {
				if err := mage.ApplyEffect(game, body, source.ID(), player.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want := 20
				if source == blue {
					want = 19
				}
				if game.GetPlayer(player.PlayerID()).Life() != want {
					t.Fatal("binding or simultaneous shield consumption")
				}
				game.DealDamageToPlayer(game.GetPlayer(player.PlayerID()), 1, source.ID())
			}
		}
		if game.GetPlayer(a.PlayerID()).Life() != 18 || game.GetPlayer(b.PlayerID()).Life() != 18 {
			t.Fatal("independent shields lost")
		}
	}
}

func TestRONGeneralDamageSourceScope(t *testing.T) {
	shield := `CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Player(You)),[PreventThatDamage])`
	branch := `If(PermanentPassesFilter(ThisPermanent,IsUntapped),[ChooseADamageSource(AnyDamageSource),` + shield + `])`
	for _, outside := range []bool{false, true} {
		actions := branch
		if outside {
			actions += "," + shield
		}
		nodes, err := Parse(strings.NewReader(`Card(Name:"Scope",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+actions+`]))])`), "scope.ron")
		if err != nil {
			t.Fatal(err)
		}
		_, err = Compile(nodes[0])
		if outside {
			if err == nil || !strings.Contains(err.Error(), "unbound or incompatible chosen damage source") {
				t.Fatal("branch choice escaped its scope", err)
			}
		} else if err != nil {
			t.Fatal("branch-local choice rejected", err)
		}
	}
}
