package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

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
	lib.MaybeResolveProject()

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
		ctx, _ := context.WithTimeout(context.Background(), 500*time.Millisecond)

		res, err := fs.GetChildProjectIdsWithPaths(ctx)

		if err != nil {
			log.Println(err.Error())

			if err.Error() == "context timeout" {
				errCh <- nil
				return
			}

			errCh <- fmt.Errorf("error getting child project ids with paths: %v", err)
			return
		}

		childProjectIdsWithPaths = res
		errCh <- nil
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil {
			term.OutputErrorAndExit("%v", err)
		}
	}

	var projectIds []string

	if lib.CurrentProjectId != "" {
		projectIds = append(projectIds, lib.CurrentProjectId)
	}

	for _, p := range parentProjectIdsWithPaths {
		projectIds = append(projectIds, p[1])
	}
	for _, p := range childProjectIdsWithPaths {
		projectIds = append(projectIds, p[1])
	}

	if len(projectIds) == 0 {
		fmt.Println("🤷‍♂️ No plans")
		fmt.Println()
		term.PrintCmds("", "new")
		return
	}

	term.StartSpinner("")
	plans, apiErr := api.Client.ListPlans(projectIds)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting plans: %v", apiErr)
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

	for projectId, plans := range plansByProjectId {
		if projectId != lib.CurrentProjectId {
			// sort non-current-project plans alphabetically
			sort.Slice(plans, func(i, j int) bool {
				return plans[i].Name < plans[j].Name
			})
		}
	}

	if len(currentProjectPlanIds) > 0 {
		currentBranchNamesByPlanId, err := lib.GetCurrentBranchNamesByPlanId(currentProjectPlanIds)

		if err != nil {
			term.OutputErrorAndExit("Error getting current branches: %v", err)
		}

		currentBranchesByPlanId, apiErr := api.Client.GetCurrentBranchByPlanId(lib.CurrentProjectId, shared.GetCurrentBranchByPlanIdRequest{
			CurrentBranchByPlanId: currentBranchNamesByPlanId,
		})

		if apiErr != nil {
			term.OutputErrorAndExit("Error getting current branches: %v", apiErr)
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoWrapText(false)
		table.SetHeader([]string{"#", "Name", "Updated" /*, "Created" /*"Branches",*/, "Branch", "Context", "Convo"})

		currentProjectPlans := plansByProjectId[lib.CurrentProjectId]
		if len(parentProjectIdsWithPaths) > 0 || len(childProjectIdsWithPaths) > 0 {
			color.New(color.Bold, term.ColorHiGreen).Print("Plans in current directory\n")
		} else {
			fmt.Println()
		}
		for i, p := range currentProjectPlans {
			num := strconv.Itoa(i + 1)
			if p.Id == lib.CurrentPlanId {
				num = color.New(color.Bold, term.ColorHiGreen).Sprint(num)
			}

			var name string
			if p.Id == lib.CurrentPlanId {
				name = color.New(color.Bold, term.ColorHiGreen).Sprint(p.Name) + fmt.Sprint(" 👈")
			} else {
				name = p.Name
			}

			currentBranch := currentBranchesByPlanId[p.Id]

			row := []string{
				num,
				name,
				format.Time(p.UpdatedAt),
				// format.Time(p.CreatedAt),
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
					{tablewriter.Bold},
				}
			}

			table.Rich(row, style)

		}
		table.Render()

	} else {
		fmt.Println("🤷‍♂️ No plans in current directory")
	}

	var addPathToTreeFn func(tree treeprint.Tree, basePath, localPath, projectId string, isParent bool)
	addPathToTreeFn = func(tree treeprint.Tree, basePath, localPath, projectId string, isParent bool) {
		var base string
		var tail string
		split := strings.Split(localPath, string(os.PathSeparator))

		var baseBranch treeprint.Tree
		for _, part := range split {
			base = filepath.Join(base, part)
			tail = strings.TrimPrefix(localPath, base+string(os.PathSeparator))

			var searchBranch string
			if isParent {
				baseFull := filepath.Join(fs.HomeDir, basePath, base)
				baseRel, _ := filepath.Rel(fs.Cwd, baseFull)
				searchBranch = fmt.Sprintf("%s (%s)", base, baseRel)

				// 	log.Println("Project root:", fs.Cwd)
				// 	log.Println("searchBranch:", searchBranch)
				// 	log.Println("base:", base)
				// 	log.Println("tail:", tail)
				// 	log.Println("basePath:", basePath)
				// 	log.Println("baseFull:", baseFull)
				// 	log.Println("baseRel:", baseRel)
			} else {
				searchBranch = base
			}

			baseBranch = tree.FindByValue(searchBranch)
			if baseBranch != nil {
				addPathToTreeFn(baseBranch, filepath.Join(basePath, base), tail, projectId, isParent)
				return
			}
		}

		if baseBranch == nil {
			label := localPath
			if isParent {
				pathFull := filepath.Join(fs.HomeDir, basePath, localPath)
				pathRel, _ := filepath.Rel(fs.Cwd, pathFull)
				label = fmt.Sprintf("%s (%s)", localPath, pathRel)

			}

			branch := tree.AddBranch(label)
			plans := plansByProjectId[projectId]

			for _, p := range plans {
				branch.AddNode(color.New(term.ColorHiCyan).Sprint(p.Name))
			}
		}
	}

	var c color.Attribute
	if term.IsDarkBg {
		c = color.FgWhite
	} else {
		c = color.FgBlack
	}

	if len(parentProjectIdsWithPaths) > 0 {
		fmt.Println()

		color.New(color.Bold).Println("Plans in parent directories")
		color.New(c).Println("cd into a directory to work on a plan in that directory")
		parentTree := treeprint.NewWithRoot("~")

		for i := len(parentProjectIdsWithPaths) - 1; i >= 0; i-- {
			p := parentProjectIdsWithPaths[i]

			rel, err := filepath.Rel(fs.HomeDir, p[0])

			if err != nil {
				term.OutputErrorAndExit("Error getting relative path: %v", err)
			}

			addPathToTreeFn(parentTree, "", rel, p[1], true)
		}
		fmt.Print(parentTree.String())
	}

	if len(childProjectIdsWithPaths) > 0 {
		fmt.Println()
		color.New(color.Bold).Println("Plans in child directories")
		color.New(c).Println("cd into a directory to work on a plan in that directory")
		childTree := treeprint.New()
		for _, p := range childProjectIdsWithPaths {
			rel, err := filepath.Rel(fs.Cwd, p[0])

			if err != nil {
				term.OutputErrorAndExit("Error getting relative path: %v", err)
			}

			addPathToTreeFn(childTree, "", rel, p[1], false)
		}
		fmt.Println(childTree.String())
	} else {
		fmt.Println()
	}

	fmt.Println()
	if len(currentProjectPlanIds) > 0 {
		term.PrintCmds("", "new", "cd", "delete-plan")
	} else {
		term.PrintCmds("", "new")
	}
}
