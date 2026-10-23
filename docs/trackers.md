# Brancher un autre tracker

[← Retour au README](../README.md)

Le tracker est un **détail d'infrastructure**. Le domaine (`internal/domain`) et
l'application (`internal/application`) ne connaissent que des interfaces : passer
de Jira à Linear, GitHub Projects, Azure Boards ou un export CSV ne touche ni
l'un ni l'autre.

## Le contrat

```go
// internal/application/ports/tracker.go
type Tracker interface {
    TaskProvider // FetchTasks(ctx, domain.Query) ([]domain.Task, error)
    UserResolver // Resolve(ctx, term string) (string, error)
}
```

`FetchTasks` rend des `domain.Task` : clé, résumé, assigné, type, priorité,
statut (chaîne brute — c'est le domaine qui la classe via `ClassifyStatus`),
root cause, correction, dates, temps passé, commentaires. `Resolve` traduit un
terme lisible (« Ada Lovelace », un email) en l'identifiant que le tracker
attend dans ses requêtes ; rendre le terme inchangé est une implémentation
valide.

## Les trois étapes

### 1. Le dossier d'adapter

`internal/infrastructure/<outil>/`, qui dépend **uniquement** de
`internal/domain`. Il y met :

- son `Config` et sa méthode `Validate() error`, avec ses erreurs sentinelles
  exportées (`ErrMissingConfig`…) — les règles de credentials appartiennent à
  l'adapter, pas au composition root ;
- le client HTTP (ou le lecteur de fichier) et le mapping vers `domain.Task`,
  les `nil` convertis dès la frontière pour qu'aucun pointeur n'entre dans le
  domaine.

### 2. Le builder

`internal/config/<outil>.go`, sur le modèle de `jira.go` :

```go
func buildLinearTracker() (ports.Tracker, error) {
    config := linear.Config{APIKey: env("LINEAR_API_KEY"), ...}
    if err := config.Validate(); err != nil {
        return nil, err
    }
    return linear.NewClient(config, &http.Client{Timeout: httpTimeout}), nil
}
```

### 3. L'entrée dans le registre

`internal/config/tracker.go` :

```go
const TrackerLinear TrackerName = "linear"

var trackerRegistry = map[TrackerName]trackerEntry{
    TrackerJira:   {build: buildJiraTracker, defaultUserEnv: jiraEmailEnv},
    TrackerLinear: {build: buildLinearTracker, defaultUserEnv: "LINEAR_EMAIL"},
}
```

`defaultUserEnv` nomme la variable du compte connecté : la personne ciblée
quand ni `--user` ni `TRACKER_USER` ne sont fournis. Vide si l'outil n'en a
pas — le rapport couvre alors tout le board.

C'est tout. `TRACKER=linear` dans `.env` suffit à basculer. Aucun `switch` à
rallonger ailleurs, aucun fichier du domaine ou de l'application à rouvrir
(open/closed).

## Ce qui reste à vérifier

- **Documentation** : ajouter les variables du nouvel adapter au tableau de
  [configuration.md](configuration.md) et au `.env.example`.
- **Tests** : le `Validate()` de l'adapter, son mapping vers `domain.Task`
  (via `httptest.Server`, jamais de mock), et le passage dans le registre.
- **Frontières** : `go test ./internal/architecture/...` échoue si l'adapter
  fuit vers le domaine ou l'application.
- **Sécurité** : transport HTTPS obligatoire dès qu'un credential circule,
  timeout sur le client HTTP, aucun secret loggué.
