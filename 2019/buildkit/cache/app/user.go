package app

import (
	"fmt"
	"log"
	"net/http"

	"bitbucket.org/okteto/okteto/backend/logger"

	"bitbucket.org/okteto/okteto/backend/config"
	"bitbucket.org/okteto/okteto/backend/model"
	"github.com/pkg/errors"
)

//GetUserByToken gets a user from the DB based on the API token
func (s *Server) GetUserByToken(token string) (*model.User, error) {

	u, err := s.getUser(model.User{Token: token})
	if err != nil {
		log.Printf("failed to retrieve the user by security token: %s", err.Error())
	}

	return u, err
}

// GetOrCreateUser tries to get a user from the DB, if missing, it creates it
func (s *Server) GetOrCreateUser(email string, verified bool, mixpanel string) (*model.User, *model.AppError) {

	user, err := s.GetUser(email)
	if err == nil {
		return user, nil
	}

	if err.Status != http.StatusNotFound {
		return nil, err
	}

	return s.createUser(email, verified, mixpanel)
}

//GetUser gets a user from the DB based on the email
func (s *Server) GetUser(email string) (*model.User, *model.AppError) {
	if email == "" {
		return nil, &model.AppError{Code: model.InvalidEmail, Status: http.StatusBadRequest, Message: fmt.Sprintf("empty email when getting user")}
	}

	u, err := s.getUser(model.User{Email: email})
	if err != nil {
		if err == model.ErrNotFound {
			return nil, &model.AppError{Status: http.StatusNotFound, Code: model.EntityNotFound, Message: err.Error()}
		}

		return nil, &model.AppError{Status: http.StatusInternalServerError, Code: model.InternalServerError, Message: err.Error()}

	}

	return u, nil
}

//GetUserByID gets a user from the DB based on the id
func (s *Server) GetUserByID(id string) (*model.User, error) {
	if id == "" {
		return nil, errors.New("missing-id")
	}

	return s.getUser(model.User{Model: model.Model{ID: id}})
}

//GetUserByGithubID gets a user from the DB based on the github id
func (s *Server) GetUserByGithubID(id int) (*model.User, error) {
	if id == 0 {
		return nil, errors.New("missing-id")
	}

	installation := model.GHInstallation{GithubID: id}
	result := s.DB.Where(installation).First(&installation)

	if result.RecordNotFound() {
		return nil, errors.Wrapf(model.ErrNotFound, "ghinstallation with github-user-%d not found", id)
	}

	if result.Error != nil {
		return nil, errors.Wrap(result.Error, model.ErrUnknown.Error())
	}

	if installation.UserID == "" {
		return nil, errors.Wrapf(model.ErrNotFound, "github-user-%d not found", id)
	}

	u, err := s.getUser(model.User{Model: model.Model{ID: installation.UserID}})
	if err == model.ErrNotFound {
		logger.Error(fmt.Errorf("user-%s doesn't exist but linked on ghinstallation-%s", installation.UserID, installation.ID))
	}

	return u, err
}

//InviteUser creates a user and sends an invitation via email, if the user exists.
func (s *Server) InviteUser(authenticatedUser *model.User, email string, project string, role model.ProjectRole) *model.AppError {
	u, err := s.GetUser(email)
	if err == nil {
		if u.Verified {
			// user is already verified
			return &model.AppError{Status: http.StatusBadRequest, Code: model.InvalidEmail}
		}

	} else if err.Code != model.EntityNotFound {
		return err
	}

	p, err := s.GetProject(project, authenticatedUser.ID)
	if err != nil {
		return &model.AppError{Status: http.StatusBadRequest, Code: model.MissingProject, Message: fmt.Sprintf("project-%s doesn't exist: %+v", project, err)}
	}

	// todo: If we already send an invitation email, do we send another one? maybe only do it once every 24 hours?
	if u == nil {
		u, err = s.createUser(email, false, "")
		if err != nil {
			return err
		}
	}

	result := s.DB.Create(&model.ProjectACL{ProjectID: project, UserID: u.ID, Role: role})
	if result.Error != nil {
		return &model.AppError{Status: http.StatusInternalServerError, Code: model.FailToSendEmail}
	}

	// todo: map this to the email, and implement the actual unsubscribe URL
	unsubscribeKey := model.GenerateRandomString(40)
	emailErr := s.Email.sendInviteEmail(u.Email, p.Name, config.GetBaseURL(), fmt.Sprintf("%s/unsubscribe/%s", config.GetBaseURL(), unsubscribeKey))
	if emailErr != nil {
		log.Printf("failed to sendInviteEmail to %s", u.Email)
		return &model.AppError{Status: http.StatusInternalServerError, Code: model.FailToSendEmail, Message: emailErr.Error()}
	}

	return nil

}

