package main

import (
	"net/url"
	"strings"
)

// Teams and Outlook hand out links wrapped in Microsoft Safe Links, with the real link in the
// "url" query parameter. Rules must see that real link: the wrapper host matches nothing, and a
// match pattern over the whole wrapper would hit words inside the encoded link (a repository named
// acme_x on github.com/OTHER would go to the acme profile).
type wrapper struct {
	host  string // the host itself or any subdomain of it
	param string
}

var wrappers = []wrapper{
	{host: "safelinks.protection.outlook.com", param: "url"},
	{host: "teams.public.onecdn.static.microsoft", param: "url"},
	{host: "statics.teams.cdn.office.net", param: "url"},
}

// unwrapRedirect returns the link a known wrapper points to, following nested wrappers a few
// levels deep; any other link comes back unchanged. Only the rule choice uses the result: the
// wrapper itself is opened, so the Safe Links check at click time still runs.
func unwrapRedirect(link string) string {
	for range 3 {
		inner, ok := unwrapOnce(link)
		if !ok {
			break
		}
		link = inner
	}
	return link
}

func unwrapOnce(link string) (string, bool) {
	u, err := url.Parse(link)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", false
	}
	host := strings.ToLower(u.Hostname())
	for _, w := range wrappers {
		if host != w.host && !strings.HasSuffix(host, "."+w.host) {
			continue
		}
		inner := u.Query().Get(w.param)
		if v, err := url.Parse(inner); err == nil && (v.Scheme == "http" || v.Scheme == "https") && v.Host != "" {
			return inner, true
		}
	}
	return "", false
}
