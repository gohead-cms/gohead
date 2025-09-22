import React, { useEffect, useState } from "react";
import { Flex, Box, Spinner, Text, Badge, Link, Icon, List, ListItem, Heading } from "@chakra-ui/react";
import { FiPlus } from "react-icons/fi";
import { apiFetchWithAuth } from "../../services/api";

// A placeholder component to display agent details
function AgentDetails({ agent }: { agent: any }) {
  if (!agent || !agent.schema) return null;

  const { name, schema } = agent;
  const { llmConfig, trigger, functions } = schema;

  return (
    <Box>
      <Heading size="lg" mb={4}>{name}</Heading>
      <Box mb={6}>
        <Text fontWeight="bold" fontSize="sm" textTransform="uppercase" color="gray.500">LLM Config</Text>
        <Text>Provider: <Badge colorScheme="teal">{llmConfig.provider}</Badge></Text>
        <Text>Model: <Badge colorScheme="cyan">{llmConfig.model}</Badge></Text>
      </Box>
       <Box mb={6}>
        <Text fontWeight="bold" fontSize="sm" textTransform="uppercase" color="gray.500">Trigger</Text>
        <Text>Type: <Badge colorScheme="purple">{trigger.type}</Badge></Text>
        {trigger.event_trigger && (
          <Text>On: '{trigger.event_trigger.collection}' [{trigger.event_trigger.events.join(', ')}]</Text>
        )}
      </Box>
      <Box>
        <Text fontWeight="bold" fontSize="sm" textTransform="uppercase" color="gray.500">Functions ({functions.length})</Text>
        <List spacing={2} mt={2}>
          {functions.map((func: any) => (
            <ListItem key={func.name} fontSize="sm">
              <Badge colorScheme="gray" mr={2}>{func.name}</Badge>
              {func.description}
            </ListItem>
          ))}
        </List>
      </Box>
    </Box>
  );
}


export default function AgentsPage() {
  const [agents, setAgents] = useState<{ name: string }[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [agentDetails, setAgentDetails] = useState<any | null>(null);
  const [loading, setLoading] = useState(true);
  const [agentLoading, setAgentLoading] = useState(false);

  // Effect to fetch the list of all agents
  useEffect(() => {
    setLoading(true);
    // Assumes an endpoint /admin/agents exists
    apiFetchWithAuth("/admin/agents")
      .then(async (res) => {
        if (!res.ok) throw new Error("Failed to fetch agents");
        const json = await res.json();
        const list = Array.isArray(json.data) ? json.data : [];
        // Assumes each agent object has a top-level `name` property
        setAgents(list.map((c: any) => ({ name: c.name })));
        setSelected(list.length ? list[0].name : null);
      })
      .finally(() => setLoading(false));
  }, []);

  // Effect to fetch the details of the selected agent
  useEffect(() => {
    if (!selected) {
      setAgentDetails(null);
      return;
    }
    setAgentLoading(true);
    // Assumes an endpoint /admin/agents/:name exists
    apiFetchWithAuth(`/admin/agents/${selected}`)
      .then(async (res) => {
        if (!res.ok) throw new Error("Failed to fetch agent details");
        const json = await res.json();
        setAgentDetails(json.data || null);
      })
      .finally(() => setAgentLoading(false));
  }, [selected]);

  return (
    <Flex h="100%" w="100%">
      {/* Sidebar */}
      <Box
        w="320px"
        borderRight="1px solid"
        borderColor="gray.200"
        overflowY="auto"
        px={6}
        py={8}
        bg="gray.100"
      >
        <Text
          mb={2}
          fontWeight="bold"
          fontSize="sm"
          letterSpacing="wide"
          textTransform="uppercase"
        >
          Agents
          <Badge
            variant="surface"
            colorScheme="purple"
            fontSize="0.75em"
            borderRadius="full"
            px={2}
            py={0.5}
            m={2}
            fontWeight="normal"
          >
            {agents.length}
          </Badge>
        </Text>
        {loading ? (
          <Spinner />
        ) : (
          <List>
            {agents.map(({ name }) => (
              <ListItem
                key={name}
                cursor="pointer"
                py={1}
                px={5}
                ms={2}
                fontWeight={selected === name ? "bold" : "normal"}
                color={selected === name ? "purple.600" : "inherit"}
                bg={selected === name ? "purple.100" : "transparent"}
                borderRadius="lg"
                onClick={() => setSelected(name)}
              >
                {name}
              </ListItem>
            ))}
          </List>
        )}
        <Box mt={6} px={3}>
          <Link
            color="blue.500"
            fontWeight="semibold"
            fontSize="md"
            display="flex"
            alignItems="center"
            gap={2}
            cursor="pointer"
            href="/agents/studio" // Link to a potential "create agent" page
            _hover={{ textDecoration: "underline" }}
          >
            <Icon as={FiPlus} boxSize={4} mr={1} />
            Create Agent
          </Link>
        </Box>
      </Box>
      
      {/* Main Content */}
      <Box
        flex="1"
        p={10}
        overflowY="auto"
        bg="gray.50"
        minH="100%"
      >
        {agentLoading ? (
          <Spinner />
        ) : selected && agentDetails ? (
          <AgentDetails agent={agentDetails} />
        ) : (
          <Text color="gray.400">Select an agent to view its details</Text>
        )}
      </Box>
    </Flex>
  );
}