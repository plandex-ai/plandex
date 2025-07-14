package lib

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/fs"
	"plandex-cli/term"
	"plandex-cli/types"
	shared "plandex-shared"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/fatih/color"
)

type ProviderAuthStatus int

const (
	FullySatisfied ProviderAuthStatus = iota
	PartiallySatisfied
	FullyMissing
)

type ProviderCredentialStatus struct {
	ProviderComposite string
	Status            ProviderAuthStatus
	MissingVars       []string
}

type PublisherCredentialStatus struct {
	Publisher        shared.ModelPublisher
	SelectedProvider *ProviderCredentialStatus
	PartialProviders []ProviderCredentialStatus
}

type CredentialCheckResult struct {
	AllSatisfied bool
	Publishers   []PublisherCredentialStatus
	AuthVars     map[string]string
}

func CheckCredentialStatus(opts shared.ModelProviderOptions, claudeMaxEnabled bool) (CredentialCheckResult, error) {
	publishersToProviders := groupProvidersByPublisher(opts)

	selectedAuthVars := map[string]string{}
	var publisherStatuses []PublisherCredentialStatus
	allSatisfied := true

	for publisher, providers := range publishersToProviders {
		var selectedProvider *ProviderCredentialStatus
		partialProviders := []ProviderCredentialStatus{}

		for _, provider := range providers {
			if provider.Config.HasClaudeMaxAuth && !claudeMaxEnabled {
				continue
			}

			authVars, err := ResolveProviderAuthVars(provider.Config)
			if err != nil {
				return CredentialCheckResult{}, fmt.Errorf("error checking API keys/credentials: %v", err)
			}
			status, missingVars, err := checkProviderCredentialStatus(provider.Config, authVars)
			if err != nil {
				return CredentialCheckResult{}, fmt.Errorf("error checking API keys/credentials: %v", err)
			}

			providerStatus := ProviderCredentialStatus{
				ProviderComposite: provider.Config.ToComposite(),
				Status:            status,
				MissingVars:       missingVars,
			}

			if status == FullySatisfied {
				selectedProvider = &providerStatus
				mergeAuthVars(selectedAuthVars, authVars)
				break // first fully satisfied provider found, stop looking further
			} else if status == PartiallySatisfied {
				partialProviders = append(partialProviders, providerStatus)
			}
			// otherwise,if fully missing, don't set selected provider
		}

		if selectedProvider == nil {
			allSatisfied = false
		}

		publisherStatuses = append(publisherStatuses, PublisherCredentialStatus{
			Publisher:        publisher,
			SelectedProvider: selectedProvider,
			PartialProviders: partialProviders,
		})
	}

	return CredentialCheckResult{
		AllSatisfied: allSatisfied,
		Publishers:   publisherStatuses,
		AuthVars:     selectedAuthVars,
	}, nil
}

func groupProvidersByPublisher(opts shared.ModelProviderOptions) map[shared.ModelPublisher][]shared.ModelProviderOption {
	grouped := map[shared.ModelPublisher][]shared.ModelProviderOption{}
	for _, option := range opts {
		for pub := range option.Publishers {
			grouped[pub] = append(grouped[pub], option)
		}
	}
	// stable priority sort
	for pub := range grouped {
		sort.SliceStable(grouped[pub], func(i, j int) bool {
			return grouped[pub][i].Priority < grouped[pub][j].Priority
		})
	}
	return grouped
}

func checkProviderCredentialStatus(cfg *shared.ModelProviderConfigSchema, authVars map[string]string) (ProviderAuthStatus, []string, error) {
	var missing []string

	if cfg.SkipAuth {
		return FullySatisfied, nil, nil
	}

	if cfg.HasClaudeMaxAuth {
		creds, err := GetAccountCredentials()
		if err != nil {
			return FullyMissing, nil, fmt.Errorf("error getting account credentials: %v", err)
		}
		if creds == nil || creds.ClaudeMax == nil {
			return FullyMissing, nil, nil
		}
	}

	if cfg.ApiKeyEnvVar != "" && authVars[cfg.ApiKeyEnvVar] == "" {
		missing = append(missing, cfg.ApiKeyEnvVar)
	}

	for _, extra := range cfg.ExtraAuthVars {
		if extra.Required && authVars[extra.Var] == "" {
			missing = append(missing, extra.Var)
		}
	}

	numRequired := 0
	for _, extra := range cfg.ExtraAuthVars {
		if extra.Required {
			numRequired++
		}
	}
	if cfg.ApiKeyEnvVar != "" {
		numRequired++
	}

	switch {
	case len(missing) == 0:
		return FullySatisfied, nil, nil
	case len(missing) == numRequired:
		return FullyMissing, missing, nil
	default:
		return PartiallySatisfied, missing, nil
	}
}

