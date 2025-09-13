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
import type { Schema } from "./CollectionTypes";
import CollectionEdge from "./CollectionEdge";
import CollectionNode from "./CollectionNode";
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
import { apiFetchWithAuth } from "../../utils/api";
import { CollectionEdgeType } from "./CollectionEdgeData";
import dagre from "@dagrejs/dagre";
import { AttributeEditorSidebar } from "./AttributeEditorSidebar";
import { RelationModal } from "./RelationModal";

type AttributeItem = {
  name: string;
  type: string;
};

// --- Dagre layout logic start ---
const dagreGraph = new dagre.graphlib.Graph();
dagreGraph.setDefaultEdgeLabel(() => ({}));

const nodeWidth = 200;
const nodeHeight = 60;

const getLayoutedElements = (nodes: Node[], edges: Edge[], direction = "TB") => {
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
};
// --- Dagre layout logic end ---

export default function SchemaStudio() {
  const reactFlowWrapperRef = useRef<HTMLDivElement>(null);
  const toast = useToast();
  const [nodes, setNodes] = useState<Node[]>([]);
  const [edges, setEdges] = useState<CollectionEdgeType[]>([]);
  const [loading, setLoading] = useState(true);
  const [layoutDirection, setLayoutDirection] = useState("LR");
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);
  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    node: Node | null;
  } | null>(null);

  const { 
    isOpen: isEditorOpen, 
    onOpen: onEditorOpen, 
    onClose: onEditorClose 
  } = useDisclosure();
  const [editingNode, setEditingNode] = useState<Node<{ label: string; attributes: AttributeItem[] }> | null>(null);

  const { isOpen, onOpen, onClose } = useDisclosure();
  const [viewingNode, setViewingNode] = useState<Node<{ label: string; attributes: AttributeItem[] }> | null>(null);

  const { 
    isOpen: isCreateModalOpen, 
    onOpen: onCreateModalOpen, 
    onClose: onCreateModalClose 
  } = useDisclosure();
  const [newCollectionData, setNewCollectionData] = useState({
    displayName: "",
    singularId: "",
    pluralId: ""
  });
  const [validationErrors, setValidationErrors] = useState({
    displayName: "",
  });
  
  const { 
    isOpen: isRelationModalOpen, 
    onOpen: onRelationModalOpen, 
    onClose: onRelationModalClose
  } = useDisclosure();
  const [newConnection, setNewConnection] = useState<Connection | null>(null);

  const fetchDataAndLayout = useCallback((direction: string) => {
    setLoading(true);
    apiFetchWithAuth("/admin/collections").then(async (res) => {
      const json = await res.json();
      const collections: { schema: Schema }[] = json.data || [];

      const builtNodes = collections.map((col) => ({
        id: col.schema.collectionName!,
        type: "collectionNode",
        position: { x: 0, y: 0 },
        data: {
          label: col.schema.info?.displayName || col.schema.collectionName,
          attributes: Object.entries(col.schema.attributes).map(([name, attr]: any) => ({
            name,
            type: attr.type,
          })),
        },
      }));

      const builtEdges: CollectionEdgeType[] = [];
      collections.forEach((col) => {
        const collectionName = col.schema.collectionName;
        if (!collectionName) return;

        const attrs = col.schema.attributes || {};
        Object.entries(attrs).forEach(([attrName, attr]: [string, any]) => {
          if (attr.type.includes("relation") && attr.target) {
            const targetName = attr.target.split(".").pop();
            if (!targetName) return;
            const relationType = attr.relation;

            builtEdges.push({
              id: `${collectionName}-${targetName}-${attrName}`,
              source: collectionName,
              target: targetName,
              animated: true,
              style: { stroke: "#52b4ca" },
              type: "collection",
              markerEnd: {
                type: MarkerType.ArrowClosed,
              },
              data: {
                label: attrName,
                attributes: Object.keys(col.schema.attributes || {}),
                relationType: relationType,
              }
            });
          }
        });
      });

      const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(builtNodes, builtEdges, direction);

      setNodes(layoutedNodes);
      setEdges(layoutedEdges as CollectionEdgeType[]);
      setLoading(false);
    });
  }, []);

  useEffect(() => {
    fetchDataAndLayout(layoutDirection);
  }, [layoutDirection, fetchDataAndLayout]);

  const onNodesChange = useCallback(
    (changes: NodeChange[]) => setNodes((nds) => applyNodeChanges(changes, nds)),
    []
  );

  const onEdgesChange = useCallback(
    (changes: EdgeChange<CollectionEdgeType>[]) => {
      changes.forEach(change => {
        if (change.type === 'remove' && change.id) {
          // The variable 'edgeToDelete' must be typed as 'CollectionEdgeType | undefined'
          // because the 'find' method can return undefined.
          const edgeToDelete: CollectionEdgeType | undefined = edges.find(edge => edge.id === change.id);

          // The optional chaining operator '?.' handles the case where edgeToDelete or
          // its data property might be undefined.
          if (edgeToDelete !== undefined && edgeToDelete.data !== undefined) {
  setNodes(currentNodes =>
    currentNodes.map(node => {
      if (node.id === edgeToDelete.source) {
        const typedNode = node as Node<{ label: string; attributes: AttributeItem[] }>;
        const updatedAttributes = typedNode.data.attributes.filter(
          (attr: AttributeItem) => attr.name !== edgeToDelete.data!.label
        );
        return {
          ...typedNode,
          data: {
            ...typedNode.data,
            attributes: updatedAttributes
          }
        };
      }
      return node;
    })
  );
}

        }
      });
      setEdges((eds) => applyEdgeChanges(changes, eds));
    },
    [edges, setNodes, setEdges]
  );

  const onConnect = useCallback(
    (params: Connection) => {
      setNewConnection(params);
      onRelationModalOpen();
    },
    [onRelationModalOpen]
  );

  const onNodeClick = useCallback((event: React.MouseEvent, node: Node) => {
    setSelectedNode(node);
  }, []);
  
  const onNodeContextMenu = useCallback(
    (event: React.MouseEvent, node: Node) => {
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

    const newNode: Node = {
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

    setNodes((currentNodes) => [...currentNodes, newNode]);
    setEditingNode(newNode as Node<{ label: string; attributes: AttributeItem[] }>);
    onEditorOpen();
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
          if (node.id === nodeId) {
            const typedNode = node as Node<{ label: string; attributes: AttributeItem[] }>;
            return {
              ...typedNode,
              data: {
                ...typedNode.data,
                attributes: newAttributes,
              },
            };
          }
          return node;
        })
      );
    },
    []
  );
  
  const handleSaveRelation = useCallback(
    (sourceId: string, targetId: string, relation: { type: string; label: string }) => {
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
          if (node.id === newConnection.source) {
            const typedNode = node as Node<{ label: string; attributes: AttributeItem[] }>;
            const newAttribute: AttributeItem = { name: relation.label, type: `Relation: ${relation.type}` };
            return {
              ...typedNode,
              data: {
                ...typedNode.data,
                attributes: [...typedNode.data.attributes, newAttribute]
              }
            };
          }
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
    [newConnection, onRelationModalClose, setEdges, setNodes, toast]
  );
  
  const handleViewDetails = (nodeToView: Node | null) => {
    if (nodeToView) {
      setViewingNode(nodeToView as Node<{ label: string; attributes: AttributeItem[] }>);
      onOpen();
    }
    setContextMenu(null);
  };

  const handleEditCollection = (nodeToEdit: Node | null) => {
    if (nodeToEdit) {
      setEditingNode(nodeToEdit as Node<{ label: string; attributes: AttributeItem[] }>);
      onEditorOpen();
    }
    setContextMenu(null);
  };

  const handleDeleteCollection = (nodeToDelete: Node | null) => {
    if (nodeToDelete) {
      setNodes(currentNodes => currentNodes.filter(node => node.id !== nodeToDelete.id));
      setEdges(currentEdges =>
        currentEdges.filter(
          edge => edge.source !== nodeToDelete.id && edge.target !== nodeToDelete.id
        )
      );
      toast({
        title: "Collection deleted.",
        description: `The collection "${nodeToDelete.data.label}" and its relations have been removed.`,
        status: "success",
        duration: 5000,
        isClosable: true,
      });
    }
    setContextMenu(null);
  };
  
  const handleEditorClose = () => {
    setEditingNode(null);
    onEditorClose();
  };

  const nodeTypes = { collectionNode: CollectionNode };
  const edgeTypes = { collection: CollectionEdge };

  if (loading) return <Spinner />;
  
  const viewingNodeAttributes = viewingNode?.data?.attributes;

  const sourceNode = newConnection?.source
    ? (nodes.find(n => n.id === newConnection.source) as Node<{ label: string; attributes: AttributeItem[] }>)
    : undefined;
  const targetNode = newConnection?.target
    ? (nodes.find(n => n.id === newConnection.target) as Node<{ label: string; attributes: AttributeItem[] }>)
    : undefined;
  
  // NEW: Collect all node labels to pass to the RelationModal
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
              onClick={() => handleEditCollection(selectedNode)}
              disabled={!selectedNode}
            >
              <FaEdit />
            </IconButton>
            <IconButton
              size="sm"
              aria-label="Delete Collection"
              onClick={() => handleDeleteCollection(selectedNode)}
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
              <MenuItem icon={<FaEdit />} onClick={() => handleEditCollection(contextMenu.node)}>
                Edit Collection
              </MenuItem>
              <MenuItem icon={<FaTrash />} onClick={() => handleDeleteCollection(contextMenu.node)}>
                Delete Collection
              </MenuItem>
            </MenuList>
          </Menu>
        </Box>
      )}

      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Collection Details: {viewingNode?.data?.label}</ModalHeader>
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
        isOpen={isEditorOpen}
        onClose={handleEditorClose}
        node={editingNode}
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
        // NEW: Pass the list of all node labels to the modal
        allNodeLabels={allNodeLabels}
      />
    </Box>
  );
}