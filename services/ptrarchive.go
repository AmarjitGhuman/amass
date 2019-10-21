// Copyright 2017 Jeff Foley. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package services

import (
	"context"
	"fmt"
	"time"

	"github.com/OWASP/Amass/config"
	"github.com/OWASP/Amass/eventbus"
	"github.com/OWASP/Amass/net/http"
	"github.com/OWASP/Amass/requests"
)

// PTRArchive is the Service that handles access to the Exalead data source.
type PTRArchive struct {
	BaseService

	SourceType string
}

// NewPTRArchive returns he object initialized, but not yet started.
func NewPTRArchive(sys System) *PTRArchive {
	p := &PTRArchive{SourceType: requests.SCRAPE}

	p.BaseService = *NewBaseService(p, "PTRArchive", sys)
	return p
}

// Type implements the Service interface.
func (p *PTRArchive) Type() string {
	return p.SourceType
}

// OnStart implements the Service interface.
func (p *PTRArchive) OnStart() error {
	p.BaseService.OnStart()

	p.SetRateLimit(time.Second)
	return nil
}

// OnDNSRequest implements the Service interface.
func (p *PTRArchive) OnDNSRequest(ctx context.Context, req *requests.DNSRequest) {
	cfg := ctx.Value(requests.ContextConfig).(*config.Config)
	bus := ctx.Value(requests.ContextEventBus).(*eventbus.EventBus)
	if cfg == nil || bus == nil {
		return
	}

	re := cfg.DomainRegex(req.Domain)
	if re == nil {
		return
	}

	p.CheckRateLimit()
	bus.Publish(requests.SetActiveTopic, p.String())

	url := p.getURL(req.Domain)
	page, err := http.RequestWebPage(url, nil, nil, "", "")
	if err != nil {
		bus.Publish(requests.LogTopic, fmt.Sprintf("%s: %s: %v", p.String(), url, err))
		return
	}

	for _, sd := range re.FindAllString(page, -1) {
		name := cleanName(sd)
		if name == "automated_programs_unauthorized."+req.Domain {
			continue
		}

		bus.Publish(requests.NewNameTopic, &requests.DNSRequest{
			Name:   name,
			Domain: req.Domain,
			Tag:    p.SourceType,
			Source: p.String(),
		})
	}
}

func (p *PTRArchive) getURL(domain string) string {
	format := "http://ptrarchive.com/tools/search3.htm?label=%s&date=ALL"

	return fmt.Sprintf(format, domain)
}
