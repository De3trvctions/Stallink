package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/De3trvctions/Stallink/internal/auth"
	"github.com/De3trvctions/Stallink/internal/catalog"
	"github.com/De3trvctions/Stallink/internal/data"
	"github.com/De3trvctions/Stallink/internal/logging"
	"github.com/De3trvctions/Stallink/internal/reviewswitch"
	"go.uber.org/zap"
)

type API struct {
	DB                   *data.ShopStore
	SecureCookies        bool
	DataDir              string
	RequireIdempotency   bool
	BundlesEnabled       bool
	QuoteTTL             time.Duration
	MaxPricingStates     int
	ReservationsEnabled  bool
	ReservationTTL       time.Duration
	ReviewSwitch         *reviewswitch.Switch
	AllowScheduledOrders bool
	ReportTimezone       string
}

func (a *API) current(w http.ResponseWriter, r *http.Request) (data.Account, bool) {
	cookie, err := r.Cookie("stallink_session")
	if err != nil || cookie.Value == "" {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return data.Account{}, false
	}
	hash, err := auth.HashToken(cookie.Value)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return data.Account{}, false
	}
	account, err := a.DB.AccountByToken(r.Context(), hash)
	if errors.Is(err, data.ErrNotFound) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return data.Account{}, false
	}
	if err != nil {
		logging.ErrorContext(r.Context(), "session lookup failed", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return data.Account{}, false
	}
	*r = *r.WithContext(logging.WithFields(r.Context(), zap.String("user_id", account.ID.String()), zap.String("role", account.Role)))
	return account, true
}

func pathEntityID(w http.ResponseWriter, r *http.Request, key string) (data.ID, bool) {
	id, err := data.ParseID(r.PathValue(key))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_id")
		return 0, false
	}
	return id, true
}

func (a *API) staff(w http.ResponseWriter, r *http.Request) (data.Account, bool) {
	account, ok := a.current(w, r)
	if !ok {
		return data.Account{}, false
	}
	if (account.Role != "sales" && account.Role != "shop_admin") || account.ShopID == nil {
		writeErr(w, http.StatusForbidden, "forbidden")
		return data.Account{}, false
	}
	return account, true
}

func (a *API) shopAdmin(w http.ResponseWriter, r *http.Request) (data.Account, bool) {
	account, ok := a.current(w, r)
	if !ok {
		return data.Account{}, false
	}
	if account.Role != "shop_admin" || account.ShopID == nil {
		writeErr(w, http.StatusForbidden, "forbidden")
		return data.Account{}, false
	}
	return account, true
}

func (a *API) platform(w http.ResponseWriter, r *http.Request) (data.Account, bool) {
	account, ok := a.current(w, r)
	if !ok {
		return data.Account{}, false
	}
	if account.Role != "platform_admin" {
		writeErr(w, http.StatusForbidden, "forbidden")
		return data.Account{}, false
	}
	return account, true
}

func (a *API) setSession(w http.ResponseWriter, plain string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "stallink_session",
		Value:    plain,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   14 * 24 * 3600,
		Secure:   a.SecureCookies,
	})
}

func (a *API) clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: "stallink_session", Value: "", Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, MaxAge: -1, Secure: a.SecureCookies,
	})
}

// dummyPasswordHash is a throwaway bcrypt hash used to equalize login timing
// when the account does not exist, so "unknown login" and "wrong password"
// take comparable time instead of leaking account existence through latency.
var dummyPasswordHash = sync.OnceValue(func() string {
	hash, _ := auth.HashPassword("stallink-timing-equalizer-dummy-0001")
	return hash
})

