package main

import (
	"context"
	"errors"
	"flag"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"lyricfetcher/internal/cache"
	"lyricfetcher/internal/console"
	"lyricfetcher/internal/gui"
	"lyricfetcher/internal/i18n"
	"lyricfetcher/internal/metadata"
	"lyricfetcher/internal/provider"
	"lyricfetcher/internal/worker"
)

var (
	version = "1.0.0"
	commit  = "dev"
	date    = "2026-10-25"
)

var supportedExtensions = map[string]bool{
	".mp3":  true,
	".flac": true,
	".m4a":  true,
}

func isSupportedAudio(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return supportedExtensions[ext]
}

func printHelp() {
	console.Println(i18n.T("help_app_desc"))
	console.Println()
	console.Println(i18n.T("help_usage"))
	console.Println()
	console.Println(i18n.T("help_group_general"))
	console.Println("  --lang string             " + i18n.T("help_flag_lang"))
	console.Println("  --version                 " + i18n.T("help_flag_version"))
	console.Println("  --gui                     " + i18n.T("help_flag_gui"))
	console.Println("  --debug                   " + i18n.T("help_flag_debug"))
	console.Println("  --help                    " + i18n.T("help_flag_help"))
	console.Println()
	console.Println(i18n.T("help_group_processing"))
	console.Println("  --workers int             " + i18n.T("help_flag_workers"))
	console.Println("  --strict                  " + i18n.T("help_flag_strict"))
	console.Println("  --no-netease              " + i18n.T("help_flag_no_netease"))
	console.Println("  --netease-spoof-ip        " + i18n.T("help_flag_netease_spoof_ip"))
	console.Println("  --netease-spoof-ip-value  " + i18n.T("help_flag_netease_spoof_val"))
	console.Println()
	console.Println(i18n.T("help_group_output"))
	console.Println("  --output-dir string       " + i18n.T("help_flag_output_dir"))
	console.Println("  --overwrite               " + i18n.T("help_flag_overwrite"))
	console.Println("  --dry-run                 " + i18n.T("help_flag_dry_run"))
	console.Println("  --log-file string         " + i18n.T("help_flag_log_file"))
	console.Println()
	console.Println(i18n.T("help_group_cache"))
	console.Println("  --no-cache                " + i18n.T("help_flag_no_cache"))
	console.Println("  --clear-cache             " + i18n.T("help_flag_clear_cache"))
	console.Println()
	console.Println(i18n.T("help_group_examples"))
	console.Println("  lyricfetcher \"song.mp3\"")
	console.Println("      " + i18n.T("help_example_1_desc"))
	console.Println("  lyricfetcher \"/path/to/Music\"")
	console.Println("      " + i18n.T("help_example_2_desc"))
	console.Println("  lyricfetcher --workers 8 --overwrite \"/path/to/Music\"")
	console.Println("      " + i18n.T("help_example_3_desc"))
	console.Println("  lyricfetcher --output-dir \"/path/to/lyrics\" \"/path/to/Music\"")
	console.Println("      " + i18n.T("help_example_4_desc"))
	console.Println("  lyricfetcher --dry-run --strict \"song.flac\"")
	console.Println("      " + i18n.T("help_example_5_desc"))
}

