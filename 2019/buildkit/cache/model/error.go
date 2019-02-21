package model

import (
	"errors"
	"fmt"
)

const (

	// InvalidJSON is returned when a json string can't be read
	InvalidJSON AppErrorCode = "InvalidJson"

	// InvalidYAML is returned when a yaml string can't be read
	InvalidYAML AppErrorCode = "InvalidYAML"

	// InvalidYAMLWithInfo is returned when a yaml string can't be read
	// and we have some information on what went wrong
	InvalidYAMLWithInfo AppErrorCode = "InvalidYAMLWithInfo"

	// InvalidBase64 is returned when the a base64 string can't be decoded
	InvalidBase64 AppErrorCode = "InvalidBase64"

	// InvalidForm is returned when a form string can't be read
	InvalidForm AppErrorCode = "InvalidForm"

	// MissingID is returned when an entity doesn't have an ID
	MissingID AppErrorCode = "MissingID"

	// MissingName is returned when an entity doesn't have a name
	MissingName AppErrorCode = "MissingName"

	// MissingOwner is returned when an entity doesn't have an owner
	MissingOwner AppErrorCode = "MissingOwner"

	// InvalidName is returned when an entity doesn't have a valid name
	InvalidName AppErrorCode = "InvalidName"

	// UniqueName is returned when another entity in the same scope has the name name
	UniqueName AppErrorCode = "UniqueName"

	// MissingUsers is returned when a project settings don't have any users
	MissingUsers AppErrorCode = "MissingUsers"

	// MissingAdministrators is returned when a project settings don't have any admins
	MissingAdministrators AppErrorCode = "MissingAdministrators"

	// MissingProviderType is returned when the project settings doesn't include a provider
	MissingProviderType AppErrorCode = "MissingProviderType"

	// InvalidProviderType is returned when the project settings doesn't include a valid provider
	InvalidProviderType AppErrorCode = "InvalidProviderType"

	// MissingProviderAccessKey is returned when a project settings doesn't include an access key
	MissingProviderAccessKey AppErrorCode = "MissingProviderAccessKey"

	// MissingProviderSecretKey is returned when a project settings doesn't include a secret key
	MissingProviderSecretKey AppErrorCode = "MissingProviderSecretKey"

	// InvalidKubernetesConfiguration is returned when a project kubernetes settings are not valid
	InvalidKubernetesConfiguration AppErrorCode = "InvalidKubernetesConfiguration"

	// InvalidProviderConfiguration is returned when a project provider settings are not valid
	InvalidProviderConfiguration AppErrorCode = "InvalidProviderConfiguration"

	// InvalidGracePeriod is returned when an entity doesn't have a valid name
	InvalidGracePeriod AppErrorCode = "InvalidGracePeriod"

	// InvalidContainerCount is returned when the service manifest doesn't include at least one container
	InvalidContainerCount AppErrorCode = "InvalidContainerCount"

	// MissingContainerImage is returned when container doesn't have an image
	MissingContainerImage AppErrorCode = "MissingContainerImage"

	// InvalidReplicaCount is returned when the service manifest doesn't include a valid number of replicas
	InvalidReplicaCount AppErrorCode = "InvalidReplicaCount"

	// InvalidPersistentReplica is returned when the service manifest is configured to be persistent and it has more than one replica
	InvalidPersistentReplica AppErrorCode = "InvalidPersistentReplica"

	// InvalidDevContainerCount is returned when more than on container is configured with dev mode
	InvalidDevContainerCount AppErrorCode = "InvalidDevContainerCount"

	// VolumeNotDefined is returned when a volume is mentioned in the service but not defined in the list
	VolumeNotDefined AppErrorCode = "VolumeNotDefined"

	// ProjectNotEmpty is returned when a project still has active services
	ProjectNotEmpty AppErrorCode = "ProjectNotEmpty"

	// HasActiveProjects is returned when a user still has active projects
	HasActiveProjects AppErrorCode = "HasActiveProjects"

	// InsertFailed happens when an entity couldn't be saved
	InsertFailed AppErrorCode = "InsertFailed"

	// UpdateFailed happens when an entity couldn't be updated
	UpdateFailed AppErrorCode = "UpdateFailed"

	// DeleteFailed happens when an entity couldn't be deleted
	DeleteFailed AppErrorCode = "DeleteFailed"

	// EntityNotFound happens when an entity couldn't be found.
	EntityNotFound AppErrorCode = "EntityNotFound"

	// EntityForbidden happens when an entity is found, but the caller doesn't have permission to access it
	EntityForbidden AppErrorCode = "EntityForbidden"

	// InvalidServiceStatus happens when the service is not in a valid state for the action requested
	InvalidServiceStatus AppErrorCode = "InvalidServiceStatus"

	// MissingManifest happens when the service doesn't have a manifest
	MissingManifest AppErrorCode = "MissingManifest"

	// MissingProject is returned when the request is missing the project ID
	MissingProject AppErrorCode = "MissingProject"

	// MissingProjectSettings is returned when the request is missing the project ID
	MissingProjectSettings AppErrorCode = "MissingProjectSettings"

	//MissingGithubScope is returned when the github settings in the project doesn't have a scope
	MissingGithubScope AppErrorCode = "MissingGithubScope"

	//InvalidGithubScope is returned when the scope selected doesn't match the authenticated user's
	InvalidGithubScope AppErrorCode = "InvalidGithubScope"

	//AccoutNotLinkedToGithub is returned when the account selected is not yet linked to github
	AccoutNotLinkedToGithub AppErrorCode = "AccoutNotLinkedToGithub"

	// InvalidURL is returned when the request contains an invalid URL
	InvalidURL AppErrorCode = "InvalidURL"

	// InvalidEmail happens when the project settings have a malformed email
	InvalidEmail AppErrorCode = "InvalidEmail"

	// InvalidRole happens when the user invitation has an invalid role
	InvalidRole AppErrorCode = "InvalidRole"

	// InvalidProject happens when the user invitation has an invalid project
	InvalidProject AppErrorCode = "InvalidProject"

	// InvalidProjectType happens when action is not supported on the selected project type.
	InvalidProjectType AppErrorCode = "InvalidProjectType"

	// MissingToken is returned when an api request doesn't include a token
	MissingToken AppErrorCode = "MissingToken"

	// InvalidToken is returned when the api token is not valid
	InvalidToken AppErrorCode = "InvalidToken"

	//FailToSendEmail is returned when the email provider was not able to send an email
	FailToSendEmail AppErrorCode = "FailToSendEmail"

	//InternalServerError is returned when we don't know what happened
	InternalServerError AppErrorCode = "InternalServerError"

	//AccountNotLinkedToGH is returned when we try to link a service, but the user account is not yet linked
	AccountNotLinkedToGH AppErrorCode = "AccountNotLinkedToGH"
)

// ErrNotFound is returned when the element is not found in the API or DB
var ErrNotFound = errors.New("not-found")

//ErrUnknown is returned when we don't know what happened
var ErrUnknown = errors.New("unknown")

//ErrRefMismatch is returned when a webhook is raised for the wrong branch
var ErrRefMismatch = errors.New("ref-mismatch")

//ErrSHAMismatch is returned when the commit that fired the event doesn't match the head of the repo
var ErrSHAMismatch = errors.New("sha-mismatch")

//ErrProjectNotLinkedToGithub is returned when the project is not linked to github
var ErrProjectNotLinkedToGithub = errors.New("project not linked to github")

//AppErrorCode is the type of error generated by an api call
type AppErrorCode string

//AppError is an error send back from an API call.
type AppError struct {
	Code    AppErrorCode      `json:"code"`
	Status  int               `json:"status"`
	Message string            `json:"-"`
	Data    map[string]string `json:"data"`
}

// Error returns a string of the error
func (a *AppError) Error() string {
	return fmt.Sprintf("%s - %s", a.Code, a.Message)
}
