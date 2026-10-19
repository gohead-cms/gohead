import { useState } from "react";
import { Link as RouterLink } from "react-router-dom";
import {
  Box,
  Button,
  Code,
  Flex,
  HStack,
  Heading,
  Text,
  Badge,
  Link,
  List,
  ListItem,
  VStack,
  Tabs,
  TabList,
  Tab,
  TabPanels,
  TabPanel,
  Wrap,
} from "@chakra-ui/react";
import { FiEdit2, FiTrash2 } from "react-icons/fi";
import { API_BASE_URL } from "../../../services/api";
import { type AgentLimits, type AgentNodeData } from "../../../shared/types";
import { ConfirmButton } from "../../../shared/ui/ConfirmButton";
import { approvalsLink } from "../../approvals/proposals";
import { CopyButton } from "./EditorParts";
import { RunNowButton } from "./RunNowButton";
import { RunsPanel } from "./RunsPanel";
import { TestRunButton } from "./TestRunButton";

interface AgentDetailsProps {
  agent: AgentNodeData;
  /** Deletes the agent; omitted for read-only viewers. */
  onDelete?: () => Promise<void>;
  /** A run to show when the details open (the Runs tab, with that run open). */
  initialRun?: number;
}

function Label({ children }: { children: React.ReactNode }) {
  return (
    <Text fontWeight="bold" fontSize="sm" textTransform="uppercase" color="gray.500" mb={2}>
      {children}
    </Text>
  );
}

const limitLabels: [keyof AgentLimits, string][] = [
  ["max_runs_per_hour", "runs / hour"],
  ["max_runs_per_day", "runs / day"],
  ["max_tokens_per_run", "tokens / run"],
  ["max_tokens_per_day", "tokens / day"],
  ["max_cost_per_day", "USD / day"],
];

