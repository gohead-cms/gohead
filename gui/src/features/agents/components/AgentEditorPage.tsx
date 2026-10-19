import { useMemo } from "react";
import { Link as RouterLink, useParams } from "react-router-dom";
import { Alert, AlertIcon, Button, Center, Spinner } from "@chakra-ui/react";
import { Page, PageHeader } from "../../../shared/ui/PageHeader";
import { useAgentDetails } from "../hook/useAgentDetails";
import { useAgentsList } from "../hook/useAgentList";
import { useCollectionChoices, useFunctionCatalog } from "../hook/useEditorData";
import { draftFromAgent } from "./agentForm";
import { AgentForm } from "./AgentForm";

/** Create (/agents/new) or edit (/agents/:name/edit) an agent. */
export function AgentEditorPage() {
  const { name } = useParams<{ name: string }>();
  const editing = name ?? null;
  const { agentDetails, loading: agentLoading, error: agentError } = useAgentDetails(editing);
  const { agents, loading: listLoading } = useAgentsList();
  const { catalog, loading: catalogLoading, error: catalogError } = useFunctionCatalog();
  const { collections, loading: collectionsLoading, error: collectionsError } = useCollectionChoices();

  const takenNames = useMemo(() => agents.map((a) => a.name).filter((n) => n !== editing), [agents, editing]);
  const initial = useMemo(() => (editing && !agentDetails ? null : draftFromAgent(agentDetails)), [editing, agentDetails]);

  const loading = (editing && agentLoading) || listLoading || catalogLoading || collectionsLoading;
  const backTo = editing ? `/agents?name=${encodeURIComponent(editing)}` : "/agents";

  return (
    <Page maxW="5xl">
      <PageHeader
        title={editing ? `Edit ${editing}` : "New agent"}
        description="An LLM with a system prompt, the functions it may call, and the trigger that starts it."
        actions={
          <Button as={RouterLink} to={backTo} variant="ghost" size="sm">
            Back to agents
          </Button>
        }
      />
      {agentError ? (
        <Alert status="error" borderRadius="md">
          <AlertIcon />
          {agentError}
        </Alert>
      ) : loading || !initial ? (
        <Center py={16}>
          <Spinner />
        </Center>
      ) : (
        <AgentForm
          key={editing ?? "new"}
          initial={initial}
          originalName={editing}
          agentId={agentDetails?.id}
          takenNames={takenNames}
          catalog={catalog}
          catalogError={catalogError}
          collections={collections}
          collectionsError={collectionsError}
        />
      )}
    </Page>
  );
}
