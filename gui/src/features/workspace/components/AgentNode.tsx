import React, { useState } from 'react';
import { Handle, Position } from '@xyflow/react';
import {
  Box,
  VStack,
  HStack,
  Text,
  Badge,
  Divider,
  Icon,
  Tooltip,
  IconButton,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
  useColorModeValue,
  Spinner,
  AlertDialog,
  AlertDialogBody,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogContent,
  AlertDialogOverlay,
  useDisclosure,
  Button,
} from '@chakra-ui/react';
import { 
  FaRobot, 
  FaBrain, 
  FaCog, 
  FaCode,
  FaMemory,
  FaChartLine,
  FaCalculator,
  FaSuperscript,
  FaDatabase,
  FaMicrochip,
  FaLightbulb,
  FaPlay,
  FaPause,
  FaStop,
  FaEdit,
  FaEye,
  FaEllipsisV,
  FaBug,
  FaHistory,
  FaExclamationTriangle,
  FaCheckCircle,
  FaClock,
} from 'react-icons/fa';
import { 
  RiFunctionLine 
} from 'react-icons/ri';
import type { AgentNodeData } from '../../../shared/types/agents';

interface AgentNodeProps {
  data: AgentNodeData;
  selected?: boolean;
}

// Mock agent status - in real app, this would come from your backend
type AgentStatus = 'idle' | 'running' | 'error' | 'paused';

interface AgentRuntime {
  status: AgentStatus;
  lastRun?: Date;
  runCount: number;
  errorCount: number;
  isLoading: boolean;
}

