# Identifiants Jira ("creds")

[← Retour au README](../README.md)

## C'est quoi les "creds" Jira ?

"Creds" = **credentials** = les identifiants d'authentification que le CLI
utilise pour appeler l'API REST de Jira Cloud en ton nom. Concrètement, trois
choses :

1. **`JIRA_BASE_URL`** — l'adresse de ton instance Jira Cloud, ex.
   `https://your-org.atlassian.net`. Pas un secret, mais nécessaire.
2. **`JIRA_EMAIL`** — l'email du compte Atlassian (le tien, ou un compte bot
   dédié) propriétaire du token.
3. **`JIRA_API_TOKEN`** — un **token API** Atlassian. C'est le secret. Il
   remplace ton mot de passe pour l'API. Jira Cloud utilise l'authentification
   *HTTP Basic* `email:token` encodée en base64 — c'est ce que le client
   construit dans `internal/infrastructure/jira/client.go`.

> Le CLI **n'utilise pas** ton mot de passe Jira. Uniquement email + token API.

## Générer un token API Atlassian

1. Va sur https://id.atlassian.com/manage-profile/security/api-tokens
2. *Create API token* → donne-lui un label (`todo-activity-report`).
3. Copie le token (affiché une seule fois).
4. Mets-le dans `.env` :

```env
JIRA_BASE_URL=https://your-org.atlassian.net
JIRA_EMAIL=ton.email@example.com
JIRA_API_TOKEN=le_token_copié
```

## Champs custom root cause / correction

Jira n'a pas de champ standard "root cause" ni "correction". Beaucoup de boards
les rangent dans **un seul** champ personnalisé suivant le template
**Problème / Cause / Correction / Prévention**. Relever son id
(`customfield_XXXXX`) dans Jira > Paramètres > Tickets > Champs personnalisés,
puis :

```env
JIRA_ROOT_CAUSE_FIELD=customfield_XXXXX
JIRA_CORRECTION_FIELD=
```

Le mapper interprète ce template :

- la section **Cause** alimente la colonne *root cause* ;
- la section **Correction** alimente la colonne *fix* ;
- les variantes **vulgarisée** (`🧠 Cause vulgarisée`, `🛠️ Correction
  vulgarisée`) sont ignorées au profit du contenu technique ;
- un template **non rempli** (texte d'instruction `DU POINT DE VUE DU CLIENT`)
  est traité comme vide — il ne pollue pas le « top root causes ».

`JIRA_CORRECTION_FIELD` reste vide ; ne le renseigne que si un **autre** champ
porte la correction (il prend alors le pas sur la Correction extraite ci-dessus).
Texte simple **et** ADF (Atlassian Document Format) supportés. Trouver un id :
*Jira → Settings → Issues → Custom fields*, ou `GET /rest/api/3/field`. Sans
`JIRA_ROOT_CAUSE_FIELD`, les colonnes RC/Fix restent vides — le reste du rapport
fonctionne.

## Clé projet vs nom du board

L'URL `.../jira/software/c/projects/ACME/boards/2` donne la **clé** projet
`ACME` (≠ nom affiché du board). Le CLI interroge par clé : `--project ACME`, ou `TRACKER_PROJECT=ACME` dans le `.env`.

## Trouver un accountId

Pour filtrer par utilisateur, prends l'`accountId` dans l'URL d'un board filtré :
`...?assignee=712020%3A00000000-0000-0000-0000-000000000000` → `%3A` = `:`, donc
accountId = `712020:00000000-0000-0000-0000-000000000000`. À passer à `--users`.

## API utilisée

- `GET /rest/api/3/search/jql` — tâches (l'ancien `/rest/api/3/search` a été
  supprimé par Atlassian). Pagination par `nextPageToken`.
- `GET /rest/api/3/user/search` — résolution `--user` (nom/email → accountId).
- `GET /rest/api/3/issue/{key}/worklog` — temps réellement loggué dans la période (un appel par ticket).

Le CLI passe `--user` à la résolution sauf si le terme ressemble déjà à un
accountId (présence de `:` ou 24 hex) ou si `--no-resolve` est fourni.

Le filtre d'activité s'appuie sur trois clauses JQL :
- `worklogDate` — tickets avec du temps loggué le jour ciblé ;
- `status CHANGED DURING (...)` — tickets ayant changé de statut ce jour-là ;
- `updated >= ... AND updated <= ...` — toute autre mise à jour ce jour-là
  (commentaire, édition de champ, réassignation). Jira n'a pas de champ
  `commentedDate`, donc les commentaires passent par `updated`.

## Permissions requises

Le compte doit pouvoir *Browse projects* sur le projet ciblé et lire
les champs + worklogs (la clause `worklogDate` requiert l'accès aux journaux de
travail). Lecture seule : le CLI n'écrit jamais dans Jira.
