import { SimpleGrid } from "@chakra-ui/react";
import { FiBox, FiCpu, FiFileText, FiImage, FiShare2, FiUsers } from "react-icons/fi";
import type { Overview } from "../overview";
import { formatBytes } from "../overview";
import { StatCard } from "./StatCard";

const plural = (n: number, word: string) => `${n.toLocaleString()} ${word}${n === 1 ? "" : "s"}`;

/** The instance's counts: content, model, media, people and integrations. */
export function StatGrid({ counts }: { counts: Overview["counts"] }) {
  return (
    <SimpleGrid columns={{ base: 1, md: 2, xl: 3 }} spacing={4}>
      <StatCard label="Entries" value={counts.items} detail="Across all collections" icon={FiFileText} to="/contrib" />
      <StatCard
        label="Content model"
        value={plural(counts.collections, "collection")}
        detail={`${plural(counts.components, "component")} · ${plural(counts.singletons, "single type")}`}
        icon={FiBox}
        to="/model"
        color="blue"
      />
      <StatCard label="Media files" value={counts.media} detail={`${formatBytes(counts.mediaBytes)} stored`} icon={FiImage} to="/media" color="teal" />
      <StatCard label="Users" value={counts.users} detail="Admins, editors and viewers" icon={FiUsers} to="/settings/users" color="orange" />
      <StatCard label="Agents" value={counts.agents} detail="AI agents configured" icon={FiCpu} to="/agents" color="pink" />
      <StatCard
        label="Webhooks"
        value={counts.webhooks}
        detail={plural(counts.apiTokens, "API token")}
        icon={FiShare2}
        to="/settings/webhooks"
        color="cyan"
      />
    </SimpleGrid>
  );
}
