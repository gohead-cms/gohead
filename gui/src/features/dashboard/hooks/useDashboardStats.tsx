

import { useState, useEffect } from "react";
import { DashboardStats } from "../../../shared/types/dashboard"

export function useDashboardStats() {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        setIsLoading(true);
        // Replace with actual API call
        const mockStats: DashboardStats = {
          totalCollections: 12,
          activeAgents: 8,
          contentItems: 1247,
          apiCallsToday: 2341,
        };
        
        // Simulate API delay
        await new Promise(resolve => setTimeout(resolve, 1000));
        
        setStats(mockStats);
      } catch (err) {
        setError("Failed to fetch dashboard stats");
      } finally {
        setIsLoading(false);
      }
    };

    fetchStats();
  }, []);

  return { stats, isLoading, error };
}
