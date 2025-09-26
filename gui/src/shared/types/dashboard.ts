// 📁 src/features/dashboard/types/dashboard.types.ts
export interface DashboardStats {
  totalCollections: number;
  activeAgents: number;
  contentItems: number;
  apiCallsToday: number;
}

export interface ActivityItem {
  id: string;
  type: 'content' | 'agent' | 'system' | 'user';
  title: string;
  description: string;
  timestamp: string;
  status: 'success' | 'warning' | 'error' | 'info';
  user?: string;
}

export interface AgentStatus {
  id: string;
  name: string;
  status: 'active' | 'idle' | 'error' | 'high_load';
  queueCount: number;
  lastExecution: string;
  successRate: number;
}

export interface SystemHealthMetric {
  name: string;
  value: string;
  status: 'healthy' | 'warning' | 'critical';
  description: string;
}

export interface ContentInsight {
  name: string;
  value: number;
  change: number;
  trend: 'up' | 'down' | 'stable';
}

export interface QuickAction {
  name: string;
  description: string;
  icon: React.ElementType;
  href: string;
  color: string;
}

export interface NotificationItem {
  id: string;
  type: 'alert' | 'warning' | 'info' | 'success';
  title: string;
  message: string;
  timestamp: string;
  read: boolean;
}

export interface ScheduledTask {
  id: string;
  name: string;
  type: string;
  nextRun: string;
  frequency: string;
  enabled: boolean;
}