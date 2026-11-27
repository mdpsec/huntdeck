// Package scope turns platform scope identifiers into normalised targets and
// matches hosts against them.
package scope

import (
	"net"
	"net/netip"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/idna"
	"golang.org/x/net/publicsuffix"
)

// Normalised asset types shared by all platforms.
const (
	AssetWildcard      = "wildcard"
	AssetDomain        = "domain"
	AssetURL           = "url"
	AssetAPI           = "api"
	AssetIP            = "ip"
	AssetCIDR          = "cidr"
	AssetAndroid       = "android"
	AssetIOS           = "ios"
	AssetWindows       = "windows"
	AssetSourceCode    = "source_code"
	AssetExecutable    = "executable"
	AssetHardware      = "hardware"
	AssetSmartContract = "smart_contract"
	AssetAIModel       = "ai_model"
	AssetOther         = "other"
)

// Target kinds, as used by exports.
const (
	KindWildcard = "wildcard"
	KindDomain   = "domain"
	KindURL      = "url"
	KindIP       = "ip"
	KindCIDR     = "cidr"
)

// Kinds lists every target kind, in export order.
var Kinds = []string{KindWildcard, KindDomain, KindURL, KindIP, KindCIDR}

// KindRoot selects the root (registrable) domains of wildcard, domain and URL
// targets, for recon. It isn't a kind of target: a root domain is often not
// in scope itself (app.example.com in scope doesn't put example.com in).
const KindRoot = "root"

// KindApp selects Android and iOS apps, which the hunt board scores beside
// targets. It isn't a kind of target either: an app has no host to match or
// export.
const KindApp = "app"

// Target is one tool-ready value derived from a scope identifier.
type Target struct {
	Kind  string
	Value string // what exports print, e.g. "*.example.com" or "https://a.example.com/api"
	Host  string // hostname or base domain of a wildcard; IP for ip; empty for cidr
	Apex  string // registrable domain (eTLD+1), when there is one
}

// WebType classifies a web identifier for platforms that use one category
// for wildcards, hosts and URLs alike: "*.example.com" and
// "https://*.example.com/*" are wildcards, anything with a scheme or a path
// is a URL, a bare host is a domain, and prose is "other".
func WebType(ident string) string {
	rest, hadScheme := strings.CutPrefix(strings.TrimSpace(ident), "https://")
	if !hadScheme {
		rest, hadScheme = strings.CutPrefix(rest, "http://")
	}
	host, path, _ := strings.Cut(rest, "/")
	switch {
	case host == "" || strings.ContainsAny(host, " \t"):
		return AssetOther
	case strings.Contains(host, "*"):
		return AssetWildcard
	case hadScheme || (path != "" && path != "*"):
		return AssetURL
	default:
		return AssetDomain
	}
}

// webAssetTypes are the asset types whose identifiers can contain hosts.
var webAssetTypes = map[string]bool{
	AssetWildcard: true, AssetDomain: true, AssetURL: true, AssetAPI: true,
	AssetIP: true, AssetCIDR: true, AssetOther: true,
}

// Version identifies what Targets derives from an identifier. Bump it with
// any change to that: targets stored by an older version are derived again
// when HuntDeck starts.
const Version = 6

var (
	tokenSplit = regexp.MustCompile(`[\s,;|]+`)
	// altGroup is an alternation such as "(fr|de|co.uk)".
	altGroup = regexp.MustCompile(`\(([^()\s|]*(?:\|[^()\s|]*)+)\)`)
	// braceGroup is one in brace notation, "{com,co.uk,cl}" (spaces after
	// the commas allowed), read as an altGroup. Empty alternatives, as the
	// stray last one of "{com,co.uk,}", are dropped.
	braceGroup = regexp.MustCompile(`\{\s*([^{}\s,|()]*(?:\s*,\s*[^{}\s,|()]*)+)\s*\}`)
	// starGap is the stray blank in "*. example.com".
	starGap = regexp.MustCompile(`\*\.\s+`)
)

