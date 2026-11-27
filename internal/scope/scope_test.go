package scope

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestTargets(t *testing.T) {
	tests := []struct {
		name       string
		assetType  string
		identifier string
		want       []Target
	}{
		{"brace alternation", AssetWildcard, "*.brand.{com,co.uk, cl}",
			[]Target{{KindWildcard, "*.brand.com", "brand.com", "brand.com"}, {KindWildcard, "*.brand.co.uk", "brand.co.uk", "brand.co.uk"},
				{KindWildcard, "*.brand.cl", "brand.cl", "brand.cl"}}},
		{"sibling suffixes listed after a host", AssetWildcard, "*.nymhair.co.uk,.com,.uk",
			[]Target{{KindWildcard, "*.nymhair.co.uk", "nymhair.co.uk", "nymhair.co.uk"}, {KindWildcard, "*.nymhair.com", "nymhair.com", "nymhair.com"},
				{KindWildcard, "*.nymhair.uk", "nymhair.uk", "nymhair.uk"}}},
		{"a suffix in prose stays prose", AssetOther, "api.example.com using .NET, and app.example.com .net",
			[]Target{{KindDomain, "api.example.com", "api.example.com", "example.com"}, {KindDomain, "app.example.com", "app.example.com", "example.com"}}},
		{"brace alternation with an empty one", AssetWildcard, "*.nymhair.{com,co.uk,uk,}",
			[]Target{{KindWildcard, "*.nymhair.com", "nymhair.com", "nymhair.com"}, {KindWildcard, "*.nymhair.co.uk", "nymhair.co.uk", "nymhair.co.uk"},
				{KindWildcard, "*.nymhair.uk", "nymhair.uk", "nymhair.uk"}}},
		{"brace with one alternative", AssetWildcard, "*.solo.{com,}", []Target{{KindWildcard, "*.solo.com", "solo.com", "solo.com"}}},
		{"path with a wildcard", AssetURL, "https://github.com/example-org/*",
			[]Target{{KindURL, "https://github.com/example-org/*", "github.com", "github.com"}}},
		{"path without a scheme", AssetURL, "github.com/example-org/*",
			[]Target{{KindURL, "github.com/example-org/*", "github.com", "github.com"}}},
		{"host-wide url", AssetURL, "https://app.example.com/*",
			[]Target{{KindDomain, "app.example.com", "app.example.com", "example.com"}}},
		{"wildcard host with a path", AssetURL, "https://*.example.com/api/",
			[]Target{{KindURL, "https://*.example.com/api/", "example.com", "example.com"}}},
		{"ip with a path", AssetURL, "http://192.0.2.7/admin",
			[]Target{{KindURL, "http://192.0.2.7/admin", "192.0.2.7", ""}}},
		{"plain domain", AssetURL, "app.example.com",
			[]Target{{KindDomain, "app.example.com", "app.example.com", "example.com"}}},
		{"url with scheme only", AssetURL, "https://App.Example.com/",
			[]Target{{KindDomain, "app.example.com", "app.example.com", "example.com"}}},
		// A path rule covers its path only: never its whole host.
		{"url with path", AssetURL, "https://api.example.com/v2/users",
			[]Target{
				{KindURL, "https://api.example.com/v2/users", "api.example.com", "example.com"},
			}},
		{"host with port", AssetURL, "shop.example.com:8443",
			[]Target{{KindDomain, "shop.example.com", "shop.example.com", "example.com"}}},
		{"wildcard", AssetWildcard, "*.example.com",
			[]Target{{KindWildcard, "*.example.com", "example.com", "example.com"}}},
		{"wildcard in url", AssetWildcard, "https://*.example.co.uk/*",
			[]Target{{KindWildcard, "*.example.co.uk", "example.co.uk", "example.co.uk"}}},
		{"wildcard without dot", AssetWildcard, "*example.com",
			[]Target{{KindWildcard, "*.example.com", "example.com", "example.com"}}},
		{"inner wildcard", AssetWildcard, "api.*.example.com",
			[]Target{{KindWildcard, "api.*.example.com", "example.com", "example.com"}}},
		{"tld wildcard", AssetWildcard, "example.*",
			[]Target{{KindWildcard, "example.*", "", ""}}},
		{"tld wildcard with subdomains", AssetWildcard, "*.example.*",
			[]Target{{KindWildcard, "*.example.*", "", ""}}},
		{"blank after star", AssetWildcard, "*. example.com",
			[]Target{{KindWildcard, "*.example.com", "example.com", "example.com"}}},
		{"starred group", AssetWildcard, "(*).example.com",
			[]Target{{KindWildcard, "*.example.com", "example.com", "example.com"}}},
		{"star after the host", AssetWildcard, "*.example.com*",
			[]Target{{KindWildcard, "*.example.com", "example.com", "example.com"}}},
		{"star after a domain", AssetURL, "www.example.com*",
			[]Target{{KindDomain, "www.example.com", "www.example.com", "example.com"}}},
		{"star after a longer label", AssetURL, "shop.example*",
			[]Target{{KindWildcard, "shop.example*", "", ""}}},
		{"partial label", AssetWildcard, "*-vpn.example.com",
			[]Target{{KindWildcard, "*-vpn.example.com", "example.com", "example.com"}}},
		{"star inside the last label", AssetWildcard, "blog*example.com",
			[]Target{{KindWildcard, "blog*example.com", "example.com", "example.com"}}},
		{"escaped star", AssetWildcard, `\*.example.com`,
			[]Target{{KindWildcard, "*.example.com", "example.com", "example.com"}}},
		{"partial label without a host", AssetURL, "www.example-*.com",
			[]Target{{KindWildcard, "www.example-*.com", "", ""}}},
		{"star on a suffix only", AssetWildcard, "*.com", nil},
		{"public suffixes aren't hosts", AssetWildcard, "*.example.com,uk,co.uk",
			[]Target{{KindWildcard, "*.example.com", "example.com", "example.com"}}},
		{"wildcard on a public suffix", AssetWildcard, "*.gov.sg",
			[]Target{{KindWildcard, "*.gov.sg", "gov.sg", ""}}},
		{"alternation", AssetURL, "https://www.example.(com|co.uk)/",
			[]Target{
				{KindDomain, "www.example.com", "www.example.com", "example.com"},
				{KindDomain, "www.example.co.uk", "www.example.co.uk", "example.co.uk"},
			}},
		{"alternations", AssetWildcard, "(*.a|*.b).example.(com|net)",
			[]Target{
				{KindWildcard, "*.a.example.com", "a.example.com", "example.com"},
				{KindWildcard, "*.a.example.net", "a.example.net", "example.net"},
				{KindWildcard, "*.b.example.com", "b.example.com", "example.com"},
				{KindWildcard, "*.b.example.net", "b.example.net", "example.net"},
			}},
		{"alternation inside a label", AssetURL, "shop(1|2).example.com",
			[]Target{
				{KindDomain, "shop1.example.com", "shop1.example.com", "example.com"},
				{KindDomain, "shop2.example.com", "shop2.example.com", "example.com"},
			}},
		{"list with commas", AssetURL, "a.example.com, b.example.com,a.example.com",
			[]Target{
				{KindDomain, "a.example.com", "a.example.com", "example.com"},
				{KindDomain, "b.example.com", "b.example.com", "example.com"},
			}},
		{"ip", AssetIP, "192.0.2.10", []Target{{KindIP, "192.0.2.10", "192.0.2.10", ""}}},
		{"cidr normalised", AssetCIDR, "192.0.2.77/24", []Target{{KindCIDR, "192.0.2.0/24", "", ""}}},
		{"ipv6", AssetIP, "2001:db8::1", []Target{{KindIP, "2001:db8::1", "2001:db8::1", ""}}},
		{"idn to punycode", AssetDomain, "bücher.example",
			[]Target{{KindDomain, "xn--bcher-kva.example", "xn--bcher-kva.example", "xn--bcher-kva.example"}}},
		{"other with prose keeps real domains", AssetOther, "Any host under example.com (e.g. Node.js apps, v2.0)",
			[]Target{{KindDomain, "example.com", "example.com", "example.com"}}},
		{"other prose only", AssetOther, "Anything not listed above, i.e. etc.", nil},
		{"mobile app ignored", AssetAndroid, "com.example.app", nil},
		{"source code ignored", AssetSourceCode, "https://github.com/example/repo", nil},
		{"single label rejected", AssetURL, "localhost", nil},
		{"bare star ignored", AssetWildcard, "*", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Targets(tt.assetType, tt.identifier)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Targets(%q, %q)\n got  %+v\n want %+v", tt.assetType, tt.identifier, got, tt.want)
			}
		})
	}
}

