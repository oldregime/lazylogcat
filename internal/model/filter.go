package model

import "strings"

type Level string

const (
	LvlV Level = "V"
	LvlD Level = "D"
	LvlI Level = "I"
	LvlW Level = "W"
	LvlE Level = "E"
	LvlF Level = "F"
)

const lvls = "VDIWEF"

type Filter struct {
	PackageName string
	Level       Level
	Tag         string
	Text        string
}

type Format struct {
	//Single choice
	Brief      bool
	Long       bool
	Process    bool
	Raw        bool
	Tag        bool
	Thread     bool
	Threadtime bool
	Time       bool

	//Multiple choice (conflicts are silently ignored)
	Color       bool //Shows each priority level with a different color.
	Descriptive bool //Shows log buffer event descriptions. This modifier affects event log buffer messages only and has no effect on the other non-binary buffers. The event descriptions come from the event-log-tags database.
	Epoch       bool //Displays time in seconds starting from Jan 1, 1970.
	Monotonic   bool //Displays time in CPU seconds starting from the last boot.
	Printable   bool //Ensures that any binary logging content is escaped.
	Uid         bool //If permitted by access controls, displays the UID or Android ID of the logged process.
	Usec        bool //Displays the time, with precision in microseconds.
	UTC         bool //Displays the time as UTC.
	Year        bool //Adds the year to the displayed time.
	Zone        bool //Adds the local time zone to the displayed time.
}

func (f *Filter) IsEmpty() bool {
	return f.PackageName == "" &&
		(f.Level == "" || f.Level == LvlV) &&
		f.Tag == "" &&
		f.Text == ""
}

func (l Level) Next() Level {
	if l == "" || l == LvlF {
		return LvlV
	}

	i := strings.Index(lvls, string(l))
	if i == -1 {
		return LvlV
	}
	return Level(lvls[i+1])
}

func (f *Format) Value() string {
	switch {
	case f.Brief:
		return "brief"
	case f.Long:
		return "long"
	case f.Process:
		return "process"
	case f.Raw:
		return "raw"
	case f.Tag:
		return "tag"
	case f.Thread:
		return "thread"
	case f.Threadtime:
		return "threadtime"
	case f.Time:
		return "time"
	default:
		return ""
	}
}

func (f *Format) Modifiers() []string {
	var mods []string
	if f.Color {
		mods = append(mods, "color")
	}
	if f.Descriptive {
		mods = append(mods, "descriptive")
	}
	if f.Epoch {
		mods = append(mods, "epoch")
	}
	if f.Monotonic {
		mods = append(mods, "monotonic")
	}
	if f.Printable {
		mods = append(mods, "printable")
	}
	if f.Uid {
		mods = append(mods, "uid")
	}
	if f.Usec {
		mods = append(mods, "usec")
	}
	if f.UTC {
		mods = append(mods, "UTC")
	}
	if f.Year {
		mods = append(mods, "year")
	}
	if f.Zone {
		mods = append(mods, "zone")
	}
	return mods
}
