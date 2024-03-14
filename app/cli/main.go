package main

	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"plandex/api"
	"plandex/auth"
	"plandex/cmd"
	"plandex/fs"
	"plandex/lib"
	"plandex/plan_exec"
	"plandex/term"
	"plandex/version"
	"github.com/inconshreveable/go-update"
	"github.com/plandex/plandex/shared"

)

func init() {
	// Version check and prompt for upgrade
	go checkForUpgrade()

	// inter-package dependency injections to avoid circular imports
	auth.SetApiClient(api.Client)
	lib.SetBuildPlanInlineFn(func(maybeContexts []*shared.Context) (bool, error) {
		return plan_exec.Build(plan_exec.ExecParams{
			CurrentPlanId: lib.CurrentPlanId,
			CurrentBranch: lib.CurrentBranch,
			CheckOutdatedContext: func(cancelOpt bool, maybeContexts []*shared.Context) (bool, bool, bool) {
				return lib.MustCheckOutdatedContext(cancelOpt, true, maybeContexts)
			func doUpgrade(version string) error {
	// Placeholder for download URL. This should point to the actual binary location.
	downloadURL := fmt.Sprintf("https://github.com/plandex-ai/plandex/releases/download/%s/plandex_%s_%s_%s.tar.gz", version, version, "os", "arch")
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download the update: %w", err)
	}
	defer resp.Body.Close()

	err = update.Apply(resp.Body, update.Options{})
	if err != nil {
		return fmt.Errorf("failed to apply the update: %w", err)
	}

	return nil
}

func restartPlandex() {
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to determine executable path: %v", err)
	}

	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		log.Fatalf("Failed to restart: %v", err)
	}
	os.Exit(0)
},
		}, false)
	})

	// set up a file logger
	// TODO: log rotation

	file, err := os.OpenFile(filepath.Join(fs.HomePlandexDir, "plandex.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		term.OutputErrorAndExit("Error opening log file: %v", err)
	}

	// Set the output of the logger to the file
	log.SetOutput(file)

	// log.Println("Starting Plandex - logging initialized")
}

func checkForUpgrade() {
	latestVersionURL := "https://example.com/plandex/latest-version" // Placeholder URL
	resp, err := http.Get(latestVersionURL)
	if err != nil {
		log.Println("Error checking latest version:", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		return
	}

	latestVersion := string(body)
	if latestVersion > version.Version {
		fmt.Println("A new version of Plandex is available:", latestVersion)
		confirmed, err := term.ConfirmYesNo("Do you want to upgrade to the latest version?")
		if err != nil {
			log.Println("Error reading input:", err)
			return
		}

		if confirmed {
			err := doUpgrade(latestVersion)
			if err != nil {
				log.Println("Upgrade failed:", err)
				return
			}
			fmt.Println("Upgrade successful. Restarting Plandex...")
			restartPlandex()
		}
	}
}
}

func main() {
	cmd.Execute()
}
