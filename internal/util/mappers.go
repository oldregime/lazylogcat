package util

import (
	"github.com/parfenovvs/lazylogcat/internal/config"
	"github.com/parfenovvs/lazylogcat/internal/model"
)

func FilterFromConfig(c *config.Config) model.Filter {
	return model.Filter{
		PackageName: c.Filter.Pkg.Value,
		Tag:         c.Filter.Tag.Value,
		Text:        c.Filter.Txt.Value,
		Level:       model.LvlV,
	}
}

func ColorFromConfig(c *config.Config) bool {
	if c.Display.Color == nil {
		return true
	}
	return *c.Display.Color
}

func WrapFromConfig(c *config.Config) bool {
	if c.Display.Wrap == nil {
		return true
	}
	return *c.Display.Wrap
}

func ColumnsFromConfig(c *config.Config) model.Columns {
	cols := c.Display.Columns
	if cols == nil {
		return model.Columns{
			Date:    false,
			Time:    true,
			PID:     false,
			TID:     false,
			Level:   true,
			Tag:     true,
			Message: true,
		}
	}
	return model.Columns{
		Date:    boolOrDefault(cols.Date, false),
		Time:    boolOrDefault(cols.Time, true),
		PID:     boolOrDefault(cols.PID, false),
		TID:     boolOrDefault(cols.TID, false),
		Level:   boolOrDefault(cols.Level, true),
		Tag:     boolOrDefault(cols.Tag, true),
		Message: boolOrDefault(cols.Message, true),
	}
}

func boolOrDefault(ptr *bool, def bool) bool {
	if ptr == nil {
		return def
	}
	return *ptr
}
