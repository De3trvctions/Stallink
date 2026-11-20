package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/De3trvctions/Stallink/internal/auth"
	"github.com/De3trvctions/Stallink/internal/catalog"
	"github.com/De3trvctions/Stallink/internal/data"
	"github.com/De3trvctions/Stallink/internal/logging"
	"go.uber.org/zap"
)

func (a *API) UpdateShopSettings(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	var body struct {
		Name      string `json:"name"`
		Notice    string `json:"notice"`
		Slug      string `json:"slug"`
		IsEnabled bool   `json:"isEnabled"`
	}
	if err := readJSON(r, &body); err != nil || !slugPattern.MatchString(body.Slug) || body.Name == "" {
		writeErr(w, http.StatusBadRequest, "invalid_shop")
		return
	}
	if err := a.DB.UpdateShop(r.Context(), *account.ShopID, body.Name, body.Notice, body.Slug, body.IsEnabled); err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) SetFeaturedEvent(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	var body struct {
		EventID string `json:"eventId"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	eventID := data.ID(0)
	if body.EventID != "" {
		parsed, err := data.ParseID(body.EventID)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid_event")
			return
		}
		eventID = parsed
	}
	if err := a.DB.SetFeaturedEvent(r.Context(), *account.ShopID, eventID); err != nil {
		if errors.Is(err, data.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "event_not_found")
			return
		}
		writeDataErr(w, r, err)
		return
	}
	featured := ""
	if eventID != 0 {
		featured = eventID.String()
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "featuredEventId": featured})
}

func (a *API) AdminPayCodes(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	codes, err := a.DB.ListPayCodes(r.Context(), *account.ShopID, false)
	if err != nil {
		logging.ErrorContext(r.Context(), "pay codes", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	writeJSON(w, http.StatusOK, payCodesJSON(codes))
}

func (a *API) CreateCashPayCode(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	var body struct {
		Name      string `json:"name"`
		IsDefault bool   `json:"isDefault"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = "现金"
	}
	if utf8.RuneCountInString(name) > 40 {
		writeErr(w, http.StatusBadRequest, "invalid_name")
		return
	}
	id, err := a.DB.CreateCashPayCode(r.Context(), *account.ShopID, name, body.IsDefault)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "name": name, "kind": "cash", "isDefault": body.IsDefault, "imageURL": ""})
}

func (a *API) CreatePayCode(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	if err := r.ParseMultipartForm(6 << 20); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_image")
		return
	}
	file, _, err := r.FormFile("image")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_image")
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			logging.ErrorContext(r.Context(), "close upload", zap.Error(err))
		}
	}()
	name := r.FormValue("name")
	if name == "" {
		writeErr(w, http.StatusBadRequest, "invalid_image")
		return
	}
	if !validPayCodeName(name) {
		writeErr(w, http.StatusBadRequest, "invalid_pay_code_name")
		return
	}
	path, err := saveBlob(r.Context(), a.DataDir, file)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_image")
		return
	}
	isDefault := r.FormValue("isDefault") == "true"
	currencyRaw := strings.TrimSpace(r.FormValue("currency"))
	var currency *string
	if currencyRaw != "" {
		if !catalog.ValidCurrency(currencyRaw) {
			writeErr(w, http.StatusBadRequest, "invalid_pay_code_currency")
			return
		}
		currency = &currencyRaw
	}
	id, err := a.DB.CreatePayCode(r.Context(), *account.ShopID, name, path, currency, isDefault)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "name": name, "isDefault": isDefault, "imageURL": "/api/files/pay-codes/" + id.String()})
}

