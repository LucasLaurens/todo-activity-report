# État actuel de l'application

[← Retour au README](../README.md)

> Ce fichier reflète l'état courant de `todo-activity-report`. **Règle du projet**
> ([conventions.md](conventions.md)) : il doit être mis à jour à chaque changement
> de comportement, dépendance ou flag.
>
> Emplacement : `~/projects/todo-activity-report` — module
> `github.com/LucasLaurens/todo-activity-report`, binaire `todo-activity-report`.

Dernière mise à jour : 2026-09-02.

## Statut global

CLI fonctionnel, **validé contre un vrai Jira Cloud** (2789 tâches lues). Le
tracker est un adapter interchangeable : le domaine et l'application ne
dépendent que des ports. Build OK, `go vet` OK, suite de tests verte.

## Fonctionnalités livrées

- [x] Binaire `todo-activity-report`.
- [x] Lecture des tickets **processés** un jour (temps loggué OU changement de statut OU toute autre mise à jour — commentaire/édition via `updated`) via `GET /rest/api/3/search/jql` — tous statuts, pas seulement résolus.
- [x] **By status** : répartition par statut = l'état d'avancée (en cours / bloqué / release / terminé).
- [x] **i18n** : sortie + résumé IA en `fr` (défaut) ou `en` via `APP_LOCALE` ; traductions JSON embarquées.
- [x] Temps = **worklog réellement loggué dans la période** (`/issue/{key}/worklog`), filtré par date et par auteur (`--user`), pas le cumul du ticket.
- [x] Worklog récupéré en **parallèle** (pool borné à 8, ordre préservé, annulation sur 1ère erreur), validé `-race`.
- [x] Tests indépendants en `t.Parallel()` (sauf `config` à cause de `t.Setenv`).
- [x] CI GitHub Actions : 4 jobs parallèles (format / vet / lint / test) — voir [ci.md](ci.md).
- [x] Dates en `JJ/MM/AAAA` : `--date` (un jour) ou `--from`/`--to` (plage, fin inclusive 23:59).
- [x] `--user` (répétable / CSV) résolu nom·email → accountId via `/rest/api/3/user/search` ; `--no-resolve` pour court-circuiter.
- [x] Filtres : projet (`--project`, sinon `TRACKER_PROJECT` ; aucune valeur en dur), utilisateurs, plage de dates.
- [x] **Tracker interchangeable** : port `ports.Tracker` (= `TaskProvider` + `UserResolver`), registre `TRACKER` → builder dans `internal/config/tracker.go`, chaque adapter validant sa propre config (`jira.Config.Validate()`). Ajouter un outil = un dossier d'adapter + une entrée du registre, sans toucher domaine ni use case — voir [trackers.md](trackers.md).
- [x] **Frontières hexagonales encodées** : `internal/architecture` échoue si `internal/domain` ou `internal/application` importent `infrastructure`/`config`/`i18n`.
- [x] JQL généré + pagination par `nextPageToken`.
- [x] `--limit` : cap de la liste détaillée (défaut 20) avec note de troncature.
- [x] MTTR robuste : exclut les résolutions antérieures à la création (anomalies).
- [x] Audit perf : somme du temps en une passe, champs Jira calculés une fois.
- [x] Audit sécurité : `https://` exigé par l'adapter Jira (`jira.ErrInsecureBaseURL`), troncature rune-safe (UTF-8), JQL/URL échappés, timeouts HTTP.
- [x] Mapping des champs custom root cause / correction (texte + ADF). Un champ suivant le template Problème/Cause/Correction/Prévention : la Cause alimente le root cause, la Correction le fix ; variantes « vulgarisée » ignorées, template non rempli traité comme vide.
- [x] Métriques déterministes : totaux, taux de résolution, temps total/moyen,
      MTTR, par assignee, par type, par priorité, top root causes.
