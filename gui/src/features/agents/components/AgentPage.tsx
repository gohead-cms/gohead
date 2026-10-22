import { useNavigate, useSearchParams, Link as RouterLink } from "react-router-dom";
import { Flex, Box, Spinner, Text, Button } from "@chakra-ui/react";
import { FiCpu, FiPlus } from "react-icons/fi";
import { api } from "../../../services/api";
import { useNotify } from "../../../shared/hooks/useNotify";
import { EmptyState } from "../../../shared/ui/EmptyState";
import { useMe } from "../../auth/me";
import { useAgentsList } from "../hook/useAgentList";
import { useAgentDetails } from "../hook/useAgentDetails";
import { AgentSidebar } from "./AgentSidebar";
import { AgentDetails } from "./AgentDetails";
import { Notice } from "./EditorParts";

/** Agents list and the selected agent (?name=…; the first agent otherwise; ?run=… opens one of its runs). */
export default function AgentsPage() {
  const [params, setParams] = useSearchParams();
  const navigate = useNavigate();
  const notify = useNotify();
  const { isAdmin, isDemo } = useMe();
  // The demo account looks at agents and test-runs them, nothing more.
  const canManage = isAdmin && !isDemo;
  const { agents, loading: listLoading, error: listError, refetch: refetchList } = useAgentsList();
  const selectedAgent = params.get("name") || agents[0]?.name || null;
  const { agentDetails, loading: detailsLoading, error: detailsError } = useAgentDetails(selectedAgent);

  const select = (name: string) => setParams({ name });

  const remove = async (name: string) => {
    try {
      await api("DELETE", `/admin/agents/${encodeURIComponent(name)}`);
    } catch (err) {
      notify.failure(err, "The agent was not deleted");
      return;
    }
    notify.success(`Agent ${name} deleted`);
    const next = agents.find((a) => a.name !== name)?.name;
    navigate(next ? `/agents?name=${encodeURIComponent(next)}` : "/agents", { replace: true });
    refetchList();
  };

  const error = listError || detailsError;
  const noAgents = !listLoading && !listError && agents.length === 0;

  return (
    <Flex h="100%" w="100%" minW={0}>
      <AgentSidebar agents={agents} selectedAgent={selectedAgent} onSelectAgent={select} loading={listLoading} canCreate={canManage} />

      <Box flex="1" minW={0} p={{ base: 4, md: 10 }} overflowY="auto">
        {error && (
          <Notice status="error" mb={4}>
            {error}
          </Notice>
        )}
        {noAgents ? (
          <EmptyState
            icon={FiCpu}
            title="No agents yet"
            action={
              canManage && (
                <Button asChild colorPalette="teal">
                  <RouterLink to="/agents/new">
                    <FiPlus />
                    New agent
                  </RouterLink>
                </Button>
              )
            }
          >
            An agent is an LLM with a system prompt and functions, started by a content event, a webhook, a schedule or by hand.
          </EmptyState>
        ) : detailsLoading && !agentDetails ? (
          <Spinner />
        ) : agentDetails ? (
          <AgentDetails
            // Another agent, or ?run=… opening a run, starts the details again.
            key={`${agentDetails.name}:${params.get("run") ?? ""}`}
            agent={agentDetails}
            initialRun={Number(params.get("run")) || undefined}
            onDelete={canManage ? () => remove(agentDetails.name) : undefined}
            canTest={isAdmin}
          />
        ) : !listLoading && !error ? (
          <Text color="gray.400">Select an agent to view its details</Text>
        ) : null}
      </Box>
    </Flex>
  );
}
