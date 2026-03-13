# Temporal Validity Redesign

## Problem

The current feature flag model conflates flag identity with temporal ranges. Every temporal range row duplicates identity fields (name, description, default, variations). Creating a flag requires temporal fields (validFrom) even though they're conceptually separate concerns. This causes:

- Data duplication across range rows
- Confusing create flow (must provide temporal range to create a flag)
- No clear separation between "what is this flag" and "when/how does it behave"

## Design

Split the single `feature_flags` table into three entities: **FeatureFlag** (identity), **FlagRange** (time windows), and **RangeVersion** (rules with draft/publish workflow).

### Entity Model

#### FeatureFlag (identity — stored once per flag)

| Field | Type | Description |
|-------|------|-------------|
| id | BIGSERIAL | Primary key |
| project | VARCHAR(255) | Project scope |
| stage | VARCHAR(255) | Stage scope |
| key | VARCHAR(255) | Unique flag key within project+stage |
| name | VARCHAR(255) | Display name |
| description | TEXT | Optional description |
| enabled | BOOLEAN | Global kill switch |
| default_key | VARCHAR(255) | Default variation key |
| variations | JSONB | Array of variation definitions |
| created_at | TIMESTAMPTZ | Creation timestamp |
| updated_at | TIMESTAMPTZ | Last update timestamp |

Unique constraint: `(project, stage, key)`.

#### FlagRange (time window — multiple per flag)

| Field | Type | Description |
|-------|------|-------------|
| id | BIGSERIAL | Primary key |
| flag_id | BIGINT | FK to feature_flags(id) ON DELETE CASCADE |
| active | BOOLEAN | Whether this range participates in evaluation |
| valid_from | TIMESTAMPTZ | Start of validity window (inclusive) |
| valid_to | TIMESTAMPTZ | End of validity window (exclusive), NULL = unbounded |
| created_at | TIMESTAMPTZ | Creation timestamp |
| updated_at | TIMESTAMPTZ | Last update timestamp |

Ranges for the same flag must not overlap. Overlap validation happens on create, update, and activate.

#### RangeVersion (rules + draft/publish lifecycle)

| Field | Type | Description |
|-------|------|-------------|
| id | BIGSERIAL | Primary key |
| range_id | BIGINT | FK to flag_ranges(id) ON DELETE CASCADE |
| version | INTEGER | Sequential version number (1, 2, 3...) |
| status | VARCHAR(20) | "draft" or "published" |
| rules | JSONB | Targeting rules for this version |
| published_at | TIMESTAMPTZ | When this version was published (NULL if draft) |
| created_at | TIMESTAMPTZ | Creation timestamp |

Unique constraint: `(range_id, version)`.

Invariants:
- A range has at most one draft version at a time (always the latest)
- Multiple published versions exist as history
- Evaluation uses the latest published version (highest version number with status=published)
- Published versions are immutable and cannot be deleted
- Rollback when a draft already exists overwrites the existing draft's rules (does not create a new version number)

### Go Structs

```go
type FeatureFlag struct {
    ID          int64       `json:"id"`
    Project     string      `json:"project"`
    Stage       string      `json:"stage"`
    Key         string      `json:"key"`
    Name        string      `json:"name"`
    Description string      `json:"description,omitempty"`
    Enabled     bool        `json:"enabled"`
    DefaultKey  string      `json:"defaultKey"`
    Variations  []Variation `json:"variations"`
    CreatedAt   time.Time   `json:"createdAt"`
    UpdatedAt   time.Time   `json:"updatedAt"`
}

type FlagRange struct {
    ID        int64      `json:"id"`
    FlagID    int64      `json:"flagId"`
    Active    bool       `json:"active"`
    ValidFrom time.Time  `json:"validFrom"`
    ValidTo   *time.Time `json:"validTo,omitempty"`
    CreatedAt time.Time  `json:"createdAt"`
    UpdatedAt time.Time  `json:"updatedAt"`
}

type RangeVersion struct {
    ID          int64      `json:"id"`
    RangeID     int64      `json:"rangeId"`
    Version     int        `json:"version"`
    Status      string     `json:"status"`
    Rules       []Rule     `json:"rules"`
    PublishedAt *time.Time `json:"publishedAt,omitempty"`
    CreatedAt   time.Time  `json:"createdAt"`
}
```