export function AgentDetails({ agent, onDelete, initialRun }: AgentDetailsProps) {
  const { name, schema } = agent;
  const [tab, setTab] = useState(initialRun ? 1 : 0);
  // The run to open, until the user moves on (the Runs tab remounts lazily).
  const [startRun, setStartRun] = useState(initialRun);
  // Bumped after "Run now" so the Runs tab reloads.
  const [runsVersion, setRunsVersion] = useState(0);
  /** After "Run now" or "Test run": the Runs tab, reloaded, with the new run open (it updates until it ends). */
  const follow = (runId?: number) => {
    setTab(1);
    setStartRun(runId);
    setRunsVersion((v) => v + 1);
  };
  const requireApproval = Boolean(schema.requireApproval ?? schema.require_approval);
  const { llmConfig, trigger, limits } = schema;
  const functions = schema.functions ?? [];
  const events = trigger.event_trigger?.events ?? [];
  const webhookUrl = trigger.type === "webhook" && agent.id ? `${API_BASE_URL}/agents/webhook/${agent.id}` : null;
  const setLimits = limitLabels.filter(([k]) => (limits?.[k] ?? 0) > 0);
  const priced = (llmConfig.input_cost_per_million ?? 0) > 0 || (llmConfig.output_cost_per_million ?? 0) > 0;

  return (
    <VStack spacing={4} align="stretch">
      <Flex justify="space-between" align="center" gap={3} wrap="wrap">
        <Heading size="lg" wordBreak="break-word">
          {name}
        </Heading>
        {onDelete && (
          <HStack spacing={2}>
            <RunNowButton agentName={name} onQueued={follow} />
            <TestRunButton agent={agent} onQueued={follow} />
            <Button as={RouterLink} to={`/agents/${encodeURIComponent(name)}/edit`} size="sm" leftIcon={<FiEdit2 />}>
              Edit
            </Button>
            <ConfirmButton
              trigger={
                <Button size="sm" variant="outline" colorScheme="red" leftIcon={<FiTrash2 />}>
                  Delete
                </Button>
              }
              title={`Delete ${name}?`}
              body="The agent stops running, and its run history and conversation are deleted. This cannot be undone."
              onConfirm={onDelete}
            />
          </HStack>
        )}
      </Flex>
      <Tabs
        isLazy
        colorScheme="teal"
        index={tab}
        onChange={(i) => {
          setTab(i);
          setStartRun(undefined);
        }}
      >
        <TabList>
          <Tab>Overview</Tab>
          <Tab>Runs</Tab>
        </TabList>
        <TabPanels>
          <TabPanel px={0}>
            <VStack spacing={6} align="stretch">
              {schema.systemPrompt && (
                <Box>
                  <Label>System prompt</Label>
                  <Text fontSize="sm" whiteSpace="pre-wrap" bg="white" borderWidth="1px" borderRadius="md" p={3} maxH="200px" overflowY="auto">
                    {schema.systemPrompt}
                  </Text>
                  {schema.maxTurns ? (
                    <Text fontSize="sm" color="gray.500" mt={1}>
                      Up to {schema.maxTurns} turns per run
                    </Text>
                  ) : null}
                </Box>
              )}
              <Box>
                <Label>LLM Config</Label>
                <Text>
                  Provider: <Badge colorScheme="teal">{llmConfig.provider}</Badge>
                </Text>
                <Text>
                  Model: <Badge colorScheme="cyan">{llmConfig.model || "default"}</Badge>
                </Text>
                {priced && (
                  <Text fontSize="sm" color="gray.500" mt={1}>
                    ${llmConfig.input_cost_per_million ?? 0} / ${llmConfig.output_cost_per_million ?? 0} per
                    million input / output tokens
                  </Text>
                )}
              </Box>
              <Box>
                <Label>Trigger</Label>
                <Text>
                  Type: <Badge colorScheme="purple">{trigger.type}</Badge>
                </Text>
                {trigger.type === "collection_event" && trigger.event_trigger && (
                  <Text>
                    On: '{trigger.event_trigger.collection}' [{events.join(", ")}]
                  </Text>
                )}
                {trigger.type === "cron" && trigger.cron && (
                  <Text>
                    Schedule: <Code>{trigger.cron}</Code> (UTC)
                  </Text>
                )}
                {webhookUrl && (
                  <HStack mt={1} spacing={2} flexWrap="wrap">
                    <Code fontSize="sm" wordBreak="break-all">
                      POST {webhookUrl}
                    </Code>
                    <CopyButton value={webhookUrl} label="Copy URL" />
                  </HStack>
                )}
              </Box>
              {requireApproval && (
                <Box>
                  <Label>Review</Label>
                  <Text fontSize="sm">
                    <Badge colorScheme="yellow" mr={2}>
                      Requires approval
                    </Badge>
                    Content changes wait on the{" "}
                    <Link as={RouterLink} to={approvalsLink({ agent: name })} color="purple.600">
                      Approvals
                    </Link>{" "}
                    page for an editor.
                  </Text>
                </Box>
              )}
              <Box>
                <Label>Limits</Label>
                {setLimits.length > 0 ? (
                  <Wrap>
                    {setLimits.map(([k, label]) => (
                      <Badge key={k} colorScheme="orange" variant="subtle">
                        {limits?.[k]} {label}
                      </Badge>
                    ))}
                  </Wrap>
                ) : (
                  <Text fontSize="sm" color="gray.500">
                    No limits: this agent can run and spend without bound.
                  </Text>
                )}
              </Box>
              <Box>
                <Label>Functions ({functions.length})</Label>
                {functions.length === 0 && (
                  <Text fontSize="sm" color="gray.500">
                    No functions: this agent can only answer with text.
                  </Text>
                )}
                <List spacing={2} mt={2}>
                  {functions.map((func) => (
                    <ListItem key={func.name} fontSize="sm">
                      <Badge colorScheme="gray" mr={2}>
                        {func.name}
                      </Badge>
                      {func.description}
                      {func.collections && func.collections.length > 0 ? (
                        <Text as="span" ml={2} color="gray.500">
                          (only {func.collections.join(", ")})
                        </Text>
                      ) : func.impl_key?.startsWith("collections.") ? (
                        <Text as="span" ml={2} color="orange.500">
                          (all collections)
                        </Text>
                      ) : null}
                    </ListItem>
                  ))}
                </List>
              </Box>
            </VStack>
          </TabPanel>
          <TabPanel px={0}>
            <RunsPanel key={runsVersion} agentName={name} initialRun={startRun} />
          </TabPanel>
        </TabPanels>
      </Tabs>
    </VStack>
  );
}
