package problem

const baseURI = "https://postulate.dev"

// Error type URI constants.
const (
	TypeNotFound            = baseURI + "/errors/not-found"
	TypeMethodNotAllowed    = baseURI + "/errors/method-not-allowed"
	TypeValidationFailed    = baseURI + "/errors/validation-failed"
	TypeInternalServerError = baseURI + "/errors/internal-server-error"
)

// New constructs a Problem with the given fields.
func New(errorType, title string, status int, detail, instance string) *Problem {
	return &Problem{
		Type:     errorType,
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: instance,
	}
}

// NewValidation constructs a ValidationProblem for a 422 response.
func NewValidation(detail, instance string, errors []FieldError) *ValidationProblem {
	return &ValidationProblem{
		Problem: Problem{
			Type:     TypeValidationFailed,
			Title:    "Validation Failed",
			Status:   422,
			Detail:   detail,
			Instance: instance,
		},
		Errors: errors,
	}
}
