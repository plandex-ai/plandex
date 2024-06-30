pdx-1: package cmd
pdx-2: 
pdx-3: import (
pdx-4: 	"fmt"
pdx-5: 	"path/filepath"
pdx-6: 	"plandex/api"
pdx-7: 	"plandex/auth"
pdx-8: 	"plandex/lib"
pdx-9: 	"plandex/term"
pdx-10: 	"strconv"
pdx-11: 	"strings"
pdx-12: 
pdx-13: 	"github.com/plandex/plandex/shared"
pdx-14: 	"github.com/spf13/cobra"
pdx-15: )
pdx-16: 
pdx-17: func parseRange(arg string) ([]int, error) {
pdx-18: 	var indices []int 
pdx-19: 	parts := strings.Split(arg, "-")
pdx-20: 	if len(parts) == 2 {
pdx-21: 		start, err := strconv.Atoi(parts[0])
pdx-22: 		if err != nil {
pdx-23: 			return nil, err
pdx-24: 		}
pdx-25: 		end, err := strconv.Atoi(parts[1])
pdx-26: 		if err != nil {
pdx-27: 			return nil, err
pdx-28: 		}
pdx-29: 		for i := start; i <= end; i++ {
pdx-30: 			indices = append(indices, i)
pdx-31: 		}
pdx-32: 	} else {
pdx-33: 		index, err := strconv.Atoi(arg)
pdx-34: 		if err != nil {
pdx-35: 			return nil, err
pdx-36: 		}
pdx-37: 		indices = append(indices, index)
pdx-38: 	}
pdx-39: 	return indices, nil
pdx-40: }
pdx-41: 
pdx-42: func contextRm(cmd *cobra.Command, args []string) {
pdx-43: 	auth.MustResolveAuthWithOrg()
pdx-44: 	lib.MustResolveProject()
pdx-45: 
pdx-46: 	if lib.CurrentPlanId == "" {
pdx-47: 		fmt.Println("🤷‍♂️ No current plan")
pdx-48: 		return
pdx-49: 	}
pdx-50: 
pdx-51: 	term.StartSpinner("")
pdx-52: 	contexts, err := api.Client.ListContext(lib.CurrentPlanId, lib.CurrentBranch)
pdx-53: 
pdx-54: 	if err != nil {
pdx-55: 		term.OutputErrorAndExit("Error retrieving context: %v", err)
pdx-56: 	}
pdx-57: 
pdx-58: 	deleteIds := map[string]bool{}
pdx-59: 
pdx-60: 	for _, arg := range args {
pdx-61: 		indices, err := parseRange(arg)
pdx-62: 		if err != nil {
pdx-63: 			term.OutputErrorAndExit("Error parsing range: %v", err)
pdx-64: 		}
pdx-65: 
pdx-66: 		for _, index := range indices {
pdx-67: 			if index > 0 && index <= len(contexts) {
pdx-68: 				context := contexts[index-1]
pdx-69: 				deleteIds[context.Id] = true
pdx-70: 			}
pdx-71: 		}
pdx-72: 	}
pdx-73: 
pdx-74: 	for i, context := range contexts {
pdx-75: 		for _, id := range args {
pdx-76: 			if fmt.Sprintf("%d", i+1) == id || context.Name == id || context.FilePath == id || context.Url == id {
pdx-77: 				deleteIds[context.Id] = true
pdx-78: 				break
pdx-79: 			} else if context.FilePath != "" {
pdx-80: 				// Check if id is a glob pattern
pdx-81: 				matched, err := filepath.Match(id, context.FilePath)
pdx-82: 				if err != nil {
pdx-83: 					term.OutputErrorAndExit("Error matching glob pattern: %v", err)
pdx-84: 				}
pdx-85: 				if matched {
pdx-86: 					deleteIds[context.Id] = true
pdx-87: 					break
pdx-88: 				}
pdx-89: 
pdx-90: 				// Check if id is a parent directory
pdx-91: 				parentDir := context.FilePath
pdx-92: 				for parentDir != "." && parentDir != "/" && parentDir != "" {
pdx-93: 					if parentDir == id {
pdx-94: 						deleteIds[context.Id] = true
pdx-95: 						break
pdx-96: 					}
pdx-97: 					parentDir = filepath.Dir(parentDir) // Move up one directory
pdx-98: 				}
pdx-99: 			}
pdx-100: 		}
pdx-101: 	}
pdx-102: 
pdx-103: 	if len(deleteIds) > 0 {
pdx-104: 		res, err := api.Client.DeleteContext(lib.CurrentPlanId, lib.CurrentBranch, shared.DeleteContextRequest{
pdx-105: 			Ids: deleteIds,
pdx-106: 		})
pdx-107: 		term.StopSpinner()
pdx-108: 
pdx-109: 		if err != nil {
pdx-110: 			term.OutputErrorAndExit("Error deleting context: %v", err)
pdx-111: 		}
pdx-112: 
pdx-113: 		fmt.Println("✅ " + res.Msg)
pdx-114: 	} else {
pdx-115: 		term.StopSpinner()
pdx-116: 		fmt.Println("🤷‍♂️ No context removed")
pdx-117: 	}
pdx-118: }
pdx-119: 
pdx-120: func init() {
pdx-121: 	RootCmd.AddCommand(contextRmCmd)
pdx-122: }
pdx-123: 