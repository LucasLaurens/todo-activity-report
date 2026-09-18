package domain

import "fmt"

const secondsPerHour = 3600

// TimeSpent is an immutable value object holding a non-negative duration in seconds.
type TimeSpent struct {
	seconds int
}

func NewTimeSpent(seconds int) TimeSpent {
	if seconds < 0 {
		seconds = 0
	}
	return TimeSpent{seconds: seconds}
}

func (t TimeSpent) Seconds() int {
	return t.seconds
}

func (t TimeSpent) Hours() float64 {
	return float64(t.seconds) / secondsPerHour
}

func (t TimeSpent) Add(other TimeSpent) TimeSpent {
	return TimeSpent{seconds: t.seconds + other.seconds}
}

func (t TimeSpent) DividedBy(n int) TimeSpent {
	if n <= 0 {
		return TimeSpent{}
	}
	return TimeSpent{seconds: t.seconds / n}
}

// ShareOf is this duration's fraction of a total, in [0, 1]. An empty total
// yields 0 rather than a division by zero.
func (t TimeSpent) ShareOf(total TimeSpent) float64 {
	if total.seconds == 0 {
		return 0
	}
	return float64(t.seconds) / float64(total.seconds)
}

func (t TimeSpent) String() string {
	hours := t.seconds / secondsPerHour
	minutes := (t.seconds % secondsPerHour) / 60
	if hours == 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%dh %02dm", hours, minutes)
}
