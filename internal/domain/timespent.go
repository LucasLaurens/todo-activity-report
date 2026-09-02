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

func (t TimeSpent) String() string {
	hours := t.seconds / secondsPerHour
	minutes := (t.seconds % secondsPerHour) / 60
	if hours == 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%dh %02dm", hours, minutes)
}
