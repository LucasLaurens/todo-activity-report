# Captures d'écran

Images utilisées par le README et la documentation.

| Fichier | Utilisé dans | Contenu |
|---------|--------------|---------|
| `sortie-couleurs.png` | [README](../../README.md), [usage.md](../usage.md#couleurs) | rapport complet, statuts colorés (rouge / bleu / vert) |
| `sortie-brief.png` | [README](../../README.md), [usage.md](../usage.md#couleurs) | même rapport en `--detail=brief` |
| `install-premier-run.png` | [README](../../README.md), [install.md](../install.md#captures-décran) | téléchargement de la v1.3.1, empreinte, `--version` |
| `install-release.png` | [install.md](../install.md#captures-décran) | *à fournir* : page Release GitHub avec les assets |

Les rapports sont la vraie sortie du renderer (`internal/infrastructure/cli`),
alimentée par les fakes in-memory de `internal/testsupport` avec un board
**fictif** (`ACME`, `camille.martin@example.com`) et une synthèse IA écrite à la
main. La capture d'installation montre les vraies commandes et sorties sur la
Release v1.3.1.

Conventions : PNG, largeur ≤ 1400 px, terminal en thème sombre, aucune donnée
client ni identifiant visible (flouter clés de tickets et emails réels si besoin).
