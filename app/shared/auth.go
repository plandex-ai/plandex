package shared

import "fmt"

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

	ApiErrorTypeCloudInsufficientCredits ApiErrorType = "cloud_insufficient_credits"
	ApiErrorTypeCloudMonthlyMaxReached   ApiErrorType = "cloud_monthly_max_reached"
	ApiErrorTypeCloudSubscriptionPaused  ApiErrorType = "cloud_subscription_paused"
	ApiErrorTypeCloudSubscriptionOverdue ApiErrorType = "cloud_subscription_overdue"

	ApiErrorTypeOther ApiErrorType = "other"
)

type TrialPlansExceededError struct {
	MaxPlans int `json:"maxPlans"`
}

type TrialMessagesExceededError struct {
	MaxReplies int `json:"maxMessages"`
}

type BillingError struct {
	HasBillingPermission bool `json:"hasBillingPermission"`
	IsTrial              bool `json:"isTrial"`
}

type ApiError struct {
	Type   ApiErrorType `json:"type"`
	Status int          `json:"status"`
	Msg    string       `json:"msg"`

	// only used for trial plans exceeded error
	TrialPlansExceededError *TrialPlansExceededError `json:"trialPlansExceededError,omitempty"`

	// only used for trial messages exceeded error
	TrialMessagesExceededError *TrialMessagesExceededError `json:"trialMessagesExceededError,omitempty"`

	// only used for billing errors
	BillingError *BillingError `json:"billingError,omitempty"`
}

func (e *ApiError) Error() string {
	return fmt.Sprintf("%d Error: %s", e.Status, e.Msg)
}

type ClientAccount struct {
	IsCloud     bool   `json:"isCloud"`
	Host        string `json:"host"`
	Email       string `json:"email"`
	UserName    string `json:"userName"`
	UserId      string `json:"userId"`
	Token       string `json:"token"`
	IsLocalMode bool   `json:"isLocalMode"`

	IsTrial bool `json:"isTrial"` // legacy field
}

type ClientAuth struct {
	ClientAccount
	OrgId                string `json:"orgId"`
	OrgName              string `json:"orgName"`
	OrgIsTrial           bool   `json:"orgIsTrial"`
	IntegratedModelsMode bool   `json:"integratedModelsMode"`
}
