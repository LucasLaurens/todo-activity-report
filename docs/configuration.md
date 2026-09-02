# Configuration

[← Retour au README](../README.md)

## Règle d'or : secrets dans `.env`, jamais dans le code

Toute donnée sensible (tokens, clés API, emails techniques) vit **uniquement**
dans un fichier `.env` local, **jamais** commité. Le dépôt ne contient que
`.env.example` (valeurs factices / placeholders). `.gitignore` exclut `.env`.

```bash
cp .env.example .env      # copier le gabarit
# éditer .env avec les vraies valeurs — c'est tout, pas d'export
```

### Chargement automatique du `.env`

`config.Load()` charge le `.env` au démarrage, sans export manuel. Les deux
emplacements sont lus — `.env` du **répertoire courant** d'abord, puis celui à
côté de l'**exécutable** — et la **première** source à définir une clé gagne :
l'exécutable comble donc les clés manquantes. Le binaire fonctionne ainsi depuis
n'importe quel dossier, même si un `.env` sans rapport traîne dans le répertoire
courant. Les variables déjà présentes dans l'environnement **gardent la priorité**
(un `export` manuel écrase le `.env`). Fichier absent = pas d'erreur. La valeur est
découpée sur le **premier** `=` (un token en contient souvent), et les guillemets
entourants sont retirés.

## Le pont env → application (façon Laravel)

Comme Laravel sépare `.env` (secrets bruts) de `config/*.php` (config typée
dérivée via `env()`), ce projet sépare :

| Fichier | Rôle | Équivalent Laravel |
|---------|------|--------------------|
| `.env` | secrets bruts, hors git | `.env` |
| `internal/config/env.go` | helpers `env()` / `envDefault()` — seul endroit qui touche `os.Getenv` | helper `env()` |
| `internal/config/tracker.go` | registre des adapters de tracker (`TRACKER` → builder) | `config/app.php` (drivers) |
| `internal/config/jira.go` | mappe `JIRA_*` → `jira.Config`, un fichier par adapter | `config/jira.php` |
| `internal/config/anthropic.go` | mappe `ANTHROPIC_*` → `anthropicConfig` | `config/services.php` |
| `internal/config/config.go` | compose, valide, fabrique les adapters (composition root) | repository `config()` |

Aucun autre paquet ne lit l'environnement : on passe toujours par une config
typée. Changer un nom de variable = un seul fichier à toucher.

## Variables

| Variable | Requis | Défaut | Rôle |
|----------|--------|--------|------|
| `APP_LOCALE` | non | `fr` | langue de la sortie et du résumé IA (`fr` ou `en`) |
| `TRACKER` | non | `jira` | adapter de tracker à utiliser — voir [trackers.md](trackers.md) |
| `TRACKER_PROJECT` | oui\* | — | clé du projet / board par défaut (\* sauf si `--project` est passé) |
| `JIRA_BASE_URL` | oui\*\* | — | URL instance Jira Cloud (`https://org.atlassian.net`) |
| `JIRA_EMAIL` | oui\*\* | — | email du compte / bot |
| `JIRA_API_TOKEN` | oui\*\* | — | token API Atlassian |
| `JIRA_ROOT_CAUSE_FIELD` | non | — | id champ custom root cause (`customfield_XXXXX`) |
| `JIRA_CORRECTION_FIELD` | non | — | id champ custom correction |

\*\* Les variables `JIRA_*` ne sont requises que lorsque `TRACKER=jira`. Un autre
adapter lit ses propres variables et ignore celles-ci.
| `ANTHROPIC_API_KEY` | non | — | **fallback** API si le CLI `claude` est absent |
| `ANTHROPIC_MODEL` | non | `claude-sonnet-4-6` | modèle Claude utilisé par le fallback API |

Où générer chaque token, et ceux utilisés autour du dépôt sans passer par
`.env` : [README, section Tokens et secrets](../README.md#tokens-et-secrets).
| `NO_COLOR` | non | — | si défini (non vide), désactive la colorimétrie ANSI des statuts |

`NO_COLOR` et `COLUMNS` décrivent le terminal, pas l'app : ils sont lus directement
par la couche de rendu (`internal/infrastructure/cli`), sans passer par `internal/config`.

### Langue (i18n)

`APP_LOCALE` (défaut `fr`) sélectionne la langue. Les traductions sont des
fichiers JSON embarqués (`internal/i18n/locales/fr.json`, `en.json`). Une valeur
inconnue retombe sur le français. La langue s'applique aux libellés de la sortie
**et** à la langue de réponse du résumé IA.

```bash
APP_LOCALE=en ./todo-activity-report --date='08/06/2026'   # sortie + IA en anglais
```

## Validation

La validation est répartie selon le propriétaire de la règle :

- `config.Load()` échoue avec `config.ErrUnknownTracker` si `TRACKER` ne
  correspond à aucun adapter enregistré (le message liste les noms connus) ;
- le CLI échoue avec `errNoProject` si ni `--project` ni `TRACKER_PROJECT` ne
  donnent de clé de board ;
- `Config.NewTracker()` remonte le diagnostic de **l'adapter lui-même** :
  `jira.ErrMissingConfig` si l'une des trois variables Jira manque,
  `jira.ErrInsecureBaseURL` si `JIRA_BASE_URL` n'est pas en `https://` (le token
  ne doit jamais transiter en clair).

Le composition root ne connaît donc aucune règle propre à Jira : ajouter un
tracker n'oblige pas à toucher `config.Load()`.

L'IA n'est jamais obligatoire : sans clé, le CLI calcule toutes les métriques et
omet seulement le narratif.

## Source du narratif IA — priorité

Le résumé narratif est produit par la **première** option disponible :

1. **CLI `claude`** présent sur le `PATH` → appelé en headless
   (`claude -p --output-format text`). Réutilise ton login Claude Code,
   **aucune clé API nécessaire**. C'est le mode automatique.
2. sinon **`ANTHROPIC_API_KEY`** défini → appel à l'API Anthropic.
3. sinon **aucun narratif** (les métriques restent calculées).

`--no-ai` force le mode 3 quelle que soit la disponibilité.

## Confidentialité du résumé IA

Le narratif envoie les titres, root causes, corrections **et les commentaires
récents** (5 derniers par ticket, max ~280 caractères chacun) des tâches de la
période à Claude (via le CLI ou l'API). Les commentaires peuvent contenir des
échanges internes sensibles — c'est désormais transmis au service Claude pour
affiner la synthèse. Si non souhaité, passer `--no-ai`. Tout le reste
(métriques) est calculé localement, sans aucun appel.

Voir [jira-setup.md](jira-setup.md) pour obtenir les identifiants Jira, et
[trackers.md](trackers.md) pour brancher un autre outil.
