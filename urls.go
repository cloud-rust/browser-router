package main

import (
	"fmt"
	"net/url"
	"strings"
)

// siteURL is one entry of a rule's urls: a host, optionally with all its subdomains ("*.") and a
// path prefix. It looks only at host and path, so a word in a page title never moves a link.
type siteURL struct {
	host      string // lower case, without "*."
	subdomain bool   // "*.example.com": example.com and every subdomain
	path      string // lower case, "" or "/seg[/seg…]" without trailing slash
}

// parseSiteURL accepts "example.com", "*.example.com", "github.com/acme" and the same with
// http:// or https:// in front. Any other wildcard is an error: patterns belong in match.
func parseSiteURL(s string) (siteURL, error) {
	in := s
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(strings.TrimPrefix(s, "https://"), "http://")
	var u siteURL
	if strings.HasPrefix(s, "*.") {
		u.subdomain = true
		s = s[2:]
	}
	u.host, u.path, _ = strings.Cut(s, "/")
	u.path = strings.TrimRight(u.path, "/")
	if u.path != "" {
		u.path = "/" + u.path
	}
	switch {
	case u.host == "" || strings.ContainsAny(u.host, ":?#@ "):
		return siteURL{}, fmt.Errorf("url %q: needs a host such as example.com, without port", in)
	case strings.ContainsAny(u.host+u.path, "*?#"):
		return siteURL{}, fmt.Errorf("url %q: only a leading *. is allowed; use match for patterns", in)
	}
	return u, nil
}

func (u siteURL) String() string {
	if u.subdomain {
		return "*." + u.host + u.path
	}
	return u.host + u.path
}

// matches reports whether an http(s) link lies on this site. A path prefix ends at a segment
// boundary, so "github.com/acme" covers github.com/acme/x but not github.com/acme-corp.
func (u siteURL) matches(link *url.URL) bool {
	host := strings.ToLower(link.Hostname())
	if host != u.host && !(u.subdomain && strings.HasSuffix(host, "."+u.host)) {
		return false
	}
	p := strings.ToLower(link.EscapedPath())
	return u.path == "" || p == u.path || strings.HasPrefix(p, u.path+"/")
}

// specificity orders matching entries: an exact host beats a subdomain entry, a longer host a
// shorter one, then the longer path wins.
func (u siteURL) specificity() [3]int {
	exact := 1
	if u.subdomain {
		exact = 0
	}
	return [3]int{exact, len(u.host), len(u.path)}
}

func moreSpecific(a, b [3]int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

// bestURLRule returns the index of the rule with the most specific urls entry for the link, or -1.
func bestURLRule(cfg Config, link string) int {
	u, err := url.Parse(link)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return -1
	}
	best, bestSpec := -1, [3]int{}
	for i, r := range cfg.Rules {
		for _, s := range r.sites {
			if spec := s.specificity(); s.matches(u) && (best < 0 || moreSpecific(spec, bestSpec)) {
				best, bestSpec = i, spec
			}
		}
	}
	return best
}
