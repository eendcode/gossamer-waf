package plugins

import (
	"fmt"
	"gossamer/internal/gossamer"
	"gossamer/internal/plugins/browser"
	"gossamer/internal/plugins/coraza"
	"gossamer/internal/plugins/ratelimit"
	"gossamer/internal/plugins/upstream"
	"net/http"
	"sync"

	"github.com/caarlos0/env/v11"
)

var settings struct {
	IgnorePlugins   []string `env:"IGNORE_PLUGINS"`
	IgnoreModifiers []string `env:"IGNORE_MODIFIERS"`
}

var Plugins map[string]func() (Plugin, error) = map[string]func() (Plugin, error){
	"browser":   func() (Plugin, error) { return browser.New() },
	"coraza":    func() (Plugin, error) { return coraza.New() },
	"upstream":  func() (Plugin, error) { return upstream.New() },
	"ratelimit": func() (Plugin, error) { return ratelimit.New() },
}

var Modifiers map[string]func() (ResponseModifier, error) = map[string]func() (ResponseModifier, error){
	// "injector": func() (Modifier, error) { }
}

type Plugin interface {
	// An interface for gossamer plugins

	// A validator decides if a request should be blocked or not.
	// Note that all prejudicers are run in goroutines, so they don't modify the original request
	Validate(gossamer.Connection) bool

	// A verifier does the same as a prejudicer, but it does so when the response is in
	Verify(gossamer.Connection) bool

	// A preprocessor (and the same goes for postprocessors) modifies the request.
	// Note that it is blocking, so they are quite expensive.
	Preprocess(gossamer.Connection) error
	Postprocess(gossamer.Connection) error
}

type ResponseModifier interface {
	Modify(*http.Response) error
}

func InitializeModifiers() ([]ResponseModifier, error) {
	var modifierList []ResponseModifier
	if err := env.Parse(&settings); err != nil {
		return modifierList, err
	}

	for name, initFunc := range Modifiers {
	IGNORE_LOOP:
		for _, ignoreModifier := range settings.IgnoreModifiers {
			if name == ignoreModifier {
				break IGNORE_LOOP
			}
		}

		mdf, err := initFunc()
		if err != nil {
			return modifierList, err
		}

		modifierList = append(modifierList, mdf)
	}

	return modifierList, nil
}

func ModifyFunc() (func(*http.Response) error, error) {
	modifiers, err := InitializeModifiers()
	if err != nil {
		return nil, err
	}

	return func(r *http.Response) error {
		for _, m := range modifiers {
			if err := m.Modify(r); err != nil {
				return err
			}
		}

		return nil
	}, nil
}

func InitializePlugins() (map[string]Plugin, error) {

	pluginMap := make(map[string]Plugin)
	if err := env.Parse(&settings); err != nil {
		return pluginMap, err
	}

	for name, initFunc := range Plugins {
	IGNORE_LOOP:
		// check if our plugin should be ignored
		for _, ignorePlugin := range settings.IgnorePlugins {
			if name == ignorePlugin {
				break IGNORE_LOOP

			}
		}

		plgn, err := initFunc()
		if err != nil {
			return pluginMap, err
		}
		pluginMap[name] = plgn
	}

	return pluginMap, nil
}

func RunValidation(pm map[string]Plugin, c gossamer.Connection) bool {
	validationChannel := make(chan bool)
	var validationWg sync.WaitGroup

	// We run all plugins asynchronously
	for name, plugin := range pm {

		validationWg.Add(1)
		go func(p Plugin) {

			defer validationWg.Done()
			if !p.Validate(c) {
				fmt.Println(name)
				validationChannel <- false
			}

		}(plugin)
	}

	go func() {
		validationWg.Wait()
		close(validationChannel)
	}()

	// Get the results
	for result := range validationChannel {
		if !result {
			// If any plugin returns false, notify and exit
			return false
		}
	}

	return true
}

func RunPreprocessor(pm map[string]Plugin, c gossamer.Connection) error {
	for _, plugin := range pm {
		if err := plugin.Preprocess(c); err != nil {
			return err
		}
	}

	return nil
}

func RunPostprocessor(pm map[string]Plugin, c gossamer.Connection) error {
	for _, plugin := range pm {
		if err := plugin.Postprocess(c); err != nil {
			return err
		}
	}

	return nil
}

func RunVerification(pm map[string]Plugin, c gossamer.Connection) bool {
	verificationChannel := make(chan bool)
	var verificationWg sync.WaitGroup

	// We run all plugins asynchronously
	for _, plugin := range pm {

		verificationWg.Add(1)
		go func(p Plugin) {

			defer verificationWg.Done()
			if !p.Verify(c) {
				verificationChannel <- false
			}

		}(plugin)
	}

	go func() {
		verificationWg.Wait()
		close(verificationChannel)
	}()

	// Get the results
	for result := range verificationChannel {
		if !result {
			// If any plugin returns false, notify and exit
			return false
		}
	}

	return true
}
