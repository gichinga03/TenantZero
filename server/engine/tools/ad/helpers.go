package ad

import "strings"

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
