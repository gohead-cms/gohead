// 📁 src/features/dashboard/components/DashboardLayout.tsx
import React from "react";
import {
  Box,
  Container,
  VStack,
  Grid,
  GridItem,
  Text,
} from "@chakra-ui/react";
import { 
  SystemOverview,
  ActivityFeed,
  AgentStatusPanel,
  SystemHealth,
  QuickActions 
} from './components'
import { 
  useDashboardStats, 
  useActivityFeed,
  useAgentStatus,
  useSystemHealth
} from "./hooks";

export function DashboardPage() {
  const { stats, isLoading: statsLoading } = useDashboardStats();
  const { activities, isLoading: activitiesLoading } = useActivityFeed();
  const { agents, isLoading: agentsLoading } = useAgentStatus();
  const { metrics, isLoading: healthLoading } = useSystemHealth();

  if (statsLoading || activitiesLoading || agentsLoading || healthLoading) {
    return (
      <Box p={8}>
        <Text>Loading dashboard...</Text>
      </Box>
    );
  }

  return (
    <Container maxW="8xl" py={8}>
      <VStack spacing={8} align="stretch">
        <Box>
          <Text fontSize="3xl" fontWeight="bold" mb={2}>
            Dashboard
          </Text>
          <Text color="gray.600">
            Overview of your GoHead CMS system
          </Text>
        </Box>

        {/* System Overview Stats */}
        <SystemOverview stats={stats!} />

        {/* Main Content Grid */}
        <Grid templateColumns={{ base: "1fr", lg: "2fr 1fr" }} gap={8}>
          <GridItem>
            <VStack spacing={6} align="stretch">
              <ActivityFeed activities={activities!} />
              <AgentStatusPanel agents={agents!} />
            </VStack>
          </GridItem>
          <GridItem>
            <VStack spacing={6} align="stretch">
              <QuickActions />
              <SystemHealth metrics={metrics!} />
            </VStack>
          </GridItem>
        </Grid>
      </VStack>
    </Container>
  );
}