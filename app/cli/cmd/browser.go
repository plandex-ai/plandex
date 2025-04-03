package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	chrome_log "github.com/chromedp/cdproto/log"
	chrome_runtime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/spf13/cobra"
)

var timeoutSeconds int

// browserCmd is our cobra command
var browserCmd = &cobra.Command{
	Use:   "browser [urls...]",
	Short: "Open browser windows with given URLs, capturing console logs and exiting on JS errors",
	RunE:  browser,
}

func init() {
	RootCmd.AddCommand(browserCmd)
	browserCmd.Flags().IntVar(&timeoutSeconds, "timeout", 10, "Timeout in seconds for browser to load")
}

// browser is the main function for our command.
func browser(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("no URLs provided")
	}

	// See if we can find Chrome or Firefox in PATH:
	chromePath, _ := findChrome()

	if chromePath != "" {
		fmt.Println("Using Chrome (not default browser, but found in PATH).")
		return openChromeWithLogs(args)
	}

	// Fallback: open with OS default tool (open/xdg-open)
	fmt.Println("No Chrome or Firefox found; falling back to OS default opener.")
	return openWithOSDefault(args)
}

// findChrome returns the path to Chrome/Chromium if found, or "" if not found.
func findChrome() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		// macOS standard installation paths
		macPaths := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
		for _, p := range macPaths {
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	case "linux", "freebsd":
		// Linux/FreeBSD (usually in PATH)
		candidates := []string{
			"google-chrome",
			"google-chrome-stable",
			"chromium",
			"chromium-browser",
		}
		for _, c := range candidates {
			if path, err := exec.LookPath(c); err == nil {
				return path, nil
			}
		}
	}
	return "", errors.New("Chrome/Chromium not found")
}

var pages = map[target.ID]string{}

// openChromeWithLogs uses chromedp to open each URL in a visible Chrome browser,
// logs JS console messages, and exits on the first JS error (or if user presses Ctrl+C).
func openChromeWithLogs(urls []string) error {
	if len(urls) == 0 {
		return errors.New("no URLs provided")
	}

	fmt.Println("Launching Chrome with console log capture...")
	// 1) Create a cancellable context that sets up a Chrome ExecAllocator
	rootCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(),
		chromedp.Flag("headless", false), // Visible, not headless
		chromedp.Flag("disable-gpu", false),
		chromedp.Flag("no-first-run", true),             // Avoid "Welcome" dialog
		chromedp.Flag("no-default-browser-check", true), // Avoid default browser dialog
		chromedp.Flag("disable-default-apps", true),     // Prevent default apps/extensions from loading
		chromedp.Flag("remote-debugging-port", "0"),     // Ensures separate debug session
	)
	defer cancelAllocator()

	// 2) Create a new browser context from the allocator
	browserCtx, cancelBrowser := chromedp.NewContext(rootCtx)
	defer cancelBrowser()

	// Add this to get the initial target
	var initialTargetID target.ID
	chromedp.ListenTarget(browserCtx, func(ev interface{}) {
		if e, ok := ev.(*target.EventTargetCreated); ok {
			// fmt.Printf("[DEBUG] Got event: %#v\n", e)
			if e.TargetInfo.Type == "page" {
				initialTargetID = e.TargetInfo.TargetID
			}
		}
	})

	if err := chromedp.Run(
		browserCtx,
		chrome_runtime.Enable(),
		chrome_log.Enable(),
	); err != nil {
		log.Fatal(err)
	}

	// Listen for console API calls & JS exceptions
	chromedp.ListenBrowser(browserCtx, func(ev interface{}) {
		// fmt.Printf("[DEBUG] Got event: %#v\n", ev)
		switch e := ev.(type) {

		case *target.EventTargetCreated:
		case *target.EventAttachedToTarget:
			// This sometimes also includes target info
			info := e.TargetInfo
			if info.Type == "page" {
				url := info.URL
				if url == "about:blank" {
					url = urls[0]
				}
				pages[info.TargetID] = url
			}

		case *target.EventTargetDestroyed:
			// Something was destroyed
			destroyedID := e.TargetID
			// Check if it was a top-level page
			if url, ok := pages[destroyedID]; ok {
				fmt.Printf("[CHROME] Closed page: %s\n", url)
				// remove from map
				delete(pages, destroyedID)
				// If that was your last page, do something:
				if len(pages) == 0 {
					cancelBrowser()
					cancelAllocator()
					os.Exit(0)
				} else {
					fmt.Printf("num pages left: %d\n", len(pages))
				}
			}
		}
	})

	// Channels to handle signals & error detection
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan bool, 1)

	// 3) Open all URLs in new tabs
	for i, url := range urls {
		// Reuse the initial browser tab for the first URL
		var ctx context.Context
		if i == 0 {
			ctx, _ = chromedp.NewContext(browserCtx, chromedp.WithTargetID(initialTargetID))
		} else {
			// Create a new tab for each additional URL
			ctx, _ = chromedp.NewContext(browserCtx)
		}
		tabCtx, _ := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)

		chromedp.ListenTarget(tabCtx, func(ev interface{}) {
			switch e := ev.(type) {
			case *chrome_runtime.EventConsoleAPICalled:
				logType := e.Type.String()
				for _, arg := range e.Args {
					val := arg.Value
					fmt.Printf("[CHROME:%s] %v\n", logType, val)
					if logType == "error" {
						fmt.Println("❌ Chrome console error")
						errChan <- true
					}
				}
			case *chrome_log.EventEntryAdded:
				fmt.Printf("[CHROME:Log:%s] %s\n", e.Entry.Level, e.Entry.Text)
			case *chrome_runtime.EventExceptionThrown:
				fmt.Printf("[CHROME:Exception] %s\n", e.ExceptionDetails.Error())
				fmt.Println("❌ JavaScript exception")
				errChan <- true
			}
		})

		if err := chromedp.Run(
			tabCtx,
			chrome_runtime.Enable(),
			chrome_log.Enable(),
			chromedp.Navigate(url),
			chromedp.WaitReady("body", chromedp.ByQuery),
		); err != nil {
			fmt.Printf("Failed to load url %s: %v", url, err)
			cancelBrowser()
			cancelAllocator()
			os.Exit(1)
		} else {
			fmt.Printf("Opened (%d/%d): %s\n", i+1, len(urls), url)
		}
	}

	fmt.Println("Chrome is running, waiting for error or interrupt...")

	// 4) Wait forever, or until an error or signal
	select {
	case <-sigChan:
		fmt.Println("\n⚠️  Plandex browser process interrupted")
		return nil
	case <-rootCtx.Done():
		fmt.Println("\n⚠️  Plandex browser process closed")
		return nil
	case <-errChan:
		fmt.Println("\n❌ Plandex browser process exited due to JavaScript error")
		cancelBrowser()
		cancelAllocator()
		os.Exit(1)
	}
	return nil
}

func openWithOSDefault(urls []string) error {
	var openCmd string
	switch runtime.GOOS {
	case "darwin":
		openCmd = "open"
	case "linux", "freebsd":
		openCmd = "xdg-open"
	default:
		return errors.New("unsupported OS for fallback")
	}

	for _, url := range urls {
		cmd := exec.Command(openCmd, url)
		if err := cmd.Start(); err != nil {
			log.Printf("Failed to open %s with %s: %v", url, openCmd, err)
		}
	}
	return nil
}
