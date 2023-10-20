/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"plandex/lib"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var name string

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Start a new plan.",
	// Long:  ``,
	Args: cobra.ExactArgs(0),
	Run:  new,
}

func init() {
	RootCmd.AddCommand(newCmd)
	newCmd.Flags().StringVarP(&name, "name", "n", "", "Name of the new plan")
}

func new(cmd *cobra.Command, args []string) {
	isDraft := false

	if name == "" {
		name = "draft"
		isDraft = true
	}

	// Check git installed
	if !lib.IsCommandAvailable("git") {
		log.Fatalln("Error: git is required")
	}

	plandexDir, _, err := lib.FindOrCreatePlandex()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error finding or creating .plandex dir:", err)
		return
	}

	err = lib.ClearDraftPlans()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error clearing draft plans:", err)
		return
	}

	name, err = lib.DedupPlanName(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error deduping plan name:", err)
		return
	}

	rootDir := filepath.Join(plandexDir, name)

	// Set the current plan to 'name'
	err = lib.SetCurrentPlan(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error setting current plan:", err)
		return
	}

	// Create 'name' directory inside .plandex
	err = os.Mkdir(rootDir, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		fmt.Fprintln(os.Stderr, "Error creating plan dir:", err)
		return
	}
	// fmt.Fprintln(os.Stderr, "✅ Created plan at "+rootDir)

	// Create context subdirectory
	contextDir := filepath.Join(rootDir, "context")
	err = os.Mkdir(contextDir, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		fmt.Fprintln(os.Stderr, "Error creating context subdir:", err)
		return
	}
	// fmt.Fprintln(os.Stderr, "✅ Created context directory at "+contextDir)

	// Create plan subdirectory
	planDir := filepath.Join(rootDir, "plan")
	err = os.Mkdir(planDir, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		fmt.Fprintln(os.Stderr, "Error creating plan subdir:", err)
		return
	}
	// fmt.Fprintln(os.Stderr, "✅ Created plan directory at "+planDir)

	// Create conversation subdirectory
	conversationDir := filepath.Join(rootDir, "conversation")
	err = os.Mkdir(conversationDir, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		fmt.Fprintln(os.Stderr, "Error creating conversation subdir:", err)
		return
	}
	// fmt.Fprintln(os.Stderr, "✅ Created conversation directory at "+conversationDir)

	// Init empty git repositories in context, plan, and conversation directories
	// init git repo in root plan dir
	// fmt.Println("Initializing git repo in " + rootDir)
	err = lib.InitGitRepo(rootDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error initializing git repo:", err)
		return
	}
	// fmt.Println("Adding git submodules to " + rootDir)

	initAndAddSubmodule := func(dir string) {

		// Initialize the Git repo in the directory
		// fmt.Println("Initializing git repo in " + dir)
		err = lib.InitGitRepo(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error initializing git repo:", err)
			return
		}

		// Add the directory as a submodule to the root repo
		// fmt.Println("Adding git submodule " + dir + " to " + rootDir)
		err = lib.AddGitSubmodule(rootDir, dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error adding git submodules:", err)
			return
		}
	}
	err = lib.WriteInitialContextState(contextDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error writing initial context state:", err)
		return
	}

	// fmt.Println("Wrote initial context state to " + contextDir)

	initAndAddSubmodule(contextDir)
	initAndAddSubmodule(planDir)
	initAndAddSubmodule(conversationDir)

	// After initializing and adding all submodules, ensure they're checked out correctly in the rootDir repo
	// fmt.Println("Updating and initializing submodules in " + rootDir)
	err = lib.UpdateAndInitSubmodules(rootDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error updating submodules:", err)
		return
	}

	// fmt.Fprintln(os.Stderr, "✅ Initialized context and plan git repositories")

	if isDraft {
		fmt.Println("✅ Started new plan")
	} else {
		fmt.Printf("✅ Started new plan: %s\n", color.New(color.Bold, color.FgHiWhite).Sprint(name))
	}

	fmt.Println()
	lib.PrintCmds("load", "tell")

}
