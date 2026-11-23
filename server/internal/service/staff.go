package service

import (
	"encoding/json"
	"mime/multipart"
	"strings"

	"github.com/De3trvctions/Stallink/internal/logging"
	"go.uber.org/zap"
	"math"
	"net/http"
	"time"

	"github.com/De3trvctions/Stallink/internal/catalog"
	"github.com/De3trvctions/Stallink/internal/data"
	"github.com/De3trvctions/Stallink/internal/imagestore"
	"github.com/google/uuid"
)

const (
	maxGoodImages       = 8
	maxGoodImageBytes   = imagestore.MaxUploadBytes
	maxGoodUploadBytes  = 50 << 20
	goodImageMemorySize = 8 << 20
)

func (a *API) StaffEvents(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	events, err := a.DB.ListEvents(r.Context(), *account.ShopID)
	if err != nil {
		logging.ErrorContext(r.Context(), "events", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	out := make([]any, 0, len(events))
	for _, event := range events {
		out = append(out, eventJSON(event))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) CreateEvent(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	in, ok := readEvent(w, r)
	if !ok {
		return
	}
	id, err := a.DB.CreateEvent(r.Context(), *account.ShopID, in)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id.String()})
}

func (a *API) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	in, ok := readEvent(w, r)
	if !ok {
		return
	}
	eventID, valid := pathEntityID(w, r, "eventID")
	if !valid {
		return
	}
	if err := a.DB.UpdateEvent(r.Context(), *account.ShopID, eventID, in); err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) StaffListings(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	eventID, valid := pathEntityID(w, r, "eventID")
	if !valid {
		return
	}
	goods, err := a.DB.StaffListings(r.Context(), *account.ShopID, eventID)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, staffGoodsJSON(goods))
}

// StaffGoods returns the good-centric catalog: one entry per good with its
// shared stock and every event listing, so the management page can edit a good
// once instead of once per event.
func (a *API) StaffGoods(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	goods, err := a.DB.ShopGoods(r.Context(), *account.ShopID)
	if err != nil {
		logging.ErrorContext(r.Context(), "staff goods", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	out := make([]any, 0, len(goods))
	for _, good := range goods {
		listings := make([]any, 0, len(good.Listings))
		sold, reserved := 0, 0
		for _, listing := range good.Listings {
			sold += listing.SoldQty
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
				"listedQty": listing.ListedQty, "soldQty": listing.SoldQty, "reservedQty": listing.ReservedQty,
				"remaining": available, "isListed": listing.IsListed,
			})
		}
		images := make([]any, 0, len(good.ImageIDs))
		for sortOrder, imageID := range good.ImageIDs {
			url := "/api/staff/goods/" + good.ID + "/images/" + imageID
			images = append(images, map[string]any{"id": imageID, "url": url, "sortOrder": sortOrder})
		}
		var imageURL any
		if len(images) > 0 {
			imageURL = images[0].(map[string]any)["url"]
		}
		out = append(out, map[string]any{
			"id": good.ID, "name": good.Name, "description": good.Description,
			"reviewStatus": good.ReviewStatus, "reviewNote": good.ReviewNote, "revision": good.Revision,
			"totalStock": good.TotalStock, "soldQty": sold, "reservedQty": reserved, "availableQty": available,
			"listings": listings, "images": images, "imageURL": imageURL,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) CreateGood(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	var body struct {
		Name           string              `json:"name"`
		Description    string              `json:"description"`
		TotalStock     int                 `json:"totalStock"`
		EventID        data.ID             `json:"eventId"`
		Event          *data.EventInput    `json:"event"`
		ListedQty      int                 `json:"listedQty"`
		UnitPriceMinor int64               `json:"unitPriceMinor"`
		PriceTiers     []catalog.PriceTier `json:"priceTiers"`
		IsListed       bool                `json:"isListed"`
	}
	var files []*multipart.FileHeader
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		r.Body = http.MaxBytesReader(w, r.Body, maxGoodUploadBytes)
		if err := r.ParseMultipartForm(goodImageMemorySize); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid_image")
			return
		}
		if r.MultipartForm != nil {
			defer func() {
				if err := r.MultipartForm.RemoveAll(); err != nil {
					logging.ErrorContext(r.Context(), "remove multipart temp files", zap.Error(err))
				}
			}()
		}
		if err := json.Unmarshal([]byte(r.FormValue("product")), &body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if r.MultipartForm != nil {
			files = r.MultipartForm.File["images"]
		}
		if len(files) > maxGoodImages {
			writeErr(w, http.StatusBadRequest, "image_limit")
			return
		}
	} else if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if (body.EventID == 0) == (body.Event == nil) {
		writeErr(w, http.StatusBadRequest, "event_required")
		return
	}
	if body.Event != nil && !validEvent(*body.Event) {
		writeErr(w, http.StatusBadRequest, "invalid_event")
		return
	}
	if body.UnitPriceMinor <= 0 || body.TotalStock < 0 || body.Name == "" {
		writeErr(w, http.StatusBadRequest, "invalid_good")
		return
	}
	if len([]rune(body.Name)) > 120 || len([]rune(body.Description)) > 4000 {
		writeErr(w, http.StatusBadRequest, "invalid_good")
		return
	}
	if err := catalog.ValidatePriceTiers(body.UnitPriceMinor, body.PriceTiers); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_price_tiers")
		return
	}
	upload := imageUpload{api: a, shopID: *account.ShopID}
	imagePaths, err := upload.putCatalog(r.Context(), "good", files)
	if err != nil {
		writeImageError(w, r, err)
		return
	}
	goodID, listingID, eventID, err := a.DB.CreateListedGood(r.Context(), *account.ShopID, account.ID, data.ListedGood{
		Name: body.Name, Description: body.Description, TotalStock: body.TotalStock,
		EventID: body.EventID, NewEvent: body.Event,
		UnitPriceMinor: body.UnitPriceMinor, PriceTiers: body.PriceTiers, IsListed: body.IsListed,
		ImagePaths: imagePaths, Approved: a.autoApprove(r.Context()),
	})
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"goodId": goodID.String(), "listingId": listingID.String(), "eventId": eventID.String()})
}

