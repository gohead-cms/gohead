import React from 'react';
import { EdgeProps, getSmoothStepPath, EdgeLabelRenderer, BaseEdge, Edge, MarkerType, getBezierPath } from '@xyflow/react';
import { Box, HStack, Icon, Badge } from '@chakra-ui/react';
import { FaPlusCircle, FaPencilAlt, FaTrashAlt, FaBolt, FaEye } from 'react-icons/fa';

/**
 * Defines the shape of the custom `data` property for a TriggerEdge.
 */
export interface TriggerEdgeData {
  events: string[];
  [key: string]: unknown;
}

/**
 * Defines the complete, strongly-typed Edge object for a 'triggerEdge'.
 */
export type TriggerEdgeType = Edge<TriggerEdgeData, 'triggerEdge'>;

// Map event names to specific icons for a better visual cue
const eventIcons: Record<string, React.ElementType> = {
    'item:created': FaPlusCircle,
    'item:updated': FaPencilAlt,
    'item:deleted': FaTrashAlt,
};

export function TriggerEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data,
}: EdgeProps<TriggerEdgeType>) {
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  const events = data?.events || [];

  return (
    <>
      <BaseEdge 
        path={edgePath}
        style={{ stroke: '#8a52ca', strokeDasharray: '5,5', strokeWidth: 1.5 }} 
      />
      <EdgeLabelRenderer>
        {/* Eye Icon near the source (Agent) */}
        <Box
          position="absolute"
          transform={`translate(-50%, -50%) translate(${sourceX}px,${sourceY}px)`}
          pointerEvents="all"
          // Offset the icon slightly from the node's edge
          style={{ transform: `translate(-50%, -50%) translate(${sourceX + 20}px, ${sourceY}px)` }}
          zIndex={1}
        >
          <Icon as={FaEye} color="gray.400" bg="white" borderRadius="full" p={0.5} boxSize={5} />
        </Box>

        {/* Event Label in the middle */}
        <Box
          position="absolute"
          transform={`translate(-50%, -50%) translate(${labelX}px,${labelY}px)`}
          pointerEvents="all"
          zIndex={1}
        >
          {events.length > 0 && (
            <HStack
              bg="white"
              boxShadow="md"
              borderRadius="full"
              border="1px solid"
              borderColor="purple.100"
              px={2}
              py={1}
              spacing={1.5}
            >
              {events.slice(0, 1).map((event: string) => {
                 const eventName = event.split(':')[1];
                 const IconComp = eventIcons[event] || FaBolt;
                 return (
                    <React.Fragment key={event}>
                      <Icon as={IconComp} color="purple.500" boxSize={3} />
                      <Badge colorScheme='purple' variant='subtle' size='sm' textTransform="lowercase">
                        on:{eventName}
                      </Badge>
                    </React.Fragment>
                 )
              })}
               {events.length > 1 && (
                   <Badge colorScheme='purple' variant='solid' size='sm' borderRadius='full'>
                      +{events.length - 1}
                  </Badge>
               )}
            </HStack>
          )}
        </Box>
      </EdgeLabelRenderer>
    </>
  );
}

