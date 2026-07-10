package mcp

import (
	"context"
	"strings"

	"github.com/madicen/naitv-mcp/internal/instructions"
	"github.com/madicen/naitv-mcp/internal/store"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	bundleResourceURI      = "naitv://bundle"
	entryResourceTemplate  = "naitv://entry/{id}"
	entryResourceURIPrefix = "naitv://entry/"
)

func registerResources(s *sdkmcp.Server, st *store.Store) {
	readResource := func(ctx context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
		uri := req.Params.URI
		var text string

		switch {
		case uri == bundleResourceURI:
			entries, err := st.List("", nil)
			if err != nil {
				return nil, err
			}
			text = instructions.Render(instructions.FilterInit(entries))
		case strings.HasPrefix(uri, entryResourceURIPrefix):
			id := strings.TrimPrefix(uri, entryResourceURIPrefix)
			if id == "" {
				return nil, sdkmcp.ResourceNotFoundError(uri)
			}
			e, err := st.Get(id)
			if err != nil {
				return nil, sdkmcp.ResourceNotFoundError(uri)
			}
			text = formatEntry(e)
		default:
			return nil, sdkmcp.ResourceNotFoundError(uri)
		}

		return &sdkmcp.ReadResourceResult{
			Contents: []*sdkmcp.ResourceContents{{
				URI:      uri,
				MIMEType: "text/plain",
				Text:     text,
			}},
		}, nil
	}

	s.AddResource(&sdkmcp.Resource{
		URI:         bundleResourceURI,
		Name:        "init-bundle",
		Description: "Standing instructions rendered from init-delivery context entries",
		MIMEType:    "text/plain",
	}, readResource)

	s.AddResourceTemplate(&sdkmcp.ResourceTemplate{
		URITemplate: entryResourceTemplate,
		Name:        "entry",
		Description: "A single context entry by ID",
		MIMEType:    "text/plain",
	}, readResource)
}

