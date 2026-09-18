package domain

import (
	"errors"
	"time"
)

const dateLayout = "2006-01-02"

var ErrInvalidDateRange = errors.New("from date must not be after to date")

// DateRange is an inclusive period bounded by two days.
type DateRange struct {
	from time.Time
	to   time.Time
}

func NewDateRange(from, to time.Time) (DateRange, error) {
	if from.After(to) {
		return DateRange{}, ErrInvalidDateRange
	}
	return DateRange{from: from, to: to}, nil
}

func ParseDateRange(from, to string) (DateRange, error) {
	parsedFrom, err := time.Parse(dateLayout, from)
	if err != nil {
		return DateRange{}, err
	}
	parsedTo, err := time.Parse(dateLayout, to)
	if err != nil {
		return DateRange{}, err
	}
	return NewDateRange(parsedFrom, parsedTo)
}

func (r DateRange) From() time.Time {
	return r.from
}

func (r DateRange) To() time.Time {
	return r.to
}

// Contains reports whether a moment falls in the range, whose last day counts
// in full: the period is inclusive, so 23:59 of the "to" day is still inside.
func (r DateRange) Contains(moment time.Time) bool {
	return !moment.Before(r.from) && moment.Before(r.to.AddDate(0, 0, 1))
}

func (r DateRange) FromString() string {
	return r.from.Format(dateLayout)
}

func (r DateRange) ToString() string {
	return r.to.Format(dateLayout)
}