### Repository Interface

```go
type Repository interface {
    // Flag identity CRUD
    CreateFlag(ctx context.Context, flag FeatureFlag) (FeatureFlag, error)
    GetFlag(ctx context.Context, project, stage, key string) (FeatureFlag, error)
    GetFlagByID(ctx context.Context, id int64) (FeatureFlag, error)
    UpdateFlag(ctx context.Context, flag FeatureFlag) error
    DeleteFlag(ctx context.Context, id int64) error
    ListFlags(ctx context.Context, project, stage string) ([]FeatureFlag, error)

    // Range CRUD
    CreateRange(ctx context.Context, r FlagRange) (FlagRange, error)
    GetRange(ctx context.Context, id int64) (FlagRange, error)
    UpdateRange(ctx context.Context, r FlagRange) error
    DeleteRange(ctx context.Context, id int64) error
    ListRanges(ctx context.Context, flagID int64) ([]FlagRange, error)
    ActivateRange(ctx context.Context, id int64) error
    DeactivateRange(ctx context.Context, id int64) error
    GetActiveRange(ctx context.Context, flagID int64, at time.Time) (FlagRange, error)
    CheckRangeOverlap(ctx context.Context, flagID int64, validFrom time.Time, validTo *time.Time, excludeID int64) (bool, error)

    // Version CRUD
    CreateVersion(ctx context.Context, v RangeVersion) (RangeVersion, error)
    GetVersion(ctx context.Context, id int64) (RangeVersion, error)
    UpdateVersion(ctx context.Context, v RangeVersion) error
    DeleteDraftVersion(ctx context.Context, id int64) error
    ListVersions(ctx context.Context, rangeID int64) ([]RangeVersion, error)
    GetPublishedVersion(ctx context.Context, rangeID int64) (RangeVersion, error)
    PublishVersion(ctx context.Context, id int64) error
}
```

### Evaluation Flow

```
EvaluateFlag(project, stage, key, attrs):
  1. repo.GetFlag(project, stage, key) → FeatureFlag
  2. if !flag.Enabled → return default, reason=DISABLED
  3. repo.GetActiveRange(flag.ID, now) → FlagRange, err
  4. if no active range → return default, reason=NO_ACTIVE_RANGE
  5. repo.GetPublishedVersion(range.ID) → RangeVersion, err
  6. if no published version → return default, reason=NO_PUBLISHED_VERSION
  7. evaluator.Evaluate(flag, version, attrs) → result
```

The evaluator signature changes to: `Evaluate(ctx context.Context, flag FeatureFlag, v RangeVersion, attrs EvaluationContext) (EvaluationResult, error)`. It reads variations/defaultKey from the flag and rules from the version.

New evaluation reasons:
- `NO_ACTIVE_RANGE` — flag exists but no active range covers the current time
- `NO_PUBLISHED_VERSION` — active range exists but has no published version (only a draft)

### API Endpoints

#### Flag identity

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/admin/{project}/{stage}/flags` | List flags |
| POST | `/api/v1/admin/{project}/{stage}/flags` | Create flag (identity only) |
| GET | `/api/v1/admin/{project}/{stage}/flags/{key}` | Get flag by key |
| GET | `/api/v1/admin/flags/{id}` | Get flag by ID |
| PUT | `/api/v1/admin/flags/{id}` | Update flag identity |
| DELETE | `/api/v1/admin/flags/{id}` | Delete flag (cascades ranges + versions) |

#### Ranges

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/admin/flags/{flagId}/ranges` | List all ranges for a flag |
| POST | `/api/v1/admin/flags/{flagId}/ranges` | Create range + v1 draft |
| GET | `/api/v1/admin/ranges/{id}` | Get range by ID |
| PUT | `/api/v1/admin/ranges/{id}` | Update range (time window) |
| DELETE | `/api/v1/admin/ranges/{id}` | Delete range |
| POST | `/api/v1/admin/ranges/{id}/activate` | Activate range (requires published version) |
| POST | `/api/v1/admin/ranges/{id}/deactivate` | Deactivate range |

