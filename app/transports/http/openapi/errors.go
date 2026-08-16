package openapi

import "github.com/Southclaws/fault/ftag"

// KindEmailNotVerified marks an authorisation failure that was caused solely by
// an unverified email address rather than a genuine lack of permission. It is
// still a 403, but carrying a distinct problem type lets clients offer to
// resend the verification code instead of showing a dead-end error.
//
// The problem type is derived from the kind, so this surfaces as
// urn:storyden:problem:email-not-verified.
const KindEmailNotVerified ftag.Kind = "EMAIL_NOT_VERIFIED"
