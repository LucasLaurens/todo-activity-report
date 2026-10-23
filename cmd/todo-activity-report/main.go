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

var (
	errDateWithRange   = errors.New("use either --date or --from/--to, not both")
	errIncompleteRange = errors.New("provide both --from and --to (DD/MM/YYYY)")
	errUserWithAll     = errors.New("use either --user or --all, not both")
)

// helpText is the only description of the flags: the FlagSet's own usage lines
// are never printed, so the defaults are injected here from their constants.
var helpText = fmt.Sprintf(helpTemplate, defaultTaskLimit, cli.DetailLevelNames())

const helpTemplate = `todo-activity-report — ce qu'une personne a fait un jour donné : temps par ticket, avancement, cause, correction, synthèse IA.
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
  --limit        nb max de tâches détaillées (défaut: %d ; 0 = toutes)
  --detail       niveau de détail : %s (défaut: normal)
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
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// run takes stderr as a file, not a writer: the spinner only draws on a terminal.
func run(args []string, stdout io.Writer, stderr *os.File) error {
	parsed, err := parseFlags(args, stdout)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}

	// Answered before the config is loaded: asking a downloaded binary which
	// version it is must work without any tracker credential.
	if parsed.version {
		_, err := fmt.Fprintf(stdout, "todo-activity-report %s\n", version)
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
	query, err := buildQuery(parsed, period, cfg)
	if err != nil {
		return err
	}

	translator := cfg.NewTranslator()
	reporter, stop := newProgress(stderr, translator)
	defer stop()

	service, err := buildService(parsed, cfg, reporter, stderr)
	if err != nil {
		return err
	}
	report, err := service.Generate(context.Background(), query)
	if err != nil {
		return err
	}

	stop()
	return cli.NewRenderer(stdout, options, translator).Render(report)
}

// buildQuery settles which board and which people the report is about.
func buildQuery(parsed flags, period domain.DateRange, cfg config.Config) (domain.Query, error) {
	project := resolveProject(parsed.project, cfg.Project())
	if project == "" {
		return domain.Query{}, errNoProject
	}
	assignees, err := resolveAssignees(parsed, cfg.DefaultAssignee())
	if err != nil {
		return domain.Query{}, err
	}
	return domain.Query{
		Project:        project,
		Assignees:      assignees,
		Period:         period,
		IncludeTouched: parsed.touched,
	}, nil
}

func buildService(parsed flags, cfg config.Config, reporter ports.ProgressReporter, stderr io.Writer) (application.ReportService, error) {
	tracker, err := cfg.NewTracker()
	if err != nil {
		return application.ReportService{}, err
	}
	return application.NewReportService(
		tracker,
		selectUserResolver(parsed, tracker),
		selectSummarizer(parsed, cfg, stderr),
		reporter,
	), nil
}

// selectUserResolver lets --no-resolve send the raw --user terms to the
// tracker's query; otherwise the tracker resolves them itself.
func selectUserResolver(parsed flags, tracker ports.Tracker) ports.UserResolver {
	if parsed.noResolve {
		return application.PassthroughResolver{}
	}
	return tracker
}

func selectSummarizer(parsed flags, cfg config.Config, stderr io.Writer) ports.Summarizer {
	if parsed.noAI {
		return cfg.NewOfflineSummarizer()
	}
	return cfg.NewSummarizer(stderr)
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

// parseFlags leaves the help to the FlagSet: it already prints helpText on
// --help and on a parse error.
func parseFlags(args []string, out io.Writer) (flags, error) {
	var parsed flags
	if err := newFlagSet(&parsed, out).Parse(args); err != nil {
		return flags{}, err
	}
	return parsed, nil
}

// newFlagSet registers the flags without usage strings: helpText describes them.
func newFlagSet(parsed *flags, out io.Writer) *flag.FlagSet {
	set := flag.NewFlagSet("todo-activity-report", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.Usage = func() { fmt.Fprint(out, helpText) }

	set.StringVar(&parsed.project, "project", "", "")
	set.Var(&parsed.users, "user", "")
	set.BoolVar(&parsed.all, "all", false, "")
	set.StringVar(&parsed.date, "date", "", "")
	set.StringVar(&parsed.from, "from", "", "")
	set.StringVar(&parsed.to, "to", "", "")
	set.BoolVar(&parsed.touched, "touched", false, "")
	set.BoolVar(&parsed.noAI, "no-ai", false, "")
	set.BoolVar(&parsed.noResolve, "no-resolve", false, "")
	set.IntVar(&parsed.limit, "limit", defaultTaskLimit, "")
	set.StringVar(&parsed.detail, "detail", "", "")
	set.BoolVar(&parsed.version, "version", false, "")

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
		return domain.DateRange{}, errDateWithRange
	}
	if parsed.from == "" || parsed.to == "" {
		return domain.DateRange{}, errIncompleteRange
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
			return nil, errUserWithAll
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
