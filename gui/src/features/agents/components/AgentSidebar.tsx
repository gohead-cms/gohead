import React from "react";
import {
  Box,
  Text,
  Badge,
  Spinner,
  List,
  ListItem,
  Link,
  Icon,
} from "@chakra-ui/react";
import { FiPlus } from "react-icons/fi";
import { AgentNodeData } from "../../../shared/types";

interface AgentSidebarProps {
  agents: AgentNodeData[];
  selectedAgent: string | null;
  onSelectAgent: (name: string) => void;
  loading: boolean;
}

export function AgentSidebar({
  agents,
  selectedAgent,
  onSelectAgent,
  loading,
}: AgentSidebarProps) {
  return (
    <Box
      w="320px"
      borderRight="1px solid"
      borderColor="gray.200"
      overflowY="auto"
      px={6}
      py={8}
      bg="gray.50"
    >
      <Text
        mb={4}
        fontWeight="bold"
        fontSize="sm"
        letterSpacing="wide"
        textTransform="uppercase"
      >
        Agents
        <Badge
          variant="solid"
          colorScheme="purple"
          fontSize="0.75em"
          borderRadius="full"
          px={2}
          ml={2}
        >
          {agents.length}
        </Badge>
      </Text>
      {loading ? (
        <Spinner />
      ) : (
        <List spacing={1}>
          {agents.map(({ name }) => (
            <ListItem
              key={name}
              cursor="pointer"
              py={2}
              px={3}
              fontWeight={selectedAgent === name ? "bold" : "normal"}
              color={selectedAgent === name ? "purple.600" : "inherit"}
              bg={selectedAgent === name ? "purple.100" : "transparent"}
              borderRadius="md"
              onClick={() => onSelectAgent(name)}
              _hover={{ bg: "gray.100" }}
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
          display="flex"
          alignItems="center"
          gap={2}
          href="/agents/studio" // Link to a potential "create agent" page
        >
          <Icon as={FiPlus} />
          Create Agent
        </Link>
      </Box>
    </Box>
  );
}