func (a *API) AddGoodImages(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxGoodUploadBytes)
	if err := r.ParseMultipartForm(goodImageMemorySize); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_image")
		return
	}
	if r.MultipartForm != nil {
		defer func() {
			if err := r.MultipartForm.RemoveAll(); err != nil {
				logging.ErrorContext(r.Context(), "remove multipart temp files", zap.Error(err))
			}
		}()
	}
	if r.MultipartForm == nil {
		writeErr(w, http.StatusBadRequest, "invalid_image")
		return
	}
	files := r.MultipartForm.File["images"]
	if len(files) == 0 || len(files) > maxGoodImages {
		writeErr(w, http.StatusBadRequest, "image_limit")
		return
	}
	upload := imageUpload{api: a, shopID: *account.ShopID}
	paths, err := upload.putCatalog(r.Context(), "good", files)
	if err != nil {
		writeImageError(w, r, err)
		return
	}
	goodID, valid := pathEntityID(w, r, "goodID")
	if !valid {
		return
	}
	ids, err := a.DB.AddGoodImages(r.Context(), *account.ShopID, account.ID, goodID, paths, a.autoApprove(r.Context()))
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	images := make([]any, 0, len(ids))
	for index, id := range ids {
		images = append(images, map[string]any{
			"id": id, "url": "/api/staff/goods/" + goodID.String() + "/images/" + id,
			"sortOrder": index,
		})
	}
	writeJSON(w, http.StatusCreated, map[string]any{"images": images})
}

