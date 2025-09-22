import React, { useEffect, useState, useCallback, useRef } from "react";
import {
  ReactFlow,
  applyNodeChanges,
  applyEdgeChanges,
  addEdge,
  Node,
  Edge,
  NodeChange,
  EdgeChange,
  Connection,
  Panel,
  MarkerType,
  Background,
  Controls,
  BackgroundVariant
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import type { Schema, AgentNodeData } from "../../shared/types";
import CollectionEdge from "./components/CollectionEdge";
import CollectionNode from "./components/CollectionNode";
import AgentNode from "./components/AgentNode";
import { FaRobot } from "react-icons/fa";
import { List, ListItem } from "@chakra-ui/react";
import {
  Box,
  Spinner,
  Button,
  Flex,
  IconButton,
  useDisclosure,
  Text,
  VStack,
  HStack,
  Badge,
  Tabs,
  TabList,
  TabPanels,
  Tab,
  TabPanel,
  FormControl,
  FormLabel,
  Input,
  FormHelperText,
  FormErrorMessage,
  Menu,
  MenuList,
  MenuItem,
  useToast,
} from "@chakra-ui/react";
import {
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalFooter,
  ModalBody,
  ModalCloseButton,
} from '@chakra-ui/react';
import { FaPlus, FaEdit, FaTrash, FaEye, FaArrowsAltV, FaArrowsAltH } from "react-icons/fa";
import { apiFetchWithAuth } from "../../services/api";
import { CollectionEdgeType } from "../../shared/types";
import dagre from "@dagrejs/dagre";
import { AttributeEditorSidebar } from "./components/AttributeEditorSidebar";
import { RelationModal } from "./components/RelationModal";

type AttributeItem = {
  name: string;
  type: string;
};

type CollectionNode = Node<{ label: string; attributes: AttributeItem[] }, 'collectionNode'>;
type AgentNode = Node<AgentNodeData, 'agentNode'>;

type AppNode = CollectionNode | AgentNode;

// --- Dagre layout logic start ---
const dagreGraph = new dagre.graphlib.Graph();
dagreGraph.setDefaultEdgeLabel(() => ({}));

const nodeWidth = 200;
const nodeHeight = 60;

function getLayoutedElements(nodes: AppNode[], edges: Edge[], direction = "TB") {
  dagreGraph.setGraph({
    rankdir: direction,
    nodesep: 300,
    ranksep: 350,
  });
  nodes.forEach((node) => {
    dagreGraph.setNode(node.id, { width: nodeWidth, height: nodeHeight });
  });

  edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target);
  });

  dagre.layout(dagreGraph);

  const layoutedNodes = nodes.map((node) => {
    const nodeWithPosition = dagreGraph.node(node.id);

    node.position = {
      x: nodeWithPosition.x - nodeWidth / 2,
      y: nodeWithPosition.y - nodeHeight / 2,
    };

    return node;
  });

  return { nodes: layoutedNodes, edges };
}
// --- Dagre layout logic end ---

