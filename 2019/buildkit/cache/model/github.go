package model

//GHScope The github scope of the installation. This can be User or Organization.
type GHScope string

// GHInstallation represents every person that installs the okteto github application
// After creating a GHInstallation, it should be linked to at least one project
type GHInstallation struct {
	Model
	//InstallationID The ID of the installation. This is generated by github.
	InstallationID int

	//GithubID The Github ID of the account where the app was installated. This can be the id of a user or an organzation.
	GithubID int

	//GithubLogin The Github login of the account where the app was installed. This can be the id of a user or an organzation.
	GithubLogin string

	//UserID The User ID of the okteto account that claimed the installation.
	UserID string

	//Scope The scope of the installation
	Scope GHScope
}

// GHRepoLink represents every repository/branch combination linked to one (or more) okteto services
// A service can only be linked if the project was linked first
type GHRepoLink struct {
	Model
	InstallationID int    `gorm:"index:idx_installation_repo_branch"`
	RepositoryID   int    `gorm:"index:idx_installation_repo_branch"`
	Branch         string `gorm:"index:idx_installation_repo_branch"`
	Manifest       string
}

const (
	//GHUser a github user
	GHUser = GHScope("User")

	//GHOrganization a github org
	GHOrganization = GHScope("Organization")
)
