package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/mohit4bug/mo-sh/handlers"
)

func SSHKeyRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", handlers.FindAllSSHKeys)
	r.Get("/{sshKeyID}", handlers.FindSSHKeyByID)
	r.Post("/generate-key-pair", handlers.GenerateKeyPair)

	return r
}