//VerifyUser marks the user as verified, this should be called once the user logged in via a valid email provider (so we know is the true owner)
func (s *Server) VerifyUser(u *model.User) *model.AppError {
	err := s.DB.Model(u).Update("verified", true).Error
	if err != nil {
		return &model.AppError{Status: http.StatusInternalServerError, Code: model.UpdateFailed}
	}

	go s.newAccountNotification(u)

	return nil
}

//LinkToMixpanel adds the mixpanel ID to u
func (s *Server) LinkToMixpanel(u *model.User, mixpanel string) error {
	err := s.DB.Model(u).Update("mixpanel", mixpanel).Error
	if err != nil {
		return errors.Wrap(err, "Couldn't link user to mixpanel")
	}

	return nil
}

// DeleteUser deactivates the user from the DB and removes her email. There's no going back from this.
func (s *Server) DeleteUser(u *model.User) *model.AppError {
	if u.ID == "" {
		return &model.AppError{Status: http.StatusInternalServerError, Code: model.DeleteFailed, Message: "id is missing"}
	}

	tx := s.DB.Begin()
	if tx.Error != nil {
		return &model.AppError{
			Status:  http.StatusInternalServerError,
			Code:    model.DeleteFailed,
			Message: fmt.Sprintf("failed to start tx: %s", tx.Error.Error()),
		}
	}

	result := tx.Model(&u).Update("email", u.ID)

	if result.Error != nil {
		tx.Rollback()
		return &model.AppError{
			Status:  http.StatusInternalServerError,
			Code:    model.DeleteFailed,
			Message: fmt.Sprintf("failed to update user's email: %s", result.Error.Error())}
	}

	result = tx.Delete(&u)

	if result.Error != nil {
		tx.Rollback()
		return &model.AppError{
			Status:  http.StatusInternalServerError,
			Code:    model.DeleteFailed,
			Message: fmt.Sprintf("failed to delete the user: %s", result.Error.Error())}
	}

	result = tx.Commit()
	if result.Error != nil {
		tx.Rollback()
		return &model.AppError{
			Status:  http.StatusInternalServerError,
			Code:    model.DeleteFailed,
			Message: fmt.Sprintf("failed to commit the transaction: %s", result.Error.Error())}
	}

	return nil
}

func (s *Server) createUser(email string, verified bool, mixpanel string) (*model.User, *model.AppError) {

	user := model.NewUser(email)
	user.Mixpanel = mixpanel
	user.Verified = verified

	err := s.DB.Create(user).Error
	if err != nil {
		return nil, &model.AppError{Status: http.StatusInternalServerError, Code: model.FailToSendEmail}
	}

	log.Printf("created user-%s", email)
	if verified {
		go s.newAccountNotification(user)
	}

	return user, nil
}

func (s *Server) getUser(condition model.User) (*model.User, error) {
	var user model.User
	result := s.DB.Where(condition).Preload("GHInstallation").First(&user)

	if result.RecordNotFound() {
		return nil, model.ErrNotFound
	}

	if result.Error != nil {
		return nil, errors.Wrap(result.Error, "failed to get user from DB")
	}

	return &user, nil
}
