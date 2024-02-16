package shared

type AuthHeader struct {
	Token string `json:"token"`
	OrgId string `json:"orgId"`
}

type ApiErrorType string

const (
	ApiErrorTypeInvalidToken          ApiErrorType = "invalid_token"
	ApiErrorTypeTrialPlansExceeded    ApiErrorType = "trial_plans_exceeded"
	ApiErrorTypeTrialMessagesExceeded ApiErrorType = "trial_messages_exceeded"
	ApiErrorTypeTrialActionNotAllowed ApiErrorType = "trial_action_not_allowed"

	ApiErrorTypeContinueNoMessages ApiErrorType = "continue_no_messages"

	ApiErrorTypeOther ApiErrorType = "other"
)

type TrialPlansExceededError struct {
	MaxPlans int `json:"maxPlans"`
}

type TrialMessagesExceededError struct {
	MaxReplies int `json:"maxMessages"`
}

type ApiError struct {
	Type   ApiErrorType `json:"type"`
	Status int          `json:"status"`
	Msg    string       `json:"msg"`

	// only used for trial plans exceeded error
	TrialPlansExceededError *TrialPlansExceededError `json:"trialPlansExceededError,omitempty"`

	// only used for trial messages exceeded error
	TrialMessagesExceededError *TrialMessagesExceededError `json:"trialMessagesExceededError,omitempty"`
}