export default function SchemaStudio() {
  const reactFlowWrapperRef = useRef<HTMLDivElement>(null);
  const toast = useToast();
  const [nodes, setNodes] = useState<AppNode[]>([]);
  const [edges, setEdges] = useState<Edge[]>([]); // Generic Edge type for both
  const [loading, setLoading] = useState(true);
  const [layoutDirection, setLayoutDirection] = useState("LR");
  const [selectedNode, setSelectedNode] = useState<AppNode | null>(null);
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; node: AppNode; } | null>(null);

  const { isOpen, onOpen, onClose } = useDisclosure();
  // States for modals and sidebars
  const { isOpen: isViewerOpen, onOpen: onViewerOpen, onClose: onViewerClose } = useDisclosure();
  const [viewingNode, setViewingNode] = useState<AppNode | null>(null);
  
  const { isOpen: isCollectionEditorOpen, onOpen: onCollectionEditorOpen, onClose: onCollectionEditorClose } = useDisclosure();
  const [editingCollection, setEditingCollection] = useState<CollectionNode | null>(null);
  
  const { isOpen: isAgentEditorOpen, onOpen: onAgentEditorOpen, onClose: onAgentEditorClose } = useDisclosure();
  const [editingAgent, setEditingAgent] = useState<AgentNode | null>(null);

  const { isOpen: isCreateModalOpen, onOpen: onCreateModalOpen, onClose: onCreateModalClose } = useDisclosure();
  const [newCollectionData, setNewCollectionData] = useState({ displayName: "", singularId: "", pluralId: "" });
  const [validationErrors, setValidationErrors] = useState({ displayName: "" });
  
  const { isOpen: isRelationModalOpen, onOpen: onRelationModalOpen, onClose: onRelationModalClose } = useDisclosure();
  const [newConnection, setNewConnection] = useState<Connection | null>(null);
  const fetchDataAndLayout = useCallback((direction: string) => {
    setLoading(true);
    Promise.all([
      apiFetchWithAuth("/admin/collections"),
      apiFetchWithAuth("/admin/agents"),
    ]).then(async ([collectionsRes, agentsRes]) => {
      // Process Collections
      const collectionsJson = await collectionsRes.json();
      const collections: { schema: Schema }[] = collectionsJson.data || [];
      const collectionNodes: CollectionNode[] = collections.map((col) => ({
        id: col.schema.collectionName!,
        type: "collectionNode" as const,
        position: { x: 0, y: 0 },
        data: {
          label: col.schema.info?.displayName || col.schema.collectionName || '',
          attributes: Object.entries(col.schema.attributes).map(([name, attr]: [string, { type: string }]) => ({ name, type: attr.type })),
        },
      }));

      const collectionEdges: CollectionEdgeType[] = collections.flatMap((col) => {
        const collectionName = col.schema.collectionName;
        if (!collectionName) return [];
        return Object.entries(col.schema.attributes || {}).map(([attrName, attr]: [string, any]) => {
          if (attr.type.includes("relation") && attr.target) {
            const targetName = attr.target.split(".").pop();
            return {
              id: `${collectionName}-${targetName!}-${attrName}`,
              source: collectionName, target: targetName!, animated: true, style: { stroke: "#52b4ca" }, type: "collection",
              markerEnd: { type: MarkerType.ArrowClosed }, data: { label: attrName, relationType: attr.relation }
            };
          }
          return null;
        }).filter(Boolean) as CollectionEdgeType[];
      });

      // Process Agents
      const agentsJson = await agentsRes.json();
      const agents: AgentNodeData[] = agentsJson.data || [];
      const agentNodes: AgentNode[] = agents.map((agent) => ({
        id: agent.name,
        type: "agentNode" as const,
        position: { x: 0, y: 0 },
        data: agent,
      }));

      const agentTriggerEdges: Edge[] = agents.flatMap((agent) => {
        const trigger = agent.schema?.trigger;
        if (trigger?.type === 'collection_event' && trigger.event_trigger?.collection) {
          return {
            id: `trigger-${trigger.event_trigger.collection}-to-${agent.name}`,
            source: trigger.event_trigger.collection, target: agent.name, type: 'smoothstep', animated: true,
            style: { stroke: '#8a52ca', strokeDasharray: '5,5' }, markerEnd: { type: MarkerType.ArrowClosed, color: '#8a52ca' },
          };
        }
        return [];
      });

      // Combine and Layout
      const allNodes: AppNode[] = [...collectionNodes, ...agentNodes];
      const allEdges: Edge[] = [...collectionEdges, ...agentTriggerEdges];
      const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(allNodes, allEdges, direction);

      setNodes(layoutedNodes);
      setEdges(layoutedEdges);
    }).finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    fetchDataAndLayout(layoutDirection);
  }, [layoutDirection, fetchDataAndLayout]);
  const onNodesChange = useCallback(
    (changes: NodeChange[]) => 
      setNodes((nds) => applyNodeChanges(changes, nds) as AppNode[]),
    []
  );

  const onEdgesChange = useCallback((changes: EdgeChange[]) => {
    changes.forEach(change => {
      if (change.type === 'remove' && change.id) {
        const edgeToDelete = edges.find(edge => edge.id === change.id);
        if (edgeToDelete?.type === 'collection' && edgeToDelete.data) {
          const edgeData = edgeToDelete.data;
          setNodes(currentNodes =>
            currentNodes.map(node => {
              if (node.id === edgeToDelete.source && node.type === 'collectionNode') {
                const updatedAttributes = node.data.attributes.filter(attr => attr.name !== edgeData.label);
                return { ...node, data: { ...node.data, attributes: updatedAttributes } };
              }
              return node;
            })
          );
        }
      }
    });
    setEdges((eds) => applyEdgeChanges(changes, eds));
  }, [edges, setNodes, setEdges]);

  const onConnect = useCallback(
    (params: Connection) => {
      if (params.source === params.target) return;
      setNewConnection(params);
      onRelationModalOpen();
    },
    [onRelationModalOpen]
  );
  const onNodeClick = useCallback(
    (event: React.MouseEvent, node: AppNode) => {
      setSelectedNode(node);
    },
    []
  );
  
  const onNodeContextMenu = useCallback(
    (event: React.MouseEvent, node: AppNode) => {
      event.preventDefault();
      if (reactFlowWrapperRef.current) {
        const rect = reactFlowWrapperRef.current.getBoundingClientRect();
        setContextMenu({
          x: event.clientX - rect.left,
          y: event.clientY - rect.top,
          node,
        });
        setSelectedNode(node);
      }
    },
    []
  );
  const onPaneClick = useCallback(() => {
    setContextMenu(null);
    setSelectedNode(null);
  }, []);

  const handleDisplayNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const displayName = e.target.value;
    const singularId = displayName.toLowerCase().replace(/\s/g, "-");
    const pluralId = singularId ? `${singularId}s` : "";

    setNewCollectionData({
      displayName,
      singularId,
      pluralId
    });

    if (displayName.trim() === "") {
      setValidationErrors({ displayName: "Display name is required." });
    } else {
      setValidationErrors({ displayName: "" });
    }
  };

  const handleAddCollection = () => {
    setNewCollectionData({
      displayName: "",
      singularId: "",
      pluralId: ""
    });
    setValidationErrors({ displayName: "" });
    onCreateModalOpen();
  };

  const handleContinueClick = () => {
    if (newCollectionData.displayName.trim() === "") {
      setValidationErrors({ displayName: "Display name is required." });
      return;
    }

    const newNode: CollectionNode = {
      id: newCollectionData.singularId,
      type: "collectionNode",
      position: { x: 50, y: 50 },
      data: {
        label: newCollectionData.displayName,
        attributes: [],
      },
    };
    
    const nodeExists = nodes.some(n => n.id === newNode.id);
    if (nodeExists) {
      setValidationErrors({ displayName: "A collection with this ID already exists." });
      return;
    }

    // This now works because newNode is correctly typed
    setNodes((currentNodes) => [...currentNodes, newNode]);

    // Use the updated state setters from the refactor
    setEditingCollection(newNode);
    onCollectionEditorOpen();
    
    onCreateModalClose();
    setNewCollectionData({
      displayName: "",
      singularId: "",
      pluralId: ""
    });
  };

  const handleUpdateNodeAttributes = useCallback(
    (nodeId: string, newAttributes: AttributeItem[]) => {
      setNodes((currentNodes) =>
        currentNodes.map((node) => {
          // TYPE GUARD: Check both the ID and the node type.
          if (node.id === nodeId && node.type === 'collectionNode') {
            return {
              ...node,
              data: {
                ...node.data,
                attributes: newAttributes,
              },
            };
          }
          // For all other nodes (including AgentNodes), return them unmodified.
          return node;
        })
      );
    },
    []
  );
  
  const handleSaveRelation = useCallback((sourceId: string, targetId: string, relation: { type: string; label: string; }) => {
      if (!newConnection || !newConnection.source || !newConnection.target) return;

      const newEdge: CollectionEdgeType = {
        id: `${newConnection.source}-${newConnection.target}-${relation.label}`,
        source: newConnection.source,
        target: newConnection.target,
        animated: true,
        style: { stroke: "#52b4ca" },
        type: "collection",
        markerEnd: {
          type: MarkerType.ArrowClosed,
        },
        data: {
          label: relation.label,
          relationType: relation.type,
          attributes: [],
        }
      };

      setNodes(currentNodes =>
        currentNodes.map(node => {
          // TYPE GUARD: Check both the ID and the node type.
          if (node.id === newConnection.source && node.type === 'collectionNode') {
            // Inside this block, TypeScript knows `node` is a CollectionNode.
            const newAttribute: AttributeItem = { 
              name: relation.label, 
              type: `Relation: ${relation.type}` 
            };
            
            return {
              ...node,
              data: {
                ...node.data,
                attributes: [...node.data.attributes, newAttribute]
              }
            };
          }
          // For all other nodes (including AgentNodes), return them unmodified.
          return node;
        })
      );
      setEdges((eds) => addEdge(newEdge, eds));

      toast({
        title: "Relation created.",
        description: `A ${relation.type} relation named "${relation.label}" was created.`,
        status: "success",
        duration: 5000,
        isClosable: true,
      });

      onRelationModalClose();
      setNewConnection(null);
    },
    [newConnection, onRelationModalClose, toast]
  );
  
  const handleViewDetails = (nodeToView: AppNode | null) => {
    if (nodeToView) {
      // No casting is needed because nodeToView is already the correct type.
      setViewingNode(nodeToView);
      onViewerOpen(); // Use the updated disclosure function name
    }
    setContextMenu(null);
  };

  const handleEdit = (nodeToEdit: AppNode | null) => {
    if (nodeToEdit) {
      if(nodeToEdit.type === 'collectionNode') {
        setEditingCollection(nodeToEdit as CollectionNode);
        onCollectionEditorOpen();
      } else if (nodeToEdit.type === 'agentNode') {
        setEditingAgent(nodeToEdit as AgentNode);
        onAgentEditorOpen();
      }
    }
    setContextMenu(null);
  };

  const handleDelete = (nodeToDelete: AppNode | null) => {
    if (nodeToDelete) {
      setNodes(nds => nds.filter(n => n.id !== nodeToDelete.id));
      setEdges(eds => eds.filter(e => e.source !== nodeToDelete.id && e.target !== nodeToDelete.id));
      toast({
        title: `${nodeToDelete.type === 'agentNode' ? 'Agent' : 'Collection'} deleted.`,
        status: "success", duration: 5000, isClosable: true,
      });
    }
    setContextMenu(null);
  };

  const handleCollectionEditorClose = () => {
    setEditingCollection(null);
    onCollectionEditorClose();
  };


