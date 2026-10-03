package gmcm

import (
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
	slash := strings.LastIndexByte(loc, '/')
	if slash <= 0 || slash == len(loc)-1 {
		return "", 0, "", fmt.Errorf("gmcm: --set wants page/index=value")
	}
	page = loc[:slash]
	index, err = strconv.Atoi(loc[slash+1:])
	if err != nil {
		return "", 0, "", fmt.Errorf("gmcm: --set index: %w", err)
	}
	return page, index, value, nil
}

func EditFromCapture(cap Capture, page string, index int, value string) (Edit, error) {
	for _, p := range cap.Pages {
		if p.ID != page {
			continue
		}
		for _, opt := range p.Options {
			if opt.Index != index {
				continue
			}
			fieldID := ""
			if opt.FieldID != nil {
				fieldID = *opt.FieldID
			}
			parsed, err := coerceValue(opt.Kind, value)
			if err != nil {
				return Edit{}, err
			}
			return Edit{
				Page:    p.ID,
				Index:   opt.Index,
				Kind:    opt.Kind,
				FieldID: fieldID,
				Name:    opt.Name,
				Value:   parsed,
			}, nil
		}
		return Edit{}, fmt.Errorf("gmcm: no option %s/%d", page, index)
	}
	return Edit{}, fmt.Errorf("gmcm: no page %s", page)
}

func coerceValue(kind, raw string) (any, error) {
	switch kind {
	case "bool":
		return strconv.ParseBool(raw)
	case "int":
		return strconv.Atoi(raw)
	case "float":
		return strconv.ParseFloat(raw, 64)
	default:
		return raw, nil
	}
}
