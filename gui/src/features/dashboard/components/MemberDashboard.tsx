import type { IconType } from "react-icons";
import { Link as RouterLink } from "react-router-dom";
import { Badge, Box, Flex, HStack, Icon, LinkBox, LinkOverlay, SimpleGrid, Text } from "@chakra-ui/react";
import { FiEdit3, FiImage } from "react-icons/fi";
import { Page, PageHeader } from "../../../shared/ui/PageHeader";
import { type Me, mePermissions } from "../../auth/me";
import { ANY_TYPE, allows, allowsAny, MEDIA, type Permissions } from "../../auth/permissions";

const VERBS: [string, string][] = [
  ["read", "browse"],
  ["create", "create"],
  ["update", "edit"],
  ["publish", "publish"],
  ["delete", "delete"],
];

function joinWords(words: string[]): string {
  return words.length <= 1 ? (words[0] ?? "") : `${words.slice(0, -1).join(", ")} and ${words[words.length - 1]}`;
}

/** What the role may do, in a sentence, e.g. "You can browse, create and edit content in some types only and browse media." */
function describe(p: Permissions): string {
  const granted = VERBS.filter(([a]) => allowsAny(p, a));
  if (granted.length === 0 && !allows(p, "read", MEDIA)) {
    return "Your role has no access to content yet. Ask an admin to give you a role with the permissions you need.";
  }
  const parts: string[] = [];
  if (granted.length) {
    // Granted on "*" means every type, apart from those listed on their own.
    const content = p.content ?? {};
    const everywhere = granted.every(([a]) => (content[ANY_TYPE] ?? []).includes(a) && Object.values(content).every((l) => (l ?? []).includes(a)));
    parts.push(`${joinWords(granted.map(([, v]) => v))} content${everywhere ? "" : " (in some types only)"}`);
  }
  if (allows(p, "create", MEDIA)) parts.push("upload media");
  else if (allows(p, "read", MEDIA)) parts.push("browse media");
  return `You can ${joinWords(parts)}. The content model, agents and settings are managed by admins.`;
}

function Shortcut({ to, icon, title, children }: { to: string; icon: IconType; title: string; children: string }) {
  return (
    <LinkBox bg="white" borderWidth="1px" borderRadius="lg" p={5} _hover={{ borderColor: "purple.300" }}>
      <Flex gap={4} align="start">
        <Flex p={2} bg="purple.50" borderRadius="md">
          <Icon as={icon} boxSize={5} color="purple.500" />
        </Flex>
        <Box>
          <LinkOverlay as={RouterLink} to={to} fontWeight="semibold">
            {title}
          </LinkOverlay>
          <Text fontSize="sm" color="gray.500">
            {children}
          </Text>
        </Box>
      </Flex>
    </LinkBox>
  );
}

/** The dashboard of editors and viewers: where to go next. */
export function MemberDashboard({ me, canWrite }: { me: Me | null; canWrite: boolean }) {
  const perms = mePermissions(me);
  const content = allowsAny(perms, "read");
  const media = allows(perms, "read", MEDIA);
  return (
    <Page maxW="5xl">
      <PageHeader title={me ? `Welcome, ${me.username}` : "Welcome"} description="Overview of your GoHead CMS system" />
      <Box bg="white" borderWidth="1px" borderRadius="lg" p={5} mb={6}>
        <HStack spacing={3} mb={1}>
          <Text fontWeight="semibold">Your role</Text>
          <Badge colorScheme={canWrite ? "purple" : "gray"}>{me?.role ?? "unknown"}</Badge>
        </HStack>
        <Text fontSize="sm" color="gray.500">
          {describe(perms)}
        </Text>
      </Box>
      <SimpleGrid columns={{ base: 1, md: 2 }} spacing={4}>
        {content && (
          <Shortcut to="/contrib" icon={FiEdit3} title="Content editor">
            {canWrite ? "Write and update entries." : "Browse entries."}
          </Shortcut>
        )}
        {media && (
          <Shortcut to="/media" icon={FiImage} title="Media library">
            {allows(perms, "create", MEDIA) ? "Upload and manage images and files." : "Browse images and files."}
          </Shortcut>
        )}
      </SimpleGrid>
    </Page>
  );
}
