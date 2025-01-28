package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mohit4bug/mo-sh/routes"
)

func main() {
	r := chi.NewRouter()

	r.Mount("/sources", routes.SourceRoutes())
	r.Mount("/webhooks", routes.WebhookRoutes())

	http.ListenAndServe(":3000", r)
}
