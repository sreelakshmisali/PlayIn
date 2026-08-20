package health

import (
	"net/http"

	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

// Handler exposes the health service over HTTP. It translates between the
// transport and the service and holds no logic of its own.
type Handler struct {
	service *Service
}

// NewHandler wires a Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes registers the package's endpoints on mux under the given prefix.
// Each feature package registers its own routes so the central router stays a
// list of mounts rather than a list of paths.
//
// Every path is registered twice: once with its method, once without. The
// method-less pattern answers unsupported methods with a JSON 405. Without it
// the router's catch-all "/" route wins and those requests come back as 404.
func (h *Handler) Routes(mux *http.ServeMux, prefix string) {
	mux.HandleFunc("GET "+prefix+"/health", h.Get)
	mux.HandleFunc(prefix+"/health", httpx.MethodNotAllowed)
}

// Get handles GET /api/v1/health.
// A degraded report returns 503 so load balancers and orchestrators can act on
// the status code without parsing the body.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	report := h.service.Check(r.Context())

	status := http.StatusOK
	if !report.Healthy() {
		status = http.StatusServiceUnavailable
	}

	httpx.JSON(w, r, status, report)
}
