import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Alert, AlertDescription, AlertIcon, Box, Button, Center, Spinner } from "@chakra-ui/react";
import { FiPlus, FiShare2 } from "react-icons/fi";
import { EmptyState } from "../../shared/ui/EmptyState";
import { DeleteNodeDialog } from "./components/DeleteNodeDialog";
import { WorkspaceGraph } from "./components/WorkspaceGraph";
import { buildGraph, graphSignature, layoutGraph } from "./graph";
import type { LayoutDirection, NodeRef } from "./graph";
import { useWorkspaceData } from "./hooks";

/** Schema Designer: a read-only map of the saved collections, relations and agents. */
export default function WorkspaceCanvas() {
  const navigate = useNavigate();
  const { data, error, refetch } = useWorkspaceData();
  const [direction, setDirection] = useState<LayoutDirection>("LR");
  const [deleting, setDeleting] = useState<NodeRef | null>(null);

  const graph = useMemo(() => {
    if (!data) return null;
    const { nodes, edges } = buildGraph(data.collections, data.agents);
    return { nodes: layoutGraph(nodes, edges, direction), edges, signature: graphSignature(nodes, edges) };
  }, [data, direction]);

  const newCollection = () => navigate("/model/collections/new");
  const edit = (ref: NodeRef) =>
    navigate(ref.kind === "collection" ? `/model/collections/${encodeURIComponent(ref.name)}` : `/agents/${encodeURIComponent(ref.name)}/edit`);

  let content;
  if (error) {
    content = (
      <Box p={8}>
        <Alert status="error" borderRadius="md">
          <AlertIcon />
          <AlertDescription flex="1">{error}</AlertDescription>
          <Button size="sm" onClick={refetch}>
            Retry
          </Button>
        </Alert>
      </Box>
    );
  } else if (!graph) {
    content = (
      <Center h="full">
        <Spinner />
      </Center>
    );
  } else if (graph.nodes.length === 0) {
    content = (
      <Box p={8}>
        <EmptyState
          icon={FiShare2}
          title="Nothing to show yet"
          action={
            <Button colorScheme="teal" size="sm" leftIcon={<FiPlus />} onClick={newCollection}>
              New collection
            </Button>
          }
        >
          The Schema Designer maps your collections, their relations and the agents they trigger. Create a collection to
          get started.
        </EmptyState>
      </Box>
    );
  } else {
    content = (
      <WorkspaceGraph
        // A new layout or a model change starts from fresh positions.
        key={`${direction}:${graph.signature}`}
        initialNodes={graph.nodes}
        edges={graph.edges}
        direction={direction}
        onDirection={setDirection}
        onNew={newCollection}
        onRefresh={refetch}
        onEdit={edit}
        onDelete={setDeleting}
        onViewLogs={() => navigate("/activity")}
      />
    );
  }

  return (
    <Box w="100%" h="calc(100vh - 64px)" bg="#fafdff" position="relative">
      {content}
      <DeleteNodeDialog target={deleting} onClose={() => setDeleting(null)} onDeleted={refetch} />
    </Box>
  );
}
