package main

import (
	"fmt"
	"gossamer/internal/logging"
	"gossamer/internal/plugins"
	"log/slog"
	"net/http"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
)

var settings struct {
	ListenAddress                string   `env:"LISTEN_ADDR" envDefault:"0.0.0.0"`
	ListenPort                   int16    `env:"LISTEN_PORT" envDefault:"8080"`
	TlsCertificatePath           string   `env:"TLS_CERTIFICATE_PATH" envDefault:"tls.crt"`
	TlsKeyPath                   string   `env:"TLS_KEY_PATH" envDefault:"tls.key"`
	IgnorePlugins                []string `env:"IGNORE_PLUGINS"`
	HookDebugMode                bool     `env:"HOOK_DEBUG_MODE" envDefault:"false"`
	ServerHeader                 string   `env:"SERVER_HEADER" envDefault:"gossamer/0.1.0"`
	OverrideUpstreamServerHeader bool     `env:"OVERRIDE_UPSTREAM_SERVER_HEADER" envDefault:"true"`
}

var (
	logger     *slog.Logger
	validate   *validator.Validate = validator.New(validator.WithRequiredStructEnabled())
	pluginMap  map[string]plugins.Plugin
	modifyFunc func(*http.Response) error
)

func healthz(w http.ResponseWriter, _ *http.Request) {
	// Health endpoint
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func main() {

	// parse environment variables
	if err := env.Parse(&settings); err != nil {
		panic(err)
	}

	logger = logging.NewLogger()

	// Set up our plugins
	var err error
	pluginMap, err = plugins.InitializePlugins()
	if err != nil {
		logger.Error("error initializing plugins", "error", err)
		return
	}

	// Set up modifiers
	modifyFunc, err = plugins.ModifyFunc()
	if err != nil {
		logger.Error("error initializing modifiers", "error", err)
		return
	}

	// Set up the valkey connection

	mux := http.NewServeMux()
	mux.Handle("/favicon.ico", http.NotFoundHandler())
	mux.Handle("/gstatic/", http.StripPrefix("/gstatic/", http.FileServer(http.Dir("static/"))))
	mux.HandleFunc("/gshealth", healthz)
	mux.HandleFunc("/_gsHook", checkIn)
	mux.HandleFunc("/", handleProxy)

	logger.Info(
		"running gossamer",
		"listen_port", settings.ListenPort,
	)

	listenString := fmt.Sprintf("%s:%d", settings.ListenAddress, settings.ListenPort)

	if settings.ListenPort == 443 || settings.ListenPort == 8443 {
		if err := http.ListenAndServeTLS(listenString, settings.TlsCertificatePath, settings.TlsKeyPath, mux); err != nil {
			logger.Error("error occured on setting up listener", "error", err)
		}
	} else {
		if err := http.ListenAndServe(listenString, mux); err != nil {
			logger.Error("error occured on setting up listener", "error", err)
		}
	}
}
