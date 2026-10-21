import { useState } from "react";
import { Link as RouterLink } from "react-router-dom";
import { Alert, Badge, Box, Button, Code, Field, HStack, Icon, Link, Spinner, Text, Textarea, VStack, Wrap } from "@chakra-ui/react";
import { useColorModeValue } from "@/components/ui/color-mode";
import { DrawerBody, DrawerCloseTrigger, DrawerContent, DrawerFooter, DrawerHeader, DrawerRoot, DrawerTitle } from "@/components/ui/drawer";
import { FiCheck, FiEdit3, FiExternalLink, FiGlobe, FiX } from "react-icons/fi";
import { ApiError } from "../../services/api";
import { useNotify } from "../../shared/hooks/useNotify";
import { errorMessage } from "../../shared/utils/errors";
import { formatTime } from "../agents/components/runFormat";
import { useMe } from "../auth/me";
import { ChangesTable } from "../history/ChangesTable";
import { agentRunLink } from "../history/history";
import {
  approveProposal,
  entryLink,
  isStaleConflict,
  notifyProposalsChanged,
  parseEditedData,
  type Proposal,
  type ProposalDetail,
  proposalActionColor,
  proposalActionLabel,
  proposalEntryLabel,
  proposalStatusColor,
  rejectProposal,
  sameJSON,
} from "./proposals";
import { useProposal } from "./useProposals";