func rules(id int64, in bool, assetType, identifier string) []Rule {
	var out []Rule
	for _, t := range Targets(assetType, identifier) {
		out = append(out, Rule{ProgramID: id, InScope: in, Target: t})
	}
	return out
}

func TestMatch(t *testing.T) {
	var all []Rule
	all = append(all, rules(1, true, AssetWildcard, "*.example.com")...)
	all = append(all, rules(1, false, AssetURL, "admin.example.com")...)
	all = append(all, rules(1, false, AssetWildcard, "*.internal.example.com")...)
	all = append(all, rules(1, true, AssetURL, "api.internal.example.com")...)
	all = append(all, rules(2, true, AssetURL, "www.example.com")...)
	all = append(all, rules(3, true, AssetCIDR, "192.0.2.0/24")...)
	all = append(all, rules(3, false, AssetIP, "192.0.2.5")...)
	all = append(all, rules(4, true, AssetWildcard, "*-vpn.example.org")...)
	all = append(all, rules(4, true, AssetWildcard, "api.*.example.net")...)

	type verdict struct {
		program int64
		in      bool
	}
	tests := []struct {
		input string
		want  []verdict
	}{
		{"shop.example.com", []verdict{{1, true}}},
		{"https://shop.example.com/login", []verdict{{1, true}}},
		{"example.com", nil}, // wildcard covers subdomains only
		{"admin.example.com", []verdict{{1, false}}},
		{"x.internal.example.com", []verdict{{1, false}}},
		{"api.internal.example.com", []verdict{{1, true}}}, // exact in-scope beats OOS wildcard
		{"www.example.com", []verdict{{1, true}, {2, true}}},
		{"WWW.Example.COM:443", []verdict{{1, true}, {2, true}}},
		{"192.0.2.9", []verdict{{3, true}}},
		{"192.0.2.5", []verdict{{3, false}}},
		{"other.test", nil},
		{"eu-vpn.example.org", []verdict{{4, true}}},
		{"a.eu-vpn.example.org", []verdict{{4, true}}},
		{"www.example.org", nil}, // the pattern, not everything under its host
		{"api.x.example.net", []verdict{{4, true}}},
		{"web.x.example.net", nil},
	}
	for _, tt := range tests {
		got := Match(all, tt.input)
		var gotV []verdict
		for _, m := range got {
			gotV = append(gotV, verdict{m.ProgramID, m.InScope})
		}
		if !reflect.DeepEqual(gotV, tt.want) {
			t.Errorf("Match(%q) = %v, want %v", tt.input, gotV, tt.want)
		}
		// Loading only the candidate rules, as the store does, must not change the verdicts.
		if filtered := Match(candidateRules(all, tt.input), tt.input); !reflect.DeepEqual(filtered, got) {
			t.Errorf("Match(%q) on candidate rules = %+v, want %+v", tt.input, filtered, got)
		}
	}
}

