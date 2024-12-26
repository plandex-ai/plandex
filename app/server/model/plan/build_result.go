package plan

import (
	"context"
	"fmt"
	"log"
	"plandex-server/db"
	"plandex-server/syntax"
	"sort"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
)

type OverlapStrategy int

const (
	OverlapStrategySkip OverlapStrategy = iota
	OverlapStrategyError
)

type PlanResultParams struct {
	OrgId               string
	PlanId              string
	PlanBuildId         string
	ConvoMessageId      string
	FilePath            string
	PreBuildState       string
	OverlapStrategy     OverlapStrategy
	ChangesWithLineNums []*shared.StreamedChangeWithLineNums

	CheckSyntax bool

	IsFix       bool
	IsSyntaxFix bool
	IsOtherFix  bool
	FixEpoch    int
}

type PlanResultParamsUpdated struct {
	OrgId               string
	PlanId              string
	PlanBuildId         string
	ConvoMessageId      string
	FilePath            string
	PreBuildState       string
	ProposedChanges     string
	EntireFile          bool
	OverlapStrategy     OverlapStrategy
	ChangesWithLineNums []*shared.StreamedChangeWithLineNumsUpdated

	CheckSyntax bool

	IsFix       bool
	IsSyntaxFix bool
	IsOtherFix  bool
	FixEpoch    int
}