func MustVerifyAuthVars(integratedModels bool) map[string]string {
	return mustVerifyAuthVars(integratedModels, false)
}

func MustVerifyAuthVarsSilent(integratedModels bool) map[string]string {
	return mustVerifyAuthVars(integratedModels, true)
}

func mustVerifyAuthVars(integratedModels, silent bool) map[string]string {
	if !silent {
		term.StartSpinner("")
	}

	planSettings, apiErr := api.Client.GetSettings(CurrentPlanId, CurrentBranch)
	if apiErr != nil {
		term.OutputErrorAndExit("Error getting settings: %v", apiErr)
	}

	orgUserConfig := MustGetOrgUserConfig()

	opts := planSettings.GetModelProviderOptions()

	if !silent {
		if hasAnthropicModels(opts) {
			didConnect := promptClaudeMaxIfNeeded()
			term.StartSpinner("")
			if !didConnect && orgUserConfig.UseClaudeSubscription {
				didConnect = connectClaudeMaxIfNeeded()
				if !didConnect {
					refreshClaudeMaxCredsIfNeeded()
				}
			}
		}
	}

	// For IntegratedModelsMode on Cloud, we only send the connected Claude subscription api key—nothing else
	// If we're in IntegratedModelsMode and there's no connected Claude sub, return nil
	if integratedModels {
		if orgUserConfig.UseClaudeSubscription {
			creds, err := GetAccountCredentials()
			if err != nil {
				term.OutputErrorAndExit("Error getting Claude subscription credentials: %v", err)
			}

			if creds != nil && creds.ClaudeMax != nil {
				return map[string]string{
					shared.AnthropicClaudeMaxTokenEnvVar: creds.ClaudeMax.AccessToken,
				}
			}
		}
		return nil
	}

	checkResult, err := CheckCredentialStatus(opts, orgUserConfig.UseClaudeSubscription)
	if err != nil {
		term.OutputErrorAndExit("Error checking API keys/credentials: %v", err)
	}
	if checkResult.AllSatisfied {
		return checkResult.AuthVars
	}

	showCredentialErrorMessage(checkResult, opts)
	os.Exit(1)
	return nil
}

func ResolveProviderAuthVars(cfg *shared.ModelProviderConfigSchema) (map[string]string, error) {
	authVars := map[string]string{}

	if cfg.SkipAuth {
		return authVars, nil
	}

	if cfg.HasAWSAuth {
		// PLANDEX_AWS_PROFILE enables credential file loading from ~/.aws/credentials
		// this ensures it's opt-in so we only use bedrock if user explicitly intends to
		profile := os.Getenv("PLANDEX_AWS_PROFILE")
		if profile != "" {
			os.Setenv("AWS_PROFILE", profile)
			if err := loadAWSVars(authVars); err == nil {
				return authVars, nil
			}
		}
		// if no PLANDEX_AWS_PROFILE is set OR loading aws vars fails, just silently fall through to the default env var checks

		// because we're disabling the EC2 metadata service, the aws check will fail unless appropriate env vars or credentials file is found, but it's not actually a problem—just indicates AWS creds aren't set
	}

	if cfg.HasClaudeMaxAuth {
		creds, err := GetAccountCredentials()
		if err != nil {
			return nil, fmt.Errorf("error getting account credentials: %v", err)
		}

		if creds != nil && creds.ClaudeMax != nil {
			token := creds.ClaudeMax.AccessToken
			authVars[shared.AnthropicClaudeMaxTokenEnvVar] = token
		}
	}

	if cfg.ApiKeyEnvVar != "" {
		val := os.Getenv(cfg.ApiKeyEnvVar)
		if val != "" {
			authVars[cfg.ApiKeyEnvVar] = val
		}
	}

	for _, extra := range cfg.ExtraAuthVars {
		val := os.Getenv(extra.Var)
		if val == "" && extra.Default != "" {
			val = extra.Default
		}

		if extra.MaybeJSONFilePath {
			if val == "" {
				continue
			}
			content, err := maybeLoadFile(val)
			if err != nil {
				return nil, fmt.Errorf("failed to load file for %s: %v", extra.Var, err)
			}
			authVars[extra.Var] = content
		} else if val != "" {
			authVars[extra.Var] = val
		}
	}

	return authVars, nil
}

