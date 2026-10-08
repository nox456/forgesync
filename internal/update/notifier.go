package update

import "context"

type LatestFinder interface {
	LatestVersion(ctx context.Context) (string, error)
}
