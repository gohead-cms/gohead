// 📁 src/features/dashboard/components/AgentStatusPanel.tsx
import React from "react";
import {
  Box,
  Text,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  TableContainer,
  Badge,
  Progress,
  HStack,
  Icon,
  VStack,
} from "@chakra-ui/react";
import {
  FiCheckCircle,
  FiClock,
  FiAlertTriangle,
  FiActivity,
} from "react-icons/fi";
import { AgentStatus } from "../../../shared/types/dashboard"

const statusIcons: Record<string, React.ElementType> = {
  active: FiActivity,
  idle: FiClock,
  error: FiAlertTriangle,
  high_load: FiAlertTriangle,
};

const statusColors: Record<string, string> = {
  active: "green",
  idle: "gray",
  error: "red",
  high_load: "yellow",
};

interface AgentStatusPanelProps {
  agents: AgentStatus[];
}

export function AgentStatusPanel({ agents }: AgentStatusPanelProps) {
  return (
    <Box
      bg="white"
      borderWidth="1px"
      borderColor="gray.200"
      borderRadius="xl"
      p={6}
    >
      <Text fontSize="lg" fontWeight="bold" mb={4}>
        Agent Status & Performance
      </Text>
      <TableContainer>
        <Table variant="simple" size="sm">
          <Thead>
            <Tr>
              <Th>Agent</Th>
              <Th>Status</Th>
              <Th>Queue</Th>
              <Th>Success Rate</Th>
              <Th>Last Run</Th>
            </Tr>
          </Thead>
          <Tbody>
            {agents.map((agent) => (
              <Tr key={agent.id}>
                <Td>
                  <Text fontWeight="medium">{agent.name}</Text>
                </Td>
                <Td>
                  <Badge
                    colorScheme={statusColors[agent.status]}
                    variant="subtle"
                  >
                    <Icon as={statusIcons[agent.status]} boxSize={3} mr={1} />
                    {agent.status.replace('_', ' ')}
                  </Badge>
                </Td>
                <Td>
                  <Text fontSize="sm">
                    {agent.queueCount > 0 ? `${agent.queueCount} items` : '—'}
                  </Text>
                </Td>
                <Td>
                  <VStack spacing={1} align="stretch">
                    <HStack justify="space-between">
                      <Text fontSize="xs">{agent.successRate}%</Text>
                    </HStack>
                    <Progress
                      value={agent.successRate}
                      size="xs"
                      colorScheme={agent.successRate > 90 ? "green" : agent.successRate > 70 ? "yellow" : "red"}
                    />
                  </VStack>
                </Td>
                <Td>
                  <Text fontSize="xs" color="gray.600">
                    {agent.lastExecution}
                  </Text>
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </TableContainer>
    </Box>
  );
}