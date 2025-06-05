package filters

import (
	"regexp"
	"strings"
)

type FilterType string

const (
	FilterTypeContains FilterType = "contains"
	FilterTypeRegex    FilterType = "regex"
	FilterTypeExact    FilterType = "exact"
)

type Filter struct {
	Name          string
	Type          FilterType
	Pattern       string
	PriorityBoost int
	CaseSensitive bool
	regex         *regexp.Regexp
}

func NewFilter(name string, filterType FilterType, pattern string, priorityBoost int, caseSensitive bool) (*Filter, error) {
	f := &Filter{
		Name:          name,
		Type:          filterType,
		Pattern:       pattern,
		PriorityBoost: priorityBoost,
		CaseSensitive: caseSensitive,
	}

	if filterType == FilterTypeRegex {
		flags := ""
		if !caseSensitive {
			flags = "(?i)"
		}
		regex, err := regexp.Compile(flags + pattern)
		if err != nil {
			return nil, err
		}
		f.regex = regex
	}

	return f, nil
}

func (f *Filter) Matches(content string) bool {
	switch f.Type {
	case FilterTypeContains:
		if f.CaseSensitive {
			return strings.Contains(content, f.Pattern)
		}
		return strings.Contains(strings.ToLower(content), strings.ToLower(f.Pattern))
	
	case FilterTypeRegex:
		return f.regex.MatchString(content)
	
	case FilterTypeExact:
		if f.CaseSensitive {
			return content == f.Pattern
		}
		return strings.ToLower(content) == strings.ToLower(f.Pattern)
	
	default:
		return false
	}
}