- [x] Rendu terminal : distributions alignées (`tabwriter`), ligne vide entre sections, puces (`•`), wrap par mots avec indentation suspendue (largeur = `COLUMNS` sinon 100). Chaque ticket = en-tête compact `clé · statut · temps` puis résumé / Cause / Fix wrappés. Par défaut, Cause/Fix réduits à leur **gist** (1ère phrase, coupe à la phrase sinon au mot, ≤150 runes) ; `--full` affiche tout. « Par personne » masqué si une seule personne. `--brief` ne garde que les métriques chiffrées + la synthèse IA (masque Causes racines et Tickets). « Causes racines récurrentes » n'affiche que les causes touchant ≥2 tickets (sinon section masquée — les causes uniques sont déjà sous chaque ticket).
- [x] Colorimétrie ANSI : rouge pour bloqué / attente bloquante / **blocage signalé en commentaire**, bleu pour le travail en vol (`In progress dev`, `Code Review`, `To test`, `Waiting for feat env`), vert pour livré et terminal (`Deployed / Completed`, `Waiting for release`, `Fermée`), gris pour l'abandon (`Annulé`, `Obsolète`), défaut du terminal pour le non démarré. La classification vit **dans le domaine** (`domain.ClassifyStatus` → `StatusKind`), unique source pour la couleur, le tri et le marqueur ; ordre des règles significatif (`release`, `feat env` avant le générique `waiting`/`attente`). Appliqué à « Avancement », à tout le bloc de chaque ticket (en-tête + résumé + Cause + Correction) et aux lignes de ticket de la synthèse IA (colorées d'après leur ticket, blocage en commentaire inclus ; la ligne d'ouverture reste neutre). Les statuts réels du board sont couverts par un test.
- [x] `Task.HasBlockingComment()` : un commentaire récent signalant un blocage passe le ticket en rouge malgré son statut, avec marqueur `⚠ blocage signalé en commentaire` sur la ligne. Heuristique par mots-clés, sans appel réseau supplémentaire (commentaires déjà récupérés inline).
- [x] Ordre de lecture **bloquants d'abord** : tickets triés par `StatusKind.AttentionRank()` (bloqué → en vol → livré → non démarré → abandonné) avec la clé en tie-break, section « Avancement » triée de même (volume en tie-break), et prompt IA instruit de conserver l'ordre reçu.
- [x] Résumé narratif IA : auto via le CLI `claude` local (sans clé), fallback API Anthropic, sinon omis. Format **briefing oral**, budget de longueur imposé (2 phrases de vue d'ensemble max, **20 mots max par ligne de ticket**, une seule ligne par ticket, ni markdown ni titre ni ligne vide) : vue d'ensemble (volume / temps / risque) puis une ligne par ticket, **groupée par statut** et couvrant **tous** les tickets — terminés : cause racine + fix ; en cours / bloqués / code review / release : avancée + blocage/prochaine étape (déduits du résumé, statut et temps loggué). Les **commentaires récents** (5 derniers/ticket, récupérés inline avec la recherche — pas d'appel N+1) alimentent la synthèse pour la rendre précise ; ils servent de contexte, non recopiés. Envoi à Claude → voir [confidentialité](configuration.md#confidentialité-du-résumé-ia).
- [x] Config façon Laravel (env → config typée → fabriques) + auto-load `.env` (CWD **puis** dossier de l'exécutable, les deux lus, 1ère clé gagnante → l'exe comble les manques ; sans override de l'env réel).
- [x] Distribution binaire : job CI `release` (4 cibles `darwin-arm64|amd64`, `linux-amd64|arm64`, `-trimpath -ldflags="-s -w"`, `SHA256SUMS`) publiant la Release GitHub du tag. Binaire **autonome** (stdlib, statiquement lié, locales embarquées) : install = binaire + `.env` à côté, sans le dépôt — voir [install.md](install.md).
- [x] `--version` : rend le tag gravé au build (`-ldflags="-X main.version=$GITHUB_REF_NAME"` dans le job `release`), `dev` pour un binaire compilé à la main. Répond avant le chargement de la config : aucun identifiant de tracker nécessaire.
- [x] `--help` : aide complète (usage, options, exemples), gardée par un test qui échoue si un flag enregistré n'y figure pas.
- [x] Loader : spinner stderr montrant l'étape courante (port `ProgressReporter`), no-op hors terminal.
- [x] Export quotidien du brief de la veille dans l'app Notes (macOS) : [scripts/activity-notes.sh](../scripts/activity-notes.sh), planifié par un LaunchAgent — voir [usage.md](usage.md#automatisation--note-quotidienne-dans-lapp-notes-macos).
- [x] Wrapper shell `activity-report` (`~/.zshrc`, hors dépôt) : prompts date/utilisateur puis `--brief` ; rend la main au binaire dès qu'un flag est passé — voir [usage.md](usage.md#wrapper-shell-activity-report).
- [x] Tests : domain, application, infrastructure (jira/cli/llm), config, i18n, cmd — fakes in-memory. Détail par paquet dans [testing.md](testing.md#couverture-par-paquet).

## Surface CLI

Flags : `--date` | (`--from` + `--to`), `--user`, `--project`, `--no-ai`, `--no-resolve`, `--limit`, `--full`, `--brief`, `--version`, `--help`.

Variables : `TRACKER` (défaut `jira`), `TRACKER_PROJECT`, `APP_LOCALE`, les
`JIRA_*` de l'adapter, `ANTHROPIC_*`, `NO_COLOR`, `COLUMNS` — voir
[configuration.md](configuration.md#variables).

## Dépendances externes

| Service | Statut | Requis |
|---------|--------|--------|
| Jira Cloud REST API | intégré (adapter par défaut) | oui, sauf autre `TRACKER` |
| CLI `claude` local | intégré (source IA par défaut) | non |
| Anthropic Messages API | intégré (fallback IA) | non |

Aucune dépendance Go tierce : stdlib uniquement.

## Limitations connues / pistes

- [ ] Un seul adapter de tracker livré (`jira`) : le port et le registre existent, les autres restent à écrire.
- [ ] Pas de cache des réponses du tracker (chaque run rappelle l'API).
- [ ] Sortie terminal uniquement (pas de JSON/CSV export).
- [ ] Pas de Homebrew tap / installeur : le binaire se télécharge à la main (`curl`) depuis la Release.

## Modèle IA par défaut

`claude-sonnet-4-6` (override via `ANTHROPIC_MODEL`).
