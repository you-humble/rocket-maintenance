package health

import (
	"net/http"

	"github.com/you-humble/rocket-maintenance/platform/logger"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write([]byte("SERVING")); err != nil {
		logger.Error(r.Context(), "health check", logger.ErrorF(err))
	}
}
