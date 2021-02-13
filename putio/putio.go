package putio

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/putdotio/go-putio"
	"golang.org/x/oauth2"
)

// Putio struct containing client information
type Putio struct {
	client *putio.Client
	dirID  int64
	logger hclog.Logger
	queue  map[putio.File]struct{}
}

// New creates and returns Putio with client information
func New(token string) *Putio {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)

	client := putio.NewClient(oauthClient)

	return &Putio{
		client: client,
		dirID:  0,
		logger: nil,
		queue:  map[putio.File]struct{}{},
	}
}

// GetDirID STUB gets ID of directory to fetch files from
func (p *Putio) GetDirID(name string) error {
	p.dirID = 0
	return nil
}

// CleanTransfers cleans transfer list
func (p *Putio) CleanTransfers(ctx context.Context) error {
	return p.client.Transfers.Clean(ctx)
}

// FetchList returns list of files/directories inside specified directory
func (p *Putio) FetchList(ctx context.Context) ([]putio.File, error) {
	list, _, err := p.client.Files.List(ctx, p.dirID)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// RequestZip requests and waits for a zip file to be created for a file
func (p *Putio) RequestZip(ctx context.Context, file putio.File) (*putio.Zip, error) {
	zip, err := p.client.Zips.Create(ctx, file.ID)
	if err != nil {
		return nil, err
	}

	ticker := time.NewTimer(0)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			zip, err := p.client.Zips.Get(ctx, zip)
			if err != nil {
				return nil, err
			}

			if zip.URL != "" {
				return &zip, nil
			}

			ticker.Reset(time.Second)
		}
	}
}

// DeleteFile deletes a file from putio
func (p *Putio) DeleteFile(ctx context.Context, file int64) error {
	ticker := time.NewTimer(0)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("failed to delete file before timeout, %v", ctx.Err())
		case <-ticker.C:
			err := p.client.Files.Delete(ctx, file)
			if err != nil {
				p.logger.Debug("failed to delete file", "file", file, "error", err)
				ticker.Reset(5 * time.Second)
				break
			}

			return nil
		}
	}
}
