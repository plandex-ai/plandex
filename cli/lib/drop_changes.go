package lib

import (
	"plandex/shared"
	"time"
)

func rejectReplacement(replacement *shared.Replacement) {
	currentTime := time.Now().Format(time.RFC3339)
	replacement.RejectedAt = currentTime
}

func DropChanges(name string) error {
	plandexDir, _, err := FindOrCreatePlandex()
	if err != nil {
		return fmt.Errorf("error finding or creating plandex dir: %w", err)
	}

	if name == "" {
		fmt.Println("\uD83E\uDD37\u200D\u2642\uFE0F No plan specified and no current plan")
		return nil
	}

	rootDir := filepath.Join(plandexDir, name)

	_, err = os.Stat(rootDir)

	if os.IsNotExist(err) {
		fmt.Printf("\uD83E\uDD37\u200D\u2642\uFE0F Plan with name '%s' doesn't exist\n", name)
		return nil
	} else if err != nil {
		return fmt.Errorf("error checking if plan exists: %w", err)
	}

	res, err := GetCurrentPlanState()

	if err != nil {
		return fmt.Errorf("error getting current plan files: %w", err)
	}

	planResByPath := res.PlanResByPath

	for _, planResults := range planResByPath {
		for _, planResult := range planResults {
			for _, replacement := range planResult.Replacements {
				rejectReplacement(replacement)
			}
		}
	}

	return nil
}

func DropChange(name string, changeId string) error {
	plandexDir, _, err := FindOrCreatePlandex()
	if err != nil {
		return fmt.Errorf("error finding or creating plandex dir: %w", err)
	}

	if name == "" {
		fmt.Println("🤷‍♂️ No plan specified and no current plan")
		return nil
	}

	rootDir := filepath.Join(plandexDir, name)

	_, err = os.Stat(rootDir)

	if os.IsNotExist(err) {
		fmt.Printf("🤷‍♂️ Plan with name '%s' doesn't exist\n", name)
		return nil
	} else if err != nil {
		return fmt.Errorf("error checking if plan exists: %w", err)
	}

	res, err := GetCurrentPlanState()

	if err != nil {
		return fmt.Errorf("error getting current plan files: %w", err)
	}

	planResByPath := res.PlanResByPath

	for _, planResults := range planResByPath {
		for _, planResult := range planResults {
			for _, replacement := range planResult.Replacements {
				if replacement.Id == changeId {
					rejectReplacement(replacement)
					return nil
				}
			}
		}
	}

	fmt.Printf("🤷‍♂️ Change with ID '%s' doesn't exist in plan '%s'\n", changeId, name)
	return nil
}
