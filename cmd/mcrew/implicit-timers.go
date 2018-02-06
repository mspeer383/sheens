package main

// This file is an experiment in implicit timers.
//
// Use Service.ImplicitTimers to enable/disable.

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Comcast/sheens/core"
)

// consideredWalked might take some action based on the given Walked.
//
// If s.ImplicitTimers is true, then this method will automatically
// create and cancel timers based on branch patterns of certain forms.
//
// If a message-based branch pattern includes an "after" property,
// then an implicit timer is created and automatically cancelled.
// When the machine transitions to such a node, the value of the
// "after" property is passed to InterpretTime().  The result gives a
// time at which a timer, which will be automatically created, should
// fire.  When that timer fires, a message matching the branch pattern
// is sent to the machine (directly).  When the machine transitions
// away from this node, the timer is cancelled (perhaps after it has
// already fired).
//
// This method definitely adds overhead to machine execution.  ToDo:
// benchmark.
//
// ToDo: Support multiple implicit timers in a single set of branches.
func (s *Service) considerWalked(ctx context.Context, mid string, spec *core.Spec, walked *core.Walked) {

	if !s.ImplicitTimers {
		return
	}

	// When we enter a node with a timer branch, we create a
	// timer.
	//
	// When we leave a node with a timer branch, we delete the
	// timer.

	for _, stride := range walked.Strides {
		if stride.From != nil && stride.To != nil {
			// Probably cheaper just to try to remove a
			// timer even if there isn't one to remove.
			_ = s.timers.Rem(ctx, mid+"/"+stride.From.NodeName)
		}
		if stride.To != nil {
			// We are arriving at a node.  Does it have a
			// timer branch (or branches)?

			// This code silently bails out if it runs
			// into any problems.  ToDo: Something better;
			// maybe just log.

			node, have := spec.Nodes[stride.To.NodeName]
			if !have {
				// ToDo: Warn.
				continue
			}
			if node.Branches == nil {
				continue
			}
			if node.Branches.Type != "message" {
				continue
			}

			for _, b := range node.Branches.Branches {
				if b.Pattern == nil {
					continue
				}

				m, is := b.Pattern.(map[string]interface{})
				if !is {
					continue
				}

				x, have := m["after"]
				if !have {
					continue
				}

				if s, is := x.(string); is {
					if core.IsVariable(s) {
						x, have = stride.To.Bs[s]
					}
				}

				t, err := InterpretTime(x)
				if err != nil {
					s.err(err)
					continue
				}

				err = s.timers.Add(ctx,
					mid+"/"+stride.To.NodeName,
					map[string]interface{}{
						"after": x,
					},
					t.Sub(time.Now()))
				if err != nil {
					s.err(err)
				}
			}
		}
	}
}

// ClockTimeFormat is one of the time formats supported by InterpretTime.
var ClockTimeFormat = "2006-01-02 15:04:05Z"

// InterpretTime attempts to interpret is argument as a time.
//
// Current support:
//
// A number (int, int64, float64) is interpreted as "seconds from now".
//
// A string representation of an integer is interpreted as that many
// seconds from now.
//
// A string represenation of a Go Duration is interpreted as that
// duration from now.
//
// A string representation of the form ClockTimeFormat is interpreted
// as that time as parsed.
//
// ToDo: Support crontab syntax.
func InterpretTime(v interface{}) (t time.Time, err error) {

	t = ZeroTime

	switch vv := v.(type) {
	case int:
		t = time.Now().UTC().Add(time.Duration(vv) * time.Second)
		return
	case int64:
		t = time.Now().UTC().Add(time.Duration(vv) * time.Second)
		return
	case float64:
		t = time.Now().UTC().Add(time.Duration(vv) * time.Second)
		return
	case string:
		s := vv
		var secs int
		if secs, err = strconv.Atoi(s); err == nil {
			t = time.Now().UTC().Add(time.Duration(secs) * time.Second)
			return
		}
		if t, err = time.Parse(ClockTimeFormat, s); err == nil {
			return
		}
		var d time.Duration
		if d, err = time.ParseDuration(s); err == nil {
			t = time.Now().UTC().Add(d)
			return
		}
	default:
		err = fmt.Errorf(`unsupported "after" %#v (%T)`, v, v)
	}

	return
}

var ZeroTime time.Time
