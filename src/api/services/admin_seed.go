package services

import (
	"context"
	"errors"
	"log"
	"os"

	models "backend-yonathan/src/models"
	"backend-yonathan/src/repository"

	"golang.org/x/crypto/bcrypt"
)

// SeedAdminUser creates the admin user on startup if ADMIN_EMAIL is set and
// the user does not already exist. This ensures a master user is always
// available after deployment without exposing a public registration endpoint.
func SeedAdminUser(userRepo repository.UserRepository) {
	email := os.Getenv("ADMIN_EMAIL")
	password := os.Getenv("ADMIN_PASSWORD")
	username := os.Getenv("ADMIN_USERNAME")

	if email == "" || password == "" {
		return
	}
	if username == "" {
		username = "admin"
	}

	ctx := context.Background()
	if _, err := userRepo.GetUserByEmail(ctx, email); err == nil {
		return
	} else if !errors.Is(err, repository.ErrNotFound) {
		log.Printf("[admin-seed] error checking admin user: %v", err)
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[admin-seed] error hashing password: %v", err)
		return
	}

	admin := models.User{
		Email:    email,
		Password: string(hashed),
		UserName: username,
	}

	if err := userRepo.SaveUser(ctx, admin); err != nil {
		log.Printf("[admin-seed] error saving admin user: %v", err)
		return
	}

	log.Printf("[admin-seed] admin user created: %s", email)
}
