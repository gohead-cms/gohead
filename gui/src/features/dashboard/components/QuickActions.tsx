import type { IconType } from "react-icons";
import { Link as RouterLink } from "react-router-dom";
import { Button, Icon, VStack } from "@chakra-ui/react";
import { FiBookOpen, FiCpu, FiEdit3, FiExternalLink, FiPlus, FiUpload } from "react-icons/fi";
import { API_BASE_URL } from "../../../services/api";

const ACTIONS: { label: string; to: string; icon: IconType }[] = [
  { label: "New collection", to: "/model/collections/new", icon: FiPlus },
  { label: "Upload media", to: "/media", icon: FiUpload },
  { label: "New agent", to: "/agents/new", icon: FiCpu },
  { label: "Content editor", to: "/contrib", icon: FiEdit3 },
];

/** Links to the most common tasks. */
export function QuickActions() {
  return (
    <VStack align="stretch" gap={2}>
      {ACTIONS.map((a) => (
        <Button key={a.to} asChild variant="outline" size="xs" justifyContent="flex-start">
          <RouterLink to={a.to}>
            <Icon as={a.icon} />
            {a.label}
          </RouterLink>
        </Button>
      ))}
      <Button asChild variant="outline" size="xs" justifyContent="flex-start">
        <a href={`${API_BASE_URL}/api/docs`} target="_blank" rel="noopener noreferrer">
          <Icon as={FiBookOpen} />
          API reference
          <Icon as={FiExternalLink} />
        </a>
      </Button>
    </VStack>
  );
}
