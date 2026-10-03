# Upgrade Guide

## v0.4.0

### The fleet record is a concrete type over the non-generic model

Hesape `v0.47.0` removes the generic model layer, and this release moves to it
with Hesape `v0.48.0` and Framework `v0.50.2`. `Fleet` embeds the non-generic
`model.Model`, its table is declared once beside it with `model.NewTable`, and
the query that starts from it is generated beside it by `aru model:build`, in
`FleetQuery.go`. No route, migration, action, policy decision, tenant rule or
part of the control plane changed.

**`Fleets` returns the generated query.** It takes a `model.DB` -- a `*data.DB`
passes unchanged -- and returns `*fleet.FleetQuery` instead of
`*model.Model[fleet.Fleet]`. A chain that started from it keeps its text, minus
the calls that no longer exist:

| before | now |
|---|---|
| `fleet.Fleets(db).NewQuery().Where(…)` | `fleet.Fleets(db).Where(…)` |
| `fleet.Fleets(db).NewInstance(nil, false)` and `.Entity` | `fleet.Fleets(db).New()`, which returns `*Fleet` |
| `Get` → `model.Collection[fleet.Fleet]` | `Get` → `fleet.FleetCollection` (`[]*Fleet`) |
| `func(q *model.Builder[fleet.Fleet])` in a grouped `Where` | `func(q *fleet.FleetQuery)` |
| `fleet.Fleets(db).GetTable()`, `.KeyType`, `.TenantColumn` | nothing: the table is unexported, and its settings are not read off the query |

`First`, `Find` and the other row terminals still return `*Fleet`, and still
take the Grant. Every `FleetService` method keeps its signature: `Create` and
`Find` still return `*Fleet`, and `List` still returns `[]*Fleet`.

**The entity no longer carries the model's configuration.** `Fleet` embeds
`model.Model`, so the fields and methods `model.Model[Fleet]` promoted onto it
are gone: the configuration fields (`PrimaryKey`, `KeyType`, `Incrementing`,
`Timestamps`, `TenantColumn`, `Table` and the rest) live in the table, which this
package keeps unexported, and `Exists` and `WasRecentlyCreated` are methods,
`row.Exists()`. A copied row still reads its fields but refuses every write with
`model.ErrUnwired`, so keep the pointers the queries return.

**Upgrade the floor.** The module requires Hesape `v0.48.0` and Framework
`v0.50.2`, and `arandu.mod.toml` declares `framework = ">= 0.50"`. An
application that pins a Hesape below `v0.47.0` cannot compile this release:
every generic model type it would need is gone from Hesape itself.

**What changes without a compiler error.** The store route, `POST` on the
prefix, reads `name` through `Context.Input`, and since Hesape `v0.44.0` the
input of a `POST` is its body alone: a url-encoded or multipart form, or a JSON
object. A `name` sent only in the query string of the `POST` is no longer read,
and the request is refused by validation as one with no name. The listing and
the record routes are `GET` and read the query string as before.

<details>
<summary>Every incompatible symbol <code>apidiff</code> reports against v0.3.0</summary>

Most of these are the methods and fields `model.Model[Fleet]` promoted onto
`Fleet`, which left with the generic type.

