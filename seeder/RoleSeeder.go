package seeder

import (
	"errors"
	"log"
	"mein-idaas/model"

	"gorm.io/gorm"
)

func SeedRoles(db *gorm.DB) {
	// Define the default roles you want to exist
	roles := []model.Role{
		{
			Name:        "Administrator",
			Code:        "admin",
			Description: "Full system access",
			IsSystem:    true, // Cannot be deleted
		},
		{
			Name:        "Moderator",
			Code:        "moderator",
			Description: "Can manage content but not system settings",
			IsSystem:    false,
		},
		{
			Name:        "User",
			Code:        "user",
			Description: "Standard registered user",
			IsSystem:    true, // Usually standard user role should be protected
		},
	}

	log.Println("Seeding roles...")

	for _, role := range roles {
		// 1. Check if it exists by Code
		var existing model.Role
		err := db.Where("code = ?", role.Code).First(&existing).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 2. Not found -> Create it
			if err := db.Create(&role).Error; err != nil {
				log.Printf("Error creating role %s: %v", role.Code, err)
			} else {
				log.Printf("Created new role: %s", role.Code)
			}
		} else if err != nil {
			// 3. Database error (connection issue, etc.)
			log.Printf("Error checking role %s: %v", role.Code, err)
		} else {
			// 4. Found -> Do nothing
			// log.Printf("Role %s already exists. Skipping.", role.Code)
		}
	}

	log.Println("Role seeding completed.")
}