// maxVariants bounds how many identifiers alternations expand to.
const maxVariants = 64

// Targets extracts targets from one identifier. Identifiers often hold
// several values ("a.com, b.com") or prose, so it tokenises and keeps what
// parses. For "other" assets only hostnames under a real public suffix are
// accepted, so prose like "Node.js" isn't mistaken for a domain.
func Targets(assetType, identifier string) []Target {
	if !webAssetTypes[assetType] {
		return nil
	}
	strict := assetType == AssetOther
	var out []Target
	seen := map[Target]bool{}
	for _, v := range variants(identifier) {
		prev := "" // the token before, if it named a host alone, for suffixSwap
		seps := tokenSplit.FindAllString(v, -1)
		for i, tok := range tokenSplit.Split(v, -1) {
			if i > 0 && strings.Contains(seps[i-1], ",") {
				if swapped, ok := suffixSwap(prev, tok); ok {
					tok = swapped
				}
			}
			ts := parseToken(tok, strict)
			prev = ""
			if len(ts) > 0 && !strings.ContainsAny(tok, "/:") {
				prev = tok
			}
			for _, t := range ts {
				if !seen[t] {
					seen[t] = true
					out = append(out, t)
				}
			}
		}
	}
	return out
}

// suffixSwap reads a bare public suffix listed after a host as that host
// under it: in "*.example.co.uk,.com,.uk", ".com" is *.example.com and
// ".uk" *.example.uk (as the next token, prev is the swapped host).
func suffixSwap(prev, tok string) (string, bool) {
	tok = strings.ToLower(strings.TrimSpace(tok))
	if prev == "" || len(tok) < 3 || tok[0] != '.' {
		return "", false
	}
	if ps, icann := publicsuffix.PublicSuffix(tok[1:]); !icann || ps != tok[1:] {
		return "", false
	}
	host := strings.ToLower(strings.TrimSuffix(prev, "."))
	ps, icann := publicsuffix.PublicSuffix(strings.TrimPrefix(host, "*."))
	if !icann || len(host) <= len(ps)+1 {
		return "", false
	}
	return host[:len(host)-len(ps)] + tok[1:], true
}

// variants mends "(*).example.com", "\*.example.com" and "*. example.com"
// and expands alternations: "www.example.(com|co.uk)" and
// "www.example.{com,co.uk}" name www.example.com and www.example.co.uk.
// Past maxVariants the remaining groups are left as written.
func variants(s string) []string {
	s = strings.NewReplacer("(*)", "*", `\*`, "*").Replace(s)
	s = braceGroup.ReplaceAllStringFunc(s, func(g string) string {
		var alts []string
		for a := range strings.SplitSeq(strings.Trim(g, "{} \t"), ",") {
			if a = strings.TrimSpace(a); a != "" {
				alts = append(alts, a)
			}
		}
		switch len(alts) {
		case 0:
			return g
		case 1:
			return alts[0]
		}
		return "(" + strings.Join(alts, "|") + ")"
	})
	s = starGap.ReplaceAllString(s, "*.")
	out := []string{s}
	for i := 0; i < len(out); {
		loc := altGroup.FindStringSubmatchIndex(out[i])
		alts := 0
		if loc != nil {
			alts = strings.Count(out[i][loc[2]:loc[3]], "|") + 1
		}
		if loc == nil || len(out)-1+alts > maxVariants {
			i++
			continue
		}
		v := out[i]
		expanded := make([]string, 0, alts)
		for a := range strings.SplitSeq(v[loc[2]:loc[3]], "|") {
			expanded = append(expanded, v[:loc[0]]+a+v[loc[1]:])
		}
		out = slices.Replace(out, i, i+1, expanded...)
	}
	return out
}

