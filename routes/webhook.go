package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/mohit4bug/mo-sh/handlers"
)

func WebhookRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/github/redirect", handlers.HandleGithubRedirect)

	return r
}
