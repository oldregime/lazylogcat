package util

import (
	"github.com/parfenovvs/lazylogcat/internal/config"
	"github.com/parfenovvs/lazylogcat/internal/model"
)

func FilterFromConfig(c *config.Config) model.Filter {
	pkg := textFilterFromConfig(c.Filter.Pkg)
	tag := textFilterFromConfig(c.Filter.Tag)
	txt := textFilterFromConfig(c.Filter.Txt)
	return model.Filter{
		PackageName: pkg,
		Tag:         tag,
		Text:        txt,
		Level:       model.LvlV,
	}
}

// textFilterFromConfig converts a config.TextFilter to a model.TextFilter,
// mapping the mode string and pre-compiling regex patterns.
func textFilterFromConfig(cf config.TextFilter) model.TextFilter {
	tf := model.TextFilter{
		Value: cf.Value,
		Mode:  modeFromConfig(cf.Mode),
	}
	tf.Compile()
	return tf
}

// modeFromConfig maps a config mode string to a model.TextFilterMode.
// Unknown values default to FilterModeContains.
func modeFromConfig(s string) model.TextFilterMode {
	switch s {
	case "exact":
		return model.FilterModeExact
	case "regex":
		return model.FilterModeRegex
	default:
		return model.FilterModeContains
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
