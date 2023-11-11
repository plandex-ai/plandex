package model

import (
	"os"

	"github.com/sashabaranov/go-openai"
)

var PlannerModel = "gpt-4-1106-preview" //openai.GPT4
var PlanSummaryModel = PlannerModel
var BuilderModel = PlannerModel
var ShortSummaryModel = openai.GPT3Dot5Turbo
var NameModel = openai.GPT3Dot5Turbo
var CommitMsgModel = openai.GPT3Dot5Turbo

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
