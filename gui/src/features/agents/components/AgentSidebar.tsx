import { Link as RouterLink } from "react-router-dom";
import { Box, Text, Badge, Spinner, List, ListItem, Button, Flex } from "@chakra-ui/react";
import { FiPlus } from "react-icons/fi";
import { AgentNodeData } from "../../../shared/types";

interface AgentSidebarProps {
  agents: AgentNodeData[];
  selectedAgent: string | null;
  onSelectAgent: (name: string) => void;
  loading: boolean;
  canCreate?: boolean;
}

export function AgentSidebar({ agents, selectedAgent, onSelectAgent, loading, canCreate }: AgentSidebarProps) {
  return (
    <Box
      as="nav"
      aria-label="Agents"
      w={{ base: "200px", lg: "280px" }}
      flexShrink={0}
      borderRight="1px solid"
      borderColor="gray.200"
      overflowY="auto"
      px={{ base: 3, lg: 6 }}
      py={8}
      bg="gray.50"
    >
      <Flex justify="space-between" align="center" mb={4} gap={2}>
        <Text fontWeight="bold" fontSize="sm" letterSpacing="wide" textTransform="uppercase">
          Agents
          <Badge variant="solid" colorScheme="purple" fontSize="0.75em" borderRadius="full" px={2} ml={2}>
            {agents.length}
          </Badge>
        </Text>
      </Flex>
      {canCreate && (
        <Button as={RouterLink} to="/agents/new" size="sm" colorScheme="teal" leftIcon={<FiPlus />} w="full" mb={4}>
          New agent
        </Button>
      )}
      {loading ? (
        <Spinner />
      ) : (
        <List spacing={1}>
          {agents.map(({ name }) => {
            const active = selectedAgent === name;
            return (
              <ListItem key={name}>
                <Box
                  as="button"
                  type="button"
                  w="full"
                  textAlign="left"
                  py={2}
                  px={3}
                  fontWeight={active ? "bold" : "normal"}
                  color={active ? "purple.600" : "inherit"}
                  bg={active ? "purple.100" : "transparent"}
                  borderRadius="md"
                  wordBreak="break-word"
                  aria-current={active ? "page" : undefined}
                  onClick={() => onSelectAgent(name)}
                  _hover={{ bg: active ? "purple.100" : "gray.100" }}
                >
                  {name}
                </Box>
              </ListItem>
            );
          })}
        </List>
      )}
    </Box>
  );
}
