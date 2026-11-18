package server

import (
	"strings"
	"testing"

	"github.com/De3trvctions/Stallink/internal/service"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

func TestHTTPServerLimitsAndDeleteRoute(t *testing.T) {
	srv := NewHTTP("127.0.0.1:0", "stallink", &service.API{})
	if srv.ReadHeaderTimeout != readHeaderTimeout || srv.ReadTimeout != readTimeout || srv.IdleTimeout != idleTimeout {
		t.Fatalf("timeouts header=%s read=%s idle=%s", srv.ReadHeaderTimeout, srv.ReadTimeout, srv.IdleTimeout)
	}
	found := map[string]bool{}
	if err := srv.WalkRoute(func(info khttp.RouteInfo) error {
		if strings.Contains(info.Path, "/images/") {
			found[info.Method+" "+info.Path] = true
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	const imagePath = "/api/staff/goods/{goodID}/images/{imageID}"
	if !found["GET "+imagePath] || !found["DELETE "+imagePath] {
		t.Fatalf("image routes = %v", found)
	}
}