// candidateRules mirrors store.MatchRules' SQL filter.
func candidateRules(all []Rule, input string) []Rule {
	hosts, cidrs, ok := Candidates(input)
	if !ok {
		return nil
	}
	var out []Rule
	for _, r := range all {
		if slices.Contains(hosts, r.Target.Host) || (cidrs && r.Target.Kind == KindCIDR) {
			out = append(out, r)
		}
	}
	return out
}

func TestCandidates(t *testing.T) {
	tests := []struct {
		input string
		hosts []string
		cidrs bool
		ok    bool
	}{
		{"https://A.b.Example.com:8443/x", []string{"a.b.example.com", "b.example.com", "example.com", "com"}, false, true},
		{"example.com", []string{"example.com", "com"}, false, true},
		{"192.0.2.7", []string{"192.0.2.7"}, true, true},
		{"[2001:db8::1]:443", []string{"2001:db8::1"}, true, true},
		{"not a host", nil, false, false},
	}
	for _, tt := range tests {
		hosts, cidrs, ok := Candidates(tt.input)
		if !reflect.DeepEqual(hosts, tt.hosts) || cidrs != tt.cidrs || ok != tt.ok {
			t.Errorf("Candidates(%q) = %v, %v, %v; want %v, %v, %v", tt.input, hosts, cidrs, ok, tt.hosts, tt.cidrs, tt.ok)
		}
	}
}

