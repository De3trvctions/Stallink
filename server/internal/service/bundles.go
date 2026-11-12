package service

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/De3trvctions/Stallink/internal/catalog"
	"github.com/De3trvctions/Stallink/internal/data"
)

func (a *API) CreateBundle(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	if !a.BundlesEnabled {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	var body struct {
		Name           string                   `json:"name"`
		Description    string                   `json:"description"`
		EventID        data.ID                  `json:"eventId"`
		Event          *data.EventInput         `json:"event"`
		Components     []bundleComponentRequest `json:"components"`
		UnitPriceMinor int64                    `json:"unitPriceMinor"`
		PriceTiers     []priceTierRequest       `json:"priceTiers"`
		IsListed       bool                     `json:"isListed"`
	}
	var bundleFiles []*multipart.FileHeader
	componentFiles := make(map[string][]*multipart.FileHeader)
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		r.Body = http.MaxBytesReader(w, r.Body, maxGoodUploadBytes)
		if err := r.ParseMultipartForm(goodImageMemorySize); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid_image")
			return
		}
		if r.MultipartForm != nil {
			defer func() { _ = r.MultipartForm.RemoveAll() }()
			bundleFiles = r.MultipartForm.File["bundleImages"]
		}
		if err := json.Unmarshal([]byte(r.FormValue("bundle")), &body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid_json")
			return
		}
		for _, component := range body.Components {
			if component.DraftKey != "" {
				if !validDraftKey(component.DraftKey) {
					writeErr(w, http.StatusBadRequest, "invalid_bundle")
					return
				}
				componentFiles[component.DraftKey] = r.MultipartForm.File["componentImages."+component.DraftKey]
			}
		}
	} else if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if (body.EventID == 0) == (body.Event == nil) || (body.Event != nil && !validEvent(*body.Event)) {
		writeErr(w, http.StatusBadRequest, "event_required")
		return
	}
	if body.Name == "" || len(body.Components) < 2 || len(body.Components) > 20 || len(body.PriceTiers) != 0 || len(bundleFiles) > 1 {
		writeErr(w, http.StatusBadRequest, "invalid_bundle")
		return
	}
	paths := make([]string, 0, 16)
	bundleImages, err := saveGoodImages(a, r, bundleFiles)
	if err != nil {
		writeImageError(w, r, err)
		return
	}
	paths = append(paths, bundleImages...)
	components := make([]data.BundleComponentInput, 0, len(body.Components))
	for _, component := range body.Components {
		entry := data.BundleComponentInput{GoodID: component.GoodID, DraftKey: component.DraftKey, Quantity: component.Quantity}
		if component.Draft != nil {
			draft := *component.Draft
			images, imageErr := saveGoodImages(a, r, componentFiles[component.DraftKey])
			if imageErr != nil {
				removeGoodImages(r.Context(), a.DataDir, paths)
				writeImageError(w, r, imageErr)
				return
			}
			draft.ImagePaths = images
			paths = append(paths, images...)
			entry.Draft = &draft
		}
		components = append(components, entry)
	}
	priceTiers := make([]catalog.PriceTier, 0, len(body.PriceTiers))
	for _, tier := range body.PriceTiers {
		priceTiers = append(priceTiers, catalog.PriceTier{Quantity: tier.Quantity, TotalPriceMinor: tier.TotalPriceMinor})
	}
	bundleID, listingID, err := a.DB.CreateBundle(r.Context(), *account.ShopID, account.ID, data.BundleInput{
		Name: body.Name, Description: body.Description, EventID: body.EventID, NewEvent: body.Event,
		Components: components, UnitPriceMinor: body.UnitPriceMinor, PriceTiers: priceTiers,
		IsListed: body.IsListed, ImagePaths: bundleImages, Approved: a.autoApprove(r.Context()),
	})
	if err != nil {
		removeGoodImages(r.Context(), a.DataDir, paths)
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"bundleId": bundleID.String(), "bundleListingId": listingID.String(), "eventId": body.EventID.String()})
}

