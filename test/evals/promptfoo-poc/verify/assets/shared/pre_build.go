pdx-1: package cmd
pdx-2: 
pdx-3: import (
pdx-4: 	"fmt"
pdx-5: 	"path/filepath"
pdx-6: 	"plandex/api"
pdx-7: 	"plandex/auth"
pdx-8: 	"plandex/lib"
pdx-9: 	"plandex/term"
pdx-10: 
pdx-11: 	"github.com/plandex/plandex/shared"
pdx-12: 	"github.com/spf13/cobra"
pdx-13: )
pdx-14: 
pdx-15: var contextRmCmd = &cobra.Command{
pdx-16: 	Use:     "rm",
pdx-17: 	Aliases: []string{"remove", "unload"},
pdx-18: 	Short:   "Remove context",
pdx-19: 	Long:    `Remove context by index, name, or glob.`,
pdx-20: 	Args:    cobra.MinimumNArgs(1),
pdx-21: 	Run:     contextRm,
pdx-22: }
pdx-23: 
pdx-24: func contextRm(cmd *cobra.Command, args []string) {
pdx-25: 	auth.MustResolveAuthWithOrg()
pdx-26: 	lib.MustResolveProject()
pdx-27: 
pdx-28: 	if lib.CurrentPlanId == "" {
pdx-29: 		fmt.Println("🤷‍♂️ No current plan")
pdx-30: 		return
pdx-31: 	}
pdx-32: 
pdx-33: 	term.StartSpinner("")
pdx-34: 	contexts, err := api.Client.ListContext(lib.CurrentPlanId, lib.CurrentBranch)
pdx-35: 
pdx-36: 	if err != nil {
pdx-37: 		term.OutputErrorAndExit("Error retrieving context: %v", err)
pdx-38: 	}
pdx-39: 
pdx-40: 	deleteIds := map[string]bool{}
pdx-41: 
pdx-42: 	for i, context := range contexts {
pdx-43: 		for _, id := range args {
pdx-44: 			if fmt.Sprintf("%d", i+1) == id || context.Name == id || context.FilePath == id || context.Url == id {
pdx-45: 				deleteIds[context.Id] = true
pdx-46: 				break
pdx-47: 			} else if context.FilePath != "" {
pdx-48: 				// Check if id is a glob pattern
pdx-49: 				matched, err := filepath.Match(id, context.FilePath)
pdx-50: 				if err != nil {
pdx-51: 					term.OutputErrorAndExit("Error matching glob pattern: %v", err)
pdx-52: 				}
pdx-53: 				if matched {
pdx-54: 					deleteIds[context.Id] = true
pdx-55: 					break
pdx-56: 				}
pdx-57: 
pdx-58: 				// Check if id is a parent directory
pdx-59: 				parentDir := context.FilePath
pdx-60: 				for parentDir != "." && parentDir != "/" && parentDir != "" {
pdx-61: 					if parentDir == id {
pdx-62: 						deleteIds[context.Id] = true
pdx-63: 						break
pdx-64: 					}
pdx-65: 					parentDir = filepath.Dir(parentDir) // Move up one directory
pdx-66: 				}
pdx-67: 
pdx-68: 			}
pdx-69: 		}
pdx-70: 	}
pdx-71: 
pdx-72: 	if len(deleteIds) > 0 {
pdx-73: 		res, err := api.Client.DeleteContext(lib.CurrentPlanId, lib.CurrentBranch, shared.DeleteContextRequest{
pdx-74: 			Ids: deleteIds,
pdx-75: 		})
pdx-76: 		term.StopSpinner()
pdx-77: 
pdx-78: 		if err != nil {
pdx-79: 			term.OutputErrorAndExit("Error deleting context: %v", err)
pdx-80: 		}
pdx-81: 
pdx-82: 		fmt.Println("✅ " + res.Msg)
pdx-83: 	} else {
pdx-84: 		term.StopSpinner()
pdx-85: 		fmt.Println("🤷‍♂️ No context removed")
pdx-86: 	}
pdx-87: }
pdx-88: 
pdx-89: func init() {
pdx-90: 	RootCmd.AddCommand(contextRmCmd)
pdx-91: }
pdx-92: 