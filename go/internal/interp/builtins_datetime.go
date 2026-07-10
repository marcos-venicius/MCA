package interp

import (
	"time"

	"mca/internal/ast"
)

// TODO: I don't know if it still make sense the way it's currently designed.
//       Should I have a date type?

func builtinTime(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	return IntV(time.Now().Unix())
}

// adjustedUTC is the year/month/date/day/hour/minute/second builtins'
// shared pattern: an integer *hour* offset added to the current wall-clock
// time, then read back out via UTC (gmtime), not local time.
func adjustedUTC(in *Interp, arg ast.Expr) time.Time {
	offset := intOf(expectKind(arg, in.Eval(arg).Value, KInt))
	return time.Now().Add(time.Duration(offset) * time.Hour).UTC()
}

func builtinYear(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	return IntV(int64(adjustedUTC(in, args[0]).Year()))
}

func builtinMonth(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	return IntV(int64(adjustedUTC(in, args[0]).Month()))
}

func builtinDate(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	return IntV(int64(adjustedUTC(in, args[0]).Day()))
}

func builtinDay(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	// tm_wday: 0=Sunday..6=Saturday -- same numbering as Go's time.Weekday.
	return IntV(int64(adjustedUTC(in, args[0]).Weekday()))
}

func builtinHour(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	return IntV(int64(adjustedUTC(in, args[0]).Hour()))
}

func builtinMinute(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	return IntV(int64(adjustedUTC(in, args[0]).Minute()))
}

func builtinSecond(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	return IntV(int64(adjustedUTC(in, args[0]).Second()))
}

func builtinMillisecond(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	return IntV(time.Now().UnixMilli())
}