func (a *API) UpdateBundle(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	if !a.BundlesEnabled {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	var body struct {
		Name             string                   `json:"name"`
		Description      string                   `json:"description"`
		EventID          data.ID                  `json:"eventId"`
		Components       []bundleComponentRequest `json:"components"`
		UnitPriceMinor   int64                    `json:"unitPriceMinor"`
		PriceTiers       []priceTierRequest       `json:"priceTiers"`
		IsListed         bool                     `json:"isListed"`
		ExpectedRevision int64                    `json:"expectedRevision"`
		ReplaceImages    bool                     `json:"replaceImages"`
	}
	var files []*multipart.FileHeader
	componentFiles := make(map[string][]*multipart.FileHeader)
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		r.Body = http.MaxBytesReader(w, r.Body, maxGoodUploadBytes)
		if err := r.ParseMultipartForm(goodImageMemorySize); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid_image")
			return
		}
		if r.MultipartForm != nil {
			defer func() { _ = r.MultipartForm.RemoveAll() }()
		}
		if err := json.Unmarshal([]byte(r.FormValue("bundle")), &body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if r.MultipartForm != nil {
			files = r.MultipartForm.File["bundleImages"]
		}
		for _, component := range body.Components {
			if component.DraftKey != "" {
				if !validDraftKey(component.DraftKey) {
					writeErr(w, http.StatusBadRequest, "invalid_bundle")
					return
				}
				componentFiles[component.DraftKey] = r.MultipartForm.File["componentImages."+component.DraftKey]
			}
		}
	} else if err := readJSONStrict(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if body.ExpectedRevision <= 0 || body.EventID == 0 || len(body.Components) < 2 || len(body.Components) > 20 || len(body.PriceTiers) != 0 || len(files) > 1 {
		writeErr(w, http.StatusBadRequest, "invalid_bundle")
		return
	}
	if !body.ReplaceImages && len(files) > 0 {
		writeErr(w, http.StatusBadRequest, "invalid_bundle")
		return
	}
	paths, err := saveGoodImages(a, r, files)
	if err != nil {
		writeImageError(w, r, err)
		return
	}
	components := make([]data.BundleComponentInput, 0, len(body.Components))
	allPaths := append([]string{}, paths...)
	for _, component := range body.Components {
		entry := data.BundleComponentInput{GoodID: component.GoodID, DraftKey: component.DraftKey, Quantity: component.Quantity}
		if component.Draft != nil {
			draft := *component.Draft
			images, imageErr := saveGoodImages(a, r, componentFiles[component.DraftKey])
			if imageErr != nil {
				removeGoodImages(r.Context(), a.DataDir, allPaths)
				writeImageError(w, r, imageErr)
				return
			}
			draft.ImagePaths = images
			allPaths = append(allPaths, images...)
			entry.Draft = &draft
		}
		components = append(components, entry)
	}
	tiers := make([]catalog.PriceTier, 0, len(body.PriceTiers))
	for _, tier := range body.PriceTiers {
		tiers = append(tiers, catalog.PriceTier{Quantity: tier.Quantity, TotalPriceMinor: tier.TotalPriceMinor})
	}
	bundleID, valid := pathEntityID(w, r, "bundleID")
	if !valid {
		removeGoodImages(r.Context(), a.DataDir, allPaths)
		return
	}
	oldPaths, err := a.DB.UpdateBundle(r.Context(), *account.ShopID, account.ID, bundleID, body.ExpectedRevision,
		data.BundleInput{Name: body.Name, Description: body.Description, EventID: body.EventID, Components: components,
			UnitPriceMinor: body.UnitPriceMinor, PriceTiers: tiers, IsListed: body.IsListed, ImagePaths: paths}, body.ReplaceImages)
	if err != nil {
		removeGoodImages(r.Context(), a.DataDir, allPaths)
		writeDataErr(w, r, err)
		return
	}
	for _, path := range oldPaths {
		removeBlob(r.Context(), a.DataDir, path)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type bundleComponentRequest struct {
	GoodID   data.ID               `json:"goodId"`
	DraftKey string                `json:"draftKey"`
	Quantity int                   `json:"quantity"`
	Draft    *data.BundleGoodDraft `json:"draft"`
}

type priceTierRequest struct {
	Quantity        int   `json:"quantity"`
	TotalPriceMinor int64 `json:"totalPriceMinor"`
}

func validDraftKey(key string) bool {
	if len(key) == 0 || len(key) > 64 {
		return false
	}
	for _, ch := range key {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '_' && ch != '-' {
			return false
		}
	}
	return true
}

func (a *API) StaffBundles(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	if !a.BundlesEnabled {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	eventID, valid := pathEntityID(w, r, "eventID")
	if !valid {
		return
	}
	bundles, err := a.DB.StaffBundles(r.Context(), *account.ShopID, eventID)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	out := make([]any, 0, len(bundles))
	for _, bundle := range bundles {
		out = append(out, bundleJSON(bundle, "staff"))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) ReviewBundle(w http.ResponseWriter, r *http.Request) {
	account, ok := a.platform(w, r)
	if !ok {
		return
	}
	if !a.BundlesEnabled {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	var body struct {
		Status           string `json:"status"`
		Note             string `json:"note"`
		ExpectedRevision int64  `json:"expectedRevision"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	body.Status = strings.TrimSpace(body.Status)
	body.Note = strings.TrimSpace(body.Note)
	if (body.Status != "approved" && body.Status != "rejected") || len([]rune(body.Note)) > 500 || (body.Status == "rejected" && body.Note == "") || body.ExpectedRevision <= 0 {
		writeErr(w, http.StatusBadRequest, "invalid_review")
		return
	}
	bundleID, valid := pathEntityID(w, r, "bundleID")
	if !valid {
		return
	}
	if err := a.DB.ReviewBundle(r.Context(), account.ID, bundleID, body.Status, body.Note, body.ExpectedRevision); err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func bundleJSON(bundle data.BundleCard, view string) map[string]any {
	images := make([]any, 0, len(bundle.ImageIDs))
	imagePrefix := "/api/files/bundles/"
	componentPrefix := "/api/files/goods/"
	if view == "staff" {
		imagePrefix = "/api/staff/bundles/" + bundle.BundleID + "/images/"
		componentPrefix = "/api/staff/goods/"
	} else if view == "platform" {
		imagePrefix = "/api/platform/bundles/" + bundle.BundleID + "/images/"
		componentPrefix = "/api/platform/goods/"
	}
	for sortOrder, id := range bundle.ImageIDs {
		images = append(images, map[string]any{"id": id, "url": imagePrefix + id, "sortOrder": sortOrder})
	}
	components := make([]any, 0, len(bundle.Components))
	for _, component := range bundle.Components {
		componentImages := make([]any, 0, len(component.ImageIDs))
		for sortOrder, id := range component.ImageIDs {
			url := componentPrefix + component.GoodID + "/images/" + id
			if view != "staff" && view != "platform" {
				url = componentPrefix + id
			}
			componentImages = append(componentImages, map[string]any{"id": id, "url": url, "sortOrder": sortOrder})
		}
		components = append(components, map[string]any{
			"goodId": component.GoodID, "listingId": component.ListingID, "name": component.GoodName,
			"quantity": component.Quantity, "unitPriceMinor": component.UnitPriceMinor,
			"reviewStatus": component.ReviewStatus, "isListed": component.IsListed, "remaining": component.Remaining,
			"images": componentImages,
		})
	}
	var imageURL any
	if len(images) > 0 {
		imageURL = images[0].(map[string]any)["url"]
	}
	return map[string]any{
		"bundleListingId": bundle.ListingID, "bundleId": bundle.BundleID, "shopId": bundle.ShopID,
		"shopName": bundle.ShopName, "shopSlug": bundle.ShopSlug,
		"eventId": bundle.EventID, "eventName": bundle.EventName, "currency": bundle.Currency,
		"name": bundle.Name, "description": bundle.Description, "unitPriceMinor": bundle.UnitPriceMinor,
		"priceLabel": priceLabel(bundle.Currency, bundle.UnitPriceMinor), "priceTiers": bundle.PriceTiers,
		"remaining": bundle.Remaining, "isListed": bundle.IsListed, "reviewStatus": bundle.ReviewStatus,
		"reviewNote": bundle.ReviewNote, "revision": bundle.Revision, "images": images, "imageURL": imageURL,
		"components": components,
	}
}

func bundleCardsJSON(bundles []data.BundleCard, view string) []any {
	out := make([]any, 0, len(bundles))
	for _, bundle := range bundles {
		out = append(out, bundleJSON(bundle, view))
	}
	return out
}
