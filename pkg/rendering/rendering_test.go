package rendering_test

import (
	"gossamer/pkg/rendering"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func Test500(t *testing.T) {

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	rendering.RenderInternalServerError(w, r)

	if w.Code != 500 {
		t.Errorf("expected return code %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func Test429(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	rendering.RenderTooManyRequests(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected return code %d, got %d", http.StatusTooManyRequests, w.Code)
	}
}

func Test502(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	rendering.RenderBadGateway(w, r)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected return code %d, got %d", http.StatusBadGateway, w.Code)
	}
}

func Test403(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	rendering.RenderForbidden(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected return code %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestCustomResponse(t *testing.T) {
	w := httptest.NewRecorder()
	// r := httptest.NewRequest("GET", "/", nil)

	rendering.ErrUnableToRender(w)

	if w.Code != rendering.RenderReturnCode {
		t.Errorf("expected return code %d, got %d", rendering.RenderReturnCode, w.Code)
	}
}

func TestLogin(t *testing.T) {
	w := httptest.NewRecorder()

	newUuid, err := uuid.NewV7()
	if err != nil {
		t.Errorf("error on build uuid: %v", err)
	}

	rendering.RenderLogin(w, rendering.TemplateInput{
		Cookie:    newUuid.String(),
		ReturnUrl: "/",
	})

	if w.Code != 200 {
		t.Errorf("error on rendering login template: %v", err)
	}
}
