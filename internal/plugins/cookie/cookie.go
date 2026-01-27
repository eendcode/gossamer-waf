package cookie

import (
	"gossamer/pkg/rendering"
	"net/http"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/google/uuid"
)

type CookieEnforcer struct {
	ValidTime time.Duration `env:"COOKIE_VALID_TIME" envDefault:"24h"`
	Name      string        `env:"COOKIE_NAME" envDefault:"_gSession"`
	Domain    string        `env:"COOKIE_DOMAIN"`
}

func New() (*CookieEnforcer, error) {
	var ce CookieEnforcer
	if err := env.Parse(&ce); err != nil {
		return nil, err
	}

	return &ce, nil
}

func (c *CookieEnforcer) NewCookie() string {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	idString := id.String()

	return strings.ReplaceAll(idString, "-", "")
}

func (c *CookieEnforcer) Intercept(w http.ResponseWriter, r *http.Request) {
	// This function instructs the browser to set a cookie, if it hasn't been done already.

	rendering.RenderLogin(w, rendering.TemplateInput{
		ReturnUrl: r.URL.String(),
		Cookie:    c.NewCookie(),
	})

}
