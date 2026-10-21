import { Link as RouterLink } from "react-router-dom";
import { Box, Text, Badge, Spinner, List, Button, Flex, chakra } from "@chakra-ui/react";
import { FiPlus } from "react-icons/fi";
import { type AgentNodeData } from "../../../shared/types";

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
          <Badge variant="solid" colorPalette="purple" fontSize="0.75em" borderRadius="full" px={2} ml={2}>
            {agents.length}
          </Badge>
        </Text>
      </Flex>
      {canCreate && (
        <Button asChild size="xs" colorPalette="teal" w="full" mb={4}>
          <RouterLink to="/agents/new">
            <FiPlus />
            New agent
          </RouterLink>
        </Button>
      )}
      {loading ? (
        <Spinner />
      ) : (
        <List.Root gap={1} listStyleType="none">
          {agents.map(({ name }) => {
            const active = selectedAgent === name;
            return (
              <List.Item key={name}>
                <chakra.button
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
                </chakra.button>
              </List.Item>
            );
          })}
        </List.Root>
      )}
    </Box>
  );
}
