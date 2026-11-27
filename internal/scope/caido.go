package scope

import (
	"encoding/json"
	"net/netip"
	"strings"

	"golang.org/x/net/idna"
)

// Caido scope export. Caido's Scope page has no file import yet
// (caido/caido#2022), so the export is its documented configuration file
// (docs.caido.io/app/guides/config, schema
// github.com/caido/schemas proxy-v0.57.0), whose config.project.scopes[]
// are {name, allowlist, denylist}: the object the SDKs' createScope takes
// too. It's JSON, which Caido reads as the YAML it is: `caido-cli --config
// file`.
//
// Entries are host globs as Caido matches them against the Host header or
// IP: lowercase letters, digits, '.', '-', '_', '*' and '?' only, no
// schemes, ports or paths. So:
//   - wildcards and domains go as they are ("*.example.com" keeps
//     HuntDeck's meaning: subdomains, not example.com itself);
//   - an in-scope URL's host is allowed (the path can't be kept);
//   - an out-of-scope URL with a path isn't denied: denying its host would
//     shut out the rest of the host, in scope; it's left out, said why;
//   - IPv4 addresses go as they are, /8, /16, /24 and /32 ranges as globs
//     ("10.1.2.*"); other ranges and IPv6 are left out, said why.

// CaidoVersion is the Caido release the file's format is documented for.
const CaidoVersion = "0.57.0"

// CaidoSkip is a target the export leaves out, and why.
type CaidoSkip struct {
	Value, Why string
	InScope    bool
}

// CaidoLists turns in- and out-of-scope targets into Caido's allowlist and
// denylist, each without duplicates in the targets' order, and the
// targets left out.
func CaidoLists(in, out []Target) (allow, deny []string, skipped []CaidoSkip) {
	add := func(list []string, seen map[string]bool, g string) []string {
		if !seen[g] {
			seen[g] = true
			list = append(list, g)
		}
		return list
	}
	seenA, seenD := map[string]bool{}, map[string]bool{}
	for _, t := range in {
		if g, why := caidoGlob(t, true); why != "" {
			skipped = append(skipped, CaidoSkip{Value: t.Value, Why: why, InScope: true})
		} else {
			allow = add(allow, seenA, g)
		}
	}
	for _, t := range out {
		if g, why := caidoGlob(t, false); why != "" {
			skipped = append(skipped, CaidoSkip{Value: t.Value, Why: why})
		} else {
			deny = add(deny, seenD, g)
		}
	}
	return allow, deny, skipped
}

// caidoGlob is t as a Caido host glob, or why it can't be one.
func caidoGlob(t Target, inScope bool) (string, string) {
	switch t.Kind {
	case KindWildcard, KindDomain:
		return caidoHost(t.Value)
	case KindIP:
		a, err := netip.ParseAddr(t.Value)
		if err != nil || !a.Is4() {
			return "", "IPv6: Caido's scope syntax for it isn't documented"
		}
		return a.String(), ""
	case KindCIDR:
		p, err := netip.ParsePrefix(t.Value)
		if err != nil || !p.Addr().Is4() {
			return "", "IPv6 range: Caido's scope syntax for it isn't documented"
		}
		octets := strings.Split(p.Masked().Addr().String(), ".")
		switch p.Bits() {
		case 32:
			return strings.Join(octets, "."), ""
		case 24:
			return strings.Join(octets[:3], ".") + ".*", ""
		case 16:
			return strings.Join(octets[:2], ".") + ".*.*", ""
		case 8:
			return octets[0] + ".*.*.*", ""
		}
		return "", "IP range a host glob can't express (only /8, /16, /24 and /32 can be)"
	case KindURL:
		u, ok := parseTargetURL(t.Value)
		if !ok {
			return "", "not a URL HuntDeck can read"
		}
		path := strings.TrimSuffix(u.EscapedPath(), "*")
		if !inScope && (path != "" && path != "/" || u.RawQuery != "") {
			return "", "out-of-scope path: Caido scopes have no paths, and denying the whole host would shut out the rest of it (use a filter preset)"
		}
		host := u.Hostname()
		if a, err := netip.ParseAddr(host); err == nil {
			return caidoGlob(Target{Kind: KindIP, Value: a.String()}, inScope)
		}
		return caidoHost(host)
	}
	return "", "not a host"
}

// caidoHost is a hostname (wildcards allowed) as a Caido host glob:
// lowercase, international names in their ASCII form, nothing Caido
// doesn't accept.
func caidoHost(h string) (string, string) {
	h = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(h)), ".")
	if !idnaSized(h) {
		return "", "a hostname Caido can't take"
	}
	labels := strings.Split(h, ".")
	for i, l := range labels {
		if strings.ContainsAny(l, "*?") || isASCII(l) {
			continue
		}
		a, err := idna.ToASCII(l)
		if err != nil {
			return "", "a hostname Caido can't take"
		}
		labels[i] = a
	}
	h = strings.Join(labels, ".")
	if h == "" || strings.Trim(h, "*.?") == "" {
		return "", "a hostname Caido can't take"
	}
	for _, r := range h {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && !strings.ContainsRune(".-_*?", r) {
			return "", "a hostname Caido can't take"
		}
	}
	return h, ""
}

// CaidoConfig is a Caido configuration file holding one scope preset named
// name (in a project named project), with allow and deny.
func CaidoConfig(project, name string, allow, deny []string) ([]byte, error) {
	type scope struct {
		Name      string   `json:"name"`
		Allowlist []string `json:"allowlist"`
		Denylist  []string `json:"denylist"`
	}
	type projectT struct {
		Name   string  `json:"name"`
		Scopes []scope `json:"scopes"`
	}
	doc := struct {
		Version string `json:"version"`
		Config  struct {
			Project projectT `json:"project"`
		} `json:"config"`
	}{Version: CaidoVersion}
	if allow == nil {
		allow = []string{}
	}
	if deny == nil {
		deny = []string{}
	}
	doc.Config.Project = projectT{Name: project, Scopes: []scope{{Name: name, Allowlist: allow, Denylist: deny}}}
	b, err := json.MarshalIndent(doc, "", "  ")
	return append(b, '\n'), err
}
