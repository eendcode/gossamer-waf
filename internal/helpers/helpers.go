package helpers

import (
	"net/http"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/google/uuid"
)

type Settings struct {
	CookieName string `env:"COOKIE_NAME" envDefault:"_gSession"`
}

var settings Settings

func init() {
	if err := env.Parse(&settings); err != nil {
		panic(err)
	}
}

func ParseRemoteAddr(s string) (string, string) {
	lastIndex := strings.LastIndex(s, ":")
	remoteIpWithBrackets := s[:lastIndex]
	return strings.Trim(remoteIpWithBrackets, "[]"), s[lastIndex+1:]
}

func GetCookie(r *http.Request) (string, error) {

	cookie, err := r.Cookie(settings.CookieName)
	if err == http.ErrNoCookie {
		return "", err
	}
	return strings.Split(cookie.Value, ",")[0], err
}

// func randRange(min, max int) int {
// 	return rand.IntN(max-min) + min
// }

// func determineDelay(strikes int64) time.Duration {
// 	return time.Duration(math.Ceil(float64(strikes)/10.0) * 100 * float64(time.Millisecond))
// }

func GenerateCookie() (string, error) {
	id, err := uuid.NewV7()
	idString := id.String()

	return strings.ReplaceAll(idString, "-", ""), err
}