const AgentNode: React.FC<AgentNodeProps> = ({ data, selected }) => {
  // Mock runtime data - replace with real data from your backend
  const [runtime, setRuntime] = useState<AgentRuntime>({
    status: 'idle',
    lastRun: new Date(Date.now() - 300000), // 5 minutes ago
    runCount: 23,
    errorCount: 1,
    isLoading: false,
  });

  const { isOpen, onOpen, onClose } = useDisclosure();
  const cancelRef = React.useRef<HTMLButtonElement>(null);

  // Extract information from the agent data
  const agentName = data.name || 'Unnamed Agent';
  const llmModel = data.schema?.llmConfig?.model || 'Not configured';
  const llmProvider = data.schema?.llmConfig?.provider || '';
  const functions = data.schema?.functions || [];
  const trigger = data.schema?.trigger;

  // Status configuration
  const getStatusConfig = (status: AgentStatus) => {
    switch (status) {
      case 'running':
        return {
          color: 'green.400',
          bgColor: 'green.50',
          borderColor: 'green.200',
          label: 'Running',
          icon: FaPlay,
          pulse: true
        };
      case 'error':
        return {
          color: 'red.400',
          bgColor: 'red.50',
          borderColor: 'red.200',
          label: 'Error',
          icon: FaExclamationTriangle,
          pulse: false
        };
      case 'paused':
        return {
          color: 'orange.400',
          bgColor: 'orange.50',
          borderColor: 'orange.200',
          label: 'Paused',
          icon: FaPause,
          pulse: false
        };
      default:
        return {
          color: 'gray.400',
          bgColor: 'gray.50',
          borderColor: 'gray.200',
          label: 'Idle',
          icon: FaCheckCircle,
          pulse: false
        };
    }
  };

  const statusConfig = getStatusConfig(runtime.status);

  // Function to get memory icon based on type (placeholder for future use)
  const getMemoryIcon = () => {
    return FaMemory;
  };

  // Function to get function icon based on function type/name
  const getFunctionIcon = (functionName: string) => {
    if (!functionName) return FaCode;
    const name = functionName.toLowerCase();
    if (name.includes('math') || name.includes('calc')) return FaCalculator;
    if (name.includes('chart') || name.includes('graph')) return FaChartLine;
    if (name.includes('function') || name.includes('formula')) return FaSuperscript;
    return FaCode;
  };

  // Action handlers
  const handleRunAgent = () => {
    setRuntime(prev => ({ ...prev, isLoading: true, status: 'running' }));
    // Simulate API call
    setTimeout(() => {
      setRuntime(prev => ({
        ...prev,
        isLoading: false,
        status: 'idle',
        lastRun: new Date(),
        runCount: prev.runCount + 1
      }));
    }, 3000);
  };

  const handlePauseAgent = () => {
    setRuntime(prev => ({ ...prev, status: 'paused' }));
  };

  const handleStopAgent = () => {
    onOpen(); // Show confirmation dialog
  };

  const confirmStopAgent = () => {
    setRuntime(prev => ({ ...prev, status: 'idle', isLoading: false }));
    onClose();
  };

  // Format relative time
  const getRelativeTime = (date: Date) => {
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    
    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h ago`;
    const diffDays = Math.floor(diffHours / 24);
    return `${diffDays}d ago`;
  };

  const bgColor = useColorModeValue('white', 'gray.800');
  const borderColor = selected ? '#8a52ca' : statusConfig.borderColor;

  return (
    <>
      {/* Input Handle */}
      <Handle
        type="target"
        position={Position.Left}
        style={{
          background: '#8a52ca',
          width: 8,
          height: 8,
        }}
      />

      <Box
        bg={bgColor}
        borderRadius="lg"
        border="2px solid"
        borderColor={borderColor}
        boxShadow={selected ? '0 0 0 2px rgba(138, 82, 202, 0.2)' : 'sm'}
        minW="300px"
        maxW="340px"
        overflow="hidden"
        transition="all 0.2s"
        position="relative"
        _hover={{
          boxShadow: 'md',
          borderColor: '#8a52ca',
          '& .quick-actions': {
            opacity: 1,
            transform: 'translateY(0)',
          }
        }}
      >
        {/* Status Indicator */}
        <Box
          position="absolute"
          top={2}
          right={2}
          zIndex={2}
        >
          <Tooltip label={`Status: ${statusConfig.label}`}>
            <Box
              w={3}
              h={3}
              bg={statusConfig.color}
              borderRadius="50%"
              animation={statusConfig.pulse ? 'pulse 2s infinite' : 'none'}
              boxShadow="0 0 0 2px white"
            />
          </Tooltip>
        </Box>

        {/* Quick Actions (appears on hover) */}
        <Box
          className="quick-actions"
          position="absolute"
          top={2}
          left={2}
          zIndex={2}
          opacity={0}
          transform="translateY(-10px)"
          transition="all 0.2s"
        >
          <HStack spacing={1}>
            <Tooltip label="Run Agent">
              <IconButton
                size="xs"
                aria-label="Run Agent"
                icon={runtime.isLoading ? <Spinner size="xs" /> : <FaPlay />}
                colorScheme="green"
                variant="solid"
                onClick={handleRunAgent}
                isDisabled={runtime.isLoading || runtime.status === 'running'}
              />
            </Tooltip>
            
            {runtime.status === 'running' && (
              <Tooltip label="Stop Agent">
                <IconButton
                  size="xs"
                  aria-label="Stop Agent"
                  icon={<FaStop />}
                  colorScheme="red"
                  variant="solid"
                  onClick={handleStopAgent}
                />
              </Tooltip>
            )}
            
            <Menu>
              <Tooltip label="More Actions">
                <MenuButton
                  as={IconButton}
                  size="xs"
                  aria-label="More Actions"
                  icon={<FaEllipsisV />}
                  variant="solid"
                  colorScheme="gray"
                />
              </Tooltip>
              <MenuList fontSize="sm">
                <MenuItem icon={<FaEdit />}>Configure Agent</MenuItem>
                <MenuItem icon={<FaEye />}>View Details</MenuItem>
                <MenuItem icon={<FaBug />}>Debug Mode</MenuItem>
                <MenuItem icon={<FaHistory />}>View Logs</MenuItem>
                <Divider />
                <MenuItem icon={<FaPause />} onClick={handlePauseAgent}>
                  {runtime.status === 'paused' ? 'Resume' : 'Pause'} Agent
                </MenuItem>
              </MenuList>
            </Menu>
          </HStack>
        </Box>

        {/* Header */}
        <Box
          bg="linear-gradient(135deg, #8a52ca, #a855f7)"
          color="white"
          p={3}
          pt={6} // Extra padding for quick actions
        >
          <HStack spacing={2} align="center">
            <Icon as={FaRobot} boxSize={5} />
            <VStack align="start" spacing={0} flex={1}>
              <Text fontWeight="bold" fontSize="md" noOfLines={1}>
                {agentName}
              </Text>
              <Text fontSize="xs" opacity={0.9}>
                AI Agent
              </Text>
            </VStack>
            {trigger && (
              <Tooltip label={`Trigger: ${trigger.type}`}>
                <Icon as={FaLightbulb} boxSize={4} opacity={0.8} />
              </Tooltip>
            )}
          </HStack>

          {/* Runtime Stats */}
          <HStack spacing={4} mt={2} fontSize="xs" opacity={0.9}>
            <HStack spacing={1}>
              <Icon as={FaClock} boxSize={3} />
              <Text>{runtime.lastRun ? getRelativeTime(runtime.lastRun) : 'Never'}</Text>
            </HStack>
            <HStack spacing={1}>
              <Text>Runs: {runtime.runCount}</Text>
            </HStack>
            {runtime.errorCount > 0 && (
              <HStack spacing={1}>
                <Icon as={FaExclamationTriangle} boxSize={3} />
                <Text>{runtime.errorCount}</Text>
              </HStack>
            )}
          </HStack>
        </Box>

        {/* Body */}
        <VStack spacing={0} align="stretch">
          {/* Memory Section */}
          <Box p={3} bg="gray.50">
            <HStack spacing={2} align="center" mb={2}>
              <Icon as={getMemoryIcon()} color="blue.500" boxSize={4} />
              <Text fontWeight="semibold" fontSize="sm" color="gray.700">
                Memory
              </Text>
            </HStack>
            <HStack spacing={2} align="center">
              <Badge
                colorScheme="gray"
                size="sm"
                variant="subtle"
              >
                Not configured
              </Badge>
            </HStack>
          </Box>

          <Divider />

          {/* LLM Section */}
          <Box p={3} bg="white">
            <HStack spacing={2} align="center" mb={2}>
              <Icon as={FaMicrochip} color="purple.500" boxSize={4} />
              <Text fontWeight="semibold" fontSize="sm" color="gray.700">
                LLM Config
              </Text>
            </HStack>
            <VStack spacing={1} align="start">
              {llmProvider && (
                <Badge colorScheme="purple" size="sm" variant="outline">
                  {llmProvider}
                </Badge>
              )}
              <Tooltip label={llmModel}>
                <Badge colorScheme="purple" size="sm" variant="subtle">
                  <Text noOfLines={1} maxW="150px">
                    {llmModel}
                  </Text>
                </Badge>
              </Tooltip>
            </VStack>
          </Box>

          <Divider />

          {/* Functions Section */}
          <Box p={3} bg="gray.50">
            <HStack spacing={2} align="center" mb={2}>
              <Icon as={RiFunctionLine} color="green.500" boxSize={4} />
              <Text fontWeight="semibold" fontSize="sm" color="gray.700">
                Functions
              </Text>
              <Badge colorScheme="green" size="sm" variant="outline">
                {functions.length}
              </Badge>
            </HStack>
            
            {functions.length > 0 ? (
              <VStack spacing={1} align="stretch">
                {functions.slice(0, 3).map((func, index) => {
                  const funcObj = func as any;
                  const funcName = funcObj?.name || 
                                 funcObj?.functionName || 
                                 funcObj?.id || 
                                 funcObj?.title || 
                                 `Function ${index + 1}`;
                  
                  return (
                    <HStack key={index} spacing={2} align="center">
                      <Icon 
                        as={getFunctionIcon(funcName)} 
                        color="green.600" 
                        boxSize={3} 
                      />
                      <Text fontSize="xs" color="gray.600" noOfLines={1} flex={1}>
                        {funcName}
                      </Text>
                    </HStack>
                  );
                })}
                {functions.length > 3 && (
                  <Text fontSize="xs" color="gray.500" fontStyle="italic">
                    +{functions.length - 3} more...
                  </Text>
                )}
              </VStack>
            ) : (
              <Text fontSize="xs" color="gray.500" fontStyle="italic">
                No functions configured
              </Text>
            )}
          </Box>

          {/* Loading Overlay */}
          {runtime.isLoading && (
            <Box
              position="absolute"
              top={0}
              left={0}
              right={0}
              bottom={0}
              bg="rgba(255, 255, 255, 0.8)"
              display="flex"
              alignItems="center"
              justifyContent="center"
              zIndex={10}
              borderRadius="lg"
            >
              <VStack spacing={2}>
                <Spinner size="lg" color="purple.500" />
                <Text fontSize="sm" color="purple.600" fontWeight="semibold">
                  Executing...
                </Text>
              </VStack>
            </Box>
          )}
        </VStack>
      </Box>

      {/* Output Handle */}
      <Handle
        type="source"
        position={Position.Right}
        style={{
          background: '#8a52ca',
          width: 8,
          height: 8,
        }}
      />

      {/* Stop Confirmation Dialog */}
      <AlertDialog
        isOpen={isOpen}
        leastDestructiveRef={cancelRef}
        onClose={onClose}
      >
        <AlertDialogOverlay>
          <AlertDialogContent>
            <AlertDialogHeader fontSize="lg" fontWeight="bold">
              Stop Agent Execution
            </AlertDialogHeader>

            <AlertDialogBody>
              Are you sure you want to stop the agent "{agentName}"? 
              This will terminate any ongoing operations.
            </AlertDialogBody>

            <AlertDialogFooter>
              <Button ref={cancelRef} onClick={onClose}>
                Cancel
              </Button>
              <Button colorScheme="red" onClick={confirmStopAgent} ml={3}>
                Stop Agent
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogOverlay>
      </AlertDialog>

      <style>
        {`
          @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
          }
        `}
      </style>
    </>
  );
};

export default AgentNode;