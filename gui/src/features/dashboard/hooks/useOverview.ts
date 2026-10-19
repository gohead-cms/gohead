import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";
import type { OverviewResponse } from "../overview";

/** Loads GET /admin/overview (admin-only: other roles get 403). */
export function useOverview() {
  return useApiQuery<OverviewResponse>("/admin/overview", async () => {
    const json = await getJSON<OverviewResponse>("/admin/overview", "Failed to load the overview");
    return json.data;
  });
}
