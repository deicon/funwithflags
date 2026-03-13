export interface Project {
  key: string;
  name: string;
  description?: string;
  createdAt: string;
  updatedAt: string;
}

export interface Stage {
  projectKey: string;
  key: string;
  name: string;
  description?: string;
  createdAt: string;
  updatedAt: string;
}

export type VariationType = "boolean" | "string" | "number" | "object";

export interface Variation {
  key: string;
  type: VariationType;
  value: unknown;
  description?: string;
}

export type MatcherOperator =
  | "equals"
  | "not_equals"
  | "contains"
  | "starts_with"
  | "ends_with"
  | "greater_than"
  | "less_than"
  | "in"
  | "exists";

export interface Condition {
  attribute: string;
  operator: MatcherOperator;
  value: unknown;
}

export interface RolloutBucket {
  variationKey: string;
  weight: number;
}

export interface PercentageRollout {
  attribute: string;
  seed: string;
  buckets: RolloutBucket[];
}

export interface Rule {
  id: string;
  description?: string;
  conditions?: Condition[];
  variationKey?: string;
  rollout?: PercentageRollout;
}

export interface FeatureFlag {
  id: number;
  project: string;
  stage: string;
  key: string;
  name: string;
  description?: string;
  enabled: boolean;
  defaultKey: string;
  variations: Variation[];
  createdAt: string;
  updatedAt: string;
}

export interface FlagRange {
  id: number;
  flagId: number;
  active: boolean;
  validFrom: string;
  validTo?: string;
  createdAt: string;
  updatedAt: string;
}

export type VersionStatus = "draft" | "published";

export interface RangeVersion {
  id: number;
  rangeId: number;
  version: number;
  status: VersionStatus;
  rules: Rule[];
  publishedAt?: string;
  createdAt: string;
}

export interface AuthUser {
  username: string;
  role: string;
}