func maybeLoadFile(pathOrJson string) (string, error) {
	if strings.HasPrefix(strings.TrimSpace(pathOrJson), "{") {
		// var contains json directly, so we can return it as is
		return pathOrJson, nil
	}

	// see if it's base64 encoded json
	decoded, err := base64.StdEncoding.DecodeString(pathOrJson)
	if err == nil {
		s := string(decoded)
		if strings.HasPrefix(strings.TrimSpace(s), "{") {
			return s, nil
		}
	}

	content, err := os.ReadFile(pathOrJson)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func loadAWSVars(vars map[string]string) error {
	// disable IMDS to prevent slow request
	os.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %v", err)
	}

	creds, err := cfg.Credentials.Retrieve(context.Background())
	if err != nil {
		return fmt.Errorf("failed to retrieve AWS credentials: %v", err)
	}

	vars["AWS_ACCESS_KEY_ID"] = creds.AccessKeyID
	vars["AWS_SECRET_ACCESS_KEY"] = creds.SecretAccessKey
	vars["AWS_REGION"] = cfg.Region
	if creds.SessionToken != "" {
		vars["AWS_SESSION_TOKEN"] = creds.SessionToken
	}

	return nil
}

func mergeAuthVars(dest, src map[string]string) {
	for k, v := range src {
		dest[k] = v
	}
}

func showCredentialErrorMessage(res CredentialCheckResult, opts shared.ModelProviderOptions) {
	boldRed := color.New(color.Bold, term.ColorHiRed)
	cyanChip := color.New(color.BgCyan, color.FgHiWhite)
	fmt.Println(boldRed.Sprint("🚨 Required API key(s) or model credentials are missing"))

	someOK, someMissing := false, false
	for _, p := range res.Publishers {
		if p.SelectedProvider != nil && p.SelectedProvider.Status == FullySatisfied {
			someOK = true
		} else {
			someMissing = true
		}
	}
	if someOK && someMissing {
		fmt.Println()
		fmt.Println(color.New(color.Bold, term.ColorHiYellow).Sprint("⚠️  Some models are missing a provider"))
		sorted := make([]PublisherCredentialStatus, 0, len(res.Publishers))
		for _, p := range res.Publishers {
			sorted = append(sorted, p)
		}
		sort.Slice(sorted, func(i, j int) bool {
			readyI := sorted[i].SelectedProvider != nil && sorted[i].SelectedProvider.Status == FullySatisfied
			readyJ := sorted[j].SelectedProvider != nil && sorted[j].SelectedProvider.Status == FullySatisfied
			return readyI && !readyJ
		})
		for _, p := range sorted {
			ready := p.SelectedProvider != nil && p.SelectedProvider.Status == FullySatisfied
			var lbl string
			if ready {
				lbl = p.SelectedProvider.ProviderComposite
			} else {
				lbl = "missing"
			}
			fmt.Printf("%s %s models → %s\n", mark(ready), p.Publisher, lbl)
		}
	}

	var partialLines []string
	added := map[string]bool{}
	for _, pub := range res.Publishers {
		partialProviders := pub.PartialProviders
		for _, sp := range partialProviders {
			// already added this provider
			if added[sp.ProviderComposite] {
				continue
			}

			// compute vars set vs missing
			opt, ok := opts[sp.ProviderComposite]
			if !ok {
				continue
			}
			req := requiredVars(opt.Config)
			var setVars []string
			for _, v := range req {
				missing := false
				for _, mv := range sp.MissingVars {
					if mv == v {
						missing = true
						break
					}
				}
				if !missing {
					setVars = append(setVars, v)
				}
			}
			added[sp.ProviderComposite] = true
			partialLines = append(partialLines,
				fmt.Sprintf("%s\n  set → %s\n  missing → %s",
					color.New(color.Bold).Sprint(sp.ProviderComposite),
					strings.Join(setVars, ", "),
					strings.Join(sp.MissingVars, ", "),
				))
		}
	}
	if len(partialLines) > 0 {
		sort.Strings(partialLines)
		fmt.Println()
		fmt.Println(color.New(term.ColorHiYellow, color.Bold).Sprint("⚠️  Providers with partial credentials"))
		for _, l := range partialLines {
			fmt.Println(l)
		}
	}

	byPub := providersByPublisher(opts)

	allPublishersHaveOpenRouter := allPublishersHaveProvider(byPub, shared.ModelProviderOpenRouter)

	if allPublishersHaveOpenRouter {
		fmt.Println()
		fmt.Println(color.New(term.ColorHiCyan, color.Bold).Sprint("🚀 Quick option → OpenRouter.ai"))
		fmt.Println("OpenRouter allows you to use all models in the current model pack with a single account and API key. To get started:")
		fmt.Println()
		step := func(n int, txt string) { fmt.Printf("%d. %s\n", n, txt) }
		step(1, "Sign up at "+color.New(color.Bold).Sprint("https://openrouter.ai/sign-up"))
		step(2, "Buy some credits at "+color.New(color.Bold).Sprint("https://openrouter.ai/settings/credits"))
		step(3, "Generate an API key at "+color.New(color.Bold).Sprint("https://openrouter.ai/settings/keys"))
		if term.IsRepl {
			step(4, "Quit the REPL with "+cyanChip.Sprint(" \\quit "))
			step(5, "Run "+cyanChip.Sprint(" export OPENROUTER_API_KEY=… "))
			step(6, "Restart the REPL with "+cyanChip.Sprint(" plandex "))
		} else {
			step(4, "Run "+cyanChip.Sprint(" export OPENROUTER_API_KEY=… "))
		}
	}

	if len(byPub) > 0 {
		fmt.Println()
		fmt.Println(color.New(term.ColorHiCyan, color.Bold).Sprint("🔑 Other model providers"))
		if allPublishersHaveOpenRouter {
			fmt.Println("You can also use the following providers for the current model pack:")
		} else {
			fmt.Println("You can use the following providers for the current model pack:")
		}

		fmt.Println()
		pubs := make([]string, 0, len(byPub))
		for p := range byPub {
			pubs = append(pubs, string(p))
		}
		sort.Strings(pubs)
		for _, p := range pubs {
			providers := byPub[shared.ModelPublisher(p)]
			providerNames := make([]string, 0, len(providers))
			for _, provider := range providers {
				if allPublishersHaveOpenRouter && provider == shared.ModelProviderOpenRouter {
					continue
				}
				providerNames = append(providerNames, string(provider))
			}
			fmt.Printf("%s → %s\n", color.New(color.Bold).Sprint(p+" models"), strings.Join(providerNames, ", "))
		}

		fmt.Println(color.New(color.Bold, term.ColorHiCyan).Sprint("\n📖 Per-provider instructions"))
		fmt.Println("For details on the API key/credentials required for each provider, go to:\n" + color.New(color.Bold).Sprint("https://docs.plandex.ai/models/model-providers"))
	}

	fmt.Println()
}

