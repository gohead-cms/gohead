import { Link as RouterLink, Navigate, useParams } from "react-router-dom";
import { Alert, AlertDescription, AlertIcon, Button, Center, Spinner } from "@chakra-ui/react";
import { useApiQuery } from "../../shared/hooks/useApiQuery";
import { EmptyState } from "../../shared/ui/EmptyState";
import { Page, PageHeader } from "../../shared/ui/PageHeader";
import { loadSummaries, loadType } from "./api";
import { TypeEditorForm } from "./components/TypeEditorForm";
import { emptyForm, isModelKind, KIND_LABELS } from "./schema";

/** Create (/model/{kind}/new) or edit (/model/{kind}/{name}) a content type. */
export function TypeEditorPage() {
  const params = useParams<{ kind: string; name?: string }>();
  const kind = isModelKind(params.kind) ? params.kind : null;
  const name = params.name ?? null;

  const existing = useApiQuery(kind && name ? `model-type:${kind}:${name}` : null, () => loadType(kind!, name!));
  const collections = useApiQuery(kind ? "model:collections" : null, () => loadSummaries("collections"));
  const components = useApiQuery(kind ? "model:components" : null, () => loadSummaries("components"));
  const singletons = useApiQuery(kind === "singletons" ? "model:singletons" : null, () => loadSummaries("singletons"));

  if (!kind) return <Navigate to="/model" replace />;

  const { singular } = KIND_LABELS[kind];
  const backTo = kind === "collections" ? "/model" : `/model?tab=${kind}`;
  const queries = [existing, collections, components, singletons];
  const error = queries.find((q) => q.error)?.error;
  const loading = queries.some((q) => q.loading);

  if (error) {
    return (
      <Page maxW="6xl">
        <PageHeader title={name ? `Edit ${singular} "${name}"` : `New ${singular}`} />
        <Alert status="error" borderRadius="md">
          <AlertIcon />
          <AlertDescription flex="1">{error}</AlertDescription>
          <Button size="sm" onClick={() => queries.forEach((q) => q.refetch())}>
            Retry
          </Button>
        </Alert>
      </Page>
    );
  }
  if (loading) {
    return (
      <Center py={20}>
        <Spinner />
      </Center>
    );
  }
  if (name && !existing.data) {
    return (
      <Page maxW="6xl">
        <PageHeader title={`Edit ${singular}`} />
        <EmptyState
          title={`No ${singular} named "${name}"`}
          action={
            <Button as={RouterLink} to={backTo} colorScheme="teal" size="sm">
              Back to the content model
            </Button>
          }
        >
          It may have been deleted or renamed.
        </EmptyState>
      </Page>
    );
  }

  const collectionNames = (collections.data ?? []).map((c) => c.name);
  const componentNames = (components.data ?? []).map((c) => c.name);
  const sameKind = kind === "collections" ? collectionNames : kind === "components" ? componentNames : (singletons.data ?? []).map((s) => s.name);

  return (
    <Page maxW="6xl">
      <TypeEditorForm
        key={`${kind}:${name ?? "new"}`}
        kind={kind}
        existingName={name}
        initial={existing.data?.form ?? emptyForm()}
        incompleteSchema={!!existing.data?.incomplete}
        collections={collectionNames}
        components={componentNames}
        takenNames={sameKind.filter((n) => n !== name)}
      />
    </Page>
  );
}