#### Versions

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/admin/ranges/{rangeId}/versions` | List all versions |
| POST | `/api/v1/admin/ranges/{rangeId}/versions` | Create new draft version |
| GET | `/api/v1/admin/versions/{id}` | Get version by ID |
| PUT | `/api/v1/admin/versions/{id}` | Update draft version (rules) |
| DELETE | `/api/v1/admin/versions/{id}` | Discard draft version (drafts only) |
| POST | `/api/v1/admin/versions/{id}/publish` | Publish a draft |
| POST | `/api/v1/admin/versions/{id}/rollback` | Overwrite current draft with this version's rules (creates draft if none exists) |

#### Evaluation (unchanged contract)

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/api/v1/{project}/{stage}/flags/{key}/evaluate` | Evaluate flag |

#### Audit logs

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/admin/{project}/{stage}/flags/{key}/audit` | Get audit history for a flag |

### Version Lifecycle

1. **Create range** → v1 created as `draft` with provided rules (or empty)
2. **Edit draft** → update rules on the draft version in place
3. **Publish** → draft status becomes `published`, `publishedAt` set
4. **Edit published range** → `POST /ranges/{rangeId}/versions` creates new version (v2) as `draft`, published v1 stays live
5. **Publish v2** → v2 becomes `published`, v1 remains as history
6. **Rollback to v1** → if a draft exists, overwrites its rules with v1's rules; if no draft exists, creates new version as `draft` with v1's rules
7. **Discard draft** → `DELETE /versions/{id}` removes the draft version (only drafts can be deleted; published versions are immutable)

### Activation Rules

- `ActivateRange` validates that the range has at least one published version (service layer check)
- Re-enabling a flag (`enabled = true`) immediately resumes evaluation for all active ranges — the frontend should warn about this

### Transaction Boundaries

The following operations must be atomic (single database transaction):
- **Create range + v1 draft**: if version creation fails, range must not exist
- **Publish version**: must atomically set status and publishedAt
- **Rollback with draft overwrite**: must atomically update the existing draft's rules

### Migration Strategy

The existing single-table `feature_flags` must be split. This migration is **one-way** (no down migration).

1. Create new `feature_flags` table (identity)
2. Create `flag_ranges` table
3. Create `range_versions` table
4. For each distinct `(project, stage, key)` in the old table:
   - Insert one identity row using fields from the most recently updated row (name, description, enabled, default_key, variations extracted from config JSONB)
   - For each old row with that key: insert a `flag_ranges` row (active, valid_from, valid_to), and a `range_versions` row (v1, status=published, rules extracted from config JSONB)
5. Add optional `range_id` column to `audit_logs` for range-specific entries
6. Null out old `flag_id` values in `audit_logs` (they reference the old table's IDs which no longer exist; historical entries are still queryable by `flag_key`)
7. Drop old table

Note: the old `version` column (used for optimistic locking) is not carried over. Optimistic locking uses `updated_at` timestamp comparison instead.

Note: when multiple old rows exist for the same key with different `enabled` values, the most recently updated row's value wins. Operators should verify flag states after migration.

Memory repository: restructure in-memory maps to three separate stores.

### Frontend Impact

**FlagsPage (list):**
- One row per flag (no duplication)
- `enabled` toggle on identity
- No temporal fields in list view

**Create Flag modal:**
- Only: key, name, description, default, variations
- No temporal fields needed

**FlagDetailPage:**
- Top: flag identity (name, description, default, variations, enabled toggle)
- Below: ranges list showing validFrom, validTo, active badge, draft indicator, rule count
- Expand range to see: published rules, version history, draft editor

**Draft workflow in UI:**
1. "Edit Rules" on published range → creates new draft version
2. Edit freely, saves update draft
3. "Publish" with confirmation → goes live
4. "Discard Draft" to abandon

**New range workflow:**
1. "Add Range" → set validFrom, validTo
2. Range created with v1 draft → edit rules
3. Publish v1, then activate range

**Rollback:**
- In version history, "Rollback to this version" → creates new draft with that version's rules
- Review, then publish

### Audit Logging

Existing `audit_logs` table gains an optional `range_id` column. Actions recorded:

- Flag level: CREATE, UPDATE, DELETE, ENABLE, DISABLE
- Range level: CREATE_RANGE, UPDATE_RANGE, DELETE_RANGE, ACTIVATE, DEACTIVATE
- Version level: CREATE_VERSION, UPDATE_VERSION, PUBLISH, ROLLBACK

Each entry stores before/after state as JSONB.
