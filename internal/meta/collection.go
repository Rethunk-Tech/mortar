package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const (
	collectionTTL   = 15 * time.Minute
	collectionQuery = `query($slug:String!,$domain:String,$revision:Int){collectionRevision(slug:$slug,domainName:$domain,revision:$revision,viewAdultContent:true){revisionNumber collection{name slug user{name}} modFiles{fileId optional file{modId name version mod{name}}}}}`
)

type Collection struct {
	Name     string
	Author   string
	Slug     string
	Revision int
	Files    []CollectionFile
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
			RevisionNumber int `json:"revisionNumber"`
			Collection     struct {
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

func decodeCollection(slug string, raw []byte) (Collection, error) {
	var parsed collectionGQL
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Collection{}, err
	}
	if len(parsed.Errors) > 0 || parsed.Data == nil || parsed.Data.CollectionRevision == nil {
		return Collection{}, fmt.Errorf("nexus has no collection %q", slug)
	}
	rev := parsed.Data.CollectionRevision
	out := Collection{
		Name:     rev.Collection.Name,
		Author:   rev.Collection.User.Name,
		Slug:     rev.Collection.Slug,
		Revision: rev.RevisionNumber,
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
