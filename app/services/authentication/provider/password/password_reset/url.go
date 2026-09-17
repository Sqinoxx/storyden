package password_reset

import (
	"net/url"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/ftag"
)

var ErrLinkURLOffOrigin = fault.New("reset link URL must be on the instance's public web address", ftag.With(ftag.InvalidArgument))

type LinkTemplate struct {
	u  url.URL
	qp string
}

func (r *LinkTemplate) GetURL(token string) string {
	q := r.u.Query()

	q.Add(r.qp, token)

	r.u.RawQuery = q.Encode()

	return r.u.String()
}

// NewLinkTemplate builds a reset-link template from a client-supplied URL.
// A relative path is resolved against publicWebAddress; an absolute URL
// with a different scheme or host is rejected. Without this, a caller
// could supply an attacker-controlled host and have a victim's reset
// token emailed straight to it.
func NewLinkTemplate(publicWebAddress url.URL, urlString string, tokenQueryParam string) (*LinkTemplate, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return nil, fault.Wrap(err, ftag.With(ftag.InvalidArgument))
	}

	resolved := publicWebAddress.ResolveReference(u)

	if resolved.Scheme != publicWebAddress.Scheme || resolved.Host != publicWebAddress.Host {
		return nil, ErrLinkURLOffOrigin
	}

	return &LinkTemplate{
		u:  *resolved,
		qp: tokenQueryParam,
	}, nil
}
