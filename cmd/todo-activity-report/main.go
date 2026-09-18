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

const defaultTaskLimit = 20

// errNoProject asks for the board key rather than defaulting to one team's
// project: the tool is tracker- and organisation-agnostic.
var errNoProject = errors.New("provide --project, or set TRACKER_PROJECT in .env")

const helpText = `todo-activity-report — ce qu'une personne a fait un jour donné : temps par ticket, avancement, cause, correction, synthèse IA.
Périmètre par défaut : temps loggué OU changement de statut (--touched élargit à toute mise à jour).

Usage:
  todo-activity-report                              # moi, dernier jour ouvré
  todo-activity-report --date='JJ/MM/AAAA' [--user='…'] [options]
  todo-activity-report --from='JJ/MM/AAAA' --to='JJ/MM/AAAA' [--user='…'] [options]

Options:
  --date         jour à couvrir : JJ/MM/AAAA, 'hier' ou 'today' (défaut: dernier
                 jour ouvré — lundi remonte à vendredi) ; exclusif avec --from/--to
  --from         début de plage inclus (JJ/MM/AAAA, 'hier' accepté)
  --to           fin de plage incluse (JJ/MM/AAAA, jusqu'à 23:59)
  --user         nom, email ou identifiant — répétable / séparé par virgules
                 (défaut: TRACKER_USER, sinon le compte du tracker)
  --all          tout le board au lieu d'une seule personne ; exclusif avec --user
  --project      clé du projet / board (défaut: TRACKER_PROJECT du .env)
  --touched      inclut aussi les tickets simplement mis à jour ce jour-là
                 (commentaire, édition de champ, changement en masse)
  --limit        nb max de tâches détaillées (défaut: 20 ; 0 = toutes)
  --detail       niveau de détail : brief|normal|full (défaut: normal)
                 brief = chiffres + synthèse IA ; full = textes non tronqués
  --no-ai        désactive le résumé IA
  --no-resolve   envoie les --user bruts au tracker (sans résolution d'identifiant)
  --version      affiche la version du binaire
  --help         affiche cette aide

Exemples:
  todo-activity-report                                      # ma veille ouvrée
  todo-activity-report --date='hier' --detail=brief         # brief oral d'hier
  todo-activity-report --date='08/06/2026' --user='john.doe@example.com'
  todo-activity-report --from='01/05/2026' --to='31/05/2026' --user='John Doe' --limit=0
  todo-activity-report --date='08/06/2026' --all            # tout le board ce jour-là

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
	all       bool
	date      string
	from      string
	to        string
	touched   bool
	noAI      bool
	noResolve bool
	limit     int
	detail    string
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

	period, err := resolvePeriod(parsed, time.Now())
	if err != nil {
		return err
	}

	options, err := resolveRenderOptions(parsed)
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

	assignees, err := resolveAssignees(parsed, cfg.DefaultAssignee())
	if err != nil {
		return err
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
		Project:        project,
		Assignees:      assignees,
		Period:         period,
		IncludeTouched: parsed.touched,
	}

	report, err := service.Generate(context.Background(), query)
	if err != nil {
		return err
	}

	stop()
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
	set.Var(&parsed.users, "user", "assignee name, email or identifier (repeatable, comma-separated; defaults to TRACKER_USER)")
	set.BoolVar(&parsed.all, "all", false, "report on the whole board instead of a single person")
	set.StringVar(&parsed.date, "date", "", "single day: DD/MM/YYYY, 'hier' or 'today' (default: last working day)")
	set.StringVar(&parsed.from, "from", "", "range start, inclusive, DD/MM/YYYY")
	set.StringVar(&parsed.to, "to", "", "range end, inclusive, DD/MM/YYYY")
	set.BoolVar(&parsed.touched, "touched", false, "also include tickets merely updated in the period (comment, field edit)")
	set.BoolVar(&parsed.noAI, "no-ai", false, "disable the AI narrative even if a key is set")
	set.BoolVar(&parsed.noResolve, "no-resolve", false, "send --user terms to the tracker without identifier resolution")
	set.IntVar(&parsed.limit, "limit", defaultTaskLimit, "max tasks listed in detail (0 = all)")
	set.StringVar(&parsed.detail, "detail", "", "how much to print: "+cli.DetailLevelNames())
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

// resolvePeriod accepts a single --date, a --from/--to range, or nothing at all
// — in which case the report covers the last working day, the question asked
// nine times out of ten ("what did I do yesterday?").
func resolvePeriod(parsed flags, now time.Time) (domain.DateRange, error) {
	if parsed.from == "" && parsed.to == "" {
		day, err := domain.ParseDay(parsed.date, now)
		if err != nil {
			return domain.DateRange{}, err
		}
		return domain.NewDateRange(day, day)
	}

	if parsed.date != "" {
		return domain.DateRange{}, fmt.Errorf("use either --date or --from/--to, not both")
	}
	if parsed.from == "" || parsed.to == "" {
		return domain.DateRange{}, fmt.Errorf("provide both --from and --to (DD/MM/YYYY)")
	}
	from, err := domain.ParseDay(parsed.from, now)
	if err != nil {
		return domain.DateRange{}, err
	}
	to, err := domain.ParseDay(parsed.to, now)
	if err != nil {
		return domain.DateRange{}, err
	}
	return domain.NewDateRange(from, to)
}

// resolveAssignees targets one person by default — the configured account — so
// a bare run answers "what did I do?". --user names someone else, --all widens
// the report to the whole board. Without a configured account, the report stays
// board-wide, as before.
func resolveAssignees(parsed flags, configured string) ([]string, error) {
	if parsed.all {
		if len(parsed.users) > 0 {
			return nil, fmt.Errorf("use either --user or --all, not both")
		}
		return nil, nil
	}
	if len(parsed.users) > 0 {
		return parsed.users, nil
	}
	if configured == "" {
		return nil, nil
	}
	return []string{configured}, nil
}

func resolveRenderOptions(parsed flags) (cli.RenderOptions, error) {
	detail, err := cli.ParseDetailLevel(parsed.detail)
	if err != nil {
		return cli.RenderOptions{}, err
	}
	return cli.RenderOptions{TaskLimit: parsed.limit, Detail: detail}, nil
}