func (a *API) UpdatePayCode(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	var body struct {
		Name      string  `json:"name"`
		IsEnabled bool    `json:"isEnabled"`
		IsDefault bool    `json:"isDefault"`
		Currency  *string `json:"currency"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" || !validPayCodeName(body.Name) {
		writeErr(w, http.StatusBadRequest, "invalid_pay_code_name")
		return
	}
	id, ok := pathEntityID(w, r, "id")
	if !ok {
		return
	}
	var currency *string
	if body.Currency != nil {
		trimmed := strings.TrimSpace(*body.Currency)
		if trimmed != "" {
			if !catalog.ValidCurrency(trimmed) {
				writeErr(w, http.StatusBadRequest, "invalid_pay_code_currency")
				return
			}
			currency = &trimmed
		}
	}
	if err := a.DB.UpdatePayCode(r.Context(), *account.ShopID, id, body.Name, currency, body.IsEnabled, body.IsDefault); err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func orderDateFilter(value string) (string, bool) {
	if value == "" {
		return "", true
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, reportLocation)
	if err != nil || parsed.After(time.Now().In(reportLocation)) {
		return "", false
	}
	return parsed.Format("2006-01-02"), true
}

func (a *API) AdminOrders(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	page, limit, search, valid := pageParams(r)
	if !valid {
		writeErr(w, http.StatusBadRequest, "invalid_pagination")
		return
	}
	date, validDate := orderDateFilter(r.URL.Query().Get("date"))
	if !validDate {
		writeErr(w, http.StatusBadRequest, "invalid_date")
		return
	}
	orders, more, err := a.DB.ListOrderPage(r.Context(), *account.ShopID, "", date, search, r.URL.Query().Get("status"), a.ReportTimezone, limit, (page-1)*limit)
	if err != nil {
		logging.ErrorContext(r.Context(), "admin orders", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": ordersJSON(orders), "page": page, "hasMore": more})
}

func (a *API) AdminAudit(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	page, limit, search, valid := pageParams(r)
	if !valid {
		writeErr(w, http.StatusBadRequest, "invalid_pagination")
		return
	}
	entries, more, err := a.DB.ListAuditPage(r.Context(), *account.ShopID, search, limit, (page-1)*limit)
	if err != nil {
		logging.ErrorContext(r.Context(), "audit", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	out := make([]any, 0, len(entries))
	for _, entry := range entries {
		out = append(out, map[string]string{
			"action": entry.Action, "actorLogin": entry.ActorLogin,
			"subjectType": entry.SubjectType, "subjectId": entry.SubjectID.String(), "createdAt": entry.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out, "page": page, "hasMore": more})
}

func (a *API) ListSales(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	sales, err := a.DB.ListSales(r.Context(), *account.ShopID)
	if err != nil {
		logging.ErrorContext(r.Context(), "sales", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	out := make([]any, 0, len(sales))
	for _, person := range sales {
		out = append(out, map[string]any{"id": person.ID, "loginName": person.LoginName, "enabled": person.Enabled})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) CreateSales(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	var body struct {
		LoginName string `json:"loginName"`
		Password  string `json:"password"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	login := strings.TrimSpace(body.LoginName)
	if !validLoginName(login) {
		writeErr(w, http.StatusBadRequest, "invalid_login_name")
		return
	}
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_profile")
		return
	}
	id, err := a.DB.CreateAccount(r.Context(), login, hash, "sales", account.ShopID, "", "")
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id.String()})
}

func (a *API) DisableSales(w http.ResponseWriter, r *http.Request) {
	a.setSales(w, r, false)
}

func (a *API) EnableSales(w http.ResponseWriter, r *http.Request) {
	a.setSales(w, r, true)
}

func (a *API) setSales(w http.ResponseWriter, r *http.Request, enabled bool) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	userID, ok := pathEntityID(w, r, "userID")
	if !ok {
		return
	}
	if err := a.DB.SetSalesEnabled(r.Context(), *account.ShopID, userID, enabled); err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) ResetSalesPassword(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_profile")
		return
	}
	userID, ok := pathEntityID(w, r, "userID")
	if !ok {
		return
	}
	if err := a.DB.SetSalesPassword(r.Context(), *account.ShopID, userID, hash); err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) OpenShop(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platform(w, r); !ok {
		return
	}
	var body struct {
		Name          string `json:"name"`
		Slug          string `json:"slug"`
		Notice        string `json:"notice"`
		AdminLogin    string `json:"adminLogin"`
		AdminPassword string `json:"adminPassword"`
	}
	if err := readJSON(r, &body); err != nil || !slugPattern.MatchString(body.Slug) || body.Name == "" {
		writeErr(w, http.StatusBadRequest, "invalid_shop")
		return
	}
	hash, err := auth.HashPassword(body.AdminPassword)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_profile")
		return
	}
	id, err := a.DB.OpenShop(r.Context(), body.Name, body.Slug, body.Notice, body.AdminLogin, hash)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id.String(), "slug": body.Slug, "adminLogin": body.AdminLogin})
}