interface ProposalDrawerProps {
  /** The proposal shown; null closes the drawer. */
  proposalId: number | null;
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

/** Status, action and locale of a proposal. */
export function ProposalBadges({ proposal }: { proposal: Pick<Proposal, "status" | "action" | "locale"> }) {
  return (
    <HStack gap={1.5} display="inline-flex" wrap="wrap">
      <Badge colorPalette={proposalActionColor[proposal.action] ?? "gray"}>{proposalActionLabel(proposal)}</Badge>
      {proposal.locale && <LocaleBadge locale={proposal.locale} />}
    </HStack>
  );
}

export function LocaleBadge({ locale }: { locale: string }) {
  return (
    <Badge
      colorPalette="purple"
      variant="outline"
      textTransform="none"
      display="inline-flex"
      alignItems="center"
      gap={0.5}
      title={`The ${locale} translation`}
    >
      <Icon as={FiGlobe} boxSize="0.8em" />
      {locale}
    </Badge>
  );
}

/**
 * One proposal: what the agent wants to change and why, against the entry
 * as it is now; approving it (as is, or with edited values), or rejecting it.
 */
export function ProposalDrawer({ proposalId, onClose }: ProposalDrawerProps) {
  const { detail, loading, error, refetch } = useProposal(proposalId);
  return (
    <DrawerRoot
      open={proposalId !== null}
      onOpenChange={(e) => {
        if (!e.open) onClose();
      }}
      placement="end"
      size="xl"
    >
      <DrawerContent maxW={{ base: "100vw", lg: "4xl" }}>
        <DrawerHeader borderBottomWidth="1px">
          <DrawerTitle display="flex" alignItems="center" gap={3} flexWrap="wrap" fontSize="xl" fontWeight="semibold">
            <Text as="span">Proposal #{proposalId}</Text>
            {detail && (
              <Badge colorPalette={proposalStatusColor[detail.proposal.status] ?? "gray"} fontSize="0.7em">
                {detail.proposal.status}
              </Badge>
            )}
          </DrawerTitle>
        </DrawerHeader>
        {loading && !detail && (
          <DrawerBody>
            <Spinner mt={4} />
          </DrawerBody>
        )}
        {error && !detail && (
          <DrawerBody>
            <Alert.Root role="alert" status="error" mt={4} borderRadius="md">
              <Alert.Indicator />
              <Alert.Content>
                <Alert.Description>{error}</Alert.Description>
              </Alert.Content>
            </Alert.Root>
          </DrawerBody>
        )}
        {/* A new proposal starts the review from scratch. */}
        {detail && <ProposalReview key={detail.proposal.id} detail={detail} reload={refetch} onClose={onClose} />}
        <DrawerCloseTrigger />
      </DrawerContent>
    </DrawerRoot>
  );
}

/** Values as pretty JSON. */
function JsonBlock({ value, label }: { value: unknown; label: string }) {
  const bg = useColorModeValue("gray.50", "gray.800");
  return (
    <Code display="block" whiteSpace="pre-wrap" wordBreak="break-word" p={3} fontSize="xs" bg={bg} maxH="360px" overflowY="auto" w="full" aria-label={label}>
      {JSON.stringify(value, null, 2)}
    </Code>
  );
}

type Mode = "review" | "edit" | "reject";

function ProposalReview({ detail, reload, onClose }: { detail: ProposalDetail; reload: () => void; onClose: () => void }) {
  const { proposal: p, stale, changes, currentStatus, proposed } = detail;
  const { isAdmin } = useMe();
  const notify = useNotify();
  const quoteBg = useColorModeValue("gray.50", "gray.700");
  const [mode, setMode] = useState<Mode>("review");
  // Editing starts from what approving would write (for a stale update, the
  // entry now with the agent's changes), not the agent's raw values.
  const startText = JSON.stringify(proposed ?? p.data ?? {}, null, 2);
  const [edited, setEdited] = useState(startText);
  // The text the edit started from: values equal to it are the agent's change.
  const [editBase, setEditBase] = useState(startText);
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState<"approve" | "force" | "reject" | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  // The approval was refused because the entry changed since: offer to force it.
  const [staleRefused, setStaleRefused] = useState(false);
  // Edits refused as stale: the details being reloaded, to restart the edit from them.
  const [resetFrom, setResetFrom] = useState<ProposalDetail | null>(null);
  // The edit box was reset to the entry as it is now: the reviewer has to look again.
  const [editReset, setEditReset] = useState(false);

  // Reloaded details: an untouched edit box follows them; one refused as
  // stale restarts from them.
  const [seen, setSeen] = useState(detail);
  if (seen !== detail) {
    setSeen(detail);
    if (resetFrom !== null || edited === editBase) {
      setEdited(startText);
      setEditBase(startText);
    }
    if (resetFrom !== null) {
      setResetFrom(null);
      setEditReset(true);
    }
  }

  const pending = p.status === "pending";
  const parsed = mode === "edit" ? parseEditedData(edited) : null;
  const working = busy !== null || resetFrom !== null;
  // Values unchanged from where the edit started stay the agent's change,
  // which the server rebases on the entry as it is now.
  const edits = mode === "edit" && parsed?.value && !sameJSON(parsed.value, parseEditedData(editBase).value) ? parsed.value : undefined;

  const decided = (title: string, description?: string) => {
    notify.success(title, description);
    notifyProposalsChanged();
    onClose();
  };

  const approve = async (force: boolean) => {
    if (working || (mode === "edit" && parsed?.error)) return;
    // Edits made over the entry reloaded after a stale refusal were reviewed
    // against it: apply them although the entry changed since the proposal.
    const forced = force || (edits !== undefined && editReset);
    setBusy(force ? "force" : "approve");
    setActionError(null);
    try {
      await approveProposal(p.id, { ...(edits ? { data: edits } : {}), ...(forced ? { force: true } : {}) });
      decided("Change applied", `${proposalEntryLabel(p)}: the agent's ${proposalActionLabel(p)} was approved${edits ? " with your edits" : ""}.`);
    } catch (err) {
      if (isStaleConflict(err)) {
        if (edits) {
          // Never force edited values over changes made since: start the
          // edit again from the entry as it is now.
          setStaleRefused(false);
          setEditReset(false);
          setResetFrom(detail);
        } else {
          setStaleRefused(true);
        }
        reload();
      } else {
        setActionError(errorMessage(err));
        // 422: the proposal is now failed (bad data, or the collection, entry
        // or locale is gone); 409: someone else decided it.
        if (err instanceof ApiError && (err.status === 422 || err.status === 409)) {
          notifyProposalsChanged();
          reload();
        }
      }
    } finally {
      setBusy(null);
    }
  };

  const reject = async () => {
    if (working) return;
    setBusy("reject");
    setActionError(null);
    try {
      await rejectProposal(p.id, note);
      decided("Proposal rejected", "Nothing was changed.");
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

  const showStale = pending && (stale || staleRefused || editReset);
  // Forcing applies the agent's change only, never edited values.
  const canForce = staleRefused && edits === undefined;

  return (
    <>
      <DrawerBody>
        <VStack align="stretch" gap={5} py={4}>
          <Wrap gap={4} fontSize="sm" align="center">
            <ProposalBadges proposal={p} />
            <Text>
              <Text as="span" color="gray.500">
                Agent{" "}
              </Text>
              <Text as="span" fontWeight="semibold">
                {p.agentName}
              </Text>
              {p.agentRunId !== undefined &&
                (isAdmin ? (
                  <Link asChild color="purple.600" ml={1}>
                    <RouterLink to={agentRunLink(p.agentName, p.agentRunId)}>run #{p.agentRunId}</RouterLink>
                  </Link>
                ) : (
                  <Text as="span" color="gray.500" ml={1}>
                    run #{p.agentRunId}
                  </Text>
                ))}
            </Text>
            <Text color="gray.500">Proposed {formatTime(p.createdAt)}</Text>
          </Wrap>

          <Section title="Entry">
            <HStack gap={2} wrap="wrap">
              <Text fontWeight="medium">{proposalEntryLabel(p)}</Text>
              {p.locale && <LocaleBadge locale={p.locale} />}
              {currentStatus && (
                <Badge colorPalette={currentStatus === "published" ? "green" : "gray"} variant="subtle">
                  {currentStatus}
                </Badge>
              )}
              {p.itemId !== undefined && (
                <Button asChild size="2xs" variant="ghost">
                  <RouterLink to={entryLink(p.collection, p.itemId)}>
                    Open entry
                    <FiExternalLink />
                  </RouterLink>
                </Button>
              )}
            </HStack>
          </Section>

          {p.reason && (
            <Section title="Agent's reason">
              <Text fontSize="sm" whiteSpace="pre-wrap" bg={quoteBg} borderLeftWidth="3px" borderColor="purple.300" px={3} py={2} borderRadius="sm">
                {p.reason}
              </Text>
            </Section>
          )}

          {!pending && (
            <Alert.Root role="alert" status={p.status === "approved" ? "success" : p.status === "failed" ? "error" : "info"} borderRadius="md" alignItems="start">
              <Alert.Indicator />
              <Alert.Content>
                <Alert.Title textTransform="capitalize">{p.status === "applying" ? "Being applied" : p.status}</Alert.Title>
                <Alert.Description fontSize="sm">
                  {[p.reviewedBy && `By ${p.reviewedBy}`, p.decidedAt && formatTime(p.decidedAt)].filter(Boolean).join(", ")}
                  {p.reviewNote && <Text mt={1}>“{p.reviewNote}”</Text>}
                  {p.error && <Text mt={1}>{p.error}</Text>}
                </Alert.Description>
              </Alert.Content>
            </Alert.Root>
          )}

          {showStale && (
            <Alert.Root role="alert" status="warning" borderRadius="md" alignItems="start" data-testid="stale-warning">
              <Alert.Indicator />
              <Alert.Content flex="1">
                <Alert.Title>The entry changed since the agent proposed this</Alert.Title>
                <Alert.Description fontSize="sm" display="block">
                  The “current” column shows the entry as it is now, and “proposed” what approving writes: only the fields the agent
                  changed are applied over it.
                  {canForce && " Approve anyway to apply them."}
                </Alert.Description>
                {editReset && mode === "edit" && (
                  <Text fontSize="sm" mt={2} fontWeight="medium" data-testid="edit-reset-note">
                    The entry changed while you were editing: the values to apply were reset to the entry as it is now with the agent's
                    changes. Review them, edit them again if needed, and approve.
                  </Text>
                )}
                {canForce && (
                  <Button mt={2} size="xs" colorPalette="orange" alignSelf="start" onClick={() => approve(true)} loading={busy === "force"} disabled={working}>
                    Apply anyway
                  </Button>
                )}
              </Alert.Content>
            </Alert.Root>
          )}

          <Section title={p.action === "delete" ? "Deletes the entry" : pending ? "Changes" : "Proposed values"}>
            {p.action === "delete" && (
              <Text fontSize="sm" mb={2}>
                The agent proposes to delete this entry{p.locale ? ` (its ${p.locale} translation)` : ""}.
              </Text>
            )}
            {!pending ? (
              // Decided: only what the agent proposed is kept.
              p.action !== "delete" && <JsonBlock value={p.data ?? {}} label="Proposed values" />
            ) : changes && changes.length > 0 ? (
              <ChangesTable changes={changes} beforeLabel={p.action === "create" ? "Before" : "Current"} afterLabel="Proposed" />
            ) : (
              <Text fontSize="sm" color="gray.500">
                {changes ? "No field differs from the entry as it is now." : "The collection no longer exists: this proposal can only be rejected."}
              </Text>
            )}
          </Section>

          {pending && mode === "edit" && (
            <Field.Root invalid={Boolean(parsed?.error)}>
              <Field.Label fontSize="sm">Values to apply</Field.Label>
              <Textarea
                rows={12}
                fontFamily="mono"
                fontSize="sm"
                value={edited}
                onChange={(e) => setEdited(e.target.value)}
                disabled={working}
                aria-label="Values to apply (JSON)"
                spellCheck={false}
              />
              {parsed?.error ? (
                <Field.ErrorText>{parsed.error}</Field.ErrorText>
              ) : (
                <Field.HelperText>
                  The entry's values after the change{p.locale ? `, in ${p.locale}` : ""}, as JSON. They replace the agent's values.
                </Field.HelperText>
              )}
            </Field.Root>
          )}

          {pending && mode === "reject" && (
            <Field.Root>
              <Field.Label fontSize="sm">Note (optional)</Field.Label>
              <Textarea rows={3} disabled={working} value={note} onChange={(e) => setNote(e.target.value)} placeholder="Why the change is rejected" />
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
              <Button variant="outline" colorPalette="red" onClick={() => setMode("reject")} mr="auto" disabled={working}>
                <FiX />
                Reject
              </Button>
              {p.action !== "delete" && (
                <Button variant="outline" onClick={() => setMode("edit")} disabled={working}>
                  <FiEdit3 />
                  Edit and approve
                </Button>
              )}
              <Button colorPalette="green" onClick={() => approve(false)} loading={busy === "approve"} disabled={working}>
                <FiCheck />
                Approve
              </Button>
            </>
          )}
          {mode === "edit" && (
            <>
              <Button variant="ghost" onClick={() => setMode("review")} disabled={working}>
                Cancel
              </Button>
              <Button colorPalette="green" onClick={() => approve(false)} loading={busy === "approve"} disabled={working || Boolean(parsed?.error)}>
                <FiCheck />
                Approve with edits
              </Button>
            </>
          )}
          {mode === "reject" && (
            <>
              <Button variant="ghost" onClick={() => setMode("review")} disabled={working}>
                Cancel
              </Button>
              <Button colorPalette="red" onClick={reject} loading={busy === "reject"} disabled={working}>
                <FiX />
                Reject proposal
              </Button>
            </>
          )}
        </DrawerFooter>
      )}
    </>
  );
}
