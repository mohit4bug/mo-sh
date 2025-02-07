package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/mohit4bug/mo-sh/handlers"
)

func ServerRoutes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", handlers.CreateServer)
	r.Get("/", handlers.FindAllServers)
	r.Get("/{serverID}/validate", handlers.ValidateServer) // TODO: make this /{serverID}/validate-and-configure

	return r
}
