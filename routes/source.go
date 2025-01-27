package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/mohit4bug/mo-sh/handlers"
)

func SourceRoutes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", handlers.CreateSource)
	r.Get("/", handlers.FindAllSources)

	return r
}
