# Tests

[← Retour au README](../README.md)

```bash
go test -race -cover ./...
```

Tests indépendants marqués `t.Parallel()` → exécution concurrente. Les tests de
`config` (via `t.Setenv`) restent séquentiels — `t.Setenv` est incompatible avec
`t.Parallel`. `-race` valide notamment le pool worklog concurrent.

## Principes

- **AAA** — chaque test est découpé en `// Arrange`, `// Act`, `// Assert`.
- **Pas de mocks** — on utilise des implémentations in-memory concrètes
  (`internal/testsupport`) qui implémentent réellement le comportement, plus un
  `httptest.Server` réel pour l'adapter Jira. Aucun framework de mock.
- **Builder** — `testsupport.TaskBuilder` produit des `domain.Task` avec des
  défauts sensés et une API fluide ; chaque test n'override que le champ qu'il
  teste.

## Doublures de test (fakes concrets)

| Fake | Implémente | Rôle |
|------|-----------|------|
| `InMemoryTaskProvider` | `ports.TaskProvider` | sert une slice de tâches, enregistre la dernière `Query` |
| `InMemorySummarizer` | `ports.Summarizer` | renvoie un narratif canné, enregistre le rapport reçu |
| `InMemoryUserResolver` | `ports.UserResolver` | table nom/email → identifiant, sans appel réseau |
| `application.PassthroughResolver` | `ports.UserResolver` | null object de production, réutilisable en test |
| `RecordingProgress` | `ports.ProgressReporter` | enregistre les étapes reçues, dans l'ordre, pour vérifier la séquence exacte |
| `httptest.Server` | (réseau) | vrai serveur HTTP renvoyant du JSON Jira pour tester le client |

## Couverture par paquet

Chaque paquet est testé au niveau où son comportement est observable. La liste
décrit les paquets, pas les fichiers : ajouter un fichier de test ne périme pas
ce tableau.

| Paquet | Niveau | Ce que les tests vérifient |
|--------|--------|----------------------------|
| `internal/domain` | unit | Invariants des value objects, sans aucune dépendance : `TimeSpent` (clamp du négatif, addition, division par zéro, format), `Optional[T]` (présence / absence / fallback), `DateRange` (bornes, fin inclusive à 23:59, rejet d'un intervalle inversé), `Task` (prédicats métier : `IsResolvedWithin` la période, blocage signalé en commentaire, `StatusKind`), `ParseDay` (JJ/MM/AAAA, `hier`/`today`, dernier jour ouvré), `Query` et `Report`. |
| `internal/application` | feature | Le use case contre des fakes in-memory : `ReportService` (orchestration, forward de la `Query` au provider, résolution des assignees en accountId, étapes de progression rapportées, rapport sans narratif, propagation d'erreur) et `MetricsCalculator` (totaux, moyenne, MTTR, ranking des root causes, regroupements par assignee / statut / type / priorité, jeu vide). |
| `internal/i18n` | unit | Parsing de la locale, message localisé, formatage des arguments, clé inconnue retournée telle quelle ; chaque catalogue embarqué se parse, fr et en ont exactement les mêmes clés, chaque étape de progression (`ports.Stage*`) est traduite. |
| `internal/config` | intégration | Pont env → config typée : `TRACKER` inconnu rejeté, tracker par défaut, board lu depuis `TRACKER_PROJECT`, diagnostic de l'adapter remonté tel quel, parsing du `.env` (les deux emplacements, première clé gagnante, sans écraser l'env réel), sélection du modèle Anthropic, fabriques d'adapters, compte par défaut (`TRACKER_USER`, sinon le compte déclaré par l'entrée du registre), avertissement sur un `.env` lu partiellement. Séquentiels — `t.Setenv` interdit `t.Parallel`. |
| `internal/architecture` | architecture | Frontières hexagonales : aucun fichier de `internal/domain` ou `internal/application` (tests inclus) n'importe `infrastructure`, `config` ou `i18n`. Échoue avec le fichier et l'import fautifs. |
| `internal/infrastructure/jira` | intégration | Adapter Jira contre un `httptest.Server` réel : validation de sa config (credentials manquants, `http://` refusé), JQL généré et échappé, Basic auth, pagination par `nextPageToken`, erreurs HTTP, mapping des champs (dont custom root cause / correction en texte et en ADF), extraction du template Problème/Cause/Correction/Prévention, résolution des utilisateurs. |
| `internal/infrastructure/llm` | intégration | Les trois summarizers : CLI `claude` local (narratif, sortie vide, binaire en échec), API Anthropic (requête, réponse, non-200, contenu vide, modèle par défaut) et `NoopSummarizer`. Le prompt est testé séparément — métriques et tickets présents, tickets ouverts couverts au même titre que les terminés, commentaires inclus, liste longue tronquée. |
| `internal/infrastructure/sanitize` | unit | `Printable` : séquences ANSI et caractères de contrôle retirés, retour chariot supprimé, accents / retours ligne / tabulations préservés. Son application est vérifiée aux deux frontières (mapper du tracker, réponse des summarizers). |
| `internal/infrastructure/cli` | intégration | Rendu terminal : sections overview / tâches, narratif, état vide, wrap sans casser l'UTF-8, causes en gist par défaut et complètes en `--detail=full`, `--detail=brief`, répartition du temps (part du total, tri, cap `--limit`, tickets sans temps loggé, libellé « en tant qu'assigné » et sa note), `ParseDetailLevel`, cap de la liste avec note de troncature, locale anglaise. Plus la colorimétrie des statuts (classement rouge/vert/bleu par mot-clé, alignement `tabwriter` préservé, sortie plate sans terminal) et le `Spinner` (étape rendue puis effacée, arrêt idempotent, arrêt sans démarrage qui rend la main). |
| `cmd/todo-activity-report` | feature | Surface CLI : parsing des flags, résolution de la période (`--date` exclusif de `--from`/`--to`, formats rejetés), `--project` prioritaire sur `TRACKER_PROJECT`, absence de board signalée par `errNoProject`, identifiants manquants remontés par l'adapter, sortie propre sur `--help`, aide imprimée une seule fois sur un flag inconnu, garde-fou « chaque flag enregistré figure dans `helpText` » (défaut de `--limit` injecté depuis sa constante), choix du résolveur selon `--no-resolve`. |

Chiffres à jour de la couverture :

```bash
go test -race -cover ./...
```

## Ajouter un test

1. Arrange avec `testsupport.NewTask()...Build()` ou un fake in-memory.
2. Une seule action dans Act.
3. Assert sur l'état / la sortie observable, jamais sur "comment" c'est appelé.

Voir les règles globales dans [conventions.md](conventions.md).
