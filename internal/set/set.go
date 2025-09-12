package set

import "strings"

type Set[ValueType comparable] map[ValueType]struct{}

func New[ValueType comparable]() Set[ValueType] {
	return make(Set[ValueType])
}

func (set Set[ValueType]) Add(value ValueType)      { set[value] = struct{}{} }
func (set Set[ValueType]) Has(value ValueType) bool { _, ok := set[value]; return ok }
func (set Set[ValueType]) Remove(value ValueType)   { delete(set, value) }

// String-specific helper as a function (not a method).
func KeysWithPrefix(set Set[string], prefix string) []string {
	out := make([]string, 0)
	for key := range set {
		if strings.HasPrefix(key, prefix) {
			out = append(out, key)
		}
	}
	return out
}
