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

	"github.com/spf13/cobra"
)

var context []string
var name string

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:   "new [prompt]",
	Short: "",
	Long:  ``,
	Args:  cobra.RangeArgs(0, 1),
	Run:   new,
}

func init() {
	addSharedContextFlags(newCmd)
	newCmd.Flags().StringSliceVarP(&context, "context", "c", []string{}, "Context to load into the new plan (file paths or urls)")
	newCmd.Flags().StringVarP(&name, "name", "n", "", "Name of the new plan")
	RootCmd.AddCommand(newCmd)
}

// const minActionDelay = 1000 * time.Millisecond

func new(cmd *cobra.Command, args []string) {
	var prompt string
	if len(args) > 0 {
		prompt = args[0]
	}

	if name == "" && prompt == "" {
		log.Fatalln("Error: either prompt or --name/-n flag must be provided")
	}

	if name == "" {
		summaryResp, err := lib.ApiSummarize(prompt)
		if err != nil {
			log.Fatalf("Failed to get a name for the prompt: %v", err)
		}
		name = lib.GetFileNameWithoutExt(summaryResp.FileName)
	}

	// Check git installed
	if !lib.IsCommandAvailable("git") {
		log.Fatalln("Error: git is required")
	}
	// fmt.Fprintln(os.Stderr, "✅ Confirmed git is installed")

	// Search recursively upward for a .plandex directory
	plandexDir, newd, err := lib.FindOrCreatePlandex()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error finding or creating .plandex dir:", err)
		return
	}
	if newd {
		fmt.Fprintln(os.Stderr, "✅ Created .plandex directory at "+plandexDir)
	} else {
		fmt.Fprintln(os.Stderr, "✅ Found .plandex directory at "+plandexDir)
	}

	// If 'name' directory already exists, tack on an integer to differentiate
	rootDir := filepath.Join(plandexDir, name)
	_, err = os.Stat(rootDir)
	exists := !os.IsNotExist(err)

	postfix := 1
	for exists {
		postfix += 1
		nameWithPostfix := fmt.Sprintf("%s.%d", name, postfix)
		rootDir = filepath.Join(plandexDir, nameWithPostfix)
		_, err = os.Stat(rootDir)
		exists = !os.IsNotExist(err)
	}

	// Create current plan json file with current plan name
	planFilePath := filepath.Join(plandexDir, "current_plan.json")
	planFile, err := os.OpenFile(planFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating current_plan.json:", err)
		return
	}
	defer planFile.Close()
	_, err = planFile.WriteString(fmt.Sprintf(`{"name": "%s"}`, name))

	// Create 'name' directory inside .plandex
	err = os.Mkdir(rootDir, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		fmt.Fprintln(os.Stderr, "Error creating plan dir:", err)
		return
	}
	fmt.Fprintln(os.Stderr, "✅ Created plan at "+rootDir)

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
	fmt.Fprintln(os.Stderr, "✅ Created plan directory at "+planDir)

	// Create conversation subdirectory
	conversationDir := filepath.Join(rootDir, "conversation")
	err = os.Mkdir(conversationDir, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		fmt.Fprintln(os.Stderr, "Error creating conversation subdir:", err)
		return
	}
	fmt.Fprintln(os.Stderr, "✅ Created conversation directory at "+conversationDir)

	// Init empty git repositories in context, plan, and conversation directories
	// init git repo in root plan dir
	// fmt.Println("Initializing git repo in " + rootDir)
	err = lib.InitGitRepo(rootDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error initializing git repo:", err)
		return
	}
	fmt.Println("Adding git submodules to " + rootDir)

	initAndAddSubmodule := func(dir string) {

		// Initialize the Git repo in the directory
		fmt.Println("Initializing git repo in " + dir)
		err = lib.InitGitRepo(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error initializing git repo:", err)
			return
		}

		// Add the directory as a submodule to the root repo
		fmt.Println("Adding git submodule " + dir + " to " + rootDir)
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

	fmt.Println("Wrote initial context state to " + contextDir)

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

	fmt.Fprintln(os.Stderr, "✅ Initialized context and plan git repositories")

	err = lib.LoadCurrentPlan()

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error setting current plan:", err)
		return
	}

	if len(context) > 0 {
		lib.LoadContextOrDie(&lib.LoadContextParams{
			Note:      note,
			MaxTokens: maxTokens,
			Recursive: recursive,
			MaxDepth:  maxDepth,
			NamesOnly: namesOnly,
			Truncate:  truncate,
			Resources: context,
		})
	}

	if prompt != "" {
		err = lib.Prompt(prompt, false)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Prompt error:", err)
			return
		}
	}

}