func GetPlanResultUpdated(ctx context.Context, params PlanResultParamsUpdated) (*db.PlanFileResult, string, bool, error) {
	orgId := params.OrgId
	planId := params.PlanId
	planBuildId := params.PlanBuildId
	filePath := params.FilePath
	preBuildState := params.PreBuildState
	streamedChangesWithLineNums := params.ChangesWithLineNums

	preBuildState = shared.AddLineNums(preBuildState)
	preBuildStateLines := strings.Split(preBuildState, "\n")
	proposedChangesLines := strings.Split(params.ProposedChanges, "\n")

	var replacements []*shared.Replacement

	if params.EntireFile {
		replacements = append(replacements, &shared.Replacement{
			Old:                   preBuildState,
			New:                   params.ProposedChanges,
			StreamedChangeUpdated: nil,
			EntireFile:            true,
		})
	} else {

		var highestEndLineOld int = 0
		var highestEndLineNew int = 0

		for _, streamedChange := range streamedChangesWithLineNums {
			if !streamedChange.HasChange {
				continue
			}

			getNewSection := func(highestEndLineNewPointer *int) func() (string, error) {
				highestEndLineNew := *highestEndLineNewPointer

				return func() (string, error) {
					log.Printf("getNewSection - File %s: Getting new section\n", filePath)
					startLineNew, endLineNew, err := streamedChange.New.GetLinesWithPrefix("pdx-new-")
					if err != nil {
						log.Printf("getPlanResult - File %s: Error getting lines from streamedChange: %v\n", filePath, err)

						spew.Dump(streamedChange.New)

						return "", fmt.Errorf("error getting lines from streamedChange: %v", err)
					}

					if startLineNew > len(proposedChangesLines) {
						log.Printf("getPlanResult - File %s: Start line is greater than proposedChangesLines length: %d > %d\n", filePath, startLineNew, len(proposedChangesLines))
						return "", fmt.Errorf("start line is greater than proposedChangesLines length: %d > %d", startLineNew, len(proposedChangesLines))
					}

					if endLineNew < 1 {
						log.Printf("getPlanResult - File %s: End line is less than 1: %d\n", filePath, endLineNew)
						return "", fmt.Errorf("end line is less than 1: %d", endLineNew)
					}
					if endLineNew > len(proposedChangesLines) {
						log.Printf("getPlanResult - File %s: End line is greater than proposedChangesLines length: %d > %d\n", filePath, endLineNew, len(proposedChangesLines))
						return "", fmt.Errorf("end line is greater than proposedChangesLines length: %d > %d", endLineNew, len(proposedChangesLines))
					}

					if startLineNew < highestEndLineNew {
						log.Printf("getPlanResult - File %s: Start line is less than highestEndLine: %d < %d\n", filePath, startLineNew, highestEndLineNew)

						// log.Printf("getPlanResult - File %s: streamedChange:\n", filePath)
						// log.Println(spew.Sdump(streamedChangesWithLineNums))

						spew.Dump(streamedChangesWithLineNums)

						if params.OverlapStrategy == OverlapStrategyError {
							return "", fmt.Errorf("start line is less than highestEndLine: %d < %d", startLineNew,
								highestEndLineNew)
						} else {
							return "", nil
						}
					}

					if endLineNew < highestEndLineNew {
						if params.OverlapStrategy == OverlapStrategyError {
							log.Printf("getPlanResult - File %s: End line is less than highestEndLine: %d < %d\n", filePath, endLineNew, highestEndLineNew)
							return "", fmt.Errorf("end line is less than highestEndLine: %d < %d", endLineNew, highestEndLineNew)
						} else {
							return "", nil
						}
					}

					if endLineNew > highestEndLineNew {
						// log.Printf("getPlanResult - File %s: End line is greater than highestEndLine: %d > %d\n", filePath, endLineNew, highestEndLineNew)
						// log.Println("Setting highestEndLineNew to endLineNew:", endLineNew)
						highestEndLineNew = endLineNew
					}

					var new string
					if startLineNew == endLineNew {
						new = proposedChangesLines[startLineNew-1]
					} else {
						new = strings.Join(proposedChangesLines[startLineNew-1:endLineNew], "\n")
					}

					return new, nil
				}
			}(&highestEndLineNew)

			var old string
			var new string

			newSection, err := getNewSection()
			if err != nil {
				log.Printf("getPlanResult - File %s: Error getting new section: %v\n", filePath, err)
				return nil, "", false, fmt.Errorf("error getting new section: %v", err)
			} else if newSection == "" {
				continue
			}

			if streamedChange.InsertBefore != nil && streamedChange.InsertBefore.ShouldInsertBefore {
				lineNum, err := shared.ExtractLineNumber(streamedChange.InsertBefore.Line)
				if err != nil {
					log.Printf("getPlanResult - File %s: Error extracting line number from firstLine: %v\n", filePath, err)
					return nil, "", false, fmt.Errorf("error extracting line number from firstLine: %v", err)
				}
				line := preBuildStateLines[lineNum-1]

				old = line
				new = newSection + "\n" + line

				replacements = append(replacements, &shared.Replacement{
					Old:                   old,
					New:                   new,
					StreamedChangeUpdated: streamedChange,
				})

				continue
			}

			if streamedChange.InsertAfter != nil && streamedChange.InsertAfter.ShouldInsertAfter {
				lineNum, err := shared.ExtractLineNumber(streamedChange.InsertAfter.Line)
				if err != nil {
					log.Printf("getPlanResult - File %s: Error extracting line number from firstLine: %v\n", filePath, err)
					return nil, "", false, fmt.Errorf("error extracting line number from firstLine: %v", err)
				}
				line := preBuildStateLines[lineNum-1]

				old = line
				new = line + "\n" + newSection

				replacements = append(replacements, &shared.Replacement{
					Old:                   old,
					New:                   new,
					StreamedChangeUpdated: streamedChange,
				})

				continue
			}

			startLineOld, endLineOld, err := streamedChange.Old.GetLines()
			if err != nil {
				spew.Dump(streamedChange)

				log.Printf("getPlanResult - File %s: Error getting lines from streamedChange: %v\n", filePath, err)
				return nil, "", false, fmt.Errorf("error getting lines from streamedChange: %v", err)
			}

			if startLineOld > len(preBuildStateLines) {
				log.Printf("getPlanResult - File %s: Start line is greater than preBuildStateLines length: %d > %d\n", filePath, startLineOld, len(preBuildStateLines))
				return nil, "", false, fmt.Errorf("start line is greater than preBuildStateLines length: %d > %d", startLineOld, len(preBuildStateLines))
			}

			if endLineOld < 1 {
				log.Printf("getPlanResult - File %s: End line is less than 1: %d\n", filePath, endLineOld)
				return nil, "", false, fmt.Errorf("end line is less than 1: %d", endLineOld)
			}
			if endLineOld > len(preBuildStateLines) {
				log.Printf("getPlanResult - File %s: End line is greater than preBuildStateLines length: %d > %d\n", filePath, endLineOld, len(preBuildStateLines))
				return nil, "", false, fmt.Errorf("end line is greater than preBuildStateLines length: %d > %d", endLineOld, len(preBuildStateLines))
			}

			if startLineOld < highestEndLineOld {
				log.Printf("getPlanResult - File %s: Start line is less than highestEndLine: %d < %d\n", filePath, startLineOld, highestEndLineOld)

				log.Printf("getPlanResult - File %s: streamedChange:\n", filePath)
				log.Println(spew.Sdump(streamedChangesWithLineNums))

				if params.OverlapStrategy == OverlapStrategyError {
					return nil, "", false, fmt.Errorf("start line is less than highestEndLine: %d < %d", startLineOld,
						highestEndLineOld)
				} else {
					continue
				}
			}

			if endLineOld < highestEndLineOld {
				if params.OverlapStrategy == OverlapStrategyError {
					// log.Printf("getPlanResult - File %s: End line is less than highestEndLine: %d < %d\n", filePath, endLineOld, highestEndLineOld)
					// return nil, "", false, fmt.Errorf("end line is less than highestEndLine: %d < %d", endLineOld, highestEndLineOld)
				} else {
					continue
				}
			}

			if endLineOld > highestEndLineOld {
				highestEndLineOld = endLineOld
			}

			if startLineOld == endLineOld {
				old = preBuildStateLines[startLineOld-1]
			} else {
				old = strings.Join(preBuildStateLines[startLineOld-1:endLineOld], "\n")
			}

			replacement := &shared.Replacement{
				Old:                   old,
				New:                   newSection,
				StreamedChangeUpdated: streamedChange,
			}

			replacements = append(replacements, replacement)
		}
	}

	log.Printf("getPlanResult - File %s: Will apply replacements\n", filePath)

	// spew.Dump(replacements)

	// sort replacements by index of the 'old' string property in the original file
	sort.Slice(replacements, func(i, j int) bool {
		oldI := replacements[i].Old
		oldJ := replacements[j].Old

		idxI := strings.Index(preBuildState, oldI)
		idxJ := strings.Index(preBuildState, oldJ)

		return idxI < idxJ
	})

	updated, allSucceeded := shared.ApplyReplacements(preBuildState, replacements, true)

	// log.Printf("getPlanResult - File %s: updated:\n", filePath)
	// log.Println(updated)

	updated = shared.RemoveLineNums(updated)

	for _, replacement := range replacements {
		id := uuid.New().String()
		replacement.Id = id
	}

	res := db.PlanFileResult{
		TypeVersion:         1,
		ReplaceWithLineNums: true,
		OrgId:               orgId,
		PlanId:              planId,
		PlanBuildId:         planBuildId,
		ConvoMessageId:      params.ConvoMessageId,
		Content:             "",
		Path:                filePath,
		Replacements:        replacements,
		AnyFailed:           !allSucceeded,
	}

	if params.CheckSyntax {
		validationRes, err := syntax.Validate(ctx, filePath, updated)
		if err != nil {
			log.Printf("getPlanResult - File %s: Error validating syntax: %v\n", filePath, err)
			return nil, "", false, fmt.Errorf("error validating syntax: %v", err)
		}

		res.WillCheckSyntax = validationRes.HasParser && !validationRes.TimedOut
		res.SyntaxValid = validationRes.Valid
		res.SyntaxErrors = validationRes.Errors
	}

	return &res, updated, allSucceeded, nil
}