func (a *API) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		LoginName string `json:"loginName"`
		Password  string `json:"password"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	account, err := a.DB.AccountByLogin(r.Context(), strings.TrimSpace(body.LoginName))
	if errors.Is(err, data.ErrNotFound) {
		// Run a bcrypt comparison against the dummy hash so the unknown-login
		// branch costs about the same as a real password check.
		auth.CheckPassword(dummyPasswordHash(), body.Password)
		writeErr(w, http.StatusUnauthorized, "login_failed")
		return
	}
	if err != nil {
		logging.ErrorContext(r.Context(), "login failed", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	if !account.Enabled || !auth.CheckPassword(account.PasswordHash, body.Password) {
		writeErr(w, http.StatusUnauthorized, "login_failed")
		return
	}
	a.startSession(w, r, account)
}

func (a *API) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		LoginName   string `json:"loginName"`
		Password    string `json:"password"`
		DisplayName string `json:"displayName"`
		Contact     string `json:"contact"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	login := strings.TrimSpace(body.LoginName)
	if !validLoginName(login) || !validProfileName(body.DisplayName, body.Contact) {
		writeErr(w, http.StatusBadRequest, "invalid_profile")
		return
	}
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_profile")
		return
	}
	id, err := a.DB.CreateAccount(r.Context(), login, hash, "customer", nil, body.DisplayName, body.Contact)
	if errors.Is(err, data.ErrConflict) {
		writeErr(w, http.StatusConflict, "login_taken")
		return
	}
	if err != nil {
		logging.ErrorContext(r.Context(), "register account failed", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	account := data.Account{ID: id, LoginName: login, Role: "customer", DisplayName: body.DisplayName, Contact: body.Contact, Enabled: true}
	a.startSession(w, r, account)
}

func (a *API) startSession(w http.ResponseWriter, r *http.Request, account data.Account) {
	plain, hash, err := auth.NewToken()
	if err != nil {
		logging.ErrorContext(r.Context(), "create session token failed", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	if err := a.DB.InsertSession(r.Context(), account.ID, hash, time.Now().Add(14*24*time.Hour)); err != nil {
		logging.ErrorContext(r.Context(), "store session failed", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	a.setSession(w, plain)
	writeJSON(w, http.StatusOK, meJSON(account))
}

func (a *API) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("stallink_session"); err == nil {
		if hash, err := auth.HashToken(cookie.Value); err == nil {
			if err := a.DB.DeleteSession(r.Context(), hash); err != nil {
				logging.ErrorContext(r.Context(), "delete session failed", zap.Error(err))
			}
		}
	}
	a.clearSession(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) Me(w http.ResponseWriter, r *http.Request) {
	account, ok := a.current(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, meJSON(account))
}

func (a *API) UpdateMe(w http.ResponseWriter, r *http.Request) {
	account, ok := a.current(w, r)
	if !ok {
		return
	}
	if account.Role != "customer" {
		writeErr(w, http.StatusForbidden, "forbidden")
		return
	}
	var body struct {
		DisplayName string `json:"displayName"`
		Contact     string `json:"contact"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if !validProfileName(body.DisplayName, body.Contact) {
		writeErr(w, http.StatusBadRequest, "invalid_profile")
		return
	}
	if err := a.DB.UpdateProfile(r.Context(), account.ID, body.DisplayName, body.Contact); err != nil {
		writeDataErr(w, r, err)
		return
	}
	account.DisplayName = body.DisplayName
	account.Contact = body.Contact
	writeJSON(w, http.StatusOK, meJSON(account))
}

func (a *API) Landing(w http.ResponseWriter, r *http.Request) {
	shop, err := a.DB.SoleEnabledShop(r.Context())
	if errors.Is(err, data.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "no_booth")
		return
	}
	if err != nil {
		logging.ErrorContext(r.Context(), "load landing page failed", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	a.writeShop(w, r, shop)
}

func (a *API) Shop(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if !slugPattern.MatchString(slug) {
		writeErr(w, http.StatusNotFound, "no_booth")
		return
	}
	shop, err := a.DB.EnabledShop(r.Context(), slug)
	if errors.Is(err, data.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "no_booth")
		return
	}
	if err != nil {
		logging.ErrorContext(r.Context(), "load shop failed", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	a.writeShop(w, r, shop)
}

func (a *API) writeShop(w http.ResponseWriter, r *http.Request, shop data.Shop) {
	events, goods, err := a.DB.PublicCatalog(r.Context(), shop.ID)
	if err != nil {
		logging.ErrorContext(r.Context(), "load public catalog failed", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	bundles := []data.BundleCard{}
	if a.BundlesEnabled {
		bundles, err = a.DB.PublicBundles(r.Context(), shop.ID, 0)
		if err != nil {
			logging.ErrorContext(r.Context(), "load public bundles failed", zap.Error(err))
			writeErr(w, http.StatusInternalServerError, "internal")
			return
		}
	}
	writeJSON(w, http.StatusOK, shopJSON(shop, events, goods, bundles))
}

func (a *API) Event(w http.ResponseWriter, r *http.Request) {
	shop, ok := a.publicShop(w, r)
	if !ok {
		return
	}
	eventID, valid := pathEntityID(w, r, "eventID")
	if !valid {
		return
	}
	event, goods, err := a.DB.PublicEventGoods(r.Context(), shop.ID, eventID)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	bundles := []any{}
	if a.BundlesEnabled {
		rows, err := a.DB.PublicBundles(r.Context(), shop.ID, event.ID)
		if err != nil {
			writeDataErr(w, r, err)
			return
		}
		bundles = bundleCardsJSON(rows, "public")
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"event":   eventJSON(event),
		"goods":   goodsJSON(goods),
		"bundles": bundles,
	})
}

func (a *API) PublicBundle(w http.ResponseWriter, r *http.Request) {
	shop, ok := a.publicShop(w, r)
	if !ok {
		return
	}
	if !a.BundlesEnabled {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	bundles, err := a.DB.PublicBundles(r.Context(), shop.ID, 0)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	listingID, valid := pathEntityID(w, r, "bundleListingID")
	if !valid {
		return
	}
	for _, bundle := range bundles {
		if bundle.ListingID == listingID.String() {
			writeJSON(w, http.StatusOK, bundleJSON(bundle, "public"))
			return
		}
	}
	writeErr(w, http.StatusNotFound, "not_found")
}

func (a *API) Listing(w http.ResponseWriter, r *http.Request) {
	shop, ok := a.publicShop(w, r)
	if !ok {
		return
	}
	listingID, valid := pathEntityID(w, r, "listingID")
	if !valid {
		return
	}
	good, err := a.DB.PublicListing(r.Context(), shop.ID, listingID)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	body := goodJSON(good)
	body["description"] = good.Description
	writeJSON(w, http.StatusOK, body)
}

func (a *API) publicShop(w http.ResponseWriter, r *http.Request) (data.Shop, bool) {
	slug := r.PathValue("slug")
	if !slugPattern.MatchString(slug) {
		writeErr(w, http.StatusNotFound, "no_booth")
		return data.Shop{}, false
	}
	shop, err := a.DB.EnabledShop(r.Context(), slug)
	if errors.Is(err, data.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "no_booth")
		return data.Shop{}, false
	}
	if err != nil {
		logging.ErrorContext(r.Context(), "load shop failed", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return data.Shop{}, false
	}
	return shop, true
}

func (a *API) GoodImage(w http.ResponseWriter, r *http.Request) {
	imageID, valid := pathEntityID(w, r, "imageID")
	if !valid {
		return
	}
	path, err := a.DB.GoodImagePath(r.Context(), imageID)
	if errors.Is(err, data.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		logging.ErrorContext(r.Context(), "load image failed", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	a.serveBlob(w, r, path)
}

func (a *API) BundleImage(w http.ResponseWriter, r *http.Request) {
	if !a.BundlesEnabled {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	imageID, valid := pathEntityID(w, r, "imageID")
	if !valid {
		return
	}
	path, err := a.DB.PublicBundleImagePath(r.Context(), imageID)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	a.serveBlob(w, r, path)
}

func (a *API) StaffBundleImage(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	bundleID, valid := pathEntityID(w, r, "bundleID")
	if !valid {
		return
	}
	imageID, valid := pathEntityID(w, r, "imageID")
	if !valid {
		return
	}
	path, err := a.DB.BundleImagePathForShop(r.Context(), *account.ShopID, bundleID, imageID)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	a.serveBlob(w, r, path)
}

func (a *API) PlatformBundleImage(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platform(w, r); !ok {
		return
	}
	bundleID, valid := pathEntityID(w, r, "bundleID")
	if !valid {
		return
	}
	imageID, valid := pathEntityID(w, r, "imageID")
	if !valid {
		return
	}
	path, err := a.DB.BundleImagePathForPlatform(r.Context(), bundleID, imageID)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	a.serveBlob(w, r, path)
}

func (a *API) StaffGoodImage(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	goodID, valid := pathEntityID(w, r, "goodID")
	if !valid {
		return
	}
	imageID, valid := pathEntityID(w, r, "imageID")
	if !valid {
		return
	}
	path, err := a.DB.GoodImagePathForShop(r.Context(), *account.ShopID, goodID, imageID)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	a.serveBlob(w, r, path)
}

func (a *API) PlatformGoodImage(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platform(w, r); !ok {
		return
	}
	goodID, valid := pathEntityID(w, r, "goodID")
	if !valid {
		return
	}
	imageID, valid := pathEntityID(w, r, "imageID")
	if !valid {
		return
	}
	path, err := a.DB.GoodImagePathForPlatform(r.Context(), goodID, imageID)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	a.serveBlob(w, r, path)
}

func removeBlob(ctx context.Context, dir, name string) {
	if name == "" {
		return
	}
	path := filepath.Join(dir, "blobs", filepath.Base(name))
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		logging.ErrorContext(ctx, "remove image blob failed", zap.Error(err))
	}
}

func (a *API) PayCodeImage(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	id, valid := pathEntityID(w, r, "id")
	if !valid {
		return
	}
	shopID, path, err := a.DB.PayCodePath(r.Context(), id)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	if account.ShopID == nil || *account.ShopID != shopID {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	a.serveBlob(w, r, path)
}

func (a *API) serveBlob(w http.ResponseWriter, r *http.Request, name string) {
	base := filepath.Base(name)
	full := filepath.Join(a.DataDir, "blobs", base)
	rel, err := filepath.Rel(filepath.Join(a.DataDir, "blobs"), full)
	if err != nil || strings.HasPrefix(rel, "..") {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	file, err := os.Open(full)
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			logging.ErrorContext(r.Context(), "close image file failed", zap.Error(err))
		}
	}()
	stat, err := file.Stat()
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	http.ServeContent(w, r, base, stat.ModTime(), file)
}

func meJSON(account data.Account) map[string]any {
	var shopID, shopSlug any
	if account.ShopID != nil {
		shopID = *account.ShopID
	}
	if account.ShopSlug != nil {
		shopSlug = *account.ShopSlug
	}
	return map[string]any{
		"id": account.ID, "loginName": account.LoginName, "role": account.Role,
		"shopId": shopID, "shopSlug": shopSlug, "displayName": account.DisplayName, "contact": account.Contact,
	}
}

func shopJSON(shop data.Shop, events []data.Event, goods []data.GoodCard, bundles []data.BundleCard) map[string]any {
	grouped := map[string][]any{
		"scheduled": make([]any, 0),
		"live":      make([]any, 0),
		"ended":     make([]any, 0),
	}
	for _, event := range events {
		grouped[event.Status] = append(grouped[event.Status], eventJSON(event))
	}
	out := map[string]any{
		"id": shop.ID, "name": shop.Name, "notice": shop.Notice, "coverURL": nil, "slug": shop.Slug,
		"events": grouped, "goods": goodsJSON(goods), "bundles": bundleCardsJSON(bundles, "public"),
	}
	if shop.FeaturedEventID != 0 {
		out["featuredEventId"] = shop.FeaturedEventID.String()
	}
	return out
}

func eventJSON(event data.Event) map[string]any {
	return map[string]any{
		"id": event.ID, "name": event.Name,
		"startsOn": event.StartsOn.Format(time.DateOnly), "endsOn": event.EndsOn.Format(time.DateOnly),
		"city": event.City, "venue": event.Venue, "boothNumber": event.BoothNumber,
		"currency": event.Currency, "status": event.Status,
	}
}

func goodsJSON(goods []data.GoodCard) []any {
	out := make([]any, 0, len(goods))
	for _, good := range goods {
		out = append(out, goodJSON(good))
	}
	return out
}

func goodJSON(good data.GoodCard) map[string]any {
	images := make([]any, 0, len(good.ImageIDs))
	for sortOrder, imageID := range good.ImageIDs {
		images = append(images, map[string]any{
			"id": imageID, "url": "/api/files/goods/" + imageID, "sortOrder": sortOrder,
		})
	}
	var image any
	if len(images) > 0 {
		image = images[0].(map[string]any)["url"]
	}
	return map[string]any{
		"listingId": good.ListingID, "goodId": good.GoodID, "eventId": good.EventID, "eventName": good.EventName,
		"name": good.Name, "description": good.Description, "currency": good.Currency,
		"unitPriceMinor": good.UnitPriceMinor, "priceLabel": priceLabel(good.Currency, good.UnitPriceMinor),
		"priceTiers": good.PriceTiers, "remaining": good.Remaining, "soldOut": good.SoldOut,
		"isListed": good.IsListed, "imageURL": image, "images": images,
	}
}

func staffGoodsJSON(goods []data.GoodCard) []any {
	out := make([]any, 0, len(goods))
	for _, good := range goods {
		item := goodJSON(good)
		images := make([]any, 0, len(good.ImageIDs))
		for sortOrder, imageID := range good.ImageIDs {
			url := "/api/staff/goods/" + good.GoodID + "/images/" + imageID
			images = append(images, map[string]any{"id": imageID, "url": url, "sortOrder": sortOrder})
		}
		item["images"] = images
		item["imageURL"] = nil
		if len(images) > 0 {
			item["imageURL"] = images[0].(map[string]any)["url"]
		}
		item["reviewStatus"] = good.ReviewStatus
		item["reviewNote"] = good.ReviewNote
		item["listedQty"] = good.ListedQty
		item["soldQty"] = good.SoldQty
		item["reservedQty"] = good.ReservedQty
		item["totalStock"] = good.TotalStock
		out = append(out, item)
	}
	return out
}

func priceLabel(currency string, minor int64) string {
	label, err := catalog.FormatMoney(currency, minor)
	if err != nil {
		return ""
	}
	return label
}

func writeDataErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, data.ErrNotFound):
		writeErr(w, http.StatusNotFound, "not_found")
	case errors.Is(err, data.ErrConflict):
		writeErr(w, http.StatusConflict, "conflict")
	case errors.Is(err, data.ErrIdempotencyConflict):
		writeErr(w, http.StatusConflict, "idempotency_conflict")
	case errors.Is(err, data.ErrCheckoutConflict):
		writeErr(w, http.StatusConflict, "checkout_conflict")
	case errors.Is(err, data.ErrCommandInProgress):
		writeErr(w, http.StatusConflict, "command_in_progress")
	case errors.Is(err, data.ErrCommandUnauthorized):
		writeErr(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, data.ErrReservationExpired):
		writeErr(w, http.StatusConflict, "reservation_expired")
	case errors.Is(err, data.ErrQuoteExpired):
		writeErr(w, http.StatusConflict, "quote_expired")
	case errors.Is(err, data.ErrQuoteStale):
		writeErr(w, http.StatusConflict, "quote_stale")
	case errors.Is(err, data.ErrQuoteInvalid):
		writeErr(w, http.StatusConflict, "quote_invalid")
	case errors.Is(err, data.ErrStatus):
		writeErr(w, http.StatusConflict, "invalid_status")
	case errors.Is(err, data.ErrPayCode):
		writeErr(w, http.StatusConflict, "pay_code_required")
	case errors.Is(err, data.ErrPayCodeCurrencyMismatch):
		writeErr(w, http.StatusBadRequest, "pay_code_currency_mismatch")
	case errors.Is(err, data.ErrInvalidCashReceived):
		writeErr(w, http.StatusBadRequest, "invalid_cash_received")
	case errors.Is(err, data.ErrEventNotOpen):
		writeErr(w, http.StatusBadRequest, "event_not_open")
	case errors.Is(err, data.ErrNotListed):
		writeErr(w, http.StatusBadRequest, "not_listed")
	case errors.Is(err, data.ErrInvalidBundle):
		writeErr(w, http.StatusBadRequest, "invalid_bundle")
	case errors.Is(err, data.ErrBundleComponents):
		writeErr(w, http.StatusConflict, "bundle_components_not_eligible")
	case errors.Is(err, data.ErrImageLimit):
		writeErr(w, http.StatusBadRequest, "image_limit")
	case errors.Is(err, catalog.ErrNotEnough):
		writeErr(w, http.StatusConflict, "not_enough_stock")
	case errors.Is(err, catalog.ErrStockExceeded):
		writeErr(w, http.StatusConflict, "stock_exceeded")
	case errors.Is(err, catalog.ErrInvalidPrice):
		writeErr(w, http.StatusBadRequest, "invalid_price_tiers")
	case errors.Is(err, catalog.ErrInvalidCart):
		writeErr(w, http.StatusBadRequest, "invalid_cart")
	case errors.Is(err, catalog.ErrPricingTooLarge):
		writeErr(w, http.StatusUnprocessableEntity, "pricing_complexity_exceeded")
	default:
		logging.ErrorContext(r.Context(), "request failed", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
	}
}

const maxJSONBodyBytes = 1 << 20

// readJSON bounds request allocation and rejects trailing JSON values.
func readJSON(r *http.Request, dest any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxJSONBodyBytes)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dest); err != nil {
		return err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func readJSONStrict(r *http.Request, dest any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxJSONBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		return err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}
