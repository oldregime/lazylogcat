package util

import (
	"github.com/parfenovvs/lazylogcat/internal/config"
	"github.com/parfenovvs/lazylogcat/internal/model"
)

func FilterFromConfig(c *config.Config) model.Filter {
	return model.Filter{
		PackageName: c.Session.Pkg,
		Tag:         c.Session.Tag,
		Text:        c.Session.Txt,
		Level:       model.LvlV,
	}
}

func FormatFromConfig(c *config.Config) model.Format {
	f := model.Format{}

	switch c.Prefs.Format {
	case "brief":
		f.Brief = true
	case "long":
		f.Long = true
	case "process":
		f.Process = true
	case "raw":
		f.Raw = true
	case "tag":
		f.Tag = true
	case "thread":
		f.Thread = true
	case "threadtime":
		f.Threadtime = true
	case "time":
		f.Time = true
	}

	for _, m := range c.Prefs.Modifiers {
		switch m {
		case "color":
			f.Color = true
		case "descriptive":
			f.Descriptive = true
		case "epoch":
			f.Epoch = true
		case "monotonic":
			f.Monotonic = true
		case "printable":
			f.Printable = true
		case "uid":
			f.Uid = true
		case "usec":
			f.Usec = true
		case "UTC":
			f.UTC = true
		case "year":
			f.Year = true
		case "zone":
			f.Zone = true
		}
	}

	return f
}