func GetPlanResult(ctx context.Context, params PlanResultParams) (*db.PlanFileResult, string, bool, error) {
	orgId := params.OrgId
	planId := params.PlanId
	planBuildId := params.PlanBuildId
	filePath := params.FilePath
	preBuildState := params.PreBuildState
	streamedChangesWithLineNums := params.ChangesWithLineNums

	preBuildState = shared.AddLineNums(preBuildState)

	preBuildStateLines := strings.Split(preBuildState, "\n")

	var replacements []*shared.Replacement

	var highestEndLine int = 0

	for _, streamedChange := range streamedChangesWithLineNums {
		if !streamedChange.HasChange {
			continue
		}

		var old string
		new := streamedChange.New

		if streamedChange.Old.EntireFile {
			replacements = append(replacements, &shared.Replacement{
				EntireFile:     true,
				Old:            old,
				New:            new,
				StreamedChange: streamedChange,
			})
			continue
		}

		startLine, endLine, err := streamedChange.Old.GetLines()
		if err != nil {
			log.Printf("getPlanResult - File %s: Error getting lines from streamedChange: %v\n", filePath, err)
			return nil, "", false, fmt.Errorf("error getting lines from streamedChange: %v", err)
		}

		if startLine > len(preBuildStateLines) {
			log.Printf("getPlanResult - File %s: Start line is greater than preBuildStateLines length: %d > %d\n", filePath, startLine, len(preBuildStateLines))
			return nil, "", false, fmt.Errorf("start line is greater than preBuildStateLines length: %d > %d", startLine, len(preBuildStateLines))
		}

		if endLine < 1 {
			log.Printf("getPlanResult - File %s: End line is less than 1: %d\n", filePath, endLine)
			return nil, "", false, fmt.Errorf("end line is less than 1: %d", endLine)
		}
		if endLine > len(preBuildStateLines) {
			log.Printf("getPlanResult - File %s: End line is greater than preBuildStateLines length: %d > %d\n", filePath, endLine, len(preBuildStateLines))
			return nil, "", false, fmt.Errorf("end line is greater than preBuildStateLines length: %d > %d", endLine, len(preBuildStateLines))
		}

		if startLine < highestEndLine {
			log.Printf("getPlanResult - File %s: Start line is less than highestEndLine: %d < %d\n", filePath, startLine, highestEndLine)

			log.Printf("getPlanResult - File %s: streamedChange:\n", filePath)
			log.Println(spew.Sdump(streamedChangesWithLineNums))

			if params.OverlapStrategy == OverlapStrategyError {
				return nil, "", false, fmt.Errorf("start line is less than highestEndLine: %d < %d", startLine,
					highestEndLine)
			} else {
				continue
			}
		}

		if endLine < highestEndLine {
			if params.OverlapStrategy == OverlapStrategyError {
				log.Printf("getPlanResult - File %s: End line is less than highestEndLine: %d < %d\n", filePath, endLine, highestEndLine)
				return nil, "", false, fmt.Errorf("end line is less than highestEndLine: %d < %d", endLine, highestEndLine)
			} else {
				continue
			}
		}

		if endLine > highestEndLine {
			highestEndLine = endLine
		}

		if startLine == endLine {
			old = preBuildStateLines[startLine-1]
		} else {
			old = strings.Join(preBuildStateLines[startLine-1:endLine], "\n")
		}

		replacement := &shared.Replacement{
			Old:            old,
			New:            new,
			StreamedChange: streamedChange,
		}

		replacements = append(replacements, replacement)
	}

	log.Printf("getPlanResult - File %s: Will apply replacements\n", filePath)

	updated, allSucceeded := shared.ApplyReplacements(preBuildState, replacements, true)

	updated = shared.RemoveLineNums(updated)

	for _, replacement := range replacements {
		id := uuid.New().String()
		replacement.Id = id
	}

	res := db.PlanFileResult{
		TypeVersion:         1,
		ReplaceWithLineNums: true,
		OrgId:               orgId,
		PlanId:              planId,
		PlanBuildId:         planBuildId,
		ConvoMessageId:      params.ConvoMessageId,
		Content:             "",
		Path:                filePath,
		Replacements:        replacements,
		AnyFailed:           !allSucceeded,
	}

	if params.CheckSyntax {
		validationRes, err := syntax.Validate(ctx, filePath, updated)
		if err != nil {
			log.Printf("getPlanResult - File %s: Error validating syntax: %v\n", filePath, err)
			return nil, "", false, fmt.Errorf("error validating syntax: %v", err)
		}

		res.WillCheckSyntax = validationRes.HasParser && !validationRes.TimedOut
		res.SyntaxValid = validationRes.Valid
		res.SyntaxErrors = validationRes.Errors
	}

	return &res, updated, allSucceeded, nil
}