func parseToken(tok string, strict bool) []Target {
	tok = strings.Trim(tok, `()[]{}<>"'.,;:`+"`")
	if tok == "" || tok == "*" {
		return nil
	}
	if p, err := netip.ParsePrefix(tok); err == nil {
		return []Target{{Kind: KindCIDR, Value: p.Masked().String()}}
	}
	if a, err := netip.ParseAddr(tok); err == nil {
		return []Target{{Kind: KindIP, Value: a.String(), Host: a.String()}}
	}

	var host, urlValue string
	if strings.Contains(tok, "://") {
		// url.Parse rejects '*' in hosts, so swap it for a placeholder first.
		u, err := url.Parse(strings.ReplaceAll(tok, "*", "wildcard-placeholder"))
		if err != nil || u.Hostname() == "" {
			return nil
		}
		host = strings.ReplaceAll(u.Hostname(), "wildcard-placeholder", "*")
		path := strings.ReplaceAll(u.EscapedPath(), "wildcard-placeholder", "*")
		if (path != "" && path != "/" && path != "/*") || u.RawQuery != "" {
			urlValue = strings.ToLower(u.Scheme) + "://" + strings.ToLower(strings.ReplaceAll(u.Host, "wildcard-placeholder", "*")) + path
			if u.RawQuery != "" {
				urlValue += "?" + u.RawQuery
			}
		}
	} else {
		var path string
		host, path, _ = strings.Cut(tok, "/")
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		// "github.com/org/*" without a scheme is a path rule too.
		if path != "" && path != "*" {
			urlValue = strings.ToLower(host) + "/" + path
		}
	}

	host = trimTrailingStar(host)

	// A rule with a path covers only that path: it's a URL target alone,
	// never its whole host as a domain.
	if a, err := netip.ParseAddr(host); err == nil {
		if urlValue != "" {
			return []Target{{Kind: KindURL, Value: urlValue, Host: a.String()}}
		}
		return []Target{{Kind: KindIP, Value: a.String(), Host: a.String()}}
	}

	if strings.Contains(host, "*") {
		t, ok := wildcard(host, strict)
		if !ok {
			return nil
		}
		if urlValue != "" {
			return []Target{{Kind: KindURL, Value: urlValue, Host: t.Host, Apex: t.Apex}}
		}
		return []Target{t}
	}

	h, ok := normaliseHost(host, strict)
	if !ok {
		return nil
	}
	if suffix, icann := publicsuffix.PublicSuffix(h); icann && suffix == h {
		return nil // "co.uk" alone is a suffix, not a host ("*.gov.sg" is fine)
	}
	apex := apexOf(h)
	if urlValue != "" {
		return []Target{{Kind: KindURL, Value: urlValue, Host: h, Apex: apex}}
	}
	return []Target{{Kind: KindDomain, Value: h, Host: h, Apex: apex}}
}

// trimTrailingStar drops a '*' glued to a host ending in a short TLD, as in
// "www.example.com*": it stands for what follows the host (a port, a path).
// After a longer label, as in "shop.example*", it stays part of a pattern.
func trimTrailingStar(host string) string {
	h, ok := strings.CutSuffix(host, "*")
	if !ok {
		return host
	}
	tld := strings.ToLower(h[strings.LastIndex(h, ".")+1:])
	if len(tld) < 2 || len(tld) > 3 || strings.Trim(tld, "abcdefghijklmnopqrstuvwxyz") != "" {
		return host
	}
	return h
}

var (
	patternRE      = regexp.MustCompile(`^[a-z0-9_*-]+(\.[a-z0-9_*-]+)*$`)
	patternLabelRE = regexp.MustCompile(`^[a-z0-9_*-]+$`)
)

