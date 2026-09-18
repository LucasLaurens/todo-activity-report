package application

import (
	"sort"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

// buildTimeBreakdown ranks the tickets by the time logged on them and gives
// each its share of the period's total, so the report answers "where did the
// day go?" at a glance. Tickets touched without logged time carry no share and
// are only counted.
// O(n log n) time, O(n) space.
func buildTimeBreakdown(tasks []domain.Task, total domain.TimeSpent) ([]domain.TaskTime, int) {
	timed := make([]domain.TaskTime, 0, len(tasks))
	untimed := 0
	for _, task := range tasks {
		if task.TimeSpent.Seconds() == 0 {
			untimed++
			continue
		}
		timed = append(timed, domain.TaskTime{
			Key:       task.Key,
			Summary:   task.Summary,
			Status:    task.Status,
			TimeSpent: task.TimeSpent,
			Share:     task.TimeSpent.ShareOf(total),
		})
	}
	sort.Slice(timed, func(i, j int) bool {
		if timed[i].TimeSpent.Seconds() != timed[j].TimeSpent.Seconds() {
			return timed[i].TimeSpent.Seconds() > timed[j].TimeSpent.Seconds()
		}
		return timed[i].Key < timed[j].Key
	})
	return timed, untimed
}
