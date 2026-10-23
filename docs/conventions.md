# Conventions & bonnes pratiques

[← Retour au README](../README.md)

Source de vérité normative du projet : les règles ci-dessous s'appliquent à
tout changement, et ce fichier montre **où** chacune se matérialise dans le code.
Les règles de test ont leur propre page ([testing.md](testing.md)), les commandes
à passer avant commit sont dans [ci.md](ci.md).

## SOLID

| Principe | Application |
|----------|-------------|
| **S**RP | `MetricsCalculator` calcule, `Renderer` affiche, `Client` parle à Jira — une raison de changer chacun |
| **O**CP | nouveau tracker = nouvel adapter + une entrée dans `trackerRegistry` ; ni le domaine ni le use case ne changent |
| **L**SP | `NoopSummarizer`/`ClaudeSummarizer` et `PassthroughResolver`/tracker interchangeables derrière leur port |
| **I**SP | ports minimaux : `TaskProvider`, `UserResolver`, `Summarizer`, `ProgressReporter` — une méthode chacun ; `Tracker` ne fait que composer les deux premiers |
| **D**IP | `ReportService` dépend des interfaces `ports.*`, jamais des structs concrets |

## Design patterns

- **Adapter** — `jira.Client`, `llm.ClaudeSummarizer` adaptent un service externe à un port.
- **Strategy** — narratif via `ClaudeCLISummarizer` / `ClaudeSummarizer` / `NoopSummarizer`, choisi à l'exécution.
- **Registry** — `config.trackerRegistry` : `TRACKER` → `trackerEntry` (builder d'adapter + variable du compte par défaut). Le seul endroit qui nomme un tracker concret.
- **Factory / Composition root** — `config.Config.NewTracker()` / `NewSummarizer()` / `NewOfflineSummarizer()`.
- **Null Object** — `NoopSummarizer`, `NoopProgress`, `application.PassthroughResolver` évitent les dépendances nil.
- **Repository** — `TaskProvider` abstrait la source de données.
- **Builder** — `testsupport.TaskBuilder`.
- **Value Object** — `TimeSpent`, `DateRange`, `Optional[T]` immuables.

## Data structures & algorithmes

- Agrégations en O(n) via maps (`groupByAssignee`, `countLabels`).
- `sortLabelCounts` : tri par rang (attention pour les statuts, égal sinon), puis
  count décroissant, **label en tie-break** → sortie déterministe (donc
  testable). Les rangs sont calculés une fois par label / ticket, jamais à
  chaque comparaison (`sortByAttention` aussi).
- `Optional[T]` générique : pas de sentinelle magique, pas de nil.

## Pas de valeurs magiques

Pas de chaîne ni d'entier en dur dès qu'une valeur porte un sens ou est
réutilisée : constante nommée ou type dédié. `TrackerName` plutôt qu'une chaîne
libre, `httpTimeout`/`maxWorklogConcurrency`/`pageSize` plutôt que des littéraux,
la clé du board lue dans `TRACKER_PROJECT` plutôt qu'écrite dans le code.

## Loi de Demeter

Le domaine expose le comportement plutôt que ses entrailles :
`task.IsResolved()`, `task.ResolutionDuration()`, `metrics.ResolutionRate()`.
Pas de chaîne `a.b().c().d()` qui traverse plusieurs types.

## Non-null

`Optional[T]` partout où une valeur peut manquer (`ResolvedAt`, `MTTR`,
`Narrative`). Les pointeurs nil ne franchissent jamais la frontière du domaine —
ils sont convertis en `Optional` dès le mapper Jira. Dépendances jamais nil :
`NoopSummarizer` remplace un `nil`.

## Performance

- Le temps total est sommé **une seule fois** par `Calculate`, puis réutilisé
  pour la moyenne (pas de double parcours).
- La liste de champs Jira est calculée **une fois** à la construction du client
  (`buildRequestedFields`), pas à chaque page de pagination.
- Slices alloués avec capacité connue (`make(..., 0, len(...))`).
- Complexité dominante O(n) sur les tâches ; les agrégations sont des passes
  linéaires sur maps.
- Le worklog par ticket (N+1 appels) est récupéré avec un **pool concurrent
  borné** (`maxWorklogConcurrency = 8`) : chaque goroutine écrit un index
  distinct du slice (pas de mutation partagée), l'ordre est préservé et la
  première erreur annule le contexte. Validé sans data race (`go test -race`).
  Borne = compromis vitesse / pression sur l'API Jira (rate-limit).

## Sécurité

- Secrets en `.env` (hors git), lus via `internal/config`, jamais loggués.
- Chaque adapter **valide sa propre config** : `jira.Config.Validate()` exige
  `https://` pour `JIRA_BASE_URL` (`jira.ErrInsecureBaseURL`), le token ne part
  jamais en clair. La règle vit avec l'adapter qui la subit, pas dans le
  composition root.
- Clients HTTP avec timeout (30s) — pas de pendaison.
- Toute entrée interpolée dans JQL est échappée (`quote`) ; les paramètres d'URL
  passent par `url.Values.Encode` — pas d'injection.
- La réponse du LLM est **affichée**, jamais exécutée.
- Le contenu des tickets est une **entrée non fiable** pour le modèle : le CLI
  `claude` tourne sans outil, sans MCP, sans réglage projet, dans un dossier
  vide — un commentaire piégé ne peut rien déclencher sur le poste
  (voir [configuration.md](configuration.md#isolation-du-cli-claude)).
- Le résumé IA envoie titres, causes et commentaires à Anthropic dès que le CLI
  `claude` ou `ANTHROPIC_API_KEY` est disponible ; `--no-ai` le coupe
  (voir [configuration.md](configuration.md#confidentialité-du-résumé-ia)).
- Toolchain Go **patché** : `toolchain` de `go.mod` = `GO_VERSION` de la CI
  (un test les garde synchrones), `govulncheck ./...` doit rester vide.

## Commentaires

Pas de commentaires qui paraphrasent le code. On commente seulement le *pourquoi*
non évident (ex. tie-break déterministe, support ADF, choix du no-op vs nil).

## Frontières encodées, pas seulement documentées

`internal/architecture` ne contient aucun code de production : c'est un test qui
parse les imports de `internal/domain` et `internal/application` et échoue si
l'un d'eux atteint `infrastructure`, `config` ou `i18n`. La règle d'architecture
casse donc la CI au lieu d'attendre une relecture.

## Tests

Voir [testing.md](testing.md) : AAA + pas de mocks + Faker-style via builder.

## Documentation toujours à jour

Toute modification de comportement, flag, variable d'env ou couche met à jour
le doc correspondant **dans le même changement**. Voir [status.md](status.md)
pour l'état courant de l'application.
