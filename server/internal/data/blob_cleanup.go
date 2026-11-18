package data

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// ReferencedBlobPaths returns every persisted image/payment blob path. Callers
// may remove old files only when their full path is absent from this set.
func (s *ShopStore) ReferencedBlobPaths(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT path FROM good_images
		UNION SELECT path FROM bundle_images
		UNION SELECT image_path FROM pay_codes`)
	if err != nil {
		return nil, fmt.Errorf("list referenced blobs: %w", err)
	}
	defer rows.Close()
	paths := make([]string, 0)
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, fmt.Errorf("scan referenced blob: %w", err)
		}
		paths = append(paths, path)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read referenced blobs: %w", err)
	}
	return paths, nil
}

// ReferencedBlobNames keeps bare filenames that can be compared with a blob
// directory entry. Database rows store the random hex name, not the absolute
// path. Directory-like values are ignored so they cannot protect another file.
func ReferencedBlobNames(paths []string) map[string]struct{} {
	names := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if path == "" || strings.ContainsAny(path, `/\`) {
			continue
		}
		name := filepath.Base(path)
		if name == "." || name == ".." || name != path {
			continue
		}
		names[name] = struct{}{}
	}
	return names
}
