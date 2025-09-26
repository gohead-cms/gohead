// 📁 src/features/dashboard/hooks/useSystemHealth.tsx
import { useState, useEffect } from "react";
import { SystemHealthMetric } from '../../../shared/types/dashboard'

export function useSystemHealth() {
  const [metrics, setMetrics] = useState<SystemHealthMetric[] | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchSystemHealth = async () => {
      try {
        setIsLoading(true);
        // Replace with actual API call
        const mockMetrics: SystemHealthMetric[] = [
          {
            name: "API Response Time",
            value: "120ms",
            status: "healthy",
            description: "Average response time",
          },
          {
            name: "Database Performance",
            value: "98.5%",
            status: "healthy",
            description: "Query success rate",
          },
          {
            name: "OpenAI API",
            value: "Available",
            status: "healthy",
            description: "External service status",
          },
          {
            name: "Storage Usage",
            value: "78%",
            status: "warning",
            description: "Disk space utilization",
          },
          {
            name: "Memory Usage",
            value: "45%",
            status: "healthy",
            description: "RAM utilization",
          },
        ];
        
        await new Promise(resolve => setTimeout(resolve, 700));
        setMetrics(mockMetrics);
      } catch (err) {
        setError("Failed to fetch system health");
      } finally {
        setIsLoading(false);
      }
    };

    fetchSystemHealth();
  }, []);

  return { metrics, isLoading, error };
}