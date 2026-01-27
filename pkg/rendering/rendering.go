package rendering

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type TemplateInput struct {
	StatusCode int
	Status     string
	Subtitle   string
	Color      string
	Subtext    string
	ReturnUrl  string
	Timeout    int64
	Counter    int
	Cookie     string
}

const (
	repoName               string = "gossamer"
	templateName           string = "templates"
	foreverHtml            string = "forever.html"
	statusCodeHtml         string = "status_code.html"
	limboHtml              string = "limbo.html"
	loginHtml              string = "login.html"
	RenderReturnCode       int    = 707
	UnworthyReturnCode     int    = 496
	BadBehaviourReturnCode int    = 394
)

var templateFolder string

func init() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	var rootFolder string
	if !strings.HasSuffix(cwd, repoName) {

		splitCwd := strings.Split(cwd, "/")
		for index, split := range splitCwd {
			if split == repoName {
				splitCwd = splitCwd[:len(splitCwd)-index+2]
				break
			}
		}
		rootFolder = strings.Join(splitCwd, "/")
	}

	templateFolder = filepath.Join(rootFolder, templateName)
}

func ErrUnableToRender(w http.ResponseWriter) {
	// Raise a custom error when rendering is going wrong.
	// We use a custom code to make testing easier, basically.
	w.WriteHeader(RenderReturnCode)
	w.Write([]byte("Rendering error"))

}

func RenderInternalServerError(w http.ResponseWriter, _ *http.Request) {

	w.WriteHeader(http.StatusInternalServerError)

	tmpl, err := template.ParseFiles(
		filepath.Join(templateFolder, statusCodeHtml),
	)
	if err != nil {
		ErrUnableToRender(w)
		return
	}

	if err := tmpl.Execute(w, TemplateInput{
		StatusCode: http.StatusInternalServerError,
		Status:     "Internal Server Error",
		Color:      "primary",
		Subtitle:   "Something went wrong on our end",
		Subtext:    "We're sorry, we're going through a lot right now. It's not you, it's us.",
	}); err != nil {
		ErrUnableToRender(w)
		return
	}

}

func RenderTooManyRequests(w http.ResponseWriter, _ *http.Request) {

	w.WriteHeader(http.StatusTooManyRequests)

	tmpl, err := template.ParseFiles(
		filepath.Join(templateFolder, statusCodeHtml),
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
		return
	}

	if err := tmpl.Execute(w, TemplateInput{
		StatusCode: http.StatusTooManyRequests,
		Status:     "Too Many Requests",
		Color:      "warning",
		Subtitle:   "Slow down there.",
		Subtext:    "Touch grass or something, idk.",
	}); err != nil {
		ErrUnableToRender(w)
		return
	}

}

func RenderBadGateway(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadGateway)

	tmpl, err := template.ParseFiles(
		filepath.Join(templateFolder, statusCodeHtml),
	)
	if err != nil {
		ErrUnableToRender(w)
		return
	}

	if err := tmpl.Execute(w, TemplateInput{
		StatusCode: http.StatusBadGateway,
		Status:     "Bad Gateway",
		Color:      "warning",
		Subtitle:   "Upstream unavailable",
		Subtext:    fmt.Sprintf("The upstream <code>%s</code> seems to be unavailable", r.URL.Host),
	}); err != nil {
		ErrUnableToRender(w)
		return
	}

}

func RenderForbidden(w http.ResponseWriter, _ *http.Request) {

	w.WriteHeader(http.StatusForbidden)

	tmpl, err := template.ParseFiles(
		filepath.Join(templateFolder, statusCodeHtml),
	)
	if err != nil {
		ErrUnableToRender(w)
		return
	}

	if err := tmpl.Execute(w, TemplateInput{
		StatusCode: http.StatusForbidden,
		Status:     "Forbidden",
		Color:      "warning",
		Subtitle:   "You don't have the right",
		Subtext:    "Put these foolish ambitions to rest",
	}); err != nil {
		ErrUnableToRender(w)
		return
	}
}

func RenderLogin(w http.ResponseWriter, t TemplateInput) {
	// Render the `login` page of gossamer

	// Set Accept-CH headers for browser identification
	w.Header().Set(
		"Accept-CH",
		"Sec-CH-UA, Sec-CH-UA-Bitness, Sec-CH-UA-Full-Version, Sec-CH-UA-Mobile, Sec-CH-UA-Model, Sec-CH-UA-Platform, Sec-CH-DPR",
	)
	templatePath := filepath.Join(templateFolder, loginHtml)

	tmpl, err := template.ParseFiles(
		templatePath,
	)
	if err != nil {
		ErrUnableToRender(w)
		return
	}

	if err := tmpl.Execute(w, t); err != nil {
		ErrUnableToRender(w)
		return
	}

}

func RenderLimbo(w http.ResponseWriter, t TemplateInput) {

	w.WriteHeader(http.StatusPaymentRequired)
	tmpl, err := template.ParseFiles(
		filepath.Join(templateFolder, limboHtml),
	)
	if err != nil {
		ErrUnableToRender(w)
		return
	}

	if err := tmpl.Execute(w, t); err != nil {
		ErrUnableToRender(w)
		return
	}
}

func RenderUnworthy(w http.ResponseWriter, _ *http.Request) {

	w.WriteHeader(UnworthyReturnCode)
	tmpl, err := template.ParseFiles(
		filepath.Join(templateFolder, statusCodeHtml),
	)
	if err != nil {
		ErrUnableToRender(w)
		return
	}

	if err := tmpl.Execute(w, TemplateInput{
		StatusCode: UnworthyReturnCode,
		Status:     "Unworthy",
		Color:      "danger",
		Subtitle:   "Tsk tsk tsk.",
		Subtext:    "You are not worthy of accessing this resource.",
	}); err != nil {
		ErrUnableToRender(w)
		return
	}
}

func RenderForever(w http.ResponseWriter) {

	w.WriteHeader(BadBehaviourReturnCode)
	tmpl, err := template.ParseFiles(
		filepath.Join(templateFolder, foreverHtml),
	)
	if err != nil {
		ErrUnableToRender(w)
		return
	}

	if err := tmpl.Execute(w, TemplateInput{}); err != nil {
		ErrUnableToRender(w)
		return
	}

}
