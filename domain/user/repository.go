package user

import (
	"clean-architecture/domain/models"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/infrastructure"
)

// UserRepository database structure
type Repository struct {
	infrastructure.Database
	logger framework.Logger
}

// NewUserRepository creates a new user repository
func NewRepository(db infrastructure.Database, logger framework.Logger) Repository {
	return Repository{db, logger}
}

// GetRawUserFromID loads a user by primary key.
func (r *Repository) GetRawUserFromID(userID uint) (user *models.User, err error) {
	r.logger.Info("[UserRepository...GetRawUserFromID]")
	query := r.Model(&models.User{}).Where("id = ?", userID).First(&user)
	return user, query.Error
}

// ExistsByEmail checks if the user exists by email
func (r *Repository) ExistsByEmail(email string) (bool, error) {
	r.logger.Info("[UserRepository...Exists]")

	users := make([]models.User, 0, 1)
	query := r.DB.Where("email = ?", email).Limit(1).Find(&users)

	return query.RowsAffected > 0, query.Error
}
