import { useState } from "react";
import { Link as RouterLink } from "react-router-dom";
import { Alert, Badge, Box, Button, Code, Field, HStack, Link, Text, Textarea, VStack, Wrap } from "@chakra-ui/react";
import { FiCheck, FiEdit3, FiX } from "react-icons/fi";
import { useColorModeValue } from "@/components/ui/color-mode";
import { DrawerBody, DrawerFooter } from "@/components/ui/drawer";
import { ApiError } from "../../services/api";
import { useNotify } from "../../shared/hooks/useNotify";
import { errorMessage } from "../../shared/utils/errors";
import { formatTime } from "../agents/components/runFormat";
import { agentRunLink } from "../history/history";
import {
  approveCall,
  callTarget,
  notifyProposalsChanged,
  parseEditedData,
  type ProposalDetail,
  rejectProposal,
  sameJSON,
} from "./proposals";

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

function JsonBlock({ value, label }: { value: unknown; label: string }) {
  const bg = useColorModeValue("gray.50", "gray.800");
  return (
    <Code
      display="block"
      whiteSpace="pre-wrap"
      wordBreak="break-word"
      p={3}
      fontSize="xs"
      bg={bg}
      maxH="360px"
      overflowY="auto"
      w="full"
      aria-label={label}
    >
      {typeof value === "string" ? value : JSON.stringify(value, null, 2)}
    </Code>
  );
}

type Mode = "review" | "edit" | "reject";

/**
 * A call to a connected app's tool an agent asked for: what it calls, with
 * which arguments; making it (as is, or with edited arguments), or
 * rejecting it. Admins only.
 */
