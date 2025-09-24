import React from "react";
import {
  Box,
  Heading,
  Text,
  Badge,
  List,
  ListItem,
  VStack,
} from "@chakra-ui/react";
import { AgentNodeData } from "../../../shared/types";

interface AgentDetailsProps {
  agent: AgentNodeData;
}

export function AgentDetails({ agent }: AgentDetailsProps) {
  const { name, schema } = agent;
  const { llmConfig, trigger, functions } = schema;

  return (
    <VStack spacing={6} align="stretch">
      <Heading size="lg">{name}</Heading>
      <Box>
        <Text fontWeight="bold" fontSize="sm" textTransform="uppercase" color="gray.500" mb={2}>
          LLM Config
        </Text>
        <Text>
          Provider: <Badge colorScheme="teal">{llmConfig.provider}</Badge>
        </Text>
        <Text>
          Model: <Badge colorScheme="cyan">{llmConfig.model}</Badge>
        </Text>
      </Box>
      <Box>
        <Text fontWeight="bold" fontSize="sm" textTransform="uppercase" color="gray.500" mb={2}>
          Trigger
        </Text>
        <Text>
          Type: <Badge colorScheme="purple">{trigger.type}</Badge>
        </Text>
        {trigger.event_trigger && (
          <Text>
            On: '{trigger.event_trigger.collection}' [
            {trigger.event_trigger.events.join(", ")}]
          </Text>
        )}
      </Box>
      <Box>
        <Text fontWeight="bold" fontSize="sm" textTransform="uppercase" color="gray.500" mb={2}>
          Functions ({functions.length})
        </Text>
        <List spacing={2} mt={2}>
          {functions.map((func) => (
            <ListItem key={func.name} fontSize="sm">
              <Badge colorScheme="gray" mr={2}>
                {func.name}
              </Badge>
              {func.description}
            </ListItem>
          ))}
        </List>
      </Box>
    </VStack>
  );
}
