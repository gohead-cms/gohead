import { Badge } from "@chakra-ui/react";
import { roleColor } from "./settingsFormat";

export function RoleBadge({ role }: { role: string }) {
  return <Badge colorScheme={roleColor[role] ?? "gray"}>{role || "none"}</Badge>;
}
