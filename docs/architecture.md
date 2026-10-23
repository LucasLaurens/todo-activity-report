# Architecture

[← Retour au README](../README.md)

## Vue d'ensemble

Architecture hexagonale (ports & adapters). Les dépendances pointent **vers le
domaine** : le domaine ne connaît personne, l'application connaît le domaine,
l'infrastructure connaît l'application via des interfaces (ports).

```
┌───────────────────────────────────────────────────────────┐
│ cmd/todo-activity-report (composition root + flags CLI)   │
└───────────────┬───────────────────────────────┬───────────┘
                │                               │
        ┌───────▼─────────┐            ┌────────▼──────────┐
        │ internal/config │            │ infrastructure/   │
        │ (env → app,     │── wire ──▶ │ jira, llm, cli    │
        │  registre       │            │ (1 dossier =      │
        │  TRACKER)       │            │  1 adapter)       │
        └─────────────────┘            └────────┬──────────┘
                                                │ implémentent
                                       ┌────────▼─────────┐
                                       │ application/ports │
                                       │ Tracker           │
                                       │  = TaskProvider   │
                                       │  + UserResolver   │
                                       │ Summarizer        │
                                       │ ProgressReporter  │
                                       └────────┬─────────┘
                                                │ utilisés par
                                       ┌────────▼─────────┐
                                       │ application       │
                                       │ ReportService     │
                                       │ MetricsCalculator │
                                       └────────┬─────────┘
                                                │ opèrent sur
                                       ┌────────▼─────────┐
                                       │ domain            │
                                       │ Task, TimeSpent,  │
                                       │ Optional, Metrics │
                                       └──────────────────┘
```

## Couches

| Paquet | Rôle | Dépendances |
|--------|------|-------------|
| `internal/domain` | Entités + value objects. Logique métier pure. | aucune |
| `internal/application` | Use case `ReportService`, service pur `MetricsCalculator`, `PassthroughResolver` (null object). | domain, ports |
| `internal/application/ports` | Interfaces `TaskProvider`, `UserResolver`, `Tracker`, `Summarizer`, `ProgressReporter`. | domain |
| `internal/architecture` | Aucun code : un test qui échoue si le domaine ou l'application importent une couche externe. | — (tests) |
| `internal/i18n` | Traductions fr/en (JSON embarqués), `Translator`. | aucune |
| `internal/infrastructure/jira` | Adapter Jira — **une implémentation de `ports.Tracker` parmi d'autres** (HTTP + JQL + pagination + mapping + résolution d'utilisateurs + validation de sa propre config). | domain |
| `internal/infrastructure/llm` | `ClaudeCLISummarizer` (CLI local) + `ClaudeSummarizer` (API) + `NoopSummarizer`. | domain |
| `internal/infrastructure/sanitize` | `Printable` : retire séquences ANSI et caractères de contrôle de tout texte venu de l'extérieur (tracker, réponse du modèle). | aucune |
| `internal/infrastructure/cli` | Rendu terminal (`tabwriter`, couleurs ANSI d'après `domain.StatusKind`) + `Spinner`/`NoopProgress` (loader). | domain |
| `internal/config` | Lecture env → config typée → fabriques d'adapters. | tout |
| `internal/testsupport` | Implémentations in-memory concrètes (tests). | domain |

## Flux d'une exécution

1. `cmd/todo-activity-report` parse les flags (`--date` ou `--from`/`--to` en
   `JJ/MM/AAAA`, `--user`), construit un `domain.Query`.
2. `config.Load()` lit l'env et vérifie que `TRACKER` désigne un adapter enregistré.
3. `config.NewTracker()` / `NewSummarizer()` (ou `NewOfflineSummarizer()` avec
   `--no-ai`) fabriquent les adapters ; `main` choisit le résolveur (le tracker,
   ou `application.PassthroughResolver` avec `--no-resolve`). Le tracker retourné est un `ports.Tracker` : il sert à la fois de
   `TaskProvider` et de `UserResolver`, et valide lui-même ses identifiants.
4. `ReportService.Generate()`, en émettant chaque étape via `ProgressReporter`
   (un `Spinner` sur stderr en mode terminal) :
   - `resolver.Resolve()` → chaque `--user` (nom/email) → identifiant du tracker
   - `provider.FetchTasks()` → `[]domain.Task`
   - `MetricsCalculator.Calculate()` → `domain.Metrics`
   - `summarizer.Summarize()` → narratif optionnel
5. le spinner s'arrête, puis `cli.Renderer.Render()` imprime le rapport sur stdout.

## Pourquoi l'inversion de dépendance

`ReportService` ne dépend que des ports. On peut donc :

- Remplacer Jira par Linear, GitHub Projects, un CSV ou une base sans toucher au
  use case : un dossier d'adapter + une entrée dans le registre — voir
  [trackers.md](trackers.md).
- Tester avec des fakes in-memory (voir [testing.md](testing.md)) — zéro mock.
- Activer/désactiver l'IA en injectant `ClaudeSummarizer` ou `NoopSummarizer`.

Cette frontière n'est pas qu'une convention : `internal/architecture` la vérifie
à chaque `go test`. Un import de `internal/infrastructure/...` depuis `domain` ou
`application` fait échouer la suite.

Voir [conventions.md](conventions.md) pour les patterns appliqués.
