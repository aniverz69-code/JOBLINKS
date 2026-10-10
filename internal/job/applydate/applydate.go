package applydate

import (
	"errors"
	"time"
)

// MaxAgeDays is how old a stated apply date may be. A year-old application says
// nothing about whether a posting is live now, so admitting one would let stale history speak
// about the present.
const MaxAgeDays = 365

// ErrOutOfRange is the sentinel behind every rejection here, so a caller can map the
// whole family onto its own error without matching on the message it shows the person.
var ErrOutOfRange = errors.New("apply date out of range")

// rangeError carries the reason alone as its message, and answers to the sentinel through
// Is. Both callers put this text in the body of a 400, so it must read as an instruction to the
// person rather than as a chain of package prefixes.
type rangeError struct{ reason string }

func (e *rangeError) Error() string { return e.reason }

func (e *rangeError) Is(target error) bool { return target == ErrOutOfRange }

// noonHour is the hour of the stated day an application is stored at. Noon, because a
// calendar day has to become an instant and midnight UTC renders as the PREVIOUS date for every
// reader west of Greenwich — the card would show a day earlier than the one they typed.
//
// It lives beside the window on purpose. Bounding the stored instant instead of the day is a
// real defect and not an obvious one: with a noon stamp, "I applied today" is in the future for
// the whole UTC morning, and for a caller east of UTC+12 it is in the future always. One file
// owning both makes the order — validate the day, then place it in the day — hard to get wrong.
const noonHour = 12

// Instant places a stated day at the hour applications are stored at.
func Instant(day time.Time) time.Time {
	y, m, d := day.Date()
	return time.Date(y, m, d, noonHour, 0, 0, 0, time.UTC)
}

// Validate bounds a date somebody states they applied on to the window in which it can
// mean anything: not the future, and not so far back that it describes a different hiring round.
//
// It takes the DAY, never the instant Instant derives from it.
//
// It lives beside the stage vocabulary rather than in either caller because two surfaces now
// take such a date — the ghost report, where it is evidence about a posting, and the tracker's
// dated apply, where it is the candidate correcting their own history. Held separately, the two
// answers to "which dates are believable" would part company the first time one was tuned.
func Validate(appliedOn, now time.Time) error {
	if appliedOn.After(now) {
		return &rangeError{reason: "the apply date is in the future"}
	}
	if now.Sub(appliedOn) > MaxAgeDays*24*time.Hour {
		return &rangeError{reason: "the apply date is more than a year ago"}
	}
	return nil
}
