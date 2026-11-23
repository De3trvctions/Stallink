package imagestore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const folderMIME = "application/vnd.google-apps.folder"
const referencePrefix = "drive:"

var (
	ErrNotFound = errors.New("drive image not found")
	validID     = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

type Drive struct {
	client        *http.Client
	base          string
	rootID        string
	sharedDriveID string
}

type file struct {
	ID            string            `json:"id,omitempty"`
	Name          string            `json:"name,omitempty"`
	MIMEType      string            `json:"mimeType,omitempty"`
	Parents       []string          `json:"parents,omitempty"`
	AppProperties map[string]string `json:"appProperties,omitempty"`
	DriveID       string            `json:"driveId,omitempty"`
	Capabilities  struct {
		CanAddChildren bool `json:"canAddChildren"`
	} `json:"capabilities,omitempty,omitzero"`
	Trashed bool `json:"trashed,omitempty"`
}

func NewDrive(ctx context.Context, client *http.Client, rootID string) (*Drive, error) {
	if !validID.MatchString(rootID) {
		return nil, errors.New("invalid google drive root folder id")
	}
	d := &Drive{client: client, base: "https://www.googleapis.com", rootID: rootID}
	var root file
	params := url.Values{"fields": {"id,mimeType,driveId,trashed,capabilities(canAddChildren)"}}
	err := d.jsonRequest(ctx, http.MethodGet, "/drive/v3/files/"+rootID, params, "", nil, &root)
	if err != nil {
		return nil, fmt.Errorf("check google drive root folder: %w", err)
	}
	if root.Trashed || root.MIMEType != folderMIME || !root.Capabilities.CanAddChildren {
		return nil, errors.New("google drive root must be a writable folder")
	}
	d.sharedDriveID = root.DriveID
	return d, nil
}

func (d *Drive) Put(ctx context.Context, shopID int64, kind, contentType string, content []byte) (string, error) {
	ext := strings.TrimPrefix(contentType, "image/")
	if ext == "jpeg" {
		ext = "jpg"
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	metadata, err := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {"application/json; charset=UTF-8"}})
	if err != nil {
		return "", err
	}
	if err := json.NewEncoder(metadata).Encode(file{
		Name:     fmt.Sprintf("shop-%d-%s-%s.%s", shopID, kind, uuid.NewString(), ext),
		MIMEType: contentType, Parents: []string{d.rootID},
		AppProperties: map[string]string{"stallinkImageRoot": d.rootID},
	}); err != nil {
		return "", err
	}
	media, err := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {contentType}})
	if err != nil {
		return "", err
	}
	if _, err := media.Write(content); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	var created file
	params := url.Values{"uploadType": {"multipart"}, "fields": {"id"}}
	mimeType := mime.FormatMediaType("multipart/related", map[string]string{"boundary": writer.Boundary()})
	if err := d.jsonRequest(ctx, http.MethodPost, "/upload/drive/v3/files", params, mimeType, &body, &created); err != nil {
		return "", err
	}
	id, err := checkedID(created.ID)
	if err != nil {
		return "", err
	}
	return referencePrefix + id, nil
}

func (d *Drive) Get(ctx context.Context, reference string) ([]byte, error) {
	id, err := referenceID(reference)
	if err != nil {
		return nil, err
	}
	resp, err := d.request(ctx, http.MethodGet, "/drive/v3/files/"+id, url.Values{"alt": {"media"}}, "", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(io.LimitReader(resp.Body, MaxUploadBytes+1))
	if err != nil {
		return nil, err
	}
	if len(result) > MaxUploadBytes {
		return nil, ErrImageTooLarge
	}
	return result, nil
}

func (d *Drive) Trash(ctx context.Context, reference string) error {
	id, err := referenceID(reference)
	if err != nil {
		return err
	}
	err = d.jsonRequest(ctx, http.MethodPatch, "/drive/v3/files/"+id, nil, "application/json", strings.NewReader(`{"trashed":true}`), nil)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

// Sweep trashes only images uploaded by Stallink under the configured root and older than cutoff.
func (d *Drive) Sweep(ctx context.Context, referenced map[string]bool, cutoff time.Time) (int, error) {
	params := url.Values{"q": {fmt.Sprintf("trashed = false and appProperties has { key='stallinkImageRoot' and value='%s' } and createdTime < '%s'", d.rootID, cutoff.UTC().Format(time.RFC3339))}}
	var orphaned []string
	for {
		page, err := d.list(ctx, params)
		if err != nil {
			return 0, err
		}
		for _, candidate := range page.Files {
			if ref := referencePrefix + candidate.ID; !referenced[ref] {
				orphaned = append(orphaned, ref)
			}
		}
		if page.NextPageToken == "" {
			break
		}
		params.Set("pageToken", page.NextPageToken)
	}
	for i, ref := range orphaned {
		if err := d.Trash(ctx, ref); err != nil {
			return i, err
		}
	}
	return len(orphaned), nil
}

type filePage struct {
	Files            []file `json:"files"`
	NextPageToken    string `json:"nextPageToken"`
	IncompleteSearch bool   `json:"incompleteSearch"`
}

func (d *Drive) list(ctx context.Context, params url.Values) (filePage, error) {
	params.Set("fields", "nextPageToken,incompleteSearch,files(id)")
	params.Set("includeItemsFromAllDrives", "true")
	if d.sharedDriveID != "" {
		params.Set("corpora", "drive")
		params.Set("driveId", d.sharedDriveID)
	}
	var page filePage
	err := d.jsonRequest(ctx, http.MethodGet, "/drive/v3/files", params, "", nil, &page)
	if err == nil && page.IncompleteSearch {
		err = errors.New("google drive returned an incomplete file listing")
	}
	return page, err
}

func (d *Drive) jsonRequest(ctx context.Context, method, path string, params url.Values, contentType string, body io.Reader, out any) error {
	resp, err := d.request(ctx, method, path, params, contentType, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if out == nil {
		_, err = io.Copy(io.Discard, resp.Body)
		return err
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (d *Drive) request(ctx context.Context, method, path string, params url.Values, contentType string, body io.Reader) (*http.Response, error) {
	if params == nil {
		params = make(url.Values)
	}
	params.Set("supportsAllDrives", "true")
	req, err := http.NewRequestWithContext(ctx, method, d.base+path+"?"+params.Encode(), body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google drive %s: %w", method, err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	var apiError struct {
		Error struct {
			Errors []struct{ Reason string } `json:"errors"`
		} `json:"error"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 16<<10)).Decode(&apiError) == nil && len(apiError.Error.Errors) > 0 {
		return nil, fmt.Errorf("google drive %s returned %d (%s)", method, resp.StatusCode, apiError.Error.Errors[0].Reason)
	}
	return nil, fmt.Errorf("google drive %s returned %d", method, resp.StatusCode)
}

func checkedID(id string) (string, error) {
	if !validID.MatchString(id) {
		return "", errors.New("invalid google drive file id")
	}
	return id, nil
}

func referenceID(reference string) (string, error) {
	id, found := strings.CutPrefix(reference, referencePrefix)
	if !found || !validID.MatchString(id) {
		return "", ErrNotFound
	}
	return id, nil
}
