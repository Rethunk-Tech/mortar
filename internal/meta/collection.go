package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const (
	collectionTTL   = 15 * time.Minute
	collectionQuery = `query($slug:String!,$domain:String,$revision:Int){collectionRevision(slug:$slug,domainName:$domain,revision:$revision,viewAdultContent:true){revisionNumber downloadLink installationInfo externalResources{name resourceType resourceUrl optional version author} collection{name slug user{name}} modFiles{fileId optional file{modId name version mod{name}}}}}`
)

type Collection struct {
	Name     string
	Author   string
	Slug     string
	Revision int
	Files    []CollectionFile
	// Instructions is the curator's installationInfo text, shown before importing.
	Instructions string
	// External lists what the curator points to outside Nexus; Mortar never fetches these.
	External []CollectionExternal
	// DownloadLink is the Nexus API path that resolves to the curator's collection archive.
	DownloadLink string
}

// CollectionExternal is one off-Nexus resource of a collection. Type is Nexus's resourceType ("direct", "browse" or "manual").
type CollectionExternal struct {
	Name     string
	Type     string
	URL      string
	Version  string
	Author   string
	Optional bool
}

type CollectionFile struct {
	ModID    int
	FileID   int
	Optional bool
}

func (c *Client) Collection(ctx context.Context, domain, slug string, revision int) (Collection, error) {
	name := "nexus-collection-" + domain + "-" + slug + "-" + strconv.Itoa(revision) + ".json"
	return Cached(c, name, collectionTTL, func() (Collection, error) {
		return c.fetchCollection(ctx, domain, slug, revision)
	})
}

func (c *Client) fetchCollection(ctx context.Context, domain, slug string, revision int) (Collection, error) {
	vars := map[string]any{"slug": slug, "domain": domain}
	if revision != 0 {
		vars["revision"] = revision
	}
	body, err := json.Marshal(map[string]any{"query": collectionQuery, "variables": vars})
	if err != nil {
		return Collection{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.nexusmods.com/v2/graphql", bytes.NewReader(body))
	if err != nil {
		return Collection{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mortar")
	req.Header.Set("Accept", "application/json")
	raw, err := c.do(req, maxPage)
	if err != nil {
		return Collection{}, err
	}
	return decodeCollection(slug, raw)
}

type collectionGQL struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
	Data *struct {
		CollectionRevision *struct {
			RevisionNumber    int    `json:"revisionNumber"`
			DownloadLink      string `json:"downloadLink"`
			InstallationInfo  string `json:"installationInfo"`
			ExternalResources []struct {
				Name         string `json:"name"`
				ResourceType string `json:"resourceType"`
				ResourceURL  string `json:"resourceUrl"`
				Version      string `json:"version"`
				Author       string `json:"author"`
				Optional     bool   `json:"optional"`
			} `json:"externalResources"`
			Collection struct {
				Name string `json:"name"`
				Slug string `json:"slug"`
				User struct {
					Name string `json:"name"`
				} `json:"user"`
			} `json:"collection"`
			ModFiles []struct {
				FileID   int  `json:"fileId"`
				Optional bool `json:"optional"`
				File     *struct {
					ModID int `json:"modId"`
				} `json:"file"`
			} `json:"modFiles"`
		} `json:"collectionRevision"`
	} `json:"data"`
}

// webPage keeps a curator's resource link only when it is an http or https page, since the import dialog opens it
// with the system's URL handler, which would also launch other schemes' apps.
func webPage(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return ""
	}
	return raw
}

func decodeCollection(slug string, raw []byte) (Collection, error) {
	var parsed collectionGQL
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Collection{}, err
	}
	if len(parsed.Errors) > 0 || parsed.Data == nil || parsed.Data.CollectionRevision == nil {
		return Collection{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("nexus has no collection %q", slug))
	}
	rev := parsed.Data.CollectionRevision
	out := Collection{
		Name:     rev.Collection.Name,
		Author:   rev.Collection.User.Name,
		Slug:     rev.Collection.Slug,
		Revision: rev.RevisionNumber,

		Instructions: strings.TrimSpace(rev.InstallationInfo),
		DownloadLink: rev.DownloadLink,
	}
	for _, r := range rev.ExternalResources {
		out.External = append(out.External, CollectionExternal{
			Name: r.Name, Type: r.ResourceType, URL: webPage(r.ResourceURL), Version: r.Version, Author: r.Author, Optional: r.Optional,
		})
	}
	for _, mf := range rev.ModFiles {
		if mf.File == nil {
			continue
		}
		out.Files = append(out.Files, CollectionFile{
			ModID:    mf.File.ModID,
			FileID:   mf.FileID,
			Optional: mf.Optional,
		})
	}
	return out, nil
}
