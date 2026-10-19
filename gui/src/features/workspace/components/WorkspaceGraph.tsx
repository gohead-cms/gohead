import { createContext, useCallback, useContext, useState } from "react";
import { applyNodeChanges, Background, BackgroundVariant, Controls, Panel, ReactFlow } from "@xyflow/react";
import type { Edge, NodeChange, NodeProps } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Box, Button, ButtonGroup, Flex, IconButton, Text, Tooltip } from "@chakra-ui/react";
import { FaArrowsAltH, FaArrowsAltV } from "react-icons/fa";
import { FiEdit2, FiPlus, FiRefreshCw, FiTrash2 } from "react-icons/fi";
import type { AgentNodeType } from "../../../shared/types";
import type { AppNode } from "../../../shared/types/workspace";
import { AgentNode, TriggerEdge } from "../../agents";
import { CollectionEdge, CollectionNode } from "../../collections";
import { parseNodeId } from "../graph";
import type { LayoutDirection, NodeRef } from "../graph";

interface AgentHandlers {
  onEdit: (ref: NodeRef) => void;
  onDelete: (ref: NodeRef) => void;
  onViewLogs: (name: string) => void;
}

// Agent node actions come through a context so nodeTypes stays a stable
// module-level object (React Flow re-mounts every node when it changes).
const HandlersContext = createContext<AgentHandlers | null>(null);

function AgentNodeWithActions(props: NodeProps<AgentNodeType>) {
  const handlers = useContext(HandlersContext);
  const ref: NodeRef = { kind: "agent", name: props.data.name };
  return (
    <AgentNode
      data={props.data}
      selected={props.selected}
      onEdit={() => handlers?.onEdit(ref)}
      onDelete={() => handlers?.onDelete(ref)}
      onViewLogs={() => handlers?.onViewLogs(ref.name)}
    />
  );
}

const nodeTypes = { collectionNode: CollectionNode, agentNode: AgentNodeWithActions } as const;
const edgeTypes = { collection: CollectionEdge, triggerEdge: TriggerEdge } as const;

export interface WorkspaceGraphProps extends AgentHandlers {
  initialNodes: AppNode[];
  edges: Edge[];
  direction: LayoutDirection;
  onDirection: (d: LayoutDirection) => void;
  onNew: () => void;
  onRefresh: () => void;
}

/**
 * Read-only view of the saved model. Nodes can be moved and selected, but
 * nothing changes the model except the Edit and Delete actions.
 */
export function WorkspaceGraph({ initialNodes, edges, direction, onDirection, onNew, onRefresh, onEdit, onDelete, onViewLogs }: WorkspaceGraphProps) {
  const [nodes, setNodes] = useState<AppNode[]>(initialNodes);
  const onNodesChange = useCallback((changes: NodeChange[]) => setNodes((nds) => applyNodeChanges(changes, nds) as AppNode[]), []);
  const selected = nodes.find((n) => n.selected);
  const selectedRef = selected ? parseNodeId(selected.id) : null;

  return (
    <HandlersContext.Provider value={{ onEdit, onDelete, onViewLogs }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onNodeDoubleClick={(_, node) => {
          const ref = parseNodeId(node.id);
          if (ref) onEdit(ref);
        }}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        nodesConnectable={false}
        edgesFocusable={false}
        deleteKeyCode={null}
        fitView
        minZoom={0.2}
      >
        <Panel position="top-left">
          <Flex gap={2} p={2} bg="white" borderRadius="md" boxShadow="md" align="center" wrap="wrap">
            <Button size="sm" colorScheme="teal" leftIcon={<FiPlus />} onClick={onNew}>
              New collection
            </Button>
            <Box h="24px" w="1px" bg="gray.200" />
            <Text fontSize="sm" color="gray.600" maxW="200px" noOfLines={1}>
              {selectedRef ? `${selectedRef.kind}: ${selectedRef.name}` : "Select a node"}
            </Text>
            <Button size="sm" leftIcon={<FiEdit2 />} isDisabled={!selectedRef} onClick={() => selectedRef && onEdit(selectedRef)}>
              Edit
            </Button>
            <Button
              size="sm"
              leftIcon={<FiTrash2 />}
              colorScheme="red"
              variant="outline"
              isDisabled={!selectedRef}
              onClick={() => selectedRef && onDelete(selectedRef)}
            >
              Delete
            </Button>
            <Box h="24px" w="1px" bg="gray.200" />
            <ButtonGroup size="sm" isAttached variant="outline">
              <Tooltip label="Left to right">
                <IconButton
                  aria-label="Left-to-right layout"
                  icon={<FaArrowsAltH />}
                  isActive={direction === "LR"}
                  onClick={() => onDirection("LR")}
                />
              </Tooltip>
              <Tooltip label="Top to bottom">
                <IconButton
                  aria-label="Top-to-bottom layout"
                  icon={<FaArrowsAltV />}
                  isActive={direction === "TB"}
                  onClick={() => onDirection("TB")}
                />
              </Tooltip>
            </ButtonGroup>
            <IconButton size="sm" variant="ghost" aria-label="Reload the model" icon={<FiRefreshCw />} onClick={onRefresh} />
          </Flex>
        </Panel>
        <Controls showInteractive={false} />
        <Background variant={BackgroundVariant.Dots} gap={12} size={1} />
      </ReactFlow>
    </HandlersContext.Provider>
  );
}
