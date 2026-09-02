# todo-activity-report

CLI Go qui résume, pour un **jour** (ou une plage) et des **personnes** donnés,
tout ce qui a été **travaillé** sur un board — chaque ticket avec du temps
loggué, un changement de statut **ou** toute autre mise à jour (commentaire,
édition de champ) ce jour-là, **quel que soit son état**
(en cours, bloqué, release, terminé). Sortie **en français** (ou anglais), avec
un **résumé IA** automatique via le CLI `claude` local (sans clé API).

Le tracker est un **adapter interchangeable** : Jira est fourni, brancher un
autre outil ne touche ni le domaine ni le use case — voir
[docs/trackers.md](docs/trackers.md).

Architecture hexagonale, stdlib uniquement, zéro dépendance tierce.

## Démarrage rapide

Depuis les sources :

```bash
cp .env.example .env          # tracker + identifiants — auto-chargé, aucun export
go build -o todo-activity-report ./cmd/todo-activity-report

./todo-activity-report --date='08/06/2026' --user='john.doe@example.com'
```

Ou **sans cloner le dépôt** : le binaire des Releases est autonome (stdlib,
statiquement lié, locales embarquées), donc binaire + `.env` à côté suffisent —
voir [docs/install.md](docs/install.md).

## Exemples

```bash
# Mes tickets aujourd'hui
./todo-activity-report --date='08/06/2026' --user='john.doe@example.com'

# Tout le board sur un jour
./todo-activity-report --date='08/06/2026'

# Une plage, plusieurs personnes, toutes les lignes
./todo-activity-report --from='01/05/2026' --to='31/05/2026' \
  --user='John Doe' --user='Jane Smith' --limit=0

# En anglais, sans résumé IA
APP_LOCALE=en ./todo-activity-report --date='08/06/2026' --no-ai
```

## Aperçu de la sortie

```
ACME · Rapport d'activité
Période : 2026-06-08 → 2026-06-08   Personnes : john.doe@example.com

Synthèse
  Tickets traités            14
  Résolus                    0 (0%)
  Temps loggé                6h 55m
  Moyenne / ticket           29m
  Délai moyen de résolution  n/a

Avancement
  Bloqué       11
  Code Review  1
  En cours     1
  Pré-release  1

Tickets (14 sur 14)

  • ACME-8212  ·  Code Review  ·  2h 00m
    Toggle champ obligatoire non actionnable lors du premier paramétrage
    Cause: Pour un champ tout juste ajouté, l'application ne surveillait pas…

  • ACME-10424  ·  Bloqué  ·  2h 30m
    Catégories non ajoutées au parcours
  …

Synthèse IA
  Journée concentrée sur les plans d'action et la GED. 11 tickets bloqués…
```

## Flags

| Flag | Description |
|------|-------------|
| `--date` | un jour `JJ/MM/AAAA` (exclusif avec `--from`/`--to`) |
| `--from` / `--to` | plage `JJ/MM/AAAA` (fin incluse) |
| `--user` | nom, email ou identifiant — répétable / séparé par virgules (vide = tous) |
| `--project` | clé du projet / board (défaut : `TRACKER_PROJECT` du `.env`) |
| `--limit` | nb max de tickets détaillés (défaut 20 ; `0` = tous) |
| `--full` | textes complets (root cause / fix / résumés non tronqués) |
| `--brief` | chiffres + synthèse IA seulement (masque Causes racines et Tickets) |
| `--no-ai` | désactive le résumé IA |
| `--no-resolve` | passe les `--user` bruts au tracker (sans résolution d'identifiant) |
| `--version` | version du binaire (tag gravé au build, `dev` en local) |
| `--help` | aide complète |

Tracker choisi par `TRACKER` (`jira` par défaut) et board par `TRACKER_PROJECT`.
Avec l'adapter Jira, `JIRA_BASE_URL`, `JIRA_EMAIL` et `JIRA_API_TOKEN` sont
requis — voir [docs/jira-setup.md](docs/jira-setup.md). Langue via `APP_LOCALE`
(`fr` défaut, `en`).

## Documentation

| Doc | Contenu |
|-----|---------|
| [docs/install.md](docs/install.md) | **installer** : binaire seul (Release) ou sources, empreintes, cross-compile |
| [docs/usage.md](docs/usage.md) | flags, **exemples**, format de sortie, couleurs, métriques |
| [docs/configuration.md](docs/configuration.md) | variables d'env, langue (i18n), secrets, pont env → app |
| [docs/jira-setup.md](docs/jira-setup.md) | "creds" Jira, token, accountId, champs custom |
| [docs/trackers.md](docs/trackers.md) | **brancher un autre tracker** : le port `Tracker`, l'adapter, le registre |
| [docs/architecture.md](docs/architecture.md) | couches hexagonales, flux d'exécution, DIP |
| [docs/conventions.md](docs/conventions.md) | SOLID, patterns, perf, sécurité — où chaque règle s'applique |
| [docs/testing.md](docs/testing.md) | stratégie de test (AAA, pas de mocks, `t.Parallel`) |
| [docs/ci.md](docs/ci.md) | pipeline GitHub Actions (quality en parallèle, build / release sur tag) |
| [docs/status.md](docs/status.md) | **état courant** (livré / limites / dépendances) |

Conventions normatives : [docs/conventions.md](docs/conventions.md).

## Structure

```
cmd/todo-activity-report          point d'entrée, flags, câblage
internal/domain              entités + value objects (Task, TimeSpent, Optional, Metrics)
internal/application         ReportService + MetricsCalculator + PassthroughResolver
internal/application/ports   interfaces Tracker, TaskProvider, UserResolver, Summarizer, ProgressReporter
internal/architecture        test de frontières hexagonales (aucun code de prod)
internal/i18n                traductions (fr/en, fr par défaut)
internal/infrastructure/jira adapter Jira — une implémentation de ports.Tracker
internal/infrastructure/llm  adapters IA (CLI claude, API Anthropic, no-op)
internal/infrastructure/cli  rendu terminal (couleurs par statut) + loader
internal/config              env → config typée → registre des trackers + fabriques
internal/testsupport         fakes in-memory (tests)
scripts/                     export quotidien du brief dans l'app Notes (macOS)
docs/                        documentation détaillée
docs/images/                 captures d'écran utilisées par la doc
```