func (a *API) DeleteGoodImage(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
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
	_, err := a.DB.DeleteGoodImage(r.Context(), *account.ShopID, account.ID, goodID, imageID, a.autoApprove(r.Context()))
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) ReorderGoodImages(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	var body struct {
		ImageIDs []data.ID `json:"imageIds"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	goodID, valid := pathEntityID(w, r, "goodID")
	if !valid {
		return
	}
	if err := a.DB.ReorderGoodImages(r.Context(), *account.ShopID, account.ID, goodID, body.ImageIDs, a.autoApprove(r.Context())); err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) AddListing(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	var body struct {
		EventID        data.ID             `json:"eventId"`
		UnitPriceMinor int64               `json:"unitPriceMinor"`
		PriceTiers     []catalog.PriceTier `json:"priceTiers"`
		IsListed       bool                `json:"isListed"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if body.EventID == 0 || body.UnitPriceMinor <= 0 {
		writeErr(w, http.StatusBadRequest, "invalid_listing")
		return
	}
	if err := catalog.ValidatePriceTiers(body.UnitPriceMinor, body.PriceTiers); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_price_tiers")
		return
	}
	goodID, valid := pathEntityID(w, r, "goodID")
	if !valid {
		return
	}
	id, err := a.DB.AddListing(r.Context(), *account.ShopID, account.ID, goodID, data.ListedGood{
		EventID: body.EventID, UnitPriceMinor: body.UnitPriceMinor, PriceTiers: body.PriceTiers, IsListed: body.IsListed,
	}, a.autoApprove(r.Context()))
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id.String()})
}