func TestMatchTieGoesOutOfScope(t *testing.T) {
	var all []Rule
	all = append(all, rules(1, true, AssetURL, "app.example.com")...)
	all = append(all, rules(1, false, AssetURL, "app.example.com")...)
	got := Match(all, "app.example.com")
	if len(got) != 1 || got[0].InScope {
		t.Errorf("Match = %+v; want out of scope on a tie", got)
	}
}

func TestWebType(t *testing.T) {
	tests := map[string]string{
		"*.example.com":                  AssetWildcard,
		"https://*.example.com/*":        AssetWildcard,
		"example.com":                    AssetDomain,
		"example.com/*":                  AssetDomain,
		"https://example.com":            AssetURL,
		"http://example.com/x":           AssetURL,
		"example.com/admin":              AssetURL,
		"Main website (see description)": AssetOther,
		"":                               AssetOther,
	}
	for in, want := range tests {
		if got := WebType(in); got != want {
			t.Errorf("WebType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHiddenName(t *testing.T) {
	for in, want := range map[string]bool{"████████": true, "██ ███.███": true, "": false, "a.example.com": false, "█a": false} {
		if got := HiddenName(in); got != want {
			t.Errorf("HiddenName(%q) = %v", in, got)
		}
	}
}

// A long line of distinct non-ASCII characters is refused before the IDNA
// conversion, whose cost grows with the square of a label's length: it
// took seconds before the bound. Names up to the DNS limits still convert.
func TestNormaliseHostIDNABound(t *testing.T) {
	var long []rune
	for c := rune(0x3400); len(long) < 40000; c++ {
		long = append(long, c)
	}
	label63 := strings.Repeat("ü", 63)
	tests := []struct {
		name, host string
		want       bool
	}{
		{"idn", "bücher.example.com", true},
		{"idn label at 63 runes", label63 + ".example.com", false}, // valid runes, too long once encoded
		{"40k distinct runes", string(long), false},
		{"40k runes then a domain", string(long) + ".example.com", false},
		{"long label among short ones", "a." + strings.Repeat("ü", idnaMaxLabel+1) + ".example.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			_, ok := normaliseHost(tt.host, false)
			if ok != tt.want {
				t.Errorf("normaliseHost ok = %v, want %v", ok, tt.want)
			}
			if d := time.Since(start); d > 2*time.Second {
				t.Errorf("took %v", d)
			}
		})
	}
	if !idnaSized(strings.Repeat("ü", idnaMaxLabel)+".example") || idnaSized(strings.Repeat("a.", idnaMaxName)) {
		t.Error("idnaSized bounds wrong")
	}
	if _, why := caidoHost(string(long)); why == "" {
		t.Error("caidoHost took a 40k-character name")
	}
}
