import React from 'react';
import { EdgeProps, getSmoothStepPath, EdgeLabelRenderer, BaseEdge, Edge } from '@xyflow/react';
import { Box, HStack, Icon, Badge } from '@chakra-ui/react';
import { FaPlusCircle, FaPencilAlt, FaTrashAlt, FaBolt } from 'react-icons/fa';

/**
 * Defines the shape of the custom `data` property for a TriggerEdge.
 */
export interface TriggerEdgeData {
  events: string[];
  [key: string]: unknown;
}

/**
 * Defines the complete, strongly-typed Edge object for a 'triggerEdge'.
 * You can import and use this type in your hooks or state to ensure
 * that the edges you create have the correct shape.
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
  const [edgePath, labelX, labelY] = getSmoothStepPath({
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
                      <Badge variant='subtle' size='md' textTransform="lowercase">
                        on:{eventName}
                      </Badge>
                    </React.Fragment>
                 )
              })}
               {events.length > 1 && (
                   <Badge variant='subtle' size='sd' borderRadius='full'>
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

