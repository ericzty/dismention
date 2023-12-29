package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/httprate"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("WEBHOOK") == "" {
		log.Fatal().Msg("WEBHOOK is not set")
	}
	log.Debug().Msg("Starting up")
}

func main() {
	r := chi.NewRouter()
	r.Use(httprate.LimitByIP(10, 1*time.Minute))
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	r.Post("/accept", webmentionHandler)
	http.ListenAndServe(":8080", r)
}

func webmentionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		log.Info().Msg("Invalid content type")
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Info().Err(err).Msg("error parsing form")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Verify target (my blog post)
	tgtUrl, tgtErr := url.Parse(r.Form.Get("target"))
	if tgtErr != nil {
		log.Info().Err(tgtErr).Msg("Invalid target")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if tgtUrl.Host != "ezrizhu.com" {
		log.Info().Str("host", tgtUrl.Host).Msg("invalid host")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := siteValidator(tgtUrl); err != nil {
		log.Info().Err(err).Msg("invalid target")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Verify source
	srcUrl, srcErr := url.Parse(r.Form.Get("source"))
	if srcErr != nil {
		log.Info().Err(srcErr).Msg("invalid source")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := siteValidator(srcUrl); err != nil {
		log.Info().Err(err).Msg("invalid source")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := webhook(srcUrl.String(), tgtUrl.String()); err != nil {
		log.Error().Err(err).Msg("error sending webhook")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func siteValidator(target *url.URL) error {
	if target.Scheme != "http" && target.Scheme != "https" {
		return fmt.Errorf("invalid scheme")
	}

	resp, err := http.Get(target.String())
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return err
	}

	return nil
}

func webhook(fromUrl string, toUrl string) error {
	content := "from: " + fromUrl + " to: " + toUrl
	body, err := json.Marshal(struct {
		Content string `json:"content"`
	}{
		Content: content,
	})
	if err != nil {
		return err
	}

	resp, err := http.Post(os.Getenv("WEBHOOK"), "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	if resp.StatusCode != 204 {
		return fmt.Errorf("invalid status code, received" + strconv.Itoa(resp.StatusCode))
	}

	return nil
}
