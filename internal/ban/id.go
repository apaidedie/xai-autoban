package ban

import (
	"path/filepath"
	"strings"
)

func AuthIDAliases(id string) []string {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	out := []string{id}
	base := filepath.Base(id)
	if base != "" && base != id {
		out = append(out, base)
	}
	if strings.HasSuffix(strings.ToLower(base), ".json") {
		out = append(out, strings.TrimSuffix(base, filepath.Ext(base)))
	}
	seen := map[string]struct{}{}
	uniq := make([]string, 0, len(out))
	for _, v := range out {
		k := strings.ToLower(v)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		uniq = append(uniq, v)
	}
	return uniq
}

func AuthIDsEqual(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if strings.EqualFold(a, b) {
		return true
	}
	ab, bb := filepath.Base(a), filepath.Base(b)
	if strings.EqualFold(ab, bb) {
		return true
	}
	strip := func(s string) string {
		s = filepath.Base(s)
		if strings.HasSuffix(strings.ToLower(s), ".json") {
			s = strings.TrimSuffix(s, filepath.Ext(s))
		}
		return s
	}
	return strings.EqualFold(strip(a), strip(b))
}