// return required env‑var names (API key + required extras)
func requiredVars(cfg *shared.ModelProviderConfigSchema) []string {
	var vars []string
	if cfg.ApiKeyEnvVar != "" {
		vars = append(vars, cfg.ApiKeyEnvVar)
	}
	for _, ex := range cfg.ExtraAuthVars {
		if ex.Required {
			vars = append(vars, ex.Var)
		}
	}
	return vars
}

func providersByPublisher(opts shared.ModelProviderOptions) map[shared.ModelPublisher][]shared.ModelProvider {
	byPub := map[shared.ModelPublisher][]shared.ModelProvider{}
	sortedOpts := make([]shared.ModelProviderOption, 0, len(opts))
	for _, opt := range opts {
		sortedOpts = append(sortedOpts, opt)
	}
	sort.Slice(sortedOpts, func(i, j int) bool {
		return sortedOpts[i].Priority < sortedOpts[j].Priority
	})
	for _, opt := range sortedOpts {
		for pub := range opt.Publishers {
			byPub[pub] = append(byPub[pub], opt.Config.Provider)
		}
	}
	return byPub
}

func allPublishersHaveProvider(byPub map[shared.ModelPublisher][]shared.ModelProvider, p shared.ModelProvider) bool {
	if len(byPub) == 0 {
		return false
	}

	for _, providers := range byPub {
		found := false
		for _, provider := range providers {
			if provider == p {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}
	return true
}

// emoji for satisfied vs missing
func mark(ok bool) string {
	if ok {
		return "✅"
	}
	return "❌"
}

var cachedAccountCredentials *types.AccountCredentials

func SetAccountCredentials(creds *types.AccountCredentials) error {
	if auth.Current == nil {
		return fmt.Errorf("no authenticated user")
	}
	dir := filepath.Join(fs.HomePlandexDir, auth.Current.UserId, auth.Current.OrgId)
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return fmt.Errorf("error creating account credentials directory: %v", err)
	}
	path := filepath.Join(dir, "creds.json")
	bytes, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling account credentials: %v", err)
	}
	err = os.WriteFile(path, bytes, 0600)
	if err != nil {
		return fmt.Errorf("error writing account credentials: %v", err)
	}

	cachedAccountCredentials = creds

	return nil
}

func GetAccountCredentials() (*types.AccountCredentials, error) {
	if cachedAccountCredentials != nil {
		return cachedAccountCredentials, nil
	}

	if auth.Current == nil {
		return nil, fmt.Errorf("no authenticated user")
	}
	path := filepath.Join(fs.HomePlandexDir, auth.Current.UserId, auth.Current.OrgId, "creds.json")

	bytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var creds types.AccountCredentials
	err = json.Unmarshal(bytes, &creds)
	if err != nil {
		return nil, err
	}

	cachedAccountCredentials = &creds

	return &creds, nil
}
