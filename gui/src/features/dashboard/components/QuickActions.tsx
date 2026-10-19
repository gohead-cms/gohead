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
    <VStack align="stretch" spacing={2}>
      {ACTIONS.map((a) => (
        <Button key={a.to} as={RouterLink} to={a.to} variant="outline" size="sm" justifyContent="flex-start" leftIcon={<Icon as={a.icon} />}>
          {a.label}
        </Button>
      ))}
      <Button
        as="a"
        href={`${API_BASE_URL}/api/docs`}
        target="_blank"
        rel="noopener noreferrer"
        variant="outline"
        size="sm"
        justifyContent="flex-start"
        leftIcon={<Icon as={FiBookOpen} />}
        rightIcon={<Icon as={FiExternalLink} />}
      >
        API reference
      </Button>
    </VStack>
  );
}
