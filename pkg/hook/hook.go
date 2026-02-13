package hook

import (
	"encoding/json"
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

func (h HookCommand) ToHtml() ([]byte, error) {
	return json.Marshal(h)
}

// func generateCookie() (string, error) {
// 	id, err := uuid.NewV7()
// 	idString := id.String()

// 	return strings.ReplaceAll(idString, "-", ""), err
// }

func NewCookie(cookie string, redirectUrl string) HookCommand {

	// TODO: add domain
	rawScript := `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Gossamer</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">

  <link rel="stylesheet" href="/gstatic/css/bulma.min.css">

</head>
<body>
  <noscript>
    <p>You need JavaScript enabled in order to proceed.</p>
  </noscript>

  <section class="hero is-info">
    <div class="hero-body">
      <p class="title">Gateway</p>
      <p class="subtitle">We need to get to know you.</p>
    </div>
  </section>

  <div class="fixed-grid has-2-cols">
    <div class="grid">
      <div class="cell">
        <div class="content has-text-centered">
          <p style="font-size:256px" id="countdown" class="px-6 pt-6">3</p>
          <p>seconds remaining</p>
        </div>
      </div>
      <div class="cell">
        <div class="content has-text-centered">
          <h1><p class="px-6 py-6">What's next?</p></h1>
          <p>
            Once you're verified, we'll redirect you to <code>%s</code>.
          </p>
        </div>
      </div>
    </div>
  </div>

<script>
    let count = 3;
    const countdownEl = document.getElementById('countdown');

    const interval = setInterval(() => {
      count--;
      countdownEl.textContent = count;

      if (count === 0) {
        clearInterval(interval);


		document.cookie = "%s=%s; max-age=%d";
		document.location = "%s";
}}, 1000);
</script>
</body>
</html>
`

	script := fmt.Sprintf(rawScript, redirectUrl, CookieName, cookie, cookieValidTime, redirectUrl)

	return HookCommand{
		Script:  script,
		Message: "ok",
	}
}

func Blur() HookCommand {
	rawScript := `
<script>
	() => {
const run = () => {
    document.body.style.transition = "filter 0.3s ease";
    document.body.style.filter = "blur(5px)";

    setTimeout(() => {
      document.body.style.filter = "";
    }, 1200);
  };


  run();

  setInterval(run, 60000);
};
</script>
`

	return HookCommand{
		Script:  rawScript,
		Message: "ok",
	}

}
