package ad

import (
	"strings"

	"github.com/0x4D67/tenantzero/shared/schema"
)

// kerberoastHash formats a Kerberoast hash in Hashcat $krb5tgs format.
func kerberoastHash(user, domain, spn, hash string) string {
	return "$krb5tgs$23$*" + user + "$" + domain + "$" + spn + "*$" + hash
}

// dcComponents splits a FQDN into DC= components for LDAP distinguished names.
func dcComponents(domain string) string {
	parts := strings.Split(domain, ".")
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = "DC=" + p
	}
	return strings.Join(out, ",")
}

// flagValue returns the value of a flag from a flag map.
func flagValue(flags map[string]string, key string) string {
	return flags[strings.ToLower(key)]
}

// hasFlag returns true if the flag exists in the map.
func hasFlag(flags map[string]string, key string) bool {
	_, ok := flags[strings.ToLower(key)]
	return ok
}

// effectiveCallers returns all identities the session can act as,
// in both "DOMAIN\user" and bare "user" forms so ACL lookups match
// regardless of how the lab YAML or ticket stores the principal.
func effectiveCallers(state *schema.SessionState) []string {
	seen := map[string]bool{}
	var out []string

	add := func(s string) {
		if s == "" {
			return
		}
		key := strings.ToLower(s)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, s)

		// Also add the bare username variant so "MERIDIAN\awalker" matches "awalker"
		if idx := strings.LastIndexAny(s, `\/`); idx >= 0 {
			bare := s[idx+1:]
			bkey := strings.ToLower(bare)
			if !seen[bkey] {
				seen[bkey] = true
				out = append(out, bare)
			}
		}
	}

	add(state.ActiveUser)

	for _, t := range state.Tokens {
		if t.Domain != "" {
			add(t.Domain + `\` + t.Username)
		} else {
			add(t.Username)
		}
	}

	for _, tk := range state.Tickets {
		// Ticket identity may be bare ("awalker") or qualified ("MERIDIAN\awalker")
		add(tk.Identity)
		// If bare, also add domain-qualified form using ticket target as domain hint
		if !strings.ContainsAny(tk.Identity, `\/`) && tk.Target != "" {
			// tk.Target is the domain (e.g. "MERIDIAN.LOCAL") — extract NetBIOS prefix
			domain := tk.Target
			if idx := strings.Index(domain, "."); idx >= 0 {
				domain = domain[:idx] // "MERIDIAN.LOCAL" -> "MERIDIAN"
			}
			add(domain + `\` + tk.Identity)
		}
	}

	return out
}
