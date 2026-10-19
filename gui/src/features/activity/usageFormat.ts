import { type AgentUsage } from "../../shared/types";
import { type AgentUsageRow } from "./useActivity";

/** The highest share of a limit used by an agent (0 when it has no limits). */
export function limitPressure(u: AgentUsage): number {
  const l = u.limits ?? {};
  const ratios = [
    l.max_runs_per_hour ? u.lastHour.runs / l.max_runs_per_hour : 0,
    l.max_runs_per_day ? u.lastDay.runs / l.max_runs_per_day : 0,
    l.max_tokens_per_day ? (u.lastDay.inputTokens + u.lastDay.outputTokens) / l.max_tokens_per_day : 0,
    l.max_cost_per_day ? u.lastDay.costUsd / l.max_cost_per_day : 0,
  ];
  return Math.max(...ratios);
}

export function hasLimits(u: AgentUsage): boolean {
  const l = u.limits ?? {};
  return Boolean(l.max_runs_per_hour || l.max_runs_per_day || l.max_tokens_per_run || l.max_tokens_per_day || l.max_cost_per_day);
}

export interface UsageTotals {
  runs: number;
  tokens: number;
  cost: number;
  atLimit: number;
}

export function usageTotals(rows: AgentUsageRow[]): UsageTotals {
  const t: UsageTotals = { runs: 0, tokens: 0, cost: 0, atLimit: 0 };
  for (const r of rows) {
    if (!r.usage) continue;
    t.runs += r.usage.lastDay.runs;
    t.tokens += r.usage.lastDay.inputTokens + r.usage.lastDay.outputTokens;
    t.cost += r.usage.lastDay.costUsd;
    if (limitPressure(r.usage) >= 1) t.atLimit += 1;
  }
  return t;
}

/** Busiest agents first; agents whose usage failed to load last. */
export function sortByActivity(rows: AgentUsageRow[]): AgentUsageRow[] {
  return [...rows].sort((a, b) => (b.usage?.lastDay.runs ?? -1) - (a.usage?.lastDay.runs ?? -1) || a.name.localeCompare(b.name));
}

/** "12 / 60" when a limit is set, "12" otherwise. */
export function withLimit(value: string, limit: string | null): string {
  return limit ? `${value} / ${limit}` : value;
}
