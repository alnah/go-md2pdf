# Plan: remplacer `staticcheck` par `golangci-lint` v2

## Objectif

Remplacer l'étape `staticcheck` actuelle par `golangci-lint` v2, avec une migration progressive, reproductible en CI, et sans élargir brutalement le périmètre des checks.

## Audit de l'existant (codebase)

- `Makefile`
  - `tools` installe `staticcheck` et `gosec` via `go get -tool`.
  - `lint` exécute `go tool staticcheck ./...`.
  - `check` / `check-all` dépendent de `lint`.
- CI GitHub (`.github/workflows/ci.yml`)
  - Job `lint`: `gofmt`, `go vet`, puis `go tool staticcheck ./...`.
  - Job `security`: `go tool gosec ./...`.
- `go.mod`
  - `tool (...)` contient `honnef.co/go/tools/cmd/staticcheck` et `gosec`.
- Aucun `.golangci.yml` actuellement.

## Contraintes externes (v2)

- La config v2 impose `version: "2"`.
- Le timeout de `golangci-lint run` est désactivé par défaut: il faut le fixer explicitement en CI.
- La doc recommande d'éviter `linters.default: all` en CI pour garder des builds reproductibles.
- `stylecheck`/`gosimple`/`staticcheck` sont fusionnés dans `staticcheck` en v2.
- La commande `golangci-lint config verify` permet de valider la config contre le schéma JSON.
- En CI GitHub, la doc recommande l'action officielle `golangci/golangci-lint-action`.
- Pour l'installation locale, la doc déconseille la stratégie `go tool`/`tool directive` pour `golangci-lint` (risque d'interférences de dépendances d'outillage).

## Stratégie recommandée (adaptée au projet)

1. Migration **à parité** dans un premier temps:
   - remplacer uniquement `staticcheck` par `golangci-lint` configuré avec `linters.default: none` + `enable: [staticcheck]`.
   - conserver `go vet` et `gosec` séparés comme aujourd'hui.
2. Installer `golangci-lint` en CI via l'action officielle, version épinglée.
3. Pour local/dev:
   - privilégier un binaire versionné (Homebrew/mise/binaire release),
   - ne pas l'ajouter au `tool (...)` principal du `go.mod`.
4. Une fois la parité stabilisée, décider si on active d'autres linters (`ineffassign`, `unused`, `errcheck`, etc.) dans un second lot.

## Configuration cible proposée

Fichier `.golangci.yml` (phase 1, parité staticcheck):

```yaml
version: "2"

linters:
  default: none
  enable:
    - staticcheck

run:
  timeout: 5m
  tests: true
  modules-download-mode: readonly
  relative-path-mode: gomod

issues:
  max-issues-per-linter: 0
  max-same-issues: 0

output:
  formats:
    text:
      path: stdout
      print-linter-name: true
      print-issued-lines: true
```

Pourquoi ce profil:
- `default: none` + `staticcheck` = migration low-risk et comportement proche de l'existant.
- `modules-download-mode: readonly` protège le `go.mod`/`go.sum` en CI.
- `timeout` explicite évite les jobs bloqués.

## Plan d'exécution par patch/commit

### Patch 1 - Introduire la config v2 sans changer la CI

- Ajouter `.golangci.yml` (profil parité ci-dessus).
- Ajouter une commande de validation locale (documentation) :
  - `golangci-lint config verify`
  - `golangci-lint run`
- Commit suggéré:
  - `build: add golangci-lint v2 baseline config`

### Patch 2 - Remplacer `staticcheck` dans le Makefile

- `lint`: remplacer `go tool staticcheck ./...` par `golangci-lint run`.
- `help/tools`: ajuster les messages pour l'installation de `golangci-lint`.
- Commit suggéré:
  - `build: switch make lint to golangci-lint`

### Patch 3 - Remplacer l'étape CI lint

- Dans `.github/workflows/ci.yml`, job `lint`:
  - garder `gofmt` et `go vet`,
  - remplacer l'étape `Run staticcheck` par l'action:
    - `uses: golangci/golangci-lint-action@v9`
    - `with.version: v2.8.0` (latest observé le 2026-03-05; à réévaluer au moment du patch).
- Commit suggéré:
  - `ci: use golangci-lint action for lint job`

### Patch 4 - Nettoyage des dépendances d'outillage

- Supprimer `staticcheck` du bloc `tool (...)` dans `go.mod`.
- Ajuster la cible `tools` (ne plus installer `staticcheck`).
- Mettre à jour CONTRIBUTING si besoin (prérequis lint).
- Commit suggéré:
  - `chore: remove staticcheck tool dependency`

### Patch 5 - Stabilisation (optionnelle)

- Ajouter une étape CI explicite:
  - `golangci-lint config verify`
- Évaluer l'activation graduelle de linters additionnels (lot séparé, PR dédiée).
- Commit suggéré:
  - `ci: add golangci-lint config verification`

## Critères d'acceptation

- `make check-all` passe en local.
- Job `lint` CI passe de façon stable sur `main` et PR.
- Aucune modification implicite de `go.mod`/`go.sum` pendant lint.
- Qualité équivalente à l'existant (au minimum parité `staticcheck`).

## Risques et mitigation

- Risque: explosion de findings si on active les linters standards trop tôt.
  - Mitigation: phase 1 en parité stricte (`staticcheck` seul).
- Risque: dérive de version du linter en CI.
  - Mitigation: version épinglée (`v2.x.y` ou `v2.x`) et revue périodique.
- Risque: friction locale si `golangci-lint` absent.
  - Mitigation: documenter une voie d'installation simple (Homebrew/mise/binaire).

## Rollback

Rollback simple en 1 commit:
- rétablir `go tool staticcheck ./...` dans `Makefile`,
- remettre l'étape `Run staticcheck` dans CI,
- conserver `.golangci.yml` pour préparation future.

---

## Source Evidence Log

Consultation date: 2026-03-05

| URL | Publication / last updated | Impact sur la décision |
|---|---|---|
| https://golangci-lint.run/docs/product/migration-guide/ | Last updated on 2026-03-04 01:28:06 | Règles de migration v1->v2 (`default: none`, fusion `gosimple/stylecheck/staticcheck`, flags retirés). |
| https://golangci-lint.run/docs/configuration/file/ | Last updated on 2026-03-04 01:28:06 | Structure `.golangci.yml` v2 (`version`, `linters`, `run`, `output`, `issues`). |
| https://golangci-lint.run/docs/configuration/cli/ | Last updated on 2026-03-04 01:28:06 | Flags `run`, timeout désactivé par défaut, `config verify`, commande `migrate`. |
| https://golangci-lint.run/docs/welcome/install/ci/ | Last updated on 2026-03-04 01:28:06 | Reco officielle CI: version épinglée + action GitHub officielle. |
| https://github.com/golangci/golangci-lint-action | n/a (README GitHub dynamique) | Paramétrage action (`@v9`, `version`, `install-mode`, cache, compatibilité v2). |
| https://golangci-lint.run/docs/welcome/install/local/ | Last updated on 2026-03-01 12:36:56 | Reco locale: binaire; avertissement contre `go tool`/tools pattern pour ce linter. |
| https://go.dev/ref/mod | n/a | Cadre officiel du `tool` directive et `go get -tool` (évaluation des options d'outillage). |
| https://go.dev/doc/modules/managing-dependencies | n/a | Gouvernance des dépendances et reproductibilité des builds. |
