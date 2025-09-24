import React, { useEffect, useState, useCallback, useRef, useMemo, useContext, createContext } from "react";
import {
  ReactFlow,
  applyNodeChanges,
  applyEdgeChanges,
  addEdge,
  NodeChange,
  EdgeChange,
  Connection,
  Panel,
  MarkerType,
  Background,
  Controls,
  BackgroundVariant,
  NodeProps
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import {
  Box,
  Spinner,
  Button,
  Flex,
  IconButton,
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
  useDisclosure,
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

import {
  AgentNode,
  TriggerEdge,
} from '../agents'

import { 
  CollectionEdge, 
  CollectionNode, 
  AttributeEditorSidebar, 
  RelationModal,
} from '../collections';
// Types
import type { AttributeItem, AppNode, CollectionNode as CollectionNodeType } from "../../shared/types/workspace";
import type { CollectionEdgeType, AgentNodeType } from "../../shared/types";
import { 
  useWorkspaceData,
  useWorkspaceModals
} from './hooks'

const edgeTypes = { 
  collection: CollectionEdge, 
  triggerEdge: TriggerEdge 
};

type AgentNodeHandlers = {
  onEdit: (id: string) => void;
  onDelete: (id: string) => void;
  onViewLogs: (id: string) => void;
};

const HandlersContext = createContext<AgentNodeHandlers | null>(null);

const AgentNodeWithCtx = (props: NodeProps<AgentNodeType>) => {
const handlers = useContext(HandlersContext);
  return (
    <AgentNode
      data={props.data}
      selected={props.selected}
      onEdit={() => handlers?.onEdit(props.id)}
      onDelete={() => handlers?.onDelete(props.id)}
      onViewLogs={() => handlers?.onViewLogs(props.id)}
    />
  );
};

const nodeTypes = {
  collectionNode: CollectionNode,
  agentNode: AgentNodeWithCtx,
} as const;

export default function WorkspaceCanvas() {
  const reactFlowWrapperRef = useRef<HTMLDivElement>(null);
  const toast = useToast();
  const [layoutDirection, setLayoutDirection] = useState("LR");
  const { isOpen: isCollectionEditorOpen, onOpen: onCollectionEditorOpen, onClose: onCollectionEditorClose } = useDisclosure();
  const { isOpen: isAgentEditorOpen, onOpen: onAgentEditorOpen, onClose: onAgentEditorClose } = useDisclosure();
  // Custom hooks
  const { nodes, edges, loading, setNodes, setEdges, fetchDataAndLayout } = useWorkspaceData();
  const {
    modal,
    viewerModal,
    collectionEditorModal,
    agentEditorModal,
    createModal,
    relationModal,
    selectedNode,
    setSelectedNode,
    viewingNode,
    setViewingNode,
    editingCollection,
    setEditingCollection,
    editingAgent,
    setEditingAgent,
    contextMenu,
    setContextMenu,
    newCollectionData,
    setNewCollectionData,
    validationErrors,
    setValidationErrors,
    newConnection,
    setNewConnection,
    resetNewCollectionData,
  } = useWorkspaceModals();

  useEffect(() => {
    fetchDataAndLayout(layoutDirection);
  }, [layoutDirection, fetchDataAndLayout]);

  const onNodesChange = useCallback(
    (changes: NodeChange[]) => 
      setNodes((nds) => applyNodeChanges(changes, nds) as AppNode[]),
    [setNodes]
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
      relationModal.onOpen();
    },
    [relationModal.onOpen, setNewConnection]
  );

  const onNodeClick = useCallback(
    (event: React.MouseEvent, node: AppNode) => {
      setSelectedNode(node);
    },
    [setSelectedNode]
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
    [setContextMenu, setSelectedNode]
  );

  const onPaneClick = useCallback(() => {
    setContextMenu(null);
    setSelectedNode(null);
  }, [setContextMenu, setSelectedNode]);

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
    resetNewCollectionData();
    createModal.onOpen();
  };

  const handleContinueClick = () => {
    if (newCollectionData.displayName.trim() === "") {
      setValidationErrors({ displayName: "Display name is required." });
      return;
    }

    const newNode: CollectionNodeType = {
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
    setEditingCollection(newNode);
    collectionEditorModal.onOpen();
    createModal.onClose();
    resetNewCollectionData();
  };

  const handleUpdateNodeAttributes = useCallback(
    (nodeId: string, newAttributes: AttributeItem[]) => {
      setNodes((currentNodes) =>
        currentNodes.map((node) => {
          if (node.id === nodeId && node.type === 'collectionNode') {
            return {
              ...node,
              data: {
                ...node.data,
                attributes: newAttributes,
              },
            };
          }
          return node;
        })
      );
    },
    [setNodes]
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
          if (node.id === newConnection.source && node.type === 'collectionNode') {
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

      relationModal.onClose();
      setNewConnection(null);
    },
    [newConnection, relationModal.onClose, toast, setNodes, setEdges]
  );
  
  const handleViewDetails = (nodeToView: AppNode | null) => {
    if (nodeToView) {
      setViewingNode(nodeToView);
      viewerModal.onOpen();
    }
    setContextMenu(null);
  };

  const handleEdit = (nodeToEdit: AppNode | null) => {
    if (nodeToEdit) {
      if(nodeToEdit.type === 'collectionNode') {
        setEditingCollection(nodeToEdit as CollectionNodeType);
        collectionEditorModal.onOpen();
      } else if (nodeToEdit.type === 'agentNode') {
        setEditingAgent(nodeToEdit);
        agentEditorModal.onOpen();
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

  const handleDeleteAction = useCallback((node: AppNode | null) => {
    if (node) {
       setNodes(nds => nds.filter(n => n.id !== node.id));
       setEdges(eds => eds.filter(e => e.source !== node.id && e.target !== node.id));
       toast({ title: `${node.type === 'agentNode' ? 'Agent' : 'Collection'} deleted.`, status: "info", duration: 5000, isClosable: true });
     }
     setContextMenu(null);
   }, [setNodes, setEdges, toast]);
  
  // --- Action Handlers (wrapped in useCallback for performance) ---
  const handleViewLogsAction = useCallback((node: AppNode | null) => {
     if(node) {
         toast({
             title: `View Logs for "${node.id}"`,
             description: "This feature is not yet implemented.",
             status: "info",
             duration: 3000,
             isClosable: true,
         });
     }
   }, [toast]);

  const handleEditAction = useCallback((node: AppNode | null) => {
      if (node?.type === 'collectionNode') {
        setEditingCollection(node);
        onCollectionEditorOpen();
      } else if (node?.type === 'agentNode') {
        setEditingAgent(node);
        onAgentEditorOpen();
     }
     setContextMenu(null);
   }, [onCollectionEditorOpen, onAgentEditorOpen, setEditingCollection, setEditingAgent]);

  const nodeTypes = useMemo(() => ({
    collectionNode: CollectionNode,
    agentNode: (props: NodeProps<AgentNodeType>) => {
      const node = nodes.find(n => n.id === props.id);
      return (
        <AgentNode 
          data={props.data} 
          selected={props.selected} 
          onEdit={() => handleEditAction( node as AppNode)}
          onDelete={() => handleDeleteAction(node as AppNode)}
          onViewLogs={() => handleViewLogsAction(node as AppNode)}
        />
      );
    },
  }), [nodes, handleEditAction, handleDeleteAction, handleViewLogsAction]); 

  if (loading) return <Spinner />;
  
  const viewingNodeAttributes = viewingNode?.type === 'collectionNode' 
    ? viewingNode.data.attributes 
    : undefined;

  const sourceNode = newConnection?.source
    ? nodes.find(n => n.id === newConnection.source && n.type === 'collectionNode') as CollectionNodeType | undefined
    : undefined;

  const targetNode = newConnection?.target
    ? nodes.find(n => n.id === newConnection.target && n.type === 'collectionNode') as CollectionNodeType | undefined
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

      <Modal isOpen={modal.isOpen} onClose={modal.onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>
            Details: {
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
            <Button colorScheme="blue" onClick={modal.onClose}>
              Close
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      <Modal isOpen={createModal.isOpen} onClose={createModal.onClose}>
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
            <Button variant="ghost" mr={3} onClick={createModal.onClose}>
              Cancel
            </Button>
            <Button colorScheme="teal" onClick={handleContinueClick} isDisabled={!!validationErrors.displayName || !newCollectionData.displayName.trim()}>
              Continue
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
      
       <AttributeEditorSidebar
        isOpen={collectionEditorModal.isOpen}
        onClose={collectionEditorModal.onClose}
        node={editingCollection}
        onSave={handleUpdateNodeAttributes}
      />
      
      <RelationModal
        isOpen={relationModal.isOpen}
        onClose={() => {
          relationModal.onClose();
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