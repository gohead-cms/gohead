import { getBezierPath, EdgeLabelRenderer, BaseEdge, EdgeProps, Edge, MarkerType } from "@xyflow/react";

// The data interface remains the same
export interface CollectionEdgeData {
  label: string;
  relationType: string;
  attributes: string[];
  [key: string]: unknown;
}

// The custom edge type also remains the same
export type CustomCollectionEdge = Edge<CollectionEdgeData, 'collection'>;

export default function CollectionEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data,
}: EdgeProps<CustomCollectionEdge>) {
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  // Function to create the custom label based on the relation type
  const getRelationLabel = (relationType: string | undefined): string => {
    switch (relationType) {
      case 'oneToOne':
        return 'has one';
      case 'oneToMany':
        return 'has one or many';
      case 'manyToMany':
        return 'is related to many';
      default:
        return relationType || ''; // Fallback to the original type or an empty string
    }
  };
  
  const relationLabel = getRelationLabel(data?.relationType);

  return (
    <>
      <BaseEdge
        id={id}
        path={edgePath}
        markerEnd={MarkerType.ArrowClosed}
      />
      <EdgeLabelRenderer>
        <div
          style={{
            position: 'absolute',
            transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
            padding: "2px 8px",
            borderRadius: 6,
            background: "#52b4ca",
            color: "#fff",
            fontSize: 12,
            fontWeight: 500,
            whiteSpace: "nowrap",
          }}
          className="nodrag nopan"
        >
          {relationLabel}
        </div>
      </EdgeLabelRenderer>
    </>
  );
}