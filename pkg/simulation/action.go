package simulation

import (
	"fmt"

	actionPkg "github.com/simimpact/srsim/pkg/engine/action"
	"github.com/simimpact/srsim/pkg/engine/event"
	"github.com/simimpact/srsim/pkg/engine/info"
	"github.com/simimpact/srsim/pkg/engine/queue"
	"github.com/simimpact/srsim/pkg/engine/target"
	"github.com/simimpact/srsim/pkg/key"
	"github.com/simimpact/srsim/pkg/model"
)

// TODO: Unknown TC
//	- Does ActionStart & ActionEnd happen for inserted actions? If  so, this means
//		LC such as "In the Name of the World" buff these insert actions
//	- Do Insert abilities (follow up attacks, counters, etc) count as an Action (similar to above)?

func (sim *Simulation) InsertAction(target key.TargetID) {
	var priority info.InsertPriority
	switch sim.Targets[target] {
	case info.ClassEnemy:
		priority = info.EnemyInsertAction
	default:
		priority = info.CharInsertAction
	}

	sim.Queue.Insert(queue.Task{
		Source:   target,
		Priority: priority,
		AbortFlags: []model.BehaviorFlag{
			model.BehaviorFlag_STAT_CTRL,
			model.BehaviorFlag_DISABLE_ACTION,
		},
		Execute: func() { sim.executeAction(target, true) },
	})
}

func (sim *Simulation) InsertAbility(i info.Insert) {
	sim.Queue.Insert(queue.Task{
		Source:     i.Source,
		Priority:   i.Priority,
		AbortFlags: i.AbortFlags,
		Execute:    func() { sim.executeInsert(i) },
	})
}

func (sim *Simulation) ultCheck() error {
	ults, err := sim.eval.UltCheck()
	if err != nil {
		return err
	}
	for _, act := range ults {
		if sim.Attr.FullEnergy(act.Target) {
			sim.Queue.Insert(queue.Task{
				Source:   act.Target,
				Priority: info.CharInsertUlt,
				AbortFlags: []model.BehaviorFlag{
					model.BehaviorFlag_STAT_CTRL,
					model.BehaviorFlag_DISABLE_ACTION,
				},
				Execute: func() { sim.executeUlt(act) }, // TODO: error handling
			})

		}
	}
	return nil
}

// execute everything on the queue. After queue execution is complete, return the next stateFn
// as the next state to run. This logic will run the exitCheck on each execution. If an exit
// condition is met, will return that state instead
func (s *Simulation) executeQueue(phase info.BattlePhase, next stateFn) (stateFn, error) {
	// always ult check when calling executeQueue
	if err := s.ultCheck(); err != nil {
		return next, err
	}

	// if active is not a character, cannot prform any queue execution until after ActionEnd
	if phase < info.ActionEnd && !s.IsCharacter(s.Active) {
		return s.exitCheck(next)
	}

	for !s.Queue.IsEmpty() {
		insert := s.Queue.Pop()

		// if source has no HP, skip this insert
		if s.Attr.HPRatio(insert.Source) <= 0 {
			continue
		}

		// if the source has an abort flag, skip this insert
		if s.HasBehaviorFlag(insert.Source, insert.AbortFlags...) {
			continue
		}

		insert.Execute()

		// attempt to exit. If can exit, stop sim now
		if next, err := s.exitCheck(next); next == nil || err != nil {
			return next, err
		}
		if err := s.ultCheck(); err != nil {
			return next, err
		}
	}
	return next, nil
}

func (sim *Simulation) executeAction(id key.TargetID, isInsert bool) error {
	var executable target.ExecutableAction
	var err error

	switch sim.Targets[id] {
	case info.ClassCharacter:
		executable, err = sim.Char.ExecuteAction(id, isInsert)
		if err != nil {
			return fmt.Errorf("error building char executable action %w", err)
		}
	case info.ClassEnemy:
		executable, err = sim.Enemy.ExecuteAction(id)
		if err != nil {
			return fmt.Errorf("error building enemy executable action %w", err)
		}
	case info.ClassNeutral:
		// TODO:
	default:
		return fmt.Errorf("unsupported target type: %v", sim.Targets[id])
	}

	sim.ModifySP(executable.SPDelta)
	sim.clearActionTargets()
	sim.Event.ActionStart.Emit(event.ActionEvent{
		Owner:      id,
		AttackType: executable.AttackType,
		IsInsert:   isInsert,
	})

	// execute action
	executable.Execute()

	// end attack if in one. no-op if not in an attack
	// emit end events
	sim.Combat.EndAttack()
	sim.Event.ActionEnd.Emit(event.ActionEvent{
		Owner:      id,
		Targets:    sim.ActionTargets,
		AttackType: executable.AttackType,
		IsInsert:   isInsert,
	})
	return nil
}

func (sim *Simulation) executeUlt(act actionPkg.Action) error {
	var executable target.ExecutableUlt
	var err error

	id := act.Target
	switch sim.Targets[id] {
	case info.ClassCharacter:
		executable, err = sim.Char.ExecuteUlt(act)
		if err != nil {
			return fmt.Errorf("error building char executable ult %w", err)
		}
	default:
		return fmt.Errorf("unsupported target type: %v", sim.Targets[id])
	}

	sim.clearActionTargets()
	sim.Event.UltStart.Emit(event.ActionEvent{
		Owner:      id,
		AttackType: model.AttackType_ULT,
		IsInsert:   true,
	})

	executable.Execute()

	// end attack if in one. no-op if not in an attack
	sim.Combat.EndAttack()
	sim.Event.UltEnd.Emit(event.ActionEvent{
		Owner:      id,
		Targets:    sim.ActionTargets,
		AttackType: model.AttackType_ULT,
		IsInsert:   true,
	})
	return nil
}

func (sim *Simulation) executeInsert(i info.Insert) {
	sim.clearActionTargets()
	sim.Event.InsertStart.Emit(event.InsertEvent{
		Owner:      i.Source,
		AbortFlags: i.AbortFlags,
		Priority:   i.Priority,
	})

	// execute insert
	i.Execute()

	// end attack if in one. no-op if not in an attack
	sim.Combat.EndAttack()
	sim.Event.InsertEnd.Emit(event.InsertEvent{
		Owner:      i.Source,
		Targets:    sim.ActionTargets,
		AbortFlags: i.AbortFlags,
		Priority:   i.Priority,
	})
}

func (s *Simulation) clearActionTargets() {
	for k := range s.ActionTargets {
		delete(s.ActionTargets, k)
	}
}