func (a *API) PlatformShops(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platform(w, r); !ok {
		return
	}
	shops, err := a.DB.ListShops(r.Context())
	if err != nil {
		logging.ErrorContext(r.Context(), "platform shops", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	out := make([]any, 0, len(shops))
	for _, shop := range shops {
		out = append(out, map[string]any{
			"id": shop.ID, "name": shop.Name, "slug": shop.Slug, "notice": shop.Notice, "isEnabled": shop.Enabled,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) SetPlatformShop(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platform(w, r); !ok {
		return
	}
	var body struct {
		IsEnabled bool `json:"isEnabled"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	shopID, ok := pathEntityID(w, r, "shopID")
	if !ok {
		return
	}
	if err := a.DB.SetShopEnabled(r.Context(), shopID, body.IsEnabled); err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) PlatformGoods(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platform(w, r); !ok {
		return
	}
	shops, err := a.DB.ListPlatformGoods(r.Context())
	if err != nil {
		logging.ErrorContext(r.Context(), "platform goods", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	bundlesByShop := make(map[string][]data.BundleCard)
	if a.BundlesEnabled {
		bundleShops, err := a.DB.ListPlatformBundles(r.Context())
		if err != nil {
			logging.ErrorContext(r.Context(), "platform bundles", zap.Error(err))
			writeErr(w, http.StatusInternalServerError, "internal")
			return
		}
		for _, bundleShop := range bundleShops {
			bundlesByShop[bundleShop.ShopID] = bundleShop.Bundles
		}
	}
	out := make([]any, 0, len(shops))
	for _, shop := range shops {
		goods := make([]any, 0, len(shop.Goods))
		for _, good := range shop.Goods {
			listings := make([]any, 0, len(good.Listings))
			reserved := 0
			for _, listing := range good.Listings {
				reserved += listing.ReservedQty
			}
			available := good.TotalStock - reserved
			if available < 0 {
				available = 0
			}
			for _, listing := range good.Listings {
				listings = append(listings, map[string]any{
					"listingId": listing.ListingID, "eventId": listing.EventID, "eventName": listing.EventName,
					"currency": listing.Currency, "unitPriceMinor": listing.UnitPriceMinor,
					"priceLabel": priceLabel(listing.Currency, listing.UnitPriceMinor), "priceTiers": listing.PriceTiers,
					"remaining":   available,
					"reservedQty": listing.ReservedQty, "isListed": listing.IsListed,
				})
			}
			images := make([]any, 0, len(good.ImageIDs))
			for sortOrder, imageID := range good.ImageIDs {
				url := "/api/platform/goods/" + good.ID + "/images/" + imageID
				images = append(images, map[string]any{"id": imageID, "url": url, "sortOrder": sortOrder})
			}
			var imageURL any
			if len(images) > 0 {
				imageURL = images[0].(map[string]any)["url"]
			}
			goods = append(goods, map[string]any{
				"id": good.ID, "name": good.Name, "description": good.Description,
				"reviewStatus": good.ReviewStatus, "reviewNote": good.ReviewNote, "revision": good.Revision,
				"totalStock": good.TotalStock, "listings": listings, "images": images, "imageURL": imageURL,
			})
		}
		out = append(out, map[string]any{
			"shopId": shop.ShopID, "shopName": shop.ShopName, "shopSlug": shop.ShopSlug,
			"goods": goods, "bundles": bundleCardsJSON(bundlesByShop[shop.ShopID], "platform"),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) ReviewGood(w http.ResponseWriter, r *http.Request) {
	account, ok := a.platform(w, r)
	if !ok {
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
	if (body.Status != "approved" && body.Status != "rejected") || body.ExpectedRevision <= 0 || utf8.RuneCountInString(body.Note) > 500 || (body.Status == "rejected" && body.Note == "") {
		writeErr(w, http.StatusBadRequest, "invalid_review")
		return
	}
	if body.Status == "approved" {
		body.Note = ""
	}
	goodID, ok := pathEntityID(w, r, "goodID")
	if !ok {
		return
	}
	if err := a.DB.ReviewGood(r.Context(), account.ID, goodID, body.Status, body.Note, body.ExpectedRevision); err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

const maxBlobBytes = 6 << 20

var errBlobTooLarge = errors.New("blob exceeds size limit")

// limitWriter counts bytes written and aborts once the limit is exceeded, so
// the multipart header.Size alone cannot be trusted to enforce the cap.
type limitWriter struct {
	file    *os.File
	limit   int64
	written int64
}

func (w *limitWriter) Write(p []byte) (int, error) {
	if w.written+int64(len(p)) > w.limit {
		return 0, errBlobTooLarge
	}
	n, err := w.file.Write(p)
	w.written += int64(n)
	return n, err
}

func saveBlob(ctx context.Context, dir string, src io.Reader) (string, error) {
	buf := make([]byte, 512)
	n, err := io.ReadFull(src, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return "", err
	}
	if int64(n) > maxBlobBytes {
		return "", errBlobTooLarge
	}
	kind := http.DetectContentType(buf[:n])
	if kind != "image/jpeg" && kind != "image/png" && kind != "image/webp" {
		return "", strconv.ErrSyntax
	}
	nameBytes := make([]byte, 16)
	if _, err := rand.Read(nameBytes); err != nil {
		return "", err
	}
	name := hex.EncodeToString(nameBytes)
	folder := filepath.Join(dir, "blobs")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return "", err
	}
	file, err := os.Create(filepath.Join(folder, name))
	if err != nil {
		return "", err
	}
	path := filepath.Join(folder, name)
	stored := false
	defer func() {
		if err := file.Close(); err != nil {
			logging.ErrorContext(ctx, "close blob", zap.Error(err))
		}
		if !stored {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				logging.ErrorContext(ctx, "remove incomplete blob", zap.Error(err))
			}
		}
	}()
	writer := &limitWriter{file: file, limit: maxBlobBytes, written: int64(n)}
	if _, err := writer.Write(buf[:n]); err != nil {
		return "", err
	}
	if _, err := io.Copy(writer, src); err != nil {
		return "", err
	}
	stored = true
	return name, nil
}
