package az

// dedupeScopes returns a fresh slice holding each scope exactly once, in first
// appearance order.
//
// Returning a new slice matters as much as the deduplication. Callers hand us a
// slice they still own, and appending to it can write through to their backing
// array whenever it has spare capacity, so a token request would silently
// rewrite the caller's configuration. Copying first makes every downstream
// append safe.
func dedupeScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(scopes))
	out := make([]string, 0, len(scopes))
	for _, s := range scopes {
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// withScope returns a fresh scope slice with resource's default scope appended,
// leaving scopes untouched. An empty resource yields a plain copy.
func withScope(scopes []string, resource string) []string {
	out := dedupeScopes(scopes)
	if resource == "" {
		return out
	}
	return dedupeScopes(append(out, resource+"/.default"))
}
