import { useState } from "react";
import { useDisclosure } from "@chakra-ui/react";
import { Connection } from "@xyflow/react";
import type { 
  AppNode, 
  CollectionNode, 
  AgentNode, 
  NewCollectionData, 
  ValidationErrors,
  ContextMenuState 
} from "../../../shared/types/workspace";

export function useWorkspaceModals() {
  // Modal states
  const { isOpen, onOpen, onClose } = useDisclosure();
  const { isOpen: isViewerOpen, onOpen: onViewerOpen, onClose: onViewerClose } = useDisclosure();
  const { isOpen: isCollectionEditorOpen, onOpen: onCollectionEditorOpen, onClose: onCollectionEditorClose } = useDisclosure();
  const { isOpen: isAgentEditorOpen, onOpen: onAgentEditorOpen, onClose: onAgentEditorClose } = useDisclosure();
  const { isOpen: isCreateModalOpen, onOpen: onCreateModalOpen, onClose: onCreateModalClose } = useDisclosure();
  const { isOpen: isRelationModalOpen, onOpen: onRelationModalOpen, onClose: onRelationModalClose } = useDisclosure();

  // Node states
  const [selectedNode, setSelectedNode] = useState<AppNode | null>(null);
  const [viewingNode, setViewingNode] = useState<AppNode | null>(null);
  const [editingCollection, setEditingCollection] = useState<CollectionNode | null>(null);
  const [editingAgent, setEditingAgent] = useState<AgentNode | null>(null);
  const [contextMenu, setContextMenu] = useState<ContextMenuState>(null);

  // Form states
  const [newCollectionData, setNewCollectionData] = useState<NewCollectionData>({ 
    displayName: "", 
    singularId: "", 
    pluralId: "" 
  });
  const [validationErrors, setValidationErrors] = useState<ValidationErrors>({ displayName: "" });
  const [newConnection, setNewConnection] = useState<Connection | null>(null);

  const resetNewCollectionData = () => {
    setNewCollectionData({ displayName: "", singularId: "", pluralId: "" });
    setValidationErrors({ displayName: "" });
  };

  const handleCollectionEditorClose = () => {
    setEditingCollection(null);
    onCollectionEditorClose();
  };

  return {
    // Modal controls
    modal: { isOpen, onOpen, onClose },
    viewerModal: { isOpen: isViewerOpen, onOpen: onViewerOpen, onClose: onViewerClose },
    collectionEditorModal: { 
      isOpen: isCollectionEditorOpen, 
      onOpen: onCollectionEditorOpen, 
      onClose: handleCollectionEditorClose 
    },
    agentEditorModal: { isOpen: isAgentEditorOpen, onOpen: onAgentEditorOpen, onClose: onAgentEditorClose },
    createModal: { isOpen: isCreateModalOpen, onOpen: onCreateModalOpen, onClose: onCreateModalClose },
    relationModal: { isOpen: isRelationModalOpen, onOpen: onRelationModalOpen, onClose: onRelationModalClose },

    // Node states
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

    // Form states
    newCollectionData,
    setNewCollectionData,
    validationErrors,
    setValidationErrors,
    newConnection,
    setNewConnection,
    resetNewCollectionData,
  };
}