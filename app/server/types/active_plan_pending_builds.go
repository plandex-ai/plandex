package types

import (
	"fmt"
	"log"
	"plandex-server/db"

	"github.com/plandex/plandex/shared"
)

func (ap *ActivePlan) PendingBuildsByPath(orgId, userId string, convoMessagesArg []*db.ConvoMessage) (map[string][]*ActiveBuild, error) {
	planDescs, err := db.GetConvoMessageDescriptions(orgId, ap.Id)
	if err != nil {
		return nil, fmt.Errorf("error getting pending build descriptions: %v", err)
	}

	if !HasPendingBuilds(planDescs) {
		return map[string][]*ActiveBuild{}, nil
	}

	var convoMessages []*db.ConvoMessage
	if convoMessagesArg == nil {
		var err error
		convoMessages, err = db.GetPlanConvo(orgId, ap.Id)

		if err != nil {
			return nil, fmt.Errorf("error getting plan convo: %v", err)
		}
	} else {
		convoMessages = convoMessagesArg
	}

	convoMessagesById := map[string]*db.ConvoMessage{}
	for _, msg := range convoMessages {
		convoMessagesById[msg.Id] = msg
	}

	activeBuildsByPath := map[string][]*ActiveBuild{}

	for _, desc := range planDescs {
		if (!desc.DidBuild && len(desc.Operations) > 0) || len(desc.BuildPathsInvalidated) > 0 {
			if desc.ConvoMessageId == "" {
				log.Printf("No convo message ID for description: %v\n", desc)
				return nil, fmt.Errorf("no convo message ID for description: %v", desc)
			}

			if convoMessagesById[desc.ConvoMessageId] == nil {
				log.Printf("No convo message for ID: %s\n", desc.ConvoMessageId)
				return nil, fmt.Errorf("no convo message for ID: %s", desc.ConvoMessageId)
			}

			// convoMessage := convoMessagesById[desc.ConvoMessageId]

			// replyParser := NewReplyParser()
			// replyParser.AddChunk(convoMessage.Message, false)
			// parserRes := replyParser.FinishAndRead()

			numAdded := 0
			for _, op := range desc.Operations {

				if desc.DidBuild && !desc.BuildPathsInvalidated[op.Path] {
					continue
				}

				if activeBuildsByPath[op.Path] == nil {
					activeBuildsByPath[op.Path] = []*ActiveBuild{}
				}

				numTokens, err := shared.GetNumTokens(op.Content)

				if err != nil {
					log.Printf("Error getting num tokens for file content: %v\n", err)
					return nil, fmt.Errorf("error getting num tokens for file content: %v", err)
				}

				activeBuildsByPath[op.Path] = append(activeBuildsByPath[op.Path], &ActiveBuild{
					ReplyId:           desc.ConvoMessageId,
					FileContent:       op.Content,
					FileContentTokens: numTokens,
					Path:              op.Path,
					FileDescription:   op.Description,
				})
				numAdded++
			}

		}
	}

	// log.Println("activeBuildsByPath:")
	// spew.Dump(activeBuildsByPath)

	return activeBuildsByPath, nil
}
