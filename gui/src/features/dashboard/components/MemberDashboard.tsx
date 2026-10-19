import type { IconType } from "react-icons";
import { Link as RouterLink } from "react-router-dom";
import { Badge, Box, Flex, HStack, Icon, LinkBox, LinkOverlay, SimpleGrid, Text } from "@chakra-ui/react";
import { FiEdit3, FiImage } from "react-icons/fi";
import { Page, PageHeader } from "../../../shared/ui/PageHeader";
import type { Me } from "../../auth/me";

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
  return (
    <Page maxW="5xl">
      <PageHeader title={me ? `Welcome, ${me.username}` : "Welcome"} description="Overview of your GoHead CMS system" />
      <Box bg="white" borderWidth="1px" borderRadius="lg" p={5} mb={6}>
        <HStack spacing={3} mb={1}>
          <Text fontWeight="semibold">Your role</Text>
          <Badge colorScheme={canWrite ? "purple" : "gray"}>{me?.role ?? "unknown"}</Badge>
        </HStack>
        <Text fontSize="sm" color="gray.500">
          {canWrite
            ? "You can create, edit and publish content and upload media. The content model, agents and settings are managed by admins."
            : "You can browse published content. Ask an admin if you need to edit content or manage media."}
        </Text>
      </Box>
      <SimpleGrid columns={{ base: 1, md: 2 }} spacing={4}>
        <Shortcut to="/contrib" icon={FiEdit3} title="Content editor">
          {canWrite ? "Write and update entries." : "Browse entries."}
        </Shortcut>
        {canWrite && (
          <Shortcut to="/media" icon={FiImage} title="Media library">
            Upload and manage images and files.
          </Shortcut>
        )}
      </SimpleGrid>
    </Page>
  );
}
