# CI — GitHub Actions

[← Retour au README](../README.md)

Un seul workflow, [`.github/workflows/ci.yml`](../.github/workflows/ci.yml).

## Déclencheurs

| Événement | Ce qui tourne |
|-----------|---------------|
| push sur `main` | les 4 checks qualité |
| pull request | les 4 checks qualité |
| push d'un tag `v*` | les 4 checks, puis build + Release |

## Les 4 checks qualité (en parallèle)

Une `matrix` lance quatre jobs indépendants, `fail-fast: false` : un check rouge
n'empêche pas les trois autres de rendre leur verdict.

| Check | Commande | Ce qu'il garantit |
|-------|----------|-------------------|
| `format` | `test -z "$(gofmt -l .)"` | tout le code est formaté |
| `vet` | `go vet ./...` | analyse statique du toolchain |
| `lint` | `golangci-lint run` (v2.1.0, voir [`.golangci.yml`](../.golangci.yml)) | style et bugs courants |
| `test` | `go test -race -coverpkg=./... ./...` | suite verte, sans data race, couverture affichée |

Ce sont exactement les 4 commandes à passer en local avant de commiter :

```bash
gofmt -l .            # doit être vide
go vet ./...
golangci-lint run ./...
go test -race -cover ./...
```

La suite inclut `internal/architecture`, qui échoue si le domaine ou
l'application importent une couche externe : la frontière hexagonale est donc
vérifiée par la CI, pas seulement par une relecture.

## Build & Release (sur tag)

Le job `release` ne tourne que sur un tag `v*` et seulement après un `quality`
vert (`needs: quality`).

Un seul job compile les 4 cibles : le module est en **pur Go** (stdlib, aucun
cgo), donc la compilation croisée ne demande aucun toolchain C.

| Artifact | Plateforme |
|----------|------------|
| `todo-activity-report-darwin-arm64` | macOS Apple Silicon |
| `todo-activity-report-darwin-amd64` | macOS Intel |
| `todo-activity-report-linux-amd64` | Linux x86-64 |
| `todo-activity-report-linux-arm64` | Linux ARM64 |
| `SHA256SUMS` | empreintes des 4 binaires |

Le tag est gravé dans le binaire (`-ldflags="-X main.version=$GITHUB_REF_NAME"`),
c'est ce que rend `--version`. `-trimpath -ldflags="-s -w"` retire les chemins
absolus et les symboles (~8,5 Mo → ~6 Mo).

`softprops/action-gh-release@v2` publie les 5 fichiers sur la Release GitHub du
tag — d'où les URLs de [install.md](install.md).

## Publier une version

```bash
git tag v1.0.0
git push origin v1.0.0
```

Aucun secret à configurer : le `GITHUB_TOKEN` fourni au workflow
(`permissions: contents: write`) suffit à créer la Release.
