package seeder

import (
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
		// We use 'Code' as the unique identifier to check existence
		if err := db.Where(model.Role{Code: role.Code}).FirstOrCreate(&role).Error; err != nil {
			log.Printf("Error seeding role %s: %v", role.Code, err)
		}
	}

	log.Println("Role seeding completed.")
}
