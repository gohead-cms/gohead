import React, { useState } from 'react';
import { Handle, Position } from '@xyflow/react';
import {
  Box,
  VStack,
  HStack,
  Text,
  Badge,
  Separator,
  Icon,
  IconButton,
} from '@chakra-ui/react';
import { 
  FaRobot,
  FaEdit,
  FaTrash,
  FaHistory,
  FaEllipsisV
} from 'react-icons/fa';
import { PiFunctionFill } from "react-icons/pi"; 
import { LobeHub, OpenAI, Ollama, Anthropic } from '@lobehub/icons';
import type { AgentNodeData } from '../../../shared/types';
import { MenuContent, MenuItem, MenuRoot, MenuTrigger } from '@/components/ui/menu';
import { Tooltip } from '@/components/ui/tooltip';


interface AgentNodeProps {
  data: AgentNodeData;
  selected?: boolean;
  // Add callbacks for the actions
  onEdit: (id: string) => void;
  onDelete: (id: string) => void;
  onViewLogs: (id: string) => void;
}

// FIX: Change to a named export by adding 'export' here
export const AgentNode: React.FC<AgentNodeProps> = ({ data, selected, onEdit, onDelete, onViewLogs }) => {
  const [showActions, setShowActions] = useState(false);
  
  const agentName = data.name || 'Unnamed Agent';
  const llmModel = data.schema?.llmConfig?.model || 'Not configured';
  const llmProvider = data.schema?.llmConfig?.provider || '';
  const functions = data.schema?.functions || [];
  const isActive = true; // Placeholder for agent status

  const getProviderIcon = (provider: string) => {
    switch (provider.toLowerCase()) {
      case 'openai':
        return OpenAI;
      case 'ollama':
        return Ollama;
      case 'anthropic':
        return Anthropic;
      default:
        return LobeHub;
    }
  };


  return (
    <>
      <Handle type="target" position={Position.Left} style={{ background: '#8a52ca' }}/>
      <Box
        bg="white"
        borderRadius="lg"
        border="2px solid"
        borderColor={selected ? '#8a52ca' : '#e2e8f0'}
        boxShadow={selected ? '0 0 0 3px rgba(138, 82, 202, 0.2)' : 'sm'}
        minW="280px"
        maxW="320px"
        overflow="hidden"
        transition="all 0.2s"
        onMouseEnter={() => setShowActions(true)}
        onMouseLeave={() => setShowActions(false)}
      >
        {/* Header */}
        <Box bg="linear-gradient(135deg, #8a52ca, #a855f7)" color="white" p={3}>
          <HStack gap={2} align="center">
            <Icon as={FaRobot} boxSize={5} />
            <VStack align="start" gap={0} flex={1}>
              <Text fontWeight="bold" fontSize="md" lineClamp={1}>{agentName}</Text>
              <Text fontSize="xs" opacity={0.9}>AI Agent</Text>
            </VStack>
            {/* Action Menu */}
            <MenuRoot>
              <MenuTrigger asChild>
                <IconButton
                  size="2xs"
                  variant="ghost"
                  color="white"
                  aria-label="Agent Actions"
                  opacity={showActions ? 1 : 0}
                  transition="opacity 0.2s"
                  _hover={{ bg: 'whiteAlpha.300' }}
                  _active={{ bg: 'whiteAlpha.400' }}
                >
                  <FaEllipsisV />
                </IconButton>
              </MenuTrigger>
              <MenuContent>
                <MenuItem value="edit" color="gray.700" onClick={() => onEdit(data.name)}><FaEdit />Edit Agent</MenuItem>
                <MenuItem value="logs" color="gray.700" onClick={() => onViewLogs(data.name)}><FaHistory />View Logs</MenuItem>
                <MenuItem value="delete" color="red.500" onClick={() => onDelete(data.name)}><FaTrash />Delete Agent</MenuItem>
              </MenuContent>
            </MenuRoot>
            <Tooltip content={isActive ? "Agent is Active" : "Agent is Inactive"}>
               <Box w="8px" h="8px" bg={isActive ? "green.300" : "gray.400"} borderRadius="full" />
            </Tooltip>
          </HStack>
        </Box>

        {/* Body */}
        <VStack gap={0} align="stretch">
          {/* LLM Section */}
          <Box p={3}>
            <HStack gap={2} align="center" mb={2}>
              <Text fontWeight="semibold" fontSize="sm" color="gray.700">LLM Config</Text>
            </HStack>
            <HStack>
                <Icon as={getProviderIcon(llmProvider)} color="purple.500" boxSize={5} />
              <Tooltip content={llmModel}>
                <Badge colorPalette="purple" size="sm" variant="subtle" maxW="150px" lineClamp={1}>{llmModel}</Badge>
              </Tooltip>
            </HStack>
          </Box>

          <Separator />

          {/* Functions Section */}
          <Box p={3} bg="gray.50">
            <HStack gap={2} align="center" mb={2}>
              <Icon as={PiFunctionFill} color="green.500" boxSize={4} />
              <Text fontWeight="semibold" fontSize="sm" color="gray.700">Functions</Text>
              <Badge colorPalette="green" size="sm" variant="outline">{functions.length}</Badge>
            </HStack>
            
            {functions.length > 0 ? (
              <VStack gap={1} align="stretch">
                {functions.slice(0, 3).map((func, index) => (
                  <Text key={index} fontSize="xs" color="gray.600" lineClamp={1}>- {func.name}</Text>
                ))}
                {functions.length > 3 && (
                  <Text fontSize="xs" color="gray.500" fontStyle="italic" cursor="pointer" _hover={{textDecor: 'underline'}}>
                    + {functions.length - 3} more...
                  </Text>
                )}
              </VStack>
            ) : (
              <Text fontSize="xs" color="gray.500" fontStyle="italic">No functions configured</Text>
            )}
          </Box>
        </VStack>
      </Box>

      <Handle type="source" position={Position.Right} style={{ background: '#8a52ca' }}/>
    </>
  );
};

// FIX: Remove the incorrect export statement from the end of the file