const nodeTypes = { collectionNode: CollectionNode, agentNode: AgentNode };
  const edgeTypes = { collection: CollectionEdge };

  if (loading) return <Spinner />;
  
  const viewingNodeAttributes = viewingNode?.type === 'collectionNode' 
    ? viewingNode.data.attributes 
    : undefined;

  const sourceNode = newConnection?.source
    ? nodes.find(n => n.id === newConnection.source && n.type === 'collectionNode') as CollectionNode | undefined
    : undefined;

  const targetNode = newConnection?.target
    ? nodes.find(n => n.id === newConnection.target && n.type === 'collectionNode') as CollectionNode | undefined
    : undefined;

  const allNodeLabels = nodes.map(node => (node.data as { label: string; attributes: AttributeItem[] }).label);

  return (
    <Box w="100%" h="calc(100vh - 64px)" bg="#fafdff" ref={reactFlowWrapperRef}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeClick={onNodeClick}
        onNodeContextMenu={onNodeContextMenu}
        onPaneClick={onPaneClick}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        fitView
        fitViewOptions={{
          padding: 1.0,
        }}
      >
        <Panel position="top-right">
          <Flex gap={2} p={2} bg="white" borderRadius="md" boxShadow="md" alignItems="center">
            <Button size="sm" colorScheme="teal" onClick={handleAddCollection}>
              <Flex align="center" gap={2}>
                <FaPlus />
                <span>Add Collection</span>
              </Flex>
            </Button>
            <IconButton
              size="sm"
              aria-label="View Details"
              onClick={() => handleViewDetails(selectedNode)}
              disabled={!selectedNode}
            >
              <FaEye />
            </IconButton>
            <IconButton
              size="sm"
              aria-label="Edit Collection"
              onClick={() => handleEdit(selectedNode)}
              disabled={!selectedNode}
            >
              <FaEdit />
            </IconButton>
            <IconButton
              size="sm"
              aria-label="Delete"
              onClick={() => handleDelete(selectedNode)}
              disabled={!selectedNode}
              colorScheme="red"
            >
              <FaTrash />
            </IconButton>
            
            <Box h="24px" w="1px" bg="gray.300" mx={2} />

            <IconButton
              size="sm"
              aria-label="Top-to-Bottom Layout"
              onClick={() => setLayoutDirection("TB")}
              colorScheme={layoutDirection === "TB" ? "teal" : "gray"}
            >
              <FaArrowsAltV />
            </IconButton>
            <IconButton
              size="sm"
              aria-label="Left-to-Right Layout"
              onClick={() => setLayoutDirection("LR")}
              colorScheme={layoutDirection === "LR" ? "teal" : "gray"}
            >
              <FaArrowsAltH />
            </IconButton>
          </Flex>
        </Panel>
        
        <Controls />
        <Background variant={"dots" as BackgroundVariant} gap={12} size={1} />
      </ReactFlow>

      {contextMenu && (
        <Box
          position="absolute"
          top={`${contextMenu.y}px`}
          left={`${contextMenu.x}px`}
          zIndex={100}
        >
          <Menu isOpen>
            <MenuList>
              <MenuItem icon={<FaEye />} onClick={() => handleViewDetails(contextMenu.node)}>
                View Details
              </MenuItem>
              <MenuItem icon={<FaEdit />} onClick={() => handleEdit(contextMenu.node)}>
                Edit
              </MenuItem>
              <MenuItem icon={<FaTrash />} onClick={() => handleDelete(contextMenu.node)}>
                Delete
              </MenuItem>
            </MenuList>
          </Menu>
        </Box>
      )}

      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>
            Details: {
              // FIX: Conditionally display the correct title property
              viewingNode?.type === 'collectionNode' 
                ? viewingNode.data.label 
                : viewingNode?.data.name
            }
          </ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text mb={4} fontWeight="bold">
              Attributes:
            </Text>
            {Array.isArray(viewingNodeAttributes) && viewingNodeAttributes.length > 0 ? (
            <VStack align="start" spacing={2}>
              {viewingNodeAttributes.map((attr, index) => (
                <HStack key={index} w="100%" justifyContent="space-between">
                  <Text fontWeight="bold">{attr.name}</Text>
                  <Badge colorScheme="blue">{attr.type}</Badge>
                </HStack>
              ))}
            </VStack>
            ) : (
            <Text>This collection has no attributes.</Text>
            )}
          </ModalBody>
          <ModalFooter>
            <Button colorScheme="blue" onClick={onClose}>
              Close
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      <Modal isOpen={isCreateModalOpen} onClose={onCreateModalClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Create a single type</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Tabs variant="enclosed">
              <TabList>
                <Tab>BASE SETTINGS</Tab>
                <Tab>ADVANCED SETTINGS</Tab>
              </TabList>
              <TabPanels>
                <TabPanel>
                  <VStack spacing={4} align="stretch">
                    <FormControl id="displayName" isInvalid={!!validationErrors.displayName}>
                      <FormLabel>Display name</FormLabel>
                      <Input
                        value={newCollectionData.displayName}
                        onChange={handleDisplayNameChange}
                      />
                      <FormErrorMessage>{validationErrors.displayName}</FormErrorMessage>
                    </FormControl>
                    <FormControl id="singularId">
                      <FormLabel>API ID (Singular)</FormLabel>
                      <Input
                        value={newCollectionData.singularId}
                        isReadOnly
                        bg="gray.100"
                      />
                      <FormHelperText>
                        The API ID is used to generate the API routes and databases tables/collections.
                      </FormHelperText>
                    </FormControl>
                    <FormControl id="pluralId">
                      <FormLabel>API ID (Plural)</FormLabel>
                      <Input
                        value={newCollectionData.pluralId}
                        isReadOnly
                        bg="gray.100"
                      />
                      <FormHelperText>
                        Pluralized API ID
                      </FormHelperText>
                    </FormControl>
                  </VStack>
                </TabPanel>
                <TabPanel>
                  <Text>Advanced settings are not yet implemented.</Text>
                </TabPanel>
              </TabPanels>
            </Tabs>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onCreateModalClose}>
              Cancel
            </Button>
            <Button colorScheme="teal" onClick={handleContinueClick} isDisabled={!!validationErrors.displayName || !newCollectionData.displayName.trim()}>
              Continue
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
      
       <AttributeEditorSidebar
        isOpen={isCollectionEditorOpen}
        onClose={handleCollectionEditorClose}
        node={editingCollection}
        onSave={handleUpdateNodeAttributes}
      />
      
      <RelationModal
        isOpen={isRelationModalOpen}
        onClose={() => {
          onRelationModalClose();
          setNewConnection(null);
        }}
        sourceNodeLabel={sourceNode?.data.label || null}
        targetNodeLabel={targetNode?.data.label || null}
        onSave={handleSaveRelation}
        allNodeLabels={allNodeLabels}
      />
    </Box>
  );
}