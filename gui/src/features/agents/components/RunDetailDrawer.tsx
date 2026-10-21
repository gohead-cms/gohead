import { Badge, Box, Code, HStack, Separator, Spinner, Text, VStack } from "@chakra-ui/react";
import { useColorModeValue } from "@/components/ui/color-mode";
import { DrawerBody, DrawerCloseTrigger, DrawerContent, DrawerHeader, DrawerRoot, DrawerTitle } from "@/components/ui/drawer";
import { type AgentRunStep } from "../../../shared/types";
import { RunProposals } from "../../approvals/RunProposals";
import { RunContentChanges } from "../../history/RunContentChanges";
import { useAgentRun } from "../hook/useAgentRuns";
import { Notice } from "./EditorParts";
import { RunStatusBadge } from "./RunBadges";
import {
  formatCost,
  formatDuration,
  formatTime,
  formatTokens,
  prettyJSON,
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
      <HStack gap={2} wrap="wrap" mb={1}>
        <Text fontSize="xs" color="gray.500">
          #{step.seq}
        </Text>
        <Badge colorPalette={step.type === "llm" ? "purple" : "teal"}>
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
      <VStack align="stretch" gap={2}>
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
  const { run, loading, error, inProgress } = useAgentRun(agentName, runId);

  return (
    <DrawerRoot
      open={runId !== null}
      onOpenChange={(e) => {
        if (!e.open) onClose();
      }}
      size="lg"
      placement="end"
    >
      <DrawerContent>
        <DrawerHeader borderBottomWidth="1px" pb={4}>
          <DrawerTitle fontSize="xl" fontWeight="semibold" pe={8}>
            Run #{runId}
            <Text as="span" fontWeight="normal" color="gray.500" fontSize="md" ml={2}>
              {agentName}
            </Text>
            {run && (
              <Box as="span" ml={3}>
                <RunStatusBadge run={run} />
              </Box>
            )}
            {inProgress && <Spinner size="xs" ml={2} color="gray.400" aria-label="Updating" />}
          </DrawerTitle>
        </DrawerHeader>
        <DrawerBody>
          {/* A run that has not ended reloads in place: keep it on screen. */}
          {loading && !run && <Spinner mt={4} />}
          {error && (
            <Notice status="error" mt={4}>
              {error}
            </Notice>
          )}
          {run && (
            <VStack align="stretch" gap={5} py={4}>
              {run.status === "queued" && (
                <Notice status="info" borderRadius="md">
                  Queued: the run starts as soon as a worker picks it up.
                </Notice>
              )}
              {run.dryRun && (
                <Notice status="info" variant="subtle" borderStartWidth="4px" borderStartColor="colorPalette.solid" borderRadius="md">
                  Test run: the agent ran for real, but content functions only reported what they would change (“would_update”…). Nothing
                  was written.
                </Notice>
              )}
              <HStack gap={6} wrap="wrap" fontSize="sm">
                <Text>Started {formatTime(run.startedAt)}</Text>
                <Text>{formatDuration(run.durationMs)}</Text>
                <Text>
                  {formatTokens(run.inputTokens)} in / {formatTokens(run.outputTokens)} out
                </Text>
                <Text>{formatCost(run.costUsd)}</Text>
                {run.attempt > 1 && <Badge colorPalette="yellow">attempt {run.attempt}</Badge>}
                {run.model && (
                  <Text color="gray.500">
                    {run.provider}/{run.model}
                  </Text>
                )}
              </HStack>
              {run.error && (
                <Notice status={run.status === "limited" ? "warning" : "error"}>
                  {run.error}
                </Notice>
              )}
              <Section title={`Trigger (${run.triggerType.replace(/_/g, " ")})`}>
                <Pre text={prettyJSON(run.trigger)} />
              </Section>
              {run.input && (
                <Section title="Input">
                  <Pre text={run.input} />
                </Section>
              )}
              <Separator />
              <Section title={`Steps (${run.steps?.length ?? 0})`}>
                {run.steps && run.steps.length > 0 ? (
                  <VStack align="stretch" gap={3}>
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
              <Separator />
              {/* Reloaded once the run ends: it may propose more while it runs. */}
              {runId !== null && <RunProposals key={`${runId}-${run.status}`} runId={runId} />}
              {runId !== null && !run.dryRun && <RunContentChanges key={runId} agentName={agentName} runId={runId} />}
            </VStack>
          )}
        </DrawerBody>
        <DrawerCloseTrigger />
      </DrawerContent>
    </DrawerRoot>
  );
}
