package warmthshortenscoldnights

import (
	"github.com/simimpact/srsim/pkg/engine"
	"github.com/simimpact/srsim/pkg/engine/equip/lightcone"
	"github.com/simimpact/srsim/pkg/engine/event"
	"github.com/simimpact/srsim/pkg/engine/info"
	"github.com/simimpact/srsim/pkg/engine/modifier"
	"github.com/simimpact/srsim/pkg/engine/prop"
	"github.com/simimpact/srsim/pkg/key"
	"github.com/simimpact/srsim/pkg/model"
)

const (
	CheckNBuff   key.Modifier = "warmth_shortens_cold_nights"
)

func init() {
	lightcone.Register(key.WarmthShortensColdNights, lightcone.Config{
		CreatePassive: Create,
		Rarity:        4,
		Path:          model.Path_ABUNDANCE,
		Promotions:    promotions,
	})

	//Check if action is basic atk / skill
	modifier.Register(CheckNBuff, modifier.Config{
		Listeners: modifier.Listeners{
			OnAfterAction: healTeamOnBasicOrSkill,
		},
	})
}

func Create(engine engine.Engine, owner key.TargetID, lc info.LightCone) {
	amt := 0.12 + 0.04*float64(lc.Imposition)
	//OnStart : (Simplified to 1 call)
	engine.AddModifier(owner, info.Modifier{
		Name:   CheckNBuff,
		Source: owner,
		Stats:  info.PropMap{prop.HPPercent: amt}, //static "buff"
		State:  0.015 + 0.005 * float64(lc.Imposition), //state to pass into check logic
	})
}
//if basic atk/skill, heal the whole team by x%
func healTeamOnBasicOrSkill(mod *modifier.ModifierInstance, e event.ActionEvent) {
	amt := mod.State().(float64)
	switch e.AttackType {
	case model.AttackType_NORMAL, model.AttackType_SKILL :
		//apply team heal with % based on target
		mod.Engine().Heal(info.Heal{
			Targets:  mod.Engine().Characters(), //fetch alive allies IDs through the engine
			Source:   mod.Owner(),
			BaseHeal: info.HealMap{model.HealFormula_BY_TARGET_MAX_HP: amt},
		})
	}
}