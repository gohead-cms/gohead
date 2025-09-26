// 📁 src/features/dashboard/hooks/useAgentStatus.tsx
import { useState, useEffect } from "react";
import { AgentStatus } from "../../../shared/types/dashboard"

export function useAgentStatus() {
  const [agents, setAgents] = useState<AgentStatus[] | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchAgentStatus = async () => {
      try {
        setIsLoading(true);
        // Replace with actual API call
        const mockAgents: AgentStatus[] = [
          {
            id: "1",
            name: "SEO Agent",
            status: "active",
            queueCount: 3,
            lastExecution: "2 min ago",
            successRate: 98,
          },
          {
            id: "2",
            name: "Translation Agent",
            status: "idle",
            queueCount: 0,
            lastExecution: "1 hour ago",
            successRate: 94,
          },
          {
            id: "3",
            name: "Moderation Agent",
            status: "high_load",
            queueCount: 15,
            lastExecution: "30 sec ago",
            successRate: 89,
          },
          {
            id: "4",
            name: "Content Generator",
            status: "error",
            queueCount: 8,
            lastExecution: "Failed",
            successRate: 76,
          },
        ];
        
        await new Promise(resolve => setTimeout(resolve, 600));
        setAgents(mockAgents);
      } catch (err) {
        setError("Failed to fetch agent status");
      } finally {
        setIsLoading(false);
      }
    };

    fetchAgentStatus();
  }, []);

  return { agents, isLoading, error };
}
