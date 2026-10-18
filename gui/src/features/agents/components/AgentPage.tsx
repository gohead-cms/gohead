import { useState } from "react";
import { Flex, Box, Spinner, Text, Alert, AlertIcon } from "@chakra-ui/react";
import { useAgentsList } from "../hook/useAgentList";
import { useAgentDetails } from "../hook/useAgentDetails";
import { AgentSidebar } from "./AgentSidebar";
import { AgentDetails } from "./AgentDetails";

export default function AgentsPage() {
  const [chosenAgent, setSelectedAgent] = useState<string | null>(null);
  const { agents, loading: listLoading, error: listError } = useAgentsList();
  // The first agent is shown until another is chosen.
  const selectedAgent = chosenAgent ?? agents[0]?.name ?? null;
  const {
    agentDetails,
    loading: detailsLoading,
    error: detailsError,
  } = useAgentDetails(selectedAgent);

  const error = listError || detailsError;

  return (
    <Flex h="100%" w="100%">
      <AgentSidebar
        agents={agents}
        selectedAgent={selectedAgent}
        onSelectAgent={setSelectedAgent}
        loading={listLoading}
      />

      <Box flex="1" p={10} overflowY="auto">
        {error && (
          <Alert status="error">
            <AlertIcon />
            {error}
          </Alert>
        )}
        {detailsLoading ? (
          <Spinner />
        ) : agentDetails ? (
          <AgentDetails agent={agentDetails} />
        ) : !listLoading ? (
          <Text color="gray.400">Select an agent to view its details</Text>
        ) : null}
      </Box>
    </Flex>
  );
}
