package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/application"
	"github.com/LucasLaurens/todo-activity-report/internal/application/ports"
	"github.com/LucasLaurens/todo-activity-report/internal/config"
	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/cli"
)

// version is stamped at build time by the release job:
//
//	go build -ldflags="-X main.version=$GITHUB_REF_NAME"
//
// A binary built without it reports the placeholder.
var version = "dev"

const (
	defaultTaskLimit = 20
	dateLayoutFR     = "02/01/2006"
)

// errNoProject asks for the board key rather than defaulting to one team's
// project: the tool is tracker- and organisation-agnostic.
var errNoProject = errors.New("provide --project, or set TRACKER_PROJECT in .env")

const helpText = `todo-activity-report — résumé des tickets travaillés sur un tracker, quel que soit leur état (root cause, fix, temps, métriques).

Usage:
  todo-activity-report --date='JJ/MM/AAAA' [--user='…'] [options]
  todo-activity-report --from='JJ/MM/AAAA' --to='JJ/MM/AAAA' [--user='…'] [options]

Options:
  --date         un seul jour (JJ/MM/AAAA) ; exclusif avec --from/--to
  --from         début de plage inclus (JJ/MM/AAAA)
  --to           fin de plage incluse (JJ/MM/AAAA, jusqu'à 23:59)
  --user         nom, email ou identifiant — répétable / séparé par virgules (vide = tous)
  --project      clé du projet / board (défaut: TRACKER_PROJECT du .env)
  --limit        nb max de tâches détaillées (défaut: 20 ; 0 = toutes)
  --full         affiche les textes complets (root cause / fix / résumés non tronqués)
  --brief        n'affiche que les chiffres + la synthèse IA (masque Causes racines et Tickets)
  --no-ai        désactive le résumé IA
  --no-resolve   envoie les --user bruts au tracker (sans résolution d'identifiant)
  --version      affiche la version du binaire
  --help         affiche cette aide

Exemples:
  todo-activity-report --date='08/06/2026' --user='john.doe@example.com'
  todo-activity-report --from='01/05/2026' --to='31/05/2026' --user='John Doe' --limit=0
  todo-activity-report --date='08/06/2026'                       # tout le board ce jour-là

Config: tracker (TRACKER, défaut jira) et identifiants dans .env (auto-chargé).
Voir docs/configuration.md.
`

// stringSlice collects a repeatable, comma-separable flag (e.g. --user).
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ",") }

func (s *stringSlice) Set(raw string) error {
	for _, part := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			*s = append(*s, trimmed)
		}
	}
	return nil
}

type flags struct {
	version   bool
	project   string
	users     stringSlice
	date      string
	from      string
	to        string
	noAI      bool
	noResolve bool
	limit     int
	full      bool
	brief     bool
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, out *os.File) error {
	parsed, err := parseFlags(args, out)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	// Answered before the config is loaded: asking a downloaded binary which
	// version it is must work without any tracker credential.
	if parsed.version {
		_, err := fmt.Fprintf(out, "todo-activity-report %s\n", version)
		return err
	}

	period, err := resolvePeriod(parsed)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	project := resolveProject(parsed.project, cfg.Project())
	if project == "" {
		return errNoProject
	}

	translator := cfg.NewTranslator()
	reporter, stop := newProgress(os.Stderr, translator)
	defer stop()

	tracker, err := cfg.NewTracker()
	if err != nil {
		return err
	}
	service := application.NewReportService(
		tracker,
		cfg.NewUserResolver(tracker, parsed.noResolve),
		cfg.NewSummarizer(parsed.noAI),
		reporter,
	)

	query := domain.Query{
		Project:   project,
		Assignees: parsed.users,
		Period:    period,
	}

	report, err := service.Generate(context.Background(), query)
	if err != nil {
		return err
	}

	stop()
	options := cli.RenderOptions{TaskLimit: parsed.limit, Full: parsed.full, Brief: parsed.brief}
	return cli.NewRenderer(out, options, translator).Render(report)
}

// newProgress returns a spinner when stderr is a terminal, otherwise a no-op.
// The returned stop function is idempotent and clears the spinner line.
func newProgress(stderr *os.File, translator i18n.Translator) (ports.ProgressReporter, func()) {
	if !isTerminal(stderr) {
		return cli.NoopProgress{}, func() {}
	}
	spinner := cli.NewSpinner(stderr, translator)
	spinner.Start()
	return spinner, spinner.Stop
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) != 0
}

func parseFlags(args []string, out io.Writer) (flags, error) {
	var parsed flags
	set := newFlagSet(&parsed, out)

	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return flags{}, err
		}
		set.Usage()
		return flags{}, err
	}
	return parsed, nil
}

func newFlagSet(parsed *flags, out io.Writer) *flag.FlagSet {
	set := flag.NewFlagSet("todo-activity-report", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.Usage = func() { fmt.Fprint(out, helpText) }

	set.StringVar(&parsed.project, "project", "", "project / board key (default: TRACKER_PROJECT)")
	set.Var(&parsed.users, "user", "assignee name, email or accountId (repeatable, comma-separated)")
	set.StringVar(&parsed.date, "date", "", "single day, DD/MM/YYYY")
	set.StringVar(&parsed.from, "from", "", "range start, inclusive, DD/MM/YYYY")
	set.StringVar(&parsed.to, "to", "", "range end, inclusive, DD/MM/YYYY")
	set.BoolVar(&parsed.noAI, "no-ai", false, "disable the AI narrative even if a key is set")
	set.BoolVar(&parsed.noResolve, "no-resolve", false, "send --user terms to the tracker without identifier resolution")
	set.IntVar(&parsed.limit, "limit", defaultTaskLimit, "max tasks listed in detail (0 = all)")
	set.BoolVar(&parsed.full, "full", false, "show full root cause / fix / summaries (no truncation)")
	set.BoolVar(&parsed.brief, "brief", false, "show only the metrics and the AI summary (hide root causes and ticket list)")
	set.BoolVar(&parsed.version, "version", false, "print the binary version and exit")

	return set
}

// resolveProject prefers the flag over the configured default, so a one-off run
// can target another board without editing .env.
func resolveProject(flagValue, configured string) string {
	if flagValue != "" {
		return flagValue
	}
	return configured
}

// resolvePeriod accepts either a single --date or a --from/--to range.
func resolvePeriod(parsed flags) (domain.DateRange, error) {
	if parsed.date != "" {
		if parsed.from != "" || parsed.to != "" {
			return domain.DateRange{}, fmt.Errorf("use either --date or --from/--to, not both")
		}
		day, err := parseDateFR(parsed.date)
		if err != nil {
			return domain.DateRange{}, err
		}
		return domain.NewDateRange(day, day)
	}

	if parsed.from == "" || parsed.to == "" {
		return domain.DateRange{}, fmt.Errorf("provide --date, or both --from and --to (DD/MM/YYYY)")
	}
	from, err := parseDateFR(parsed.from)
	if err != nil {
		return domain.DateRange{}, err
	}
	to, err := parseDateFR(parsed.to)
	if err != nil {
		return domain.DateRange{}, err
	}
	return domain.NewDateRange(from, to)
}

func parseDateFR(value string) (time.Time, error) {
	parsed, err := time.Parse(dateLayoutFR, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q, expected DD/MM/YYYY", value)
	}
	return parsed, nil
}
