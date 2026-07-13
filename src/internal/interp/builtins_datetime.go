package interp

import (
	"time"
)

// TODO: I don't know if it still make sense the way it's currently designed.
//       Should I have a date type?

func builtinTime(in *Interp, c *Call) Value {
	return IntV(time.Now().Unix())
}

// adjustedUTC is the year/month/date/day/hour/minute/second builtins'
// shared pattern: an integer *hour* offset added to the current wall-clock
// time, then read back out via UTC (gmtime), not local time.
func adjustedUTC(c *Call) time.Time {
	offset := intOf(expectKindAt(c.At(0), c.Args[0], KInt))
	return time.Now().Add(time.Duration(offset) * time.Hour).UTC()
}

func builtinYear(in *Interp, c *Call) Value {
	return IntV(int64(adjustedUTC(c).Year()))
}

func builtinMonth(in *Interp, c *Call) Value {
	return IntV(int64(adjustedUTC(c).Month()))
}

func builtinDate(in *Interp, c *Call) Value {
	return IntV(int64(adjustedUTC(c).Day()))
}

func builtinDay(in *Interp, c *Call) Value {
	// tm_wday: 0=Sunday..6=Saturday -- same numbering as Go's time.Weekday.
	return IntV(int64(adjustedUTC(c).Weekday()))
}

func builtinHour(in *Interp, c *Call) Value {
	return IntV(int64(adjustedUTC(c).Hour()))
}

func builtinMinute(in *Interp, c *Call) Value {
	return IntV(int64(adjustedUTC(c).Minute()))
}

func builtinSecond(in *Interp, c *Call) Value {
	return IntV(int64(adjustedUTC(c).Second()))
}

func builtinMillisecond(in *Interp, c *Call) Value {
	return IntV(time.Now().UnixMilli())
}
