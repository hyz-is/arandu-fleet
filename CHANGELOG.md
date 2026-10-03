# Changelog

Everything worth knowing about a release of Arandu Fleet is recorded here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
the versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

A published module version is immutable: Go serves it from the proxy forever, so
a release is corrected by another release and never by moving a tag.

## [Unreleased]

### Added

- `StopAndRelease`, with `StopControl`, `StopObservation` and `ErrUnquiesced`:
  it cancels a run, confirms quiescence on every selected node, and only then
  releases the run's reservations. Unknown, missing or unrelated terminal
  receipts release nothing; while termination or release is unconfirmed the
  reservations stay held and the call answers `ErrUnquiesced`.
- `HTTPWorker.SubmitTimeout` bounds job admission separately from status and
  cancellation. Zero keeps `Client.Timeout`, and a negative value is refused.
- The `host` package, host measurements an installation calls explicitly:
  `ProbeGPUs` and `ParseGPUs` read the cards through `nvidia-smi` and refuse
  rather than report zero; `ReadAvailableRAM` reads `MemAvailable` from a named
  meminfo file and `CgroupAvailable` a cgroup v2 limit and usage; `RAMBudget`,
  `EstimateRAM` and `AdmitRAM` make the RAM admission budget explicit; and
  `Housekeeping`, `Cleanup` and `Process` say what a node is allowed to clean.

### Changed

- **Breaking.** `Fleet` embeds the non-generic `model.Model`, and its table is
  declared once beside it with `model.NewTable`. `Fleets` takes a `model.DB`
  and returns the generated `*FleetQuery`; `Get` returns `FleetCollection` and
  `New` replaces `NewInstance(nil, false)`. The fields and methods
  `model.Model[Fleet]` promoted onto `Fleet` are gone, and `Exists` is a
  method. `UPGRADE.md` names every symbol.
- Requires Hesape `v0.48.0` and Framework `v0.50.2`; `arandu.mod.toml` declares
  `framework = ">= 0.50"`. The store route reads `name` from the body of the
  `POST` only, as Hesape now reads every `POST`. Routes, migrations, actions,
  policy decisions, tenant scoping and the control plane are unchanged.

### Fixed

- Reserving nodes for a run copies the run's node list, so reconciliation no
  longer rewrites a dispatch receipt its caller kept.

## [0.3.0] - 2026-09-15

### Added

- `Job` and persisted `NodeRun` identity now carry `model_recipe` and
  `model_digest`, so a retry cannot silently reuse a process launched for a
  different admitted model representation.

### Changed

- Agent reconciliation fences current processes by model identity in addition
  to execution, generation, action, contract, and runtime identity.

### Fixed

- Release verification no longer assumes a package-local `configure.go` exists.

## [0.2.0] - 2026-09-13

### Added

- The existing record API is now explicitly recorded with its five guarded
  actions: `FleetRecordView`, `FleetRecordList`, `FleetRecordCreate`,
  `FleetRecordUpdate` and `FleetRecordDelete`. Its schema remains migration
  `20260823_0001_create_fleets`.
- Durable node process identity, generation fencing and reconciliation after an
  agent restart. A restored process is trusted only when its PID, Linux start
  time and executable still match; an unprovable process is reported as
  `unknown`.
- Idempotent submission and cancellation for the same fenced job, graceful
  process-group termination followed by a bounded forced stop, and persisted
  final output retrieved through `GET /jobs/{id}/result` with a SHA-256 digest.
- Exact-node status, dispatch, cancellation and release operations so one
  execution can reserve disjoint RPC and coordinator phases without releasing
  the cluster between them.

### Changed

- `Job`, `NodeRun` and `Run` now carry generation and runtime identity fields.
  External unkeyed composite literals must move to keyed fields when upgrading.
- Result logs are limited to 8 MiB at the agent and to a bounded escaped JSON
  response at the HTTP client.

[Unreleased]: https://github.com/tayi-ai/arandu-fleet/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/tayi-ai/arandu-fleet/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/tayi-ai/arandu-fleet/compare/v0.1.2...v0.2.0