func (a *API) UpdateListing(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	var body struct {
		UnitPriceMinor int64               `json:"unitPriceMinor"`
		PriceTiers     []catalog.PriceTier `json:"priceTiers"`
		IsListed       bool                `json:"isListed"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if body.UnitPriceMinor <= 0 {
		writeErr(w, http.StatusBadRequest, "invalid_listing")
		return
	}
	if err := catalog.ValidatePriceTiers(body.UnitPriceMinor, body.PriceTiers); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_price_tiers")
		return
	}
	listingID, valid := pathEntityID(w, r, "listingID")
	if !valid {
		return
	}
	if err := a.DB.UpdateListing(r.Context(), *account.ShopID, account.ID, listingID, body.UnitPriceMinor, body.PriceTiers, body.IsListed); err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) UpdateGood(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		TotalStock  int    `json:"totalStock"`
		Listings    []struct {
			ListingID      data.ID             `json:"listingId"`
			UnitPriceMinor int64               `json:"unitPriceMinor"`
			PriceTiers     []catalog.PriceTier `json:"priceTiers"`
			IsListed       bool                `json:"isListed"`
		} `json:"listings"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if body.Name == "" || len([]rune(body.Name)) > 120 || len([]rune(body.Description)) > 4000 || body.TotalStock < 0 {
		writeErr(w, http.StatusBadRequest, "invalid_good")
		return
	}
	if len(body.Listings) > 100 {
		writeErr(w, http.StatusBadRequest, "invalid_listing")
		return
	}
	for _, listing := range body.Listings {
		if listing.ListingID == 0 || listing.UnitPriceMinor <= 0 {
			writeErr(w, http.StatusBadRequest, "invalid_listing")
			return
		}
		if err := catalog.ValidatePriceTiers(listing.UnitPriceMinor, listing.PriceTiers); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid_price_tiers")
			return
		}
	}
	goodID, valid := pathEntityID(w, r, "goodID")
	if !valid {
		return
	}
	if len(body.Listings) == 0 {
		if err := a.DB.UpdateGoodStock(r.Context(), *account.ShopID, account.ID, goodID, body.Name, body.Description, body.TotalStock, a.autoApprove(r.Context())); err != nil {
			writeDataErr(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	edits := make([]data.ListingEdit, 0, len(body.Listings))
	for _, listing := range body.Listings {
		edits = append(edits, data.ListingEdit{
			ListingID: listing.ListingID, UnitPriceMinor: listing.UnitPriceMinor,
			PriceTiers: listing.PriceTiers, IsListed: listing.IsListed,
		})
	}
	if err := a.DB.UpdateGoodCatalog(r.Context(), *account.ShopID, account.ID, goodID, body.Name, body.Description, body.TotalStock, edits, a.autoApprove(r.Context())); err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) CreateOrder(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	var body struct {
		ClientOrderID      string  `json:"clientOrderId"`
		QuoteID            data.ID `json:"quoteId"`
		ExpectedTotalMinor int64   `json:"expectedTotalMinor"`
		BoothName          string  `json:"boothName"`
	}
	if err := readJSONStrict(r, &body); err != nil || body.QuoteID == 0 || body.ExpectedTotalMinor < 0 {
		writeErr(w, http.StatusBadRequest, "invalid_order")
		return
	}
	if len([]rune(body.BoothName)) > 80 {
		writeErr(w, http.StatusBadRequest, "invalid_order")
		return
	}
	identity, ok := a.commandIdentity(w, r, account, "order.create", body)
	if !ok {
		return
	}
	clientOrderID := strings.TrimSpace(body.ClientOrderID)
	if clientOrderID == "" {
		clientOrderID = identity.Key
	}
	parsedClientOrderID, err := uuid.Parse(clientOrderID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_client_order_id")
		return
	}
	clientOrderID = parsedClientOrderID.String()
	started := time.Now()
	result, err := a.DB.CreateOrderCommand(r.Context(), identity, clientOrderID, body.QuoteID, body.ExpectedTotalMinor,
		body.BoothName, a.ReservationsEnabled, a.ReservationTTL, a.AllowScheduledOrders)
	logOrderCommandResult(r.Context(), identity, result, err, started)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeOrderCommand(w, result)
}

func (a *API) CreateOrderQuote(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	var body struct {
		EventID     data.ID                 `json:"eventId"`
		CartVersion int64                   `json:"cartVersion"`
		Lines       []catalog.CartLineInput `json:"lines"`
	}
	if err := readJSONStrict(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if body.EventID == 0 || body.CartVersion < 0 || len(body.Lines) == 0 || len(body.Lines) > 100 {
		writeErr(w, http.StatusBadRequest, "invalid_cart")
		return
	}
	quoteTTL := a.QuoteTTL
	if quoteTTL <= 0 {
		quoteTTL = 60 * time.Second
	}
	maxStates := a.MaxPricingStates
	if maxStates <= 0 {
		maxStates = catalog.DefaultMaxStates
	}
	shopID := data.ID(0)
	if account.ShopID != nil {
		shopID = *account.ShopID
	}
	quoteStarted := time.Now()
	quote, err := a.DB.CreateOrderQuote(r.Context(), shopID, account.ID, body.EventID, body.CartVersion, body.Lines,
		quoteTTL, maxStates, a.BundlesEnabled, a.AllowScheduledOrders)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	logging.InfoContext(r.Context(), "order quote completed",
		zap.String("shop_id", shopID.String()), zap.String("actor_id", account.ID.String()),
		zap.String("event_id", body.EventID.String()), zap.Int("cart_lines", len(body.Lines)),
		zap.Int64("catalog_revision", quote.CatalogRevision), zap.Int64("total_minor", quote.TotalMinor), zap.Duration("duration", time.Since(quoteStarted)))
	writeJSON(w, http.StatusOK, orderQuoteJSON(quote))
}

func (a *API) StaffOrders(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	page, limit, search, valid := pageParams(r)
	if !valid {
		writeErr(w, http.StatusBadRequest, "invalid_pagination")
		return
	}
	orders, more, err := a.DB.ListOrderPage(r.Context(), *account.ShopID, r.URL.Query().Get("eventId"), "", search, r.URL.Query().Get("status"), a.ReportTimezone, limit, (page-1)*limit)
	if err != nil {
		logging.ErrorContext(r.Context(), "orders", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": ordersJSON(orders), "page": page, "hasMore": more})
}

func (a *API) GetStaffOrder(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	orderID, valid := pathEntityID(w, r, "orderID")
	if !valid {
		return
	}
	order, err := a.DB.GetOrder(r.Context(), *account.ShopID, orderID)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, orderJSON(order))
}

func (a *API) GetStaffCheckout(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	clientOrderID := r.PathValue("clientOrderID")
	if _, err := uuid.Parse(clientOrderID); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_client_order_id")
		return
	}
	order, err := a.DB.GetOrderByClientOrderID(r.Context(), *account.ShopID, clientOrderID)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, orderJSON(order))
}

func (a *API) PayCodes(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	codes, err := a.DB.ListPayCodes(r.Context(), *account.ShopID, true)
	if err != nil {
		logging.ErrorContext(r.Context(), "pay codes", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	writeJSON(w, http.StatusOK, payCodesJSON(codes))
}

func (a *API) Pay(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	var body struct {
		PayCodeID         data.ID `json:"payCodeId"`
		CashReceivedMinor *int64  `json:"cashReceivedMinor"`
	}
	if err := readJSON(r, &body); err != nil || body.PayCodeID == 0 {
		writeErr(w, http.StatusBadRequest, "pay_code_required")
		return
	}
	if body.CashReceivedMinor != nil && *body.CashReceivedMinor < 0 {
		writeErr(w, http.StatusBadRequest, "invalid_cash_received")
		return
	}
	orderID, valid := pathEntityID(w, r, "orderID")
	if !valid {
		return
	}
	a.orderActionCommand(w, r, account, "order.pay:"+orderID.String(), body, func(identity data.CommandIdentity) (data.OrderCommandResult, error) {
		return a.DB.PayOrderCommand(r.Context(), identity, orderID, body.PayCodeID, body.CashReceivedMinor, a.AllowScheduledOrders)
	})
}

func (a *API) ReserveOrder(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	if !a.ReservationsEnabled {
		writeErr(w, http.StatusConflict, "reservations_disabled")
		return
	}
	var body struct{}
	if err := readJSONStrict(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	reservationTTL := a.ReservationTTL
	if reservationTTL <= 0 {
		reservationTTL = 15 * time.Minute
	}
	orderID, valid := pathEntityID(w, r, "orderID")
	if !valid {
		return
	}
	a.orderActionCommand(w, r, account, "order.reserve:"+orderID.String(), body, func(identity data.CommandIdentity) (data.OrderCommandResult, error) {
		return a.DB.ReserveOrderCommand(r.Context(), identity, orderID, reservationTTL)
	})
}

func (a *API) Unpaid(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	orderID, valid := pathEntityID(w, r, "orderID")
	if !valid {
		return
	}
	order, err := a.DB.GetOrder(r.Context(), *account.ShopID, orderID)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	if order.Status != "pending" {
		writeErr(w, http.StatusConflict, "invalid_status")
		return
	}
	writeJSON(w, http.StatusOK, orderJSON(order))
}

func (a *API) Fulfill(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	orderID, valid := pathEntityID(w, r, "orderID")
	if !valid {
		return
	}
	a.orderActionCommand(w, r, account, "order.fulfill:"+orderID.String(), struct{}{}, func(identity data.CommandIdentity) (data.OrderCommandResult, error) {
		return a.DB.FulfillOrderCommand(r.Context(), identity, orderID)
	})
}

func (a *API) Cancel(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	orderID, valid := pathEntityID(w, r, "orderID")
	if !valid {
		return
	}
	a.orderActionCommand(w, r, account, "order.cancel:"+orderID.String(), struct{}{}, func(identity data.CommandIdentity) (data.OrderCommandResult, error) {
		return a.DB.CancelOrderCommand(r.Context(), identity, orderID)
	})
}

// RefundConfirm closes the refund_required exit: the shop has physically
// returned the goods and handed the money back, so stock is restored and the
// order moves to cancelled. Staff group per the confirmed setting that the
// cashier acts as shop administrator; no extra role gate is applied here.
func (a *API) RefundConfirm(w http.ResponseWriter, r *http.Request) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	orderID, valid := pathEntityID(w, r, "orderID")
	if !valid {
		return
	}
	a.orderActionCommand(w, r, account, "order.refund_confirm:"+orderID.String(), struct{}{}, func(identity data.CommandIdentity) (data.OrderCommandResult, error) {
		return a.DB.ConfirmRefundCommand(r.Context(), identity, orderID)
	})
}

func (a *API) orderAction(w http.ResponseWriter, r *http.Request, fn func(data.Account) (data.Order, error)) {
	account, ok := a.staff(w, r)
	if !ok {
		return
	}
	order, err := fn(account)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, orderJSON(order))
}

func readEvent(w http.ResponseWriter, r *http.Request) (data.EventInput, bool) {
	var in data.EventInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return data.EventInput{}, false
	}
	if !validEvent(in) {
		writeErr(w, http.StatusBadRequest, "invalid_event")
		return data.EventInput{}, false
	}
	return in, true
}

func validEvent(in data.EventInput) bool {
	if in.Name == "" || !catalog.ValidCurrency(in.Currency) || !catalog.ValidEventStatus(in.Status) {
		return false
	}
	if len([]rune(in.Name)) > 120 || len([]rune(in.Venue)) > 120 || len([]rune(in.BoothNumber)) > 40 {
		return false
	}
	start, err1 := time.Parse(time.DateOnly, in.StartsOn)
	end, err2 := time.Parse(time.DateOnly, in.EndsOn)
	return err1 == nil && err2 == nil && !end.Before(start)
}

func orderJSON(order data.Order) map[string]any {
	lines := make([]any, 0, len(order.Lines))
	for _, line := range order.Lines {
		baseTotal := int64(math.MaxInt64)
		if line.Qty > 0 && line.UnitPriceMinor <= math.MaxInt64/int64(line.Qty) {
			baseTotal = line.UnitPriceMinor * int64(line.Qty)
		}
		discount := baseTotal - line.LineTotalMinor
		if discount < 0 {
			discount = 0
		}
		lines = append(lines, map[string]any{
			"goodName": line.GoodName, "unitPriceMinor": line.UnitPriceMinor, "qty": line.Qty,
			"lineTotalMinor": line.LineTotalMinor, "discountMinor": discount,
			"priceLabel": priceLabel(order.Currency, line.LineTotalMinor),
		})
	}
	var paidBy any
	if order.PaidByLogin != "" {
		paidBy = order.PaidByLogin
	}
	return map[string]any{
		"id": order.ID, "number": order.Number, "status": order.Status, "boothName": order.BoothName,
		"currency": order.Currency, "priceLabel": priceLabel(order.Currency, order.TotalMinor),
		"totalMinor": order.TotalMinor, "payCodeName": order.PayCodeName,
		"stateVersion": order.StateVersion, "createdAt": order.CreatedAt, "reservationExpiresAt": order.ReservationExpiresAt,
		"refundedAt":        order.RefundedAt,
		"cashReceivedMinor": order.CashReceivedMinor,
		"cashChangeMinor":   order.CashChangeMinor,
		"openedByLogin":     order.OpenedByLogin, "openedByRole": order.OpenedByRole,
		"paidByLogin": paidBy, "paidByRole": order.PaidByRole,
		"eventId": order.EventID, "eventName": order.EventName, "lines": lines, "pricingGroups": order.PricingGroups,
	}
}

func orderQuoteJSON(quote data.OrderQuote) map[string]any {
	return map[string]any{
		"id": quote.ID, "eventId": quote.EventID, "cartVersion": quote.CartVersion,
		"catalogRevision": quote.CatalogRevision, "expiresAt": quote.ExpiresAt,
		"totalMinor": quote.TotalMinor, "baseMinor": quote.Snapshot.BaseMinor,
		"savingsMinor": quote.Snapshot.SavingsMinor, "itemCount": quote.Snapshot.ItemCount,
		"lines": quote.Snapshot.Lines, "groups": quote.Snapshot.Groups,
	}
}

func ordersJSON(orders []data.Order) []any {
	out := make([]any, 0, len(orders))
	for _, order := range orders {
		out = append(out, orderJSON(order))
	}
	return out
}

func payCodesJSON(codes []data.PayCode) []any {
	out := make([]any, 0, len(codes))
	for _, code := range codes {
		imageURL := ""
		if code.Kind == "qr" {
			imageURL = "/api/files/pay-codes/" + code.ID.String()
		}
		out = append(out, map[string]any{
			"id": code.ID, "name": code.Name, "kind": code.Kind, "currency": code.Currency,
			"isDefault": code.IsDefault, "isEnabled": code.IsEnabled,
			"imageURL": imageURL,
		})
	}
	return out
}
