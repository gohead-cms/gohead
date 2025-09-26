// 📁 src/features/dashboard/hooks/useActivityFeed.tsx
import { useState, useEffect } from "react";
import { ActivityItem } from "../../../shared/types/dashboard"

export function useActivityFeed() {
  const [activities, setActivities] = useState<ActivityItem[] | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchActivities = async () => {
      try {
        setIsLoading(true);
        // Replace with actual API call
        const mockActivities: ActivityItem[] = [
          {
            id: "1",
            type: "content",
            title: "New article created",
            description: "Article 'AI in Modern Development' was created",
            timestamp: "2 minutes ago",
            status: "success",
            user: "John Doe",
          },
          {
            id: "2",
            type: "agent",
            title: "SEO optimization completed",
            description: "SEO agent processed 5 articles",
            timestamp: "5 minutes ago",
            status: "success",
          },
          {
            id: "3",
            type: "system",
            title: "Database backup",
            description: "Scheduled backup completed successfully",
            timestamp: "1 hour ago",
            status: "success",
          },
          {
            id: "4",
            type: "agent",
            title: "Translation failed",
            description: "Translation agent encountered API limit",
            timestamp: "2 hours ago",
            status: "error",
          },
        ];
        
        await new Promise(resolve => setTimeout(resolve, 800));
        setActivities(mockActivities);
      } catch (err) {
        setError("Failed to fetch activity feed");
      } finally {
        setIsLoading(false);
      }
    };

    fetchActivities();
  }, []);

  return { activities, isLoading, error };
}