export function CallReview({ detail, reload, onClose }: { detail: ProposalDetail; reload: () => void; onClose: () => void }) {
  const { proposal: p, call, missing } = detail;
  const notify = useNotify();
  const target = callTarget(p);
  const args = call?.arguments ?? (p.data?.arguments as Record<string, unknown> | undefined) ?? {};
  const startText = JSON.stringify(args, null, 2);
  const [mode, setMode] = useState<Mode>("review");
  const [edited, setEdited] = useState(startText);
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState<"approve" | "reject" | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const quoteBg = useColorModeValue("gray.50", "gray.700");

  const pending = p.status === "pending";
  const parsed = mode === "edit" ? parseEditedData(edited) : null;
  const edits = mode === "edit" && parsed?.value && !sameJSON(parsed.value, args) ? parsed.value : undefined;
  const label = call?.title || target.tool;
  const app = call?.connectionLabel || target.connection;

  const approve = async () => {
    if (busy || (mode === "edit" && parsed?.error)) return;
    setBusy("approve");
    setActionError(null);
    try {
      await approveCall(p.id, edits);
      notify.success("Call made", `${app}: ${label}${edits ? ", with your arguments" : ""}.`);
      notifyProposalsChanged();
      onClose();
    } catch (err) {
      setActionError(errorMessage(err));
      // 422: the call failed and the proposal is now failed; 409: someone else decided it.
      if (err instanceof ApiError && (err.status === 422 || err.status === 409)) {
        notifyProposalsChanged();
        reload();
      }
    } finally {
      setBusy(null);
    }
  };

  const reject = async () => {
    if (busy) return;
    setBusy("reject");
    setActionError(null);
    try {
      await rejectProposal(p.id, note);
      notify.success("Call rejected", "Nothing was sent.");
      notifyProposalsChanged();
      onClose();
    } catch (err) {
      setActionError(errorMessage(err));
      if (err instanceof ApiError && err.status === 409) {
        notifyProposalsChanged();
        reload();
      }
    } finally {
      setBusy(null);
    }
  };

  const result = typeof p.data?.result === "string" ? p.data.result : undefined;

  return (
    <>
      <DrawerBody>
        <VStack align="stretch" gap={5} py={4}>
          <Wrap gap={4} fontSize="sm" align="center">
            <Badge colorPalette="purple">call</Badge>
            <Text>
              <Text as="span" color="gray.500">
                Agent{" "}
              </Text>
              <Text as="span" fontWeight="semibold">
                {p.agentName}
              </Text>
              {p.agentRunId !== undefined && (
                <Link asChild color="purple.600" ml={1}>
                  <RouterLink to={agentRunLink(p.agentName, p.agentRunId)}>run #{p.agentRunId}</RouterLink>
                </Link>
              )}
            </Text>
            <Text color="gray.500">Proposed {formatTime(p.createdAt)}</Text>
          </Wrap>

          <Section title="Calls">
            <HStack gap={2} wrap="wrap">
              <Text fontWeight="medium">{label}</Text>
              <Badge colorPalette="purple" variant="outline">
                {app}
              </Badge>
              <Code fontSize="xs" bg="transparent" px={0} color="gray.500">
                {target.connection}.{target.tool}
              </Code>
              {call?.destructive && <Badge colorPalette="red">may delete or overwrite</Badge>}
            </HStack>
            {call?.description && (
              <Text fontSize="sm" color="gray.600" mt={1}>
                {call.description}
              </Text>
            )}
          </Section>

          {missing && pending && (
            <Alert.Root role="alert" status="warning" borderRadius="md">
              <Alert.Indicator />
              <Alert.Content>
                <Alert.Description fontSize="sm">The connection no longer exists: this call can only be rejected.</Alert.Description>
              </Alert.Content>
            </Alert.Root>
          )}

          {p.reason && (
            <Section title="Agent's reason">
              <Text
                fontSize="sm"
                whiteSpace="pre-wrap"
                bg={quoteBg}
                borderLeftWidth="3px"
                borderColor="purple.300"
                px={3}
                py={2}
                borderRadius="sm"
              >
                {p.reason}
              </Text>
            </Section>
          )}

          {!pending && (
            <Alert.Root
              role="alert"
              status={p.status === "approved" ? "success" : p.status === "failed" ? "error" : "info"}
              borderRadius="md"
              alignItems="start"
            >
              <Alert.Indicator />
              <Alert.Content>
                <Alert.Title textTransform="capitalize">{p.status === "applying" ? "Being made" : p.status}</Alert.Title>
                <Alert.Description fontSize="sm">
                  {[p.reviewedBy && `By ${p.reviewedBy}`, p.decidedAt && formatTime(p.decidedAt)].filter(Boolean).join(", ")}
                  {p.reviewNote && <Text mt={1}>“{p.reviewNote}”</Text>}
                  {p.error && <Text mt={1}>{p.error}</Text>}
                </Alert.Description>
              </Alert.Content>
            </Alert.Root>
          )}

          {mode === "edit" && pending ? (
            <Field.Root invalid={Boolean(parsed?.error)}>
              <Field.Label fontSize="sm">Arguments</Field.Label>
              <Textarea
                rows={12}
                fontFamily="mono"
                fontSize="sm"
                value={edited}
                onChange={(e) => setEdited(e.target.value)}
                disabled={busy !== null}
                aria-label="Arguments (JSON)"
                spellCheck={false}
              />
              {parsed?.error ? (
                <Field.ErrorText>{parsed.error}</Field.ErrorText>
              ) : (
                <Field.HelperText>The tool is called with these arguments instead of the agent's.</Field.HelperText>
              )}
            </Field.Root>
          ) : (
            <Section title="Arguments">
              <JsonBlock value={args} label="Arguments" />
            </Section>
          )}

          {result !== undefined && (
            <Section title="Result">
              <JsonBlock value={result} label="Result" />
            </Section>
          )}

          {pending && mode === "reject" && (
            <Field.Root>
              <Field.Label fontSize="sm">Note (optional)</Field.Label>
              <Textarea
                rows={3}
                disabled={busy !== null}
                value={note}
                onChange={(e) => setNote(e.target.value)}
                placeholder="Why the call is rejected"
              />
            </Field.Root>
          )}

          {actionError && (
            <Alert.Root role="alert" status="error" borderRadius="md">
              <Alert.Indicator />
              <Alert.Content>
                <Alert.Description>{actionError}</Alert.Description>
              </Alert.Content>
            </Alert.Root>
          )}
        </VStack>
      </DrawerBody>

      {pending && (
        <DrawerFooter borderTopWidth="1px" gap={2} flexWrap="wrap">
          {mode === "review" && (
            <>
              <Button variant="outline" colorPalette="red" onClick={() => setMode("reject")} mr="auto" disabled={busy !== null}>
                <FiX />
                Reject
              </Button>
              {!missing && (
                <>
                  <Button variant="outline" onClick={() => setMode("edit")} disabled={busy !== null}>
                    <FiEdit3 />
                    Edit and call
                  </Button>
                  <Button colorPalette="green" onClick={approve} loading={busy === "approve"} disabled={busy !== null}>
                    <FiCheck />
                    Approve and call
                  </Button>
                </>
              )}
            </>
          )}
          {mode === "edit" && (
            <>
              <Button variant="ghost" onClick={() => setMode("review")} disabled={busy !== null}>
                Cancel
              </Button>
              <Button
                colorPalette="green"
                onClick={approve}
                loading={busy === "approve"}
                disabled={busy !== null || Boolean(parsed?.error)}
              >
                <FiCheck />
                Call with these arguments
              </Button>
            </>
          )}
          {mode === "reject" && (
            <>
              <Button variant="ghost" onClick={() => setMode("review")} disabled={busy !== null}>
                Cancel
              </Button>
              <Button colorPalette="red" onClick={reject} loading={busy === "reject"} disabled={busy !== null}>
                <FiX />
                Reject call
              </Button>
            </>
          )}
        </DrawerFooter>
      )}
    </>
  );
}