// wildcard normalises patterns like "*.example.com", "*example.com",
// "api.*.example.com", "*-api.example.com" or "example.*". Its host is the
// domain after the last label holding a '*'. A pattern without one, such as
// "example.*" or "www.example-*.com", is kept for exports but has no host to
// match against.
func wildcard(host string, strict bool) (Target, bool) {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if rest, ok := strings.CutPrefix(host, "*"); ok && !strings.Contains(rest, "*") && !strings.HasPrefix(rest, ".") {
		if _, ok := normaliseHost(rest, strict); ok {
			host = "*." + rest // "*example.com" and "*.example.com" mean the same in practice
		}
	}
	labels := strings.Split(host, ".")
	last := 0
	for i, l := range labels {
		if strings.Contains(l, "*") {
			last = i
		}
	}
	prefix := labels[:last+1]
	if b, ok := normaliseHost(strings.Join(labels[last+1:], "."), strict); ok {
		for _, l := range prefix {
			if !patternLabelRE.MatchString(l) {
				return Target{}, false
			}
		}
		return Target{Kind: KindWildcard, Value: strings.Join(prefix, ".") + "." + b, Host: b, Apex: apexOf(b)}, true
	}
	if strict || !patternRE.MatchString(host) || !slices.ContainsFunc(labels[:len(labels)-1], func(l string) bool { return strings.Trim(l, "*") != "" }) {
		return Target{}, false
	}
	// In "blog*example.com" the host is what follows the '*'.
	if b, ok := normaliseHost(strings.TrimPrefix(host[strings.LastIndex(host, "*")+1:], "."), false); ok {
		return Target{Kind: KindWildcard, Value: host, Host: b, Apex: apexOf(b)}, true
	}
	return Target{Kind: KindWildcard, Value: host}, true
}

var labelRE = regexp.MustCompile(`^[a-z0-9_]([a-z0-9_-]{0,61}[a-z0-9_])?$`)

// normaliseHost lowercases, converts IDNs to punycode and validates. A host
// needs at least two labels and a plausible TLD; in strict mode the TLD must
// be on the public suffix list.
func normaliseHost(host string, strict bool) (string, bool) {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	if !idnaSized(h) {
		return "", false
	}
	if a, err := idna.Lookup.ToASCII(h); err == nil {
		h = a
	} else if !isASCII(h) {
		return "", false
	}
	if len(h) > 253 {
		return "", false
	}
	labels := strings.Split(h, ".")
	if len(labels) < 2 {
		return "", false
	}
	for _, l := range labels {
		if !labelRE.MatchString(l) {
			return "", false
		}
	}
	tld := labels[len(labels)-1]
	if len(tld) < 2 || strings.Trim(tld, "abcdefghijklmnopqrstuvwxyz") != "" && !strings.HasPrefix(tld, "xn--") {
		return "", false
	}
	if strict {
		suffix, icann := publicsuffix.PublicSuffix(h)
		if !icann && !strings.Contains(suffix, ".") {
			return "", false
		}
	}
	return h, true
}

// RootDomain returns the registrable domain (eTLD+1) of a normalised host,
// or "" if it has none (an IP, a bare suffix).
func RootDomain(host string) string { return apexOf(host) }

func apexOf(h string) string {
	a, err := publicsuffix.EffectiveTLDPlusOne(h)
	if err != nil {
		return ""
	}
	return a
}

// IDNA bounds, in characters: four times what DNS allows (63 a label, 253
// a name), so nothing that maps to a valid name is turned away, while the
// conversion, whose punycode step grows with the square of a label's
// length, stays cheap on input anyone can send (a 40,000-character line
// took seconds).
const (
	idnaMaxLabel = 4 * 63
	idnaMaxName  = 4 * 253
)

// idnaSized reports whether h is short enough to convert.
func idnaSized(h string) bool {
	if utf8.RuneCountInString(h) > idnaMaxName {
		return false
	}
	for l := range strings.SplitSeq(h, ".") {
		if utf8.RuneCountInString(l) > idnaMaxLabel {
			return false
		}
	}
	return true
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// HiddenName reports whether an asset's identifier is a name the platform
// blacked out ("████████": Bugcrowd does on some engagements, paused ones
// among them) rather than a name.
func HiddenName(identifier string) bool {
	seen := false
	for _, r := range identifier {
		switch r {
		case '█':
			seen = true
		case ' ', '\t', '.':
		default:
			return false
		}
	}
	return seen
}

// HiddenLabel is what to show for a hidden name.
const HiddenLabel = "name hidden by the platform"