func main() {
	// Pre-scan for --lang flag so help or errors render in the chosen language
	selectedLang := "auto"
	for i, arg := range os.Args[1:] {
		if arg == "--lang" && i+1 < len(os.Args[1:]) {
			selectedLang = os.Args[1:][i+1]
		} else if strings.HasPrefix(arg, "--lang=") {
			selectedLang = strings.TrimPrefix(arg, "--lang=")
		}
	}
	if err := i18n.Init(selectedLang); err != nil {
		console.Eprintf("runtime error: %v\n", err)
		os.Exit(1)
	}

	fsFlags := flag.NewFlagSet("lyricfetcher", flag.ContinueOnError)
	fsFlags.Usage = func() {
		printHelp()
	}

	langFlag := fsFlags.String("lang", "auto", i18n.T("help_flag_lang"))
	versionFlag := fsFlags.Bool("version", false, i18n.T("help_flag_version"))
	guiFlag := fsFlags.Bool("gui", false, i18n.T("help_flag_gui"))
	debugFlag := fsFlags.Bool("debug", false, i18n.T("help_flag_debug"))
	workersFlag := fsFlags.Int("workers", 4, i18n.T("help_flag_workers"))
	dryRunFlag := fsFlags.Bool("dry-run", false, i18n.T("help_flag_dry_run"))
	outputDirFlag := fsFlags.String("output-dir", "", i18n.T("help_flag_output_dir"))
	overwriteFlag := fsFlags.Bool("overwrite", false, i18n.T("help_flag_overwrite"))
	strictFlag := fsFlags.Bool("strict", false, i18n.T("help_flag_strict"))
	noNeteaseFlag := fsFlags.Bool("no-netease", false, i18n.T("help_flag_no_netease"))
	neteaseSpoofIPFlag := fsFlags.Bool("netease-spoof-ip", false, i18n.T("help_flag_netease_spoof_ip"))
	neteaseSpoofValFlag := fsFlags.String("netease-spoof-ip-value", "118.89.204.198", i18n.T("help_flag_netease_spoof_val"))
	noCacheFlag := fsFlags.Bool("no-cache", false, i18n.T("help_flag_no_cache"))
	clearCacheFlag := fsFlags.Bool("clear-cache", false, i18n.T("help_flag_clear_cache"))
	logFileFlag := fsFlags.String("log-file", "lyricfetcher.log", i18n.T("help_flag_log_file"))

	if err := fsFlags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		console.Eprintf("\n%s\n", i18n.T("help_hint"))
		os.Exit(2)
	}

	// Handle --version flag
	if *versionFlag {
		console.Printf("lyricfetcher %s (%s, %s)\n", version, commit, date)
		os.Exit(0)
	}

	if *debugFlag {
		metadata.DebugMode = true
	}

	// Re-init i18n if user passed explicit --lang flag
	if *langFlag != "auto" && *langFlag != selectedLang {
		if err := i18n.Init(*langFlag); err != nil {
			console.Eprintf("runtime error: %v\n", err)
			os.Exit(1)
		}
	}

	args := fsFlags.Args()
	if *guiFlag || (len(args) == 0 && !*clearCacheFlag && gui.HasGUI()) {
		hideConsoleIfExplorer()
		gui.Run()
		return
	}

	if len(args) == 0 && !*clearCacheFlag {
		console.Println(i18n.T("usage"))
		console.Println(i18n.T("help_hint"))
		os.Exit(2)
	}

	// Initialize Cache
	var c *cache.Cache
	if !*noCacheFlag {
		var err error
		c, err = cache.New("")
		if err != nil {
			console.Printf("%s %s\n", i18n.T("warn_prefix"), err)
		} else if *clearCacheFlag {
			if err := c.Clear(); err != nil {
				console.Printf("%s %s\n", i18n.T("warn_prefix"), err)
			}
		}
	}

	if len(args) == 0 && *clearCacheFlag {
		os.Exit(0)
	}

	// Collect audio files
	var files []string
	for _, arg := range args {
		stat, err := os.Stat(arg)
		if err != nil {
			console.Printf("%s %s: %s\n", i18n.T("warn_prefix"), filepath.ToSlash(arg), i18n.Tf("err_open_file", err))
			continue
		}

		if stat.IsDir() {
			err := filepath.WalkDir(arg, func(path string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					console.Printf("%s %s: %s\n", i18n.T("warn_prefix"), filepath.ToSlash(path), i18n.Tf("err_walk_dir", walkErr))
					return nil
				}
				if !d.IsDir() && isSupportedAudio(path) {
					files = append(files, path)
				}
				return nil
			})
			if err != nil {
				console.Printf("%s %s: %s\n", i18n.T("warn_prefix"), filepath.ToSlash(arg), i18n.Tf("err_walk_dir", err))
			}
		} else {
			if !isSupportedAudio(arg) {
				console.Printf("%s %s: %s\n", i18n.T("warn_prefix"), filepath.ToSlash(arg), i18n.T("err_unsupported"))
				continue
			}
			files = append(files, arg)
		}
	}

	if len(files) == 0 {
		os.Exit(0)
	}

	lrclibProvider := provider.NewLRCLIBProvider()
	neteaseProvider := provider.NewNetEaseProvider(provider.NetEaseOptions{
		SpoofIP:      *neteaseSpoofIPFlag,
		SpoofIPValue: *neteaseSpoofValFlag,
	})

	workers := *workersFlag
	if workers < 1 {
		workers = 4
	}
	if workers > 16 {
		workers = 16
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	opts := worker.Options{
		Workers:         workers,
		LrclibProvider:  lrclibProvider,
		NeteaseProvider: neteaseProvider,
		NoNetease:       *noNeteaseFlag,
		Strict:          *strictFlag,
		DryRun:          *dryRunFlag,
		OutputDir:       *outputDirFlag,
		Overwrite:       *overwriteFlag,
		NoCache:         *noCacheFlag,
		Cache:           c,
		LogFile:         *logFileFlag,
	}

	summary := worker.Run(ctx, files, opts)
	summary.Print()

	if ctx.Err() != nil {
		os.Exit(130)
	}

	os.Exit(0)
}
