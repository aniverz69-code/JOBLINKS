package main

import (
	"context"

	"github.com/strelov1/freehire/internal/ingest/linksource"
	"github.com/strelov1/freehire/internal/ingest/telegram"
)

// linkResolver adapts the linksource registry to telegram.LinkResolver: it follows a
// post's links to full vacancies and maps each to a ResolvedJob carrying the destination
// platform's source identity.
type linkResolver struct {
	reg []linksource.Source
}

func (r linkResolver) Resolve(ctx context.Context, links []telegram.Link) ([]telegram.ResolvedJob, error) {
	urls := make([]string, len(links))
	for i, l := range links {
		urls[i] = l.URL
	}

	resolved, err := linksource.ResolveLinks(ctx, r.reg, urls)
	if err != nil {
		return nil, err
	}
	out := make([]telegram.ResolvedJob, len(resolved))
	for i, rj := range resolved {
		out[i] = telegram.ResolvedJob{
			Source:      rj.Source,
			ExternalID:  rj.Job.ExternalID,
			URL:         rj.Job.URL,
			Title:       rj.Job.Title,
			Company:     rj.Job.Company,
			Location:    rj.Job.Location,
			Description: rj.Job.Description,
			Remote:      rj.Job.Remote,
			PostedAt:    rj.Job.PostedAt,
			WorkMode:    rj.Job.WorkMode,
		}
	}
	return out, nil
}

var _ telegram.LinkResolver = linkResolver{}
