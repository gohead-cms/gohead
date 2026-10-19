import { useMemo, useState } from "react";
import { Link as RouterLink, useParams } from "react-router-dom";
import { Alert, AlertDescription, AlertIcon, Box, Button, Center, Spinner } from "@chakra-ui/react";
import { Page, PageHeader } from "../../../shared/ui/PageHeader";
import { useAgentDetails } from "../hook/useAgentDetails";
import { useAgentsList } from "../hook/useAgentList";
import { useCollectionChoices, useFunctionCatalog } from "../hook/useEditorData";
import { type AgentDraft, draftFromAgent, draftFromDefinition } from "./agentForm";
import { AgentForm } from "./AgentForm";
import { TemplateGallery } from "./TemplateGallery";

/** How a new agent starts: blank, or from a template's definition. */
interface NewStart {
  draft: AgentDraft;
  /** The template's name. */
  template?: string;
}

/** Create (/agents/new) or edit (/agents/:name/edit) an agent. */
export function AgentEditorPage() {
  const { name } = useParams<{ name: string }>();
  const editing = name ?? null;
  const { agentDetails, loading: agentLoading, error: agentError } = useAgentDetails(editing);
  const { agents, loading: listLoading } = useAgentsList();
  const { catalog, loading: catalogLoading, error: catalogError } = useFunctionCatalog();
  const { collections, loading: collectionsLoading, error: collectionsError } = useCollectionChoices();

  // Creating: the gallery until a blank agent or a template is chosen.
  const [start, setStart] = useState<NewStart | null>(null);

  const takenNames = useMemo(() => agents.map((a) => a.name).filter((n) => n !== editing), [agents, editing]);
  const editInitial = useMemo(() => (editing && agentDetails ? draftFromAgent(agentDetails) : null), [editing, agentDetails]);
  const initial = editing ? editInitial : (start?.draft ?? null);

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
      ) : loading || (editing && !initial) ? (
        <Center py={16}>
          <Spinner />
        </Center>
      ) : !initial ? (
        <TemplateGallery
          collections={collections}
          onBlank={() => setStart({ draft: draftFromAgent(null) })}
          onRendered={(def, template) => setStart({ draft: draftFromDefinition(def), template: template.name })}
        />
      ) : (
        <>
          {!editing && (
            <Alert status="info" variant="left-accent" borderRadius="md" mb={5}>
              <AlertIcon />
              <AlertDescription flex="1" fontSize="sm">
                {start?.template
                  ? `Prepared from the ${start.template} template. Review it (model, functions, limits), then create the agent.`
                  : "A blank agent: fill in each section."}
              </AlertDescription>
              <Box>
                <Button size="sm" variant="ghost" onClick={() => setStart(null)}>
                  Choose another template
                </Button>
              </Box>
            </Alert>
          )}
          {/* Choosing again goes through the gallery, which starts a new form. */}
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
        </>
      )}
    </Page>
  );
}
