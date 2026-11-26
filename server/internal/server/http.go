package server

import (
	"net/http"
	"time"

	"github.com/De3trvctions/Stallink/internal/logging"
	"github.com/De3trvctions/Stallink/internal/service"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

const (
	handlerTimeout    = 30 * time.Second
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 60 * time.Second
	writeTimeout      = 60 * time.Second
	idleTimeout       = 120 * time.Second
)

func NewHTTP(addr, serviceName string, api *service.API) *khttp.Server {
	srv := khttp.NewServer(
		khttp.Address(addr),
		khttp.Timeout(handlerTimeout),
		khttp.Filter(logging.RequestFilter),
	)
	srv.ReadHeaderTimeout = readHeaderTimeout
	srv.ReadTimeout = readTimeout
	srv.WriteTimeout = writeTimeout
	srv.IdleTimeout = idleTimeout
	health := service.NewHealth(serviceName)
	srv.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		health.ServeHTTP(w, r)
	})
	r := srv.Route("/")
	route(r, http.MethodGet, "/api/public/landing", api.Landing)
	route(r, http.MethodGet, "/api/public/catalog", api.PublicCatalog)
	route(r, http.MethodGet, "/api/public/shops/{slug}", api.Shop, "slug")
	route(r, http.MethodGet, "/api/public/shops/{slug}/events/{eventID}", api.Event, "slug", "eventID")
	route(r, http.MethodGet, "/api/public/shops/{slug}/bundles/{bundleListingID}", api.PublicBundle, "slug", "bundleListingID")
	route(r, http.MethodGet, "/api/public/shops/{slug}/listings/{listingID}", api.Listing, "slug", "listingID")
	route(r, http.MethodGet, "/api/files/goods/{imageID}", api.GoodImage, "imageID")
	route(r, http.MethodGet, "/api/files/bundles/{imageID}", api.BundleImage, "imageID")
	route(r, http.MethodGet, "/api/staff/goods/{goodID}/images/{imageID}", api.StaffGoodImage, "goodID", "imageID")
	route(r, http.MethodPost, "/api/staff/goods/{goodID}/images", api.AddGoodImages, "goodID")
	route(r, http.MethodDelete, "/api/staff/goods/{goodID}/images/{imageID}", api.DeleteGoodImage, "goodID", "imageID")
	route(r, http.MethodPatch, "/api/staff/goods/{goodID}/images", api.ReorderGoodImages, "goodID")
	route(r, http.MethodPost, "/api/staff/bundles", api.CreateBundle)
	route(r, http.MethodPatch, "/api/staff/bundles/{bundleID}", api.UpdateBundle, "bundleID")
	route(r, http.MethodGet, "/api/staff/events/{eventID}/bundles", api.StaffBundles, "eventID")
	route(r, http.MethodGet, "/api/staff/bundles/{bundleID}/images/{imageID}", api.StaffBundleImage, "bundleID", "imageID")
	route(r, http.MethodGet, "/api/files/pay-codes/{id}", api.PayCodeImage, "id")

	route(r, http.MethodPost, "/api/auth/register", api.Register)
	route(r, http.MethodPost, "/api/auth/login", api.Login)
	route(r, http.MethodPost, "/api/auth/logout", api.Logout)
	route(r, http.MethodGet, "/api/auth/me", api.Me)
	route(r, http.MethodPatch, "/api/me", api.UpdateMe)

	route(r, http.MethodGet, "/api/staff/events", api.StaffEvents)
	route(r, http.MethodPost, "/api/staff/events", api.CreateEvent)
	route(r, http.MethodPatch, "/api/staff/events/{eventID}", api.UpdateEvent, "eventID")
	route(r, http.MethodGet, "/api/staff/events/{eventID}/listings", api.StaffListings, "eventID")
	route(r, http.MethodGet, "/api/staff/goods", api.StaffGoods)
	route(r, http.MethodPost, "/api/staff/goods", api.CreateGood)
	route(r, http.MethodPost, "/api/staff/goods/{goodID}/listings", api.AddListing, "goodID")
	route(r, http.MethodPatch, "/api/staff/listings/{listingID}", api.UpdateListing, "listingID")
	route(r, http.MethodPatch, "/api/staff/goods/{goodID}", api.UpdateGood, "goodID")
	route(r, http.MethodPost, "/api/staff/orders", api.CreateOrder)
	route(r, http.MethodPost, "/api/staff/order-quotes", api.CreateOrderQuote)
	route(r, http.MethodGet, "/api/staff/orders", api.StaffOrders)
	route(r, http.MethodGet, "/api/staff/orders/{orderID}", api.GetStaffOrder, "orderID")
	route(r, http.MethodGet, "/api/staff/checkouts/{clientOrderID}", api.GetStaffCheckout, "clientOrderID")
	route(r, http.MethodGet, "/api/staff/pay-codes", api.PayCodes)
	route(r, http.MethodPost, "/api/staff/orders/{orderID}/pay", api.Pay, "orderID")
	route(r, http.MethodPost, "/api/staff/orders/{orderID}/reserve", api.ReserveOrder, "orderID")
	route(r, http.MethodPost, "/api/staff/orders/{orderID}/unpaid", api.Unpaid, "orderID")
	route(r, http.MethodPost, "/api/staff/orders/{orderID}/fulfill", api.Fulfill, "orderID")
	route(r, http.MethodPost, "/api/staff/orders/{orderID}/cancel", api.Cancel, "orderID")
	route(r, http.MethodPost, "/api/staff/orders/{orderID}/refund-confirm", api.RefundConfirm, "orderID")

	route(r, http.MethodPatch, "/api/admin/shop", api.UpdateShopSettings)
	route(r, http.MethodPatch, "/api/admin/shop/featured-event", api.SetFeaturedEvent)
	route(r, http.MethodGet, "/api/admin/pay-codes", api.AdminPayCodes)
	route(r, http.MethodPost, "/api/admin/pay-codes", api.CreatePayCode)
	route(r, http.MethodPost, "/api/admin/pay-codes/cash", api.CreateCashPayCode)
	route(r, http.MethodPatch, "/api/admin/pay-codes/{id}", api.UpdatePayCode, "id")
	route(r, http.MethodGet, "/api/admin/orders", api.AdminOrders)
	route(r, http.MethodGet, "/api/admin/audit", api.AdminAudit)
	route(r, http.MethodGet, "/api/admin/daily-reports", api.GetDailyReport)
	route(r, http.MethodPost, "/api/admin/daily-reports", api.GenerateDailyReport)
	route(r, http.MethodGet, "/api/admin/daily-reports/export", api.ExportDailyReport)
	route(r, http.MethodGet, "/api/admin/sales", api.ListSales)
	route(r, http.MethodGet, "/api/admin/daily-reports/today", api.DailyReportToday)
	route(r, http.MethodPost, "/api/admin/sales", api.CreateSales)
	route(r, http.MethodPost, "/api/admin/sales/{userID}/disable", api.DisableSales, "userID")
	route(r, http.MethodPost, "/api/admin/sales/{userID}/enable", api.EnableSales, "userID")
	route(r, http.MethodPost, "/api/admin/sales/{userID}/password", api.ResetSalesPassword, "userID")

	route(r, http.MethodPost, "/api/platform/shops", api.OpenShop)
	route(r, http.MethodGet, "/api/platform/shops", api.PlatformShops)
	route(r, http.MethodPatch, "/api/platform/shops/{shopID}", api.SetPlatformShop, "shopID")
	route(r, http.MethodGet, "/api/platform/goods", api.PlatformGoods)
	route(r, http.MethodGet, "/api/platform/goods/{goodID}/images/{imageID}", api.PlatformGoodImage, "goodID", "imageID")
	route(r, http.MethodPatch, "/api/platform/goods/{goodID}/review", api.ReviewGood, "goodID")
	route(r, http.MethodGet, "/api/platform/bundles/{bundleID}/images/{imageID}", api.PlatformBundleImage, "bundleID", "imageID")
	route(r, http.MethodPatch, "/api/platform/bundles/{bundleID}/review", api.ReviewBundle, "bundleID")
	route(r, http.MethodGet, "/api/platform/review/auto-approve", api.AutoApproveSetting)
	route(r, http.MethodPatch, "/api/platform/review/auto-approve", api.SetAutoApproveSetting)
	route(r, http.MethodGet, "/api/platform/appearance", api.PlatformAppearance)
	route(r, http.MethodPut, "/api/platform/appearance", api.SetPlatformAppearance)
	return srv
}

func route(r *khttp.Router, method, path string, h http.HandlerFunc, keys ...string) {
	handler := func(ctx khttp.Context) error {
		req := ctx.Request().WithContext(ctx)
		vars := ctx.Vars()
		for _, key := range keys {
			req.SetPathValue(key, vars.Get(key))
		}
		h(ctx.Response(), req)
		return nil
	}
	switch method {
	case http.MethodGet:
		r.GET(path, handler)
	case http.MethodPost:
		r.POST(path, handler)
	case http.MethodPut:
		r.PUT(path, handler)
	case http.MethodPatch:
		r.PATCH(path, handler)
	case http.MethodDelete:
		r.DELETE(path, handler)
	default:
		r.GET(path, handler)
	}
}
