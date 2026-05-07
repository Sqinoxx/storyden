package settings

type registrationModeEnum string

const (
	registrationModePublic     registrationModeEnum = "public"
	registrationModeInvitation registrationModeEnum = "invitation"
	registrationModeDisabled   registrationModeEnum = "disabled"
)
