import { SimpleGrid } from "@chakra-ui/react";
import { FiDatabase, FiCpu, FiFileText, FiActivity } from "react-icons/fi";
import { StatCard } from "./StatCard";
import { DashboardStats } from "../../../shared/types"

interface SystemOverviewProps {
  stats: DashboardStats;
}

export function SystemOverview({ stats }: SystemOverviewProps) {
  return (
    <SimpleGrid columns={{ base: 1, md: 2, lg: 4 }} spacing={6} mb={8}>
      <StatCard
        title="Total Collections"
        value={stats.totalCollections}
        icon={FiDatabase}
        change={12}
        colorScheme="blue"
      />
      <StatCard
        title="Active Agents"
        value={stats.activeAgents}
        icon={FiCpu}
        change={8}
        colorScheme="green"
      />
      <StatCard
        title="Content Items"
        value={stats.contentItems.toLocaleString()}
        icon={FiFileText}
        change={15}
        colorScheme="purple"
      />
      <StatCard
        title="API Calls Today"
        value={stats.apiCallsToday.toLocaleString()}
        icon={FiActivity}
        change={-3}
        colorScheme="orange"
      />
    </SimpleGrid>
  );
}
