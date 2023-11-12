package model

import (
	"os"

	"github.com/sashabaranov/go-openai"
)

const strongModel = openai.GPT4TurboPreview
const weakModel = openai.GPT3Dot5Turbo1106

var PlannerModel = strongModel
var PlanSummaryModel = strongModel
var BuilderModel = strongModel //openai.GPT4
var ShortSummaryModel = weakModel
var NameModel = weakModel
var CommitMsgModel = weakModel

func init() {
	if os.Getenv("PLANNER_MODEL") != "" {
		PlannerModel = os.Getenv("PLANNER_MODEL")
	}
	if os.Getenv("PLAN_SUMMARY_MODEL") != "" {
		PlanSummaryModel = os.Getenv("PLAN_SUMMARY_MODEL")
	}
	if os.Getenv("BUILDER_MODEL") != "" {
		BuilderModel = os.Getenv("BUILDER_MODEL")
	}
	if os.Getenv("SHORT_SUMMARY_MODEL") != "" {
		ShortSummaryModel = os.Getenv("SHORT_SUMMARY_MODEL")
	}
	if os.Getenv("NAME_MODEL") != "" {
		NameModel = os.Getenv("NAME_MODEL")
	}
	if os.Getenv("COMMIT_MSG_MODEL") != "" {
		CommitMsgModel = os.Getenv("COMMIT_MSG_MODEL")
	}
}
