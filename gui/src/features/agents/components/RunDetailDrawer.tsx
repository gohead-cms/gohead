import {
  Alert,
  AlertIcon,
  Badge,
  Box,
  Code,
  Divider,
  Drawer,
  DrawerBody,
  DrawerCloseButton,
  DrawerContent,
  DrawerHeader,
  DrawerOverlay,
  HStack,
  Spinner,
  Text,
  VStack,
  useColorModeValue,
} from "@chakra-ui/react";
import { AgentRunStep } from "../../../shared/types";
import { useAgentRun } from "../hook/useAgentRuns";
import {
  formatCost,
  formatDuration,
  formatTime,
  formatTokens,
  prettyJSON,
  statusColor,
} from "./runFormat";

interface RunDetailDrawerProps {
  agentName: string;
  runId: number | null;
  onClose: () => void;
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Box>
      <Text fontWeight="bold" fontSize="xs" textTransform="uppercase" color="gray.500" mb={1}>
        {title}
      </Text>
      {children}
    </Box>
  );
}

function Pre({ text }: { text: string }) {
  const bg = useColorModeValue("gray.50", "gray.800");
  return (
    <Code
      display="block"
      whiteSpace="pre-wrap"
      wordBreak="break-word"
      p={3}
      fontSize="xs"
      bg={bg}
      maxH="320px"
      overflowY="auto"
      w="full"
    >
      {text}
    </Code>
  );
}

function Step({ step }: { step: AgentRunStep }) {
  const border = useColorModeValue("gray.200", "gray.600");
  const tokens = step.inputTokens + step.outputTokens;
  return (
    <Box borderLeftWidth="3px" borderColor={step.error ? "red.400" : border} pl={3} py={1}>
      <HStack spacing={2} wrap="wrap" mb={1}>
        <Text fontSize="xs" color="gray.500">
          #{step.seq}
        </Text>
        <Badge colorScheme={step.type === "llm" ? "purple" : "teal"}>
          {step.type === "llm" ? "model" : "function"}
        </Badge>
        <Text fontWeight="semibold" fontSize="sm">
          {step.name}
        </Text>
        {step.implKey && (
          <Text fontSize="xs" color="gray.500">
            {step.implKey}
          </Text>
        )}
        <Text fontSize="xs" color="gray.500" ml="auto">
          {formatDuration(step.durationMs)}
          {tokens > 0 && ` · ${formatTokens(tokens)} tokens`}
          {step.costUsd > 0 && ` · ${formatCost(step.costUsd)}`}
        </Text>
      </HStack>
      <VStack align="stretch" spacing={2}>
        {step.arguments && <Pre text={prettyJSON(step.arguments)} />}
        {step.result && <Pre text={prettyJSON(step.result)} />}
        {step.error && (
          <Text fontSize="sm" color="red.500">
            {step.error}
          </Text>
        )}
      </VStack>
    </Box>
  );
}

/** Shows one run: what triggered it, every model and function call, and how it ended. */
export function RunDetailDrawer({ agentName, runId, onClose }: RunDetailDrawerProps) {
  const { run, loading, error } = useAgentRun(agentName, runId);

  return (
    <Drawer isOpen={runId !== null} onClose={onClose} size="lg" placement="right">
      <DrawerOverlay />
      <DrawerContent>
        <DrawerCloseButton />
        <DrawerHeader borderBottomWidth="1px">
          Run #{runId}
          {run && (
            <Badge ml={3} colorScheme={statusColor[run.status]}>
              {run.status}
            </Badge>
          )}
        </DrawerHeader>
        <DrawerBody>
          {loading && <Spinner mt={4} />}
          {error && (
            <Alert status="error" mt={4}>
              <AlertIcon />
              {error}
            </Alert>
          )}
          {run && !loading && (
            <VStack align="stretch" spacing={5} py={4}>
              <HStack spacing={6} wrap="wrap" fontSize="sm">
                <Text>Started {formatTime(run.startedAt)}</Text>
                <Text>{formatDuration(run.durationMs)}</Text>
                <Text>
                  {formatTokens(run.inputTokens)} in / {formatTokens(run.outputTokens)} out
                </Text>
                <Text>{formatCost(run.costUsd)}</Text>
                {run.attempt > 1 && <Badge colorScheme="yellow">attempt {run.attempt}</Badge>}
                {run.model && (
                  <Text color="gray.500">
                    {run.provider}/{run.model}
                  </Text>
                )}
              </HStack>
              {run.error && (
                <Alert status={run.status === "limited" ? "warning" : "error"}>
                  <AlertIcon />
                  {run.error}
                </Alert>
              )}
              <Section title={`Trigger (${run.triggerType.replace(/_/g, " ")})`}>
                <Pre text={prettyJSON(run.trigger)} />
              </Section>
              {run.input && (
                <Section title="Input">
                  <Pre text={run.input} />
                </Section>
              )}
              <Divider />
              <Section title={`Steps (${run.steps?.length ?? 0})`}>
                {run.steps && run.steps.length > 0 ? (
                  <VStack align="stretch" spacing={3}>
                    {run.steps.map((s) => (
                      <Step key={s.id} step={s} />
                    ))}
                  </VStack>
                ) : (
                  <Text fontSize="sm" color="gray.500">
                    No steps recorded.
                  </Text>
                )}
              </Section>
              {run.output && (
                <Section title="Output">
                  <Pre text={run.output} />
                </Section>
              )}
            </VStack>
          )}
        </DrawerBody>
      </DrawerContent>
    </Drawer>
  );
}
