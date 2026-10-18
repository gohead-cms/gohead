import {
  Box,
  Heading,
  Text,
  Badge,
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
import { AgentLimits, AgentNodeData } from "../../../shared/types";
import { RunsPanel } from "./RunsPanel";

interface AgentDetailsProps {
  agent: AgentNodeData;
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

export function AgentDetails({ agent }: AgentDetailsProps) {
  const { name, schema } = agent;
  const { llmConfig, trigger, functions, limits } = schema;
  const setLimits = limitLabels.filter(([k]) => (limits?.[k] ?? 0) > 0);
  const priced = (llmConfig.input_cost_per_million ?? 0) > 0 || (llmConfig.output_cost_per_million ?? 0) > 0;

  return (
    <VStack spacing={4} align="stretch">
      <Heading size="lg">{name}</Heading>
      <Tabs isLazy colorScheme="teal">
        <TabList>
          <Tab>Overview</Tab>
          <Tab>Runs</Tab>
        </TabList>
        <TabPanels>
          <TabPanel px={0}>
            <VStack spacing={6} align="stretch">
              <Box>
                <Label>LLM Config</Label>
                <Text>
                  Provider: <Badge colorScheme="teal">{llmConfig.provider}</Badge>
                </Text>
                <Text>
                  Model: <Badge colorScheme="cyan">{llmConfig.model}</Badge>
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
                {trigger.event_trigger && (
                  <Text>
                    On: '{trigger.event_trigger.collection}' [{trigger.event_trigger.events.join(", ")}]
                  </Text>
                )}
              </Box>
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
            <RunsPanel agentName={name} />
          </TabPanel>
        </TabPanels>
      </Tabs>
    </VStack>
  );
}
