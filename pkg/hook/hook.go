package hook

import (
	"fmt"
)

const (
	CookieName      string = "_gSession"
	cookieValidTime int    = 86400
)

// clients do api requests to an endpoint `/gshook`

type HookCommand struct {
	Script  string `json:"script"`
	Message string `json:"message"`
}

func (h HookCommand) ToHtml() []byte {
	return []byte(fmt.Sprintf("<script>%s</script>", h.Script))
}

// func generateCookie() (string, error) {
// 	id, err := uuid.NewV7()
// 	idString := id.String()

// 	return strings.ReplaceAll(idString, "-", ""), err
// }

func NewCookie(cookie string, redirectUrl string) HookCommand {

	// TODO: add domain
	rawScript := `
document.cookie = "%s=%s; max-age=%d";
document.location = "%s";
`

	script := fmt.Sprintf(rawScript, CookieName, cookie, cookieValidTime, redirectUrl)

	return HookCommand{
		Script:  script,
		Message: "ok",
	}
}

func Blur() HookCommand {
	rawScript := `() => {
const run = () => {
    document.body.style.transition = "filter 0.3s ease";
    document.body.style.filter = "blur(5px)";

    setTimeout(() => {
      document.body.style.filter = "";
    }, 1200);
  };


  run();

  setInterval(run, 60000);
};`

	return HookCommand{
		Script:  rawScript,
		Message: "ok",
	}

}
