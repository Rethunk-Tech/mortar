package gmcm

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func ParseSetFlag(spec string) (page string, index int, value string, err error) {
	eq := strings.IndexByte(spec, '=')
	if eq <= 0 || eq == len(spec)-1 {
		return "", 0, "", fmt.Errorf("gmcm: --set wants page/index=value")
	}
	loc, value := spec[:eq], spec[eq+1:]
	// GMCM's first page has the empty id, so "/2=true" sets an option on it.
	slash := strings.LastIndexByte(loc, '/')
	if slash < 0 || slash == len(loc)-1 {
		return "", 0, "", fmt.Errorf("gmcm: --set wants page/index=value")
	}
	page = loc[:slash]
	index, err = strconv.Atoi(loc[slash+1:])
	if err != nil {
		return "", 0, "", fmt.Errorf("gmcm: --set index: %w", err)
	}
	return page, index, value, nil
}

// EditFromCapture builds the pending edit that sets one captured option to raw, refusing an option the bridge cannot
// set, a value of the wrong type, one outside the option's range, and one the option does not offer.
func EditFromCapture(menu Capture, page string, index int, raw string) (Edit, error) {
	opt, err := Find(menu, page, index)
	if err != nil {
		return Edit{}, err
	}
	if !opt.Editable {
		return Edit{}, fmt.Errorf("gmcm: %q can only be changed in the game", opt.Name)
	}
	value, err := coerceValue(opt, raw)
	if err != nil {
		return Edit{}, fmt.Errorf("gmcm: %q: %w", opt.Name, err)
	}
	fieldID := ""
	if opt.FieldID != nil {
		fieldID = *opt.FieldID
	}
	return Edit{Page: page, Index: opt.Index, Kind: opt.Kind, FieldID: fieldID, Name: opt.Name, Value: value}, nil
}

// Find is the option at index on page.
func Find(menu Capture, page string, index int) (Option, error) {
	for _, p := range menu.Pages {
		if p.ID != page {
			continue
		}
		for _, opt := range p.Options {
			if opt.Index == index {
				return opt, nil
			}
		}
		return Option{}, fmt.Errorf("gmcm: no option %s/%d", page, index)
	}
	return Option{}, fmt.Errorf("gmcm: no page %q", page)
}

// Text is a captured or pending value as the option's text: true/false, a number, or the string itself.
func Text(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return fmt.Sprint(x)
	}
}

func coerceValue(opt Option, raw string) (any, error) {
	switch opt.Kind {
	case "bool":
		return strconv.ParseBool(raw)
	case "int", "float":
		var n float64
		if opt.Kind == "int" {
			i, err := strconv.Atoi(raw)
			if err != nil {
				return nil, errors.New("not a whole number")
			}
			n = float64(i)
		} else {
			f, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return nil, errors.New("not a number")
			}
			n = f
		}
		if opt.Min != nil && n < *opt.Min || opt.Max != nil && n > *opt.Max {
			return nil, fmt.Errorf("%s is outside %s to %s", raw, Text(deref(opt.Min)), Text(deref(opt.Max)))
		}
		if opt.Kind == "int" {
			return int(n), nil
		}
		return n, nil
	case "choice":
		for _, c := range opt.Choices {
			if Text(c.Value) == raw {
				return c.Value, nil
			}
		}
		return nil, fmt.Errorf("%q is not one of its choices", raw)
	default:
		return raw, nil
	}
}

func deref(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}
