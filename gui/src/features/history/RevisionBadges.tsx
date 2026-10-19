import { Badge, HStack, Icon, Link, Text, Tooltip } from "@chakra-ui/react";
import { Link as RouterLink } from "react-router-dom";
import { FiCpu, FiKey, FiServer, FiUser } from "react-icons/fi";
import type { IconType } from "react-icons";
import { type ActorKind, type Revision, type RevisionAction, actionColor, actorKindColor, agentRunLink } from "./history";

const actorIcon: Record<ActorKind, IconType> = {
  user: FiUser,
  token: FiKey,
  agent: FiCpu,
  system: FiServer,
};

export function ActionBadge({ action }: { action: RevisionAction }) {
  return <Badge colorScheme={actionColor[action] ?? "gray"}>{action}</Badge>;
}

export function StatusBadge({ status }: { status?: string }) {
  if (!status) return null;
  return (
    <Badge variant="outline" colorScheme={status === "published" ? "green" : "gray"}>
      {status}
    </Badge>
  );
}

/**
 * Who made a change: an icon and badge for the kind of actor, its name, and
 * for an agent a link to the run (links=false shows the run as text, inside
 * clickable rows).
 */
export function ActorTag({ revision, links = true }: { revision: Pick<Revision, "actorKind" | "actorName" | "agentRunId">; links?: boolean }) {
  const { actorKind, actorName, agentRunId } = revision;
  return (
    <HStack spacing={1.5} minW={0} fontSize="sm">
      <Tooltip label={actorKind}>
        <Badge colorScheme={actorKindColor[actorKind] ?? "gray"} display="inline-flex" alignItems="center" gap={1} aria-label={`${actorKind} actor`}>
          <Icon as={actorIcon[actorKind] ?? FiUser} />
          {actorKind}
        </Badge>
      </Tooltip>
      <Text as="span" noOfLines={1} title={actorName}>
        {actorName || "unknown"}
      </Text>
      {actorKind === "agent" && agentRunId !== undefined && (
        links && actorName ? (
          <Link as={RouterLink} to={agentRunLink(actorName, agentRunId)} color="purple.600" whiteSpace="nowrap" onClick={(e) => e.stopPropagation()}>
            run #{agentRunId}
          </Link>
        ) : (
          <Text as="span" color="gray.500" whiteSpace="nowrap">
            run #{agentRunId}
          </Text>
        )
      )}
    </HStack>
  );
}
