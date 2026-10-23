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
- [x] **Rapport par défaut « moi, dernier jour ouvré »** : sans flag, la période est le dernier jour ouvré (`domain.ParseDay` — lundi remonte à vendredi, sam/dim aussi) et la cible est `TRACKER_USER` (`Config.DefaultAssignee`, l'email du compte tracker en secours). `--date` accepte aussi `hier`/`yesterday`, `today`/`aujourd'hui`. `--all` élargit au board entier (exclusif avec `--user`).
- [x] **Répartition du temps** : temps loggué par ticket, décroissant, avec sa part du total (`Metrics.TimeBreakdown`), et le nombre de tickets touchés sans temps loggué (`Metrics.UntimedTasks`). Masquée si rien n'a été loggué, bornée par `--limit` comme la liste de tickets (un seul `window` générique pour les deux).
- [x] **Un seul cran de détail** : `--detail=brief|normal|full` remplace les booléens `--full` et `--brief`, qui réglaient la même chose et pouvaient se contredire.
- [x] Lecture des tickets **travaillés** un jour (temps loggué OU changement de statut) via `GET /rest/api/3/search/jql` — tous statuts, pas seulement résolus. `--touched` ajoute l'union avec `updated` (commentaire, édition de champ, changement en masse) : opt-in, car `updated` bouge pour n'importe qui et noyait le rapport d'une journée.
- [x] **By status** : répartition par statut = l'état d'avancée (en cours / bloqué / release / terminé).
- [x] **i18n** : sortie + résumé IA en `fr` (défaut) ou `en` via `APP_LOCALE` ; traductions JSON embarquées.
- [x] Temps = **worklog réellement loggué dans la période** (`/issue/{key}/worklog`), filtré par date et par auteur (`--user`), pas le cumul du ticket. Le filtre jouant aussi sur `assignee`, le total est le temps **en tant qu'assigné** : le libellé le dit et une note sous la synthèse rappelle qu'un worklog sur un ticket assigné à quelqu'un d'autre n'y figure pas. Libellé neutre et pas de note avec `--all`.
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
- [x] Résolutions comptées **dans la période** (`Task.IsResolvedWithin`, `DateRange.Contains`) : « Résolus », le taux, « Par personne » et le MTTR ignorent un ticket résolu il y a trois mois et seulement touché aujourd'hui.
- [x] MTTR robuste : exclut les résolutions antérieures à la création (anomalies).
- [x] Audit perf : somme du temps en une passe, champs Jira calculés une fois.
- [x] Audit sécurité : `https://` exigé par l'adapter Jira (`jira.ErrInsecureBaseURL`), troncature rune-safe (UTF-8), JQL/URL échappés (**antislash échappé avant le guillemet** — un terme finissant par `\` s'évadait de la chaîne JQL), timeouts HTTP, `.env` **world-writable ignoré** (avertissement sur stderr) car il porte les identifiants, tout texte venu du tracker ou du modèle passe par `sanitize.Printable` (séquences ANSI, contrôles C0/C1 et marques bidi retirés avant affichage terminal), **messages d'erreur compris** (corps d'erreur Jira/Anthropic, stderr du CLI `claude`), réponses HTTP plafonnées (16 Mo Jira, 1 Mo Anthropic), redirection `https→http` refusée (sinon `net/http` y renverrait le token Jira), pagination Jira bornée (jeton répété ou plus de 1000 pages → erreur), CLI `claude` **isolé** (aucun outil, aucun MCP, aucun réglage projet, dossier de travail vide) contre l'injection de prompt via les tickets, toolchain Go patché (`go1.26.10`, `govulncheck` vide) et CI en droits minimaux avec actions épinglées par SHA.
- [x] Mapping des champs custom root cause / correction (texte + ADF). Un champ suivant le template Problème/Cause/Correction/Prévention : la Cause alimente le root cause, la Correction le fix ; variantes « vulgarisée » ignorées, template non rempli traité comme vide.
- [x] Métriques déterministes : totaux, taux de résolution, temps total/moyen,
      MTTR, par assignee, par type, par priorité, top root causes.
- [x] Rendu terminal : distributions alignées (`tabwriter`), ligne vide entre sections, puces (`•`), wrap par mots avec indentation suspendue (largeur = `cli.RenderOptions.Width`, alimentée par `COLUMNS` dans `cmd`, sinon 100). Chaque ticket = en-tête compact `clé · statut · temps` puis résumé / Cause / Fix wrappés. Par défaut, Cause/Fix réduits à leur **gist** (1ère phrase, coupe à la phrase sinon au mot, ≤150 runes) ; `--detail=full` affiche tout. « Par personne » masqué si une seule personne. `--detail=brief` ne garde que les chiffres (Répartition du temps comprise) + la synthèse IA (masque Causes racines et Tickets). « Causes racines récurrentes » n'affiche que les causes touchant ≥2 tickets (sinon section masquée — les causes uniques sont déjà sous chaque ticket).
- [x] Colorimétrie ANSI : rouge pour bloqué / attente bloquante / **blocage signalé en commentaire**, bleu pour le travail en vol (`In progress dev`, `Code Review`, `To test`, `Waiting for feat env`), vert pour livré et terminal (`Deployed / Completed`, `Waiting for release`, `Fermée`), gris pour l'abandon (`Annulé`, `Obsolète`), défaut du terminal pour le non démarré. La classification vit **dans le domaine** (`domain.ClassifyStatus` → `StatusKind`), unique source pour la couleur, le tri et le marqueur ; ordre des règles significatif (`release`, `feat env` avant le générique `waiting`/`attente`). Appliqué à « Avancement », à tout le bloc de chaque ticket (en-tête + résumé + Cause + Correction) et aux lignes de ticket de la synthèse IA (colorées d'après leur ticket, blocage en commentaire inclus ; la ligne d'ouverture reste neutre). Les statuts réels du board sont couverts par un test.
- [x] `Task.HasBlockingComment()` : un commentaire récent signalant un blocage passe le ticket en rouge malgré son statut, avec marqueur `⚠ blocage signalé en commentaire` sur la ligne. Heuristique par mots-clés, sans appel réseau supplémentaire (commentaires déjà récupérés inline).
- [x] Ordre de lecture **bloquants d'abord** : tickets triés par `StatusKind.AttentionRank()` (bloqué → en vol → livré → non démarré → abandonné) avec la clé en tie-break, section « Avancement » triée de même (volume en tie-break), et prompt IA instruit de conserver l'ordre reçu.
- [x] **Le rapport survit à l'échec de l'IA** : le narratif passe par un décorateur `FailSafeSummarizer` — CLI tué, timeout, API injoignable → avertissement sur stderr (`warning: no AI summary (…)`), métriques affichées, code de sortie 0. Budget du CLI porté à **4 min** et un dépassement est nommé (`claude cli timed out after 4m0s`) au lieu de `signal: killed` avec un stderr vide.
- [x] **Budget du prompt : 20 tickets et 3 commentaires/ticket.** Le modèle écrit une ligne par ticket après avoir lu chaque commentaire : c'est ce qui faisait dépasser le budget. Le reste de la liste est marqué `... and N more` et le prompt ordonne de l'ignorer en silence. Les 5 commentaires restent en mémoire pour la détection de blocage, qui garde sa fenêtre.
- [x] Résumé narratif IA : auto via le CLI `claude` local (sans clé), fallback API Anthropic, sinon omis. Format **briefing oral**, budget de longueur imposé (2 phrases de vue d'ensemble max, **20 mots max par ligne de ticket**, une seule ligne par ticket, ni markdown ni titre ni ligne vide) : vue d'ensemble (volume / temps / risque) puis une ligne par ticket, **groupée par statut** et couvrant **tous** les tickets — terminés : cause racine + fix ; en cours / bloqués / code review / release : avancée + blocage/prochaine étape (déduits du résumé, statut et temps loggué). Les **commentaires récents** (5 derniers/ticket récupérés inline avec la recherche — pas d'appel N+1 ; les **3 derniers** partent dans le prompt) alimentent la synthèse pour la rendre précise ; ils servent de contexte, non recopiés. Envoi à Claude → voir [confidentialité](configuration.md#confidentialité-du-résumé-ia).
- [x] Config façon Laravel (env → config typée → fabriques) + auto-load `.env` (CWD **puis** dossier de l'exécutable, les deux lus, 1ère clé gagnante → l'exe comble les manques ; sans override de l'env réel). Un `.env` world-writable est ignoré et une lecture interrompue (ligne > 64 Kio) est signalée, tous deux par un avertissement stderr.
- [x] Distribution binaire : job CI `release` (4 cibles `darwin-arm64|amd64`, `linux-amd64|arm64`, `-trimpath -ldflags="-s -w"`, `SHA256SUMS`) publiant la Release GitHub du tag. Binaire **autonome** (stdlib, statiquement lié, locales embarquées) : install = binaire + `.env` à côté, sans le dépôt — voir [install.md](install.md).
- [x] `--version` : rend le tag gravé au build (`-ldflags="-X main.version=$GITHUB_REF_NAME"` dans le job `release`), `dev` pour un binaire compilé à la main. Répond avant le chargement de la config : aucun identifiant de tracker nécessaire.
- [x] `--help` : aide complète (usage, options, exemples), seule description des flags (les défauts de `--limit` et `--detail` y sont injectés depuis le code), gardée par un test qui échoue si un flag enregistré n'y figure pas. Un flag inconnu l'imprime une seule fois.
- [x] Loader : spinner stderr montrant l'étape courante (port `ProgressReporter`), no-op hors terminal.
- [x] Export quotidien du brief de la veille dans l'app Notes (macOS) : [scripts/activity-notes.sh](../scripts/activity-notes.sh), planifié par un LaunchAgent — voir [usage.md](usage.md#automatisation--note-quotidienne-dans-lapp-notes-macos).
- [x] Wrapper shell `activity-report` (`~/.zshrc`, hors dépôt) : prompts date/utilisateur puis `--detail=brief` ; rend la main au binaire dès qu'un flag est passé — voir [usage.md](usage.md#wrapper-shell-activity-report).
- [x] Tests : domain, application, infrastructure (jira/cli/llm), config, i18n, cmd — fakes in-memory. Détail par paquet dans [testing.md](testing.md#couverture-par-paquet).

## Surface CLI

Flags : `--date` | (`--from` + `--to`), `--user` | `--all`, `--touched`, `--project`, `--no-ai`, `--no-resolve`, `--limit`, `--detail`, `--version`, `--help`.
Aucun flag = `TRACKER_USER` sur le dernier jour ouvré.

Variables : `TRACKER` (défaut `jira`), `TRACKER_PROJECT`, `TRACKER_USER`, `APP_LOCALE`, les
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
- [ ] Ne plus installer la **v1.2.0** : compilée en Go 1.22.12, elle embarque 63 vulnérabilités
  stdlib connues (`govulncheck -mode=binary`). Les **v1.3.0** et **v1.3.1** (go1.26.10) n'en ont aucune ; la v1.3.1 ajoute les durcissements de faible gravité.
- [ ] Les binaires v1.3.0 et v1.3.1 ont été compilés en local (Actions alors bloquées par la
  facturation), avec les commandes exactes du job `release`. Depuis le passage du dépôt en
  public, la CI tourne sur Actions ; le job `release` reste à voir tourner au prochain tag.
- [ ] Pas de Homebrew tap / installeur : le binaire se télécharge à la main depuis la Release
  (`curl`, sans authentification : le dépôt est public) — voir [install.md](install.md).

## Modèle IA par défaut

`claude-sonnet-4-6` (override via `ANTHROPIC_MODEL`).
