package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"plandex/api"
	"plandex/auth"
	"plandex/format"
	"plandex/fs"
	"plandex/lib"
	"plandex/term"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
	"github.com/xlab/treeprint"
)

func init() {
	RootCmd.AddCommand(plansCmd)
}

// plansCmd represents the list command
var plansCmd = &cobra.Command{
	Use:     "plans",
	Aliases: []string{"pl"},
	Short:   "List all available plans",
	Run:     plans,
}

func plans(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	errCh := make(chan error)

	var parentProjectIdsWithPaths [][2]string
	var childProjectIdsWithPaths [][2]string

	go func() {
		res, err := fs.GetParentProjectIdsWithPaths()

		if err != nil {
			errCh <- fmt.Errorf("error getting parent project ids with paths: %v", err)
			return
		}

		parentProjectIdsWithPaths = res
		errCh <- nil
	}()

	go func() {
		res, err := fs.GetChildProjectIdsWithPaths()

		if err != nil {
			errCh <- fmt.Errorf("error getting child project ids with paths: %v", err)
			return
		}

		childProjectIdsWithPaths = res
		errCh <- nil
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	}

	projectIds := []string{lib.CurrentProjectId}
	for _, p := range parentProjectIdsWithPaths {
		projectIds = append(projectIds, p[1])
	}
	for _, p := range childProjectIdsWithPaths {
		projectIds = append(projectIds, p[1])
	}

	plans, apiErr := api.Client.ListPlans(projectIds)

	if apiErr != nil {
		fmt.Fprintln(os.Stderr, "Error getting plans:", apiErr)
		return
	}

	if len(plans) == 0 {
		fmt.Println("🤷‍♂️ No plans")
		fmt.Println()
		term.PrintCmds("", "new")
		return
	}

	plansByProjectId := make(map[string][]*shared.Plan)
	var currentProjectPlanIds []string
	for _, p := range plans {
		plansByProjectId[p.ProjectId] = append(plansByProjectId[p.ProjectId], p)
		if p.ProjectId == lib.CurrentProjectId {
			currentProjectPlanIds = append(currentProjectPlanIds, p.Id)
		}
	}

	currentBranchNamesByPlanId, err := lib.GetCurrentBranchNamesByPlanId(currentProjectPlanIds)

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting current branches:", err)
		return
	}

	currentBranchesByPlanId, apiErr := api.Client.GetCurrentBranchByPlanId(lib.CurrentProjectId, shared.GetCurrentBranchByPlanIdRequest{
		CurrentBranchByPlanId: currentBranchNamesByPlanId,
	})

	if apiErr != nil {
		fmt.Fprintln(os.Stderr, "Error getting current branches:", apiErr)
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"#", "Name", "Updated", "Created" /*"Branches",*/, "Branch", "Context", "Convo"})

	currentProjectPlans := plansByProjectId[lib.CurrentProjectId]
	if len(currentProjectPlans) > 0 {
		fmt.Println()
		if len(parentProjectIdsWithPaths) > 0 || len(childProjectIdsWithPaths) > 0 {
			color.New(color.Bold, color.FgHiMagenta).Println("Plans in current directory")
		}
		for i, p := range currentProjectPlans {
			num := strconv.Itoa(i + 1)
			if p.Id == lib.CurrentPlanId {
				num = color.New(color.Bold, color.FgHiGreen).Sprint(num)
			}

			var name string
			if p.Id == lib.CurrentPlanId {
				name = color.New(color.Bold, color.FgHiGreen).Sprint(p.Name) + color.New(color.FgWhite).Sprint(" 👈 current")
			} else {
				name = p.Name
			}

			currentBranch := currentBranchesByPlanId[p.Id]

			row := []string{
				num,
				name,
				format.Time(p.UpdatedAt),
				format.Time(p.CreatedAt),
				// strconv.Itoa(p.ActiveBranches),
				currentBranch.Name,
				strconv.Itoa(currentBranch.ContextTokens) + " 🪙",
				strconv.Itoa(currentBranch.ConvoTokens) + " 🪙",
			}

			var style []tablewriter.Colors
			if p.Name == lib.CurrentPlanId {
				style = []tablewriter.Colors{
					{tablewriter.FgHiGreenColor, tablewriter.Bold},
				}
			} else {
				style = []tablewriter.Colors{
					{tablewriter.FgHiWhiteColor, tablewriter.Bold},
					{tablewriter.FgHiWhiteColor},
				}
			}

			table.Rich(row, style)

		}
		table.Render()

		fmt.Println()
		term.PrintCmds("", "tell", "new", "cd", "delete-plan")
	} else {
		fmt.Println("🤷‍♂️ No plans in current directory")
		fmt.Println()
		term.PrintCmds("", "new")
	}

	var addPathToTreeFn func(tree treeprint.Tree, pathParts []string, projectId string, isParent bool)
	addPathToTreeFn = func(tree treeprint.Tree, pathParts []string, projectId string, isParent bool) {
		if len(pathParts) == 0 {
			if plans, ok := plansByProjectId[projectId]; ok {
				for _, plan := range plans {
					tree.AddNode(color.New(color.Bold, color.FgHiGreen).Sprint(plan.Name))
				}
			}
			return
		}
		var fullPath string
		if isParent {
			fullPath = filepath.Join(fs.HomeDir, filepath.Join(pathParts...))
		} else {
			fullPath = filepath.Join(fs.ProjectRoot, filepath.Join(pathParts...))
		}
		pathBase := fs.ProjectRoot
		if isParent {
			pathBase = fs.HomeDir
		}

		relPath, _ := filepath.Rel(pathBase, fullPath)

		branch := tree.FindByValue(relPath)
		if branch == nil {
			if len(plansByProjectId[projectId]) == 0 {
				// If there are no plans for this project, collapse the directories
				tree.AddNode(relPath)
			} else {
				branch = tree.AddBranch(relPath)
				addPathToTreeFn(branch, nil, projectId, isParent)
			}
		}
	}

	if len(parentProjectIdsWithPaths) > 0 {
		fmt.Println()
		color.New(color.Bold, color.FgHiMagenta).Println("Plans in parent directories")
		color.New(color.FgWhite).Println("cd into a directory to work on a plan in that directory")
		parentTree := treeprint.New()
		for _, p := range parentProjectIdsWithPaths {
			pathParts := strings.Split(strings.TrimPrefix(p[0], fs.HomeDir+"/"),
				"/")
			addPathToTreeFn(parentTree, pathParts, p[1], true)
		}
		fmt.Println(parentTree.String())
	}

	if len(childProjectIdsWithPaths) > 0 {
		fmt.Println()
		color.New(color.Bold, color.FgHiMagenta).Println("Plans in child directories")
		color.New(color.FgWhite).Println("cd into a directory to work on a plan in that directory")
		childTree := treeprint.New()
		for _, p := range childProjectIdsWithPaths {
			pathParts := strings.Split(strings.TrimPrefix(p[0],
				fs.ProjectRoot+"/"), "/")
			addPathToTreeFn(childTree, pathParts, p[1], true)
		}
		fmt.Println(childTree.String())
	}
}