```text
Fleet.ConnectionName
Fleet.CreatedAtColumn
Fleet.DeletedAtColumn
Fleet.Entity
Fleet.Exists
Fleet.Grammar
Fleet.Incrementing
Fleet.KeyType
Fleet.NamedScopes
Fleet.PerPage
Fleet.PrimaryKey
Fleet.Processor
Fleet.RelationResolvers
Fleet.SoftDeletes
Fleet.Table
Fleet.TenantColumn
Fleet.Timestamps
Fleet.UpdatedAtColumn
Fleet.WasRecentlyCreated
Fleets
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).AddGlobalScope, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).All, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).Append, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).AttributesToArray, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).CallNamedScope, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).Create, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).Destroy, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).DiscardChanges, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).Except, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).Find, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).FindMany, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).FindOrFail, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).FindOrNew, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).First, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).FirstOrCreate, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).FirstOrNew, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).ForceCreate, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).ForceDeleteQuietly, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).ForceDeleted, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).ForceDeleting, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).ForceDestroy, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).FreshTimestamp, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetAppends, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetConnectionName, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetCreatedAtColumn, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetDeletedAtColumn, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetForeignKey, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetGlobalScopes, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetHidden, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetIncrementing, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetKeyName, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetKeyType, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetMorphClass, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetPerPage, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetPrevious, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetQualifiedCreatedAtColumn, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetQualifiedDeletedAtColumn, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetQualifiedKeyName, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetQualifiedUpdatedAtColumn, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetQueueableConnection, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetQueueableID, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetQueueableRelations, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetRawOriginal, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetRelation, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetRelations, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetRouteKey, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetRouteKeyName, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetTable, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetTouchedRelations, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetUpdatedAtColumn, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).GetVisible, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).HasAppended, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).HasGlobalScope, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).HasNamedScope, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).Is
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).IsForceDeleting, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).IsIgnoringTouch, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).IsNot, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).IsRelation, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).IsSoftDeletable, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).LoadAggregate, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).LoadMorph, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).LoadMorphAggregate, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).LoadMorphAvg, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).LoadMorphCount, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).LoadMorphMax, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).LoadMorphMin, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).LoadMorphSum, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).MakeHidden
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).MakeVisible
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).NewBaseQueryBuilder, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).NewCollection, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).NewFromBuilder, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).NewInstance, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).NewModelQuery, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).NewQuery, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).NewQueryForRestoration, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).NewQueryWithoutRelationships, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).NewQueryWithoutScope, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).NewQueryWithoutScopes, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).NewTypedBuilder, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).On, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).OnWriteConnection, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).Only, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).OnlyTrashed, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).OriginalIsEquivalent, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).PushQuietly, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).QualifyColumn, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).QualifyColumns, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).Query, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).Ref, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).RegisterGlobalScopes, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).RegisterModelEvent, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).ReplicateQuietly, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).ResolveRouteBinding, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).ResolveRouteBindingQuery, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).ResolveSoftDeletableRouteBinding, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).RestoreQuietly, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).Restored, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).Restoring, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).SetAppends, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).SetConnection, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).SetHidden, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).SetIncrementing, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).SetKeyName, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).SetKeyType, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).SetPerPage, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).SetRelation
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).SetRelations, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).SetTable, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).SetTouchedRelations, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).SetVisible, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).SoftDeleted, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).SyncChanges, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).SyncOriginal
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).SyncOriginalAttribute, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).SyncOriginalAttributes, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).ToPrettyJSON, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).Touches, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).UnsetAttribute, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).UnsetRelation, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).UnsetRelations, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).UpdateOrCreate, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).UpdateOrFail, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).UpdateQuietly, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).UpdateTimestamps, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).UsesTimestamps, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).Where, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).WhereKey, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).With, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).WithTrashed, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).WithoutRelations, method set of *Fleet
github.com/arandu-io/hesape/database/model.(*Model[github.com/tayi-ai/arandu-fleet.Fleet]).WithoutTimestamps, method set of *Fleet
```

</details>

## v0.3.0

`Job` and persisted `NodeRun` now also carry `ModelRecipe` and `ModelDigest`.
Callers using keyed literals may omit them only for actions that do not select a
model. Inference jobs should set both values from the admitted recipe. A retry
with a different model identity is rejected instead of adopting the prior
process.

## v0.2.0

`Job`, `NodeRun` and `Run` gained fields that fence a process by generation and
identify its runtime. Code using unkeyed composite literals for these types must
replace positional values with keyed fields. Existing keyed literals continue
to compile; omitted generation fields retain their zero value for the original
diagnostics and collective actions.

Agents now persist `current.json` beside their run logs. Keep that directory on
persistent storage if a worker process must reconcile an in-flight job after a
restart. A process whose Linux PID, start time and executable cannot be proven
is reported as `unknown` and is never adopted silently.
