import { Badge, HStack, Icon, Link, Text } from "@chakra-ui/react";
import { Tooltip } from "@/components/ui/tooltip";
import { Link as RouterLink } from "react-router-dom";
import { FiCpu, FiKey, FiServer, FiUser } from "react-icons/fi";
import type { IconType } from "react-icons";
import { useMe } from "../auth/me";
import { type ActorKind, type Revision, type RevisionAction, actionColor, actionLabel, actorKindColor, agentRunLink } from "./history";

const actorIcon: Record<ActorKind, IconType> = {
  user: FiUser,
  token: FiKey,
  agent: FiCpu,
  system: FiServer,
};

export function ActionBadge({ action }: { action: RevisionAction }) {
  return (
    <Badge
      colorPalette={actionColor[action] ?? "gray"}
      title={action === "baseline" ? "Baseline: the content before history was recorded" : undefined}
    >
      {actionLabel[action] ?? action}
    </Badge>
  );
}

export function StatusBadge({ status }: { status?: string }) {
  if (!status) return null;
  return (
    <Badge variant="outline" colorPalette={status === "published" ? "green" : "gray"}>
      {status}
    </Badge>
  );
}

/**
 * Who made a change: an icon and badge for the kind of actor, its name, and
 * for an agent its run: a link for admins (the agents pages are admin-only),
 * text otherwise and with links=false (inside clickable rows).
 */
export function ActorTag({ revision, links = true }: { revision: Pick<Revision, "actorKind" | "actorName" | "agentRunId">; links?: boolean }) {
  const { actorKind, actorName, agentRunId } = revision;
  const { isAdmin } = useMe();
  return (
    <HStack gap={1.5} minW={0} fontSize="sm">
      <Tooltip content={actorKind}>
        <Badge colorPalette={actorKindColor[actorKind] ?? "gray"} display="inline-flex" alignItems="center" gap={1} aria-label={`${actorKind} actor`}>
          <Icon as={actorIcon[actorKind] ?? FiUser} />
          {actorKind}
        </Badge>
      </Tooltip>
      <Text as="span" lineClamp={1} title={actorName}>
        {actorName || "unknown"}
      </Text>
      {actorKind === "agent" && agentRunId !== undefined && (
        links && isAdmin && actorName ? (
          <Link asChild color="purple.600" whiteSpace="nowrap" onClick={(e) => e.stopPropagation()}>
            <RouterLink to={agentRunLink(actorName, agentRunId)}>run #{agentRunId}</RouterLink>
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